package core

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Conf struct {
	SAS ServiceAccounts
}

type ServiceAccount struct {
	Name  string
	Path  string
	Email string `json:"client_email"`
}
type ServiceAccounts []*ServiceAccount

func LoadServiceAccount(dir string) (ServiceAccounts, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var sas ServiceAccounts
	for _, v := range files {
		if strings.HasSuffix(v.Name(), ".json") {
			b, err := ioutil.ReadFile(filepath.Join(dir, v.Name()))
			if err != nil {
				return nil, err
			}
			var sa ServiceAccount
			err = json.Unmarshal(b, &sa)
			if err != nil {
				return nil, err
			}
			sa.Name = strings.Split(sa.Email, "@")[0]
			sa.Path = filepath.Join(dir, v.Name())
			sas = append(sas, &sa)
		}
	}
	re := regexp.MustCompile(`(\d+)`)
	sort.SliceStable(sas, func(i, j int) bool {
		if re.MatchString(sas[i].Name) && re.MatchString(sas[j].Name) {
			a := re.FindStringSubmatch(sas[i].Name)[1]
			ai, _ := strconv.Atoi(a)
			b := re.FindStringSubmatch(sas[j].Name)[1]
			bj, _ := strconv.Atoi(b)
			return ai < bj
		}
		return false
	})
	return sas, nil
}
