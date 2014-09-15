package dulbecco

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type ConfigurationType struct {
	Servers []ServerType
	Plugins []PluginType
}

type ServerType struct {
	Name         string
	Address      string
	Ssl          bool
	Channels     []string
	Nickname     string
	Altnicknames []string
	Username     string
	Realname     string
	Password     string
}

type PluginType struct {
	Name    string
	Command string
	Trigger string
}

func ReadConfig(filename string) *ConfigurationType {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("ERROR Cannot read configuration file: ", err)
	}

	var config ConfigurationType
	if err := json.Unmarshal(file, &config); err != nil {
		log.Fatal("ERROR Cannot parse configuration file: ", err)
	}

	return &config
}
