package dulbecco

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type jsonobject struct {
	Configuration ConfigurationType
}

type ConfigurationType struct {
	Nickname     string
	Altnicknames []string
	Username     string
	Realname     string
	Servers      []ServerType
	Plugins      []PluginType
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

	var jsontype jsonobject
	if err := json.Unmarshal(file, &jsontype); err != nil {
		log.Fatal("ERROR Cannot parse configuration file: ", err)
	}

	return &jsontype.Configuration
}
