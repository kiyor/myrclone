package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"text/template"

	"github.com/kiyor/myrclone/core"
)

var (
	cuser, _ = user.Current()
	home     = cuser.HomeDir
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	sas, err := core.LoadServiceAccount(filepath.Join(home, ".config/rclone"))
	if err != nil {
		log.Println(err)
	}
	for _, v := range sas {
		log.Println("found sa", v.Name)
	}
	c := core.Conf{
		SAS: sas,
	}
	b, err := ioutil.ReadFile(filepath.Join(home, ".config/rclone/tmpl"))
	if err != nil {
		log.Println(err)
	}
	tmpl, err := template.New("conf").Parse(string(b))
	if err != nil {
		log.Println(err)
	}
	path := filepath.Join(home, ".config/rclone/rclone.conf")
	f, err := os.Create(path)
	if err != nil {
		log.Println(err)
	}
	err = tmpl.Execute(f, c)
	if err != nil {
		log.Println(err)
	}
	log.Println("success write to", path)
}
