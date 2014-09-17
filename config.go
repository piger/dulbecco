package dulbecco

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Configuration struct {
	Servers []ServerConfiguration
	Plugins []PluginConfiguration
}

type ServerConfiguration struct {
	Name         string
	Address      string
	Ssl          bool
	Channels     []string
	Nickname     string
	Altnicknames []string
	Username     string
	Realname     string
	Password     string
	Nickserv     string
}

type PluginConfiguration struct {
	Name    string
	Command string
	Trigger string
}

func ReadConfig(filename string) *Configuration {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("ERROR Cannot read configuration file: ", err)
	}

	var config Configuration
	if err := json.Unmarshal(file, &config); err != nil {
		log.Fatal("ERROR Cannot parse configuration file: ", err)
	}

	return &config
}
