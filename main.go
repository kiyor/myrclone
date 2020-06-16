package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/kiyor/myrclone/core"
)

var (
	aviliableAccountlist                                                                                 []string
	reTransfer                                                                                           = regexp.MustCompile(`Transferred:\s+(\d+) / (\d+), (\d+)%`)
	maxCount, startIndex, exitIfDownloadQuotaExceeded, downloadQuotaExceeded, transfers, currentId, port int
	speedLessThan                                                                                        int64
	speed                                                                                                float64
	t1                                                                                                   time.Time = time.Now()
	magicCopy                                                                                            bool
	cuser, _                                                                                             = user.Current()
	home                                                                                                 = cuser.HomeDir
	PROC_FILE                                                                                            string
	totalBytes                                                                                           uint64
	totalTransfer, count                                                                                 int

	statTransfers int
	statBytes     uint64
)

const START_ID_FILE = ".config/rclone/myrcloneid"
const DriveServerSideAcrossConfigs = "--drive-server-side-across-configs"

func init() {
	flag.IntVar(&maxCount, "c", 5, "max count of no transfer")
	flag.IntVar(&port, "p", 7788, "listen port")
	// 	flag.IntVar(&startIndex, "s", 0, "start index")
	flag.IntVar(&transfers, "x", 10, "concurrent transfers")
	flag.Int64Var(&speedLessThan, "sp", 10000000, "speed less then")
	flag.IntVar(&exitIfDownloadQuotaExceeded, "eqe", 0, "exitIfDownloadQuotaExceeded")
	flag.BoolVar(&magicCopy, "mc", false, "magic copy (drive-server-side-across-configs)")
}

func loadStartId() {
	b, err := ioutil.ReadFile(filepath.Join(home, START_ID_FILE))
	if err == nil {
		s := strings.TrimRight(string(b), "\n")
		startIndex, _ = strconv.Atoi(s)
	}
}
func saveStartId() {
	err := ioutil.WriteFile(filepath.Join(home, START_ID_FILE), []byte(fmt.Sprint(currentId)), 0644)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()
	sas, err := core.LoadServiceAccount(filepath.Join(home, ".config/rclone"))
	for _, v := range sas {
		aviliableAccountlist = append(aviliableAccountlist, v.Name)
	}
	for {
		PROC_FILE = fmt.Sprint("/tmp/myrclone_%d", port)
		if _, err := os.Stat(PROC_FILE); err == nil {
			log.Printf("myrclone is running with port %d\n", port)
			port += 1
		} else {
			break
		}
	}
	_, err = os.Create(PROC_FILE)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	loadStartId()
	log.Println("aviliableAccount", len(aviliableAccountlist)-startIndex, "start:", startIndex)
	done := make(chan os.Signal, 1)
	finish := make(chan bool)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	exceed := make(chan struct{})

	go wapper(startIndex, flag.Args(), finish, exceed)
	go dogger(exceed)
	go dashboard()

	select {
	case <-done:
	case <-finish:
	}
	saveStartId()
	log.Println(time.Since(t1))
	os.Remove(PROC_FILE)
}

func reader(rd io.Reader, f *os.File, id int, account string, res chan error) {
	reader := bufio.NewReader(rd)
	prefix := "\t"
	var r error
	for {
		l, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Println(err)
				r = err
				break
			} else {
				r = nil
				break
			}
		}
		if strings.Contains(l, "downloadQuotaExceeded") {
			downloadQuotaExceeded += 1
		} else if strings.Contains(l, "NOTICE") {
		} else {
			f.WriteString(prefix + " " + l)
		}
		if exitIfDownloadQuotaExceeded > 0 && downloadQuotaExceeded > exitIfDownloadQuotaExceeded {
			r = fmt.Errorf("downloadQuotaExceeded %d", downloadQuotaExceeded)
			log.Println(time.Since(t1))
			os.Exit(1)
		}
		if strings.Contains(l, "userRateLimitExceeded") {
			r = fmt.Errorf(prefix + "userRateLimitExceeded")
			break
		}
	}
	res <- r
}

