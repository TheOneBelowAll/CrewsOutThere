package encryption

import (
	"io/ioutil"
	"log"
	"strings"
)

var config map[string]string

// Extract key:value pairs from cot.conf and store them in a map in a way that can be accessed from other places in crewsOutThere
func InitConf() {
	config = make(map[string]string)
	// Top ReadFile is for local machines, bottom one is for deployment machine
	// rawData, err := ioutil.ReadFile("/etc/cot.conf")
	rawData, err := ioutil.ReadFile("./../encryption/cot.conf")
	if err != nil {
		log.Fatal(err)
	}
	data := string(rawData)
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		separatorIndex := strings.Index(string(line), ":")

		if separatorIndex >= 0 {
			key := line[:separatorIndex]
			val := line[separatorIndex+1 : len(line)]
			if val[len(val)-1] == '\r' {
				val = val[:len(val)-1]
			}
			config[key] = val
		}
	}
}

func GetDBPasswords() (string, string) {
	return config["db_user_pw"], config["db_admin_pw"]
}

func GetDBIp() string {
	return config["db_ip"]
}

func GetListenerIp() string {
	return config["listener_ip"]
}

func GetPhone() string {
	return config["flowrouteNumber"]
}

func GetURL() string {
	return config["flowrouteURL"]
}
