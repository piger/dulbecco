package dulbecco

import (
	"log"
	"encoding/json"
	"io/ioutil"
)

type jsonobject struct {
	Configuration ConfigurationType
}

type ConfigurationType struct {
	Nickname string
	Altnicknames []string
	Ident string
	Realname string
	Servers []ServerType
}

type ServerType struct {
	Name string
	Address string
	Port int
	Ssl bool
	Nickname string
	Altnicknames []string
	Ident string
	Realname string
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