func dogger(exceed chan struct{}) {
	cmd := fmt.Sprintf("rclone rc --rc-addr=localhost:%d core/stats", port)
	tick := time.Tick(10 * time.Second)
	var lastBytes, diffBytes uint64
	var lastChecks, diffChecks int
	for {
		select {
		case <-tick:
			saveStartId()

			c := exec.Command("/bin/sh", "-c", cmd)
			var b bytes.Buffer
			c.Stdout = &b
			c.Run()
			var s Stats
			err := json.Unmarshal(b.Bytes(), &s)
			if err == nil {
				var sp float64
				for _, v := range s.Transferring {
					sp += v.Speed
				}
				speed = sp
				if s.Bytes > 780000000000 {
					log.Println("exceed limit by bytes")
					totalBytes += s.Bytes
					totalTransfer += s.Transfers
					exceed <- struct{}{}
				}
				statTransfers = s.Transfers
				statBytes = s.Bytes
				diffBytes = statBytes - lastBytes
				diffChecks = s.Checks - lastChecks
				lastBytes = statBytes
				lastChecks = s.Checks
			}
			if diffBytes > 0 || speed > 0 || diffChecks > 0 {
				count = 0
			} else {
				count += 1
			}
			if (count > maxCount && speed < float64(speedLessThan) && !magicCopy) || (magicCopy && count > maxCount) {
				log.Println("exceed limit by count")
				count = 0
				totalBytes += s.Bytes
				totalTransfer += s.Transfers
				exceed <- struct{}{}
			}
		}
	}
}

func dashboard() {
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-tick:
			log.Printf("id:%v account:%v transfer:%v count:%v size:%v/%v speed:%v downloadQuotaExceeded:%v\n", currentId, aviliableAccountlist[currentId], statTransfers, count, humanize.IBytes(statBytes), humanize.IBytes(totalBytes+statBytes), humanize.IBytes(uint64(speed)), downloadQuotaExceeded)
		}
	}
}

var replacer = strings.NewReplacer("【", "[", "】", "]", "（", "(", "）", ")", "“", "[", "”", "]")

func wapper(id int, args []string, finish chan bool, exceed chan struct{}) {
	if len(aviliableAccountlist) == id {
		log.Println("out of service account, try start from 0")
		id = 0
	}
	currentId = id
	cmd := fmt.Sprintf("rclone --stats 10s -v --drive-pacer-min-sleep 10ms --retries=1 --transfers=%d --checkers=20 --rc --rc-addr=localhost:%d --rc-serve ", transfers, port)
	for _, v := range args {
		if strings.Contains(v, "=") {
			v = replacer.Replace(v)
			v = strings.Replace(v, "=", aviliableAccountlist[id], 1)
		}
		if strings.Contains(v, ":") {
			v = strings.Split(v, ":")[0] + ":'" + strings.Split(v, ":")[1] + "'"
		}
		cmd += v + " "
	}
	if magicCopy {
		cmd += DriveServerSideAcrossConfigs
	}
	fmt.Println(cmd)

	c := exec.Command("/bin/sh", "-c", cmd)

	// 	po, err := c.StdoutPipe()
	// 	if err != nil {
	// 		log.Println(err)
	// 	}
	pe, err := c.StderrPipe()
	if err != nil {
		log.Println(err)
	}
	var kill bool
	// 	ro := make(chan error)
	re := make(chan error)

	// 	go reader(po, os.Stdout, "1\t", ro)
	go reader(pe, os.Stderr, id, aviliableAccountlist[id], re)
	go func() {
		select {
		case <-exceed:
			kill = true
			err := c.Process.Kill()
			if err != nil {
				log.Println(err)
			}
		case erre := <-re:
			if erre != nil {
				log.Println(erre)
				kill = true
				err := c.Process.Kill()
				if err != nil {
					log.Println(err)
				}
			}
		}
	}()
	c.Start()
	// 	erro := <-ro
	// 	if erro != nil {
	// 		log.Println(erro)
	// 	}
	c.Wait()
	// 	po.Close()
	pe.Close()
	time.Sleep(2 * time.Second)
	if kill {
		wapper(id+1, args, finish, exceed)
	}
	finish <- true
}
