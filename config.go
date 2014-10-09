package dulbecco

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
)

var defaultReplies []string

type Configuration struct {
	Servers []ServerConfiguration
	Plugins []PluginConfiguration
	Replies []string
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

	if len(config.Replies) > 0 {
		defaultReplies = config.Replies
	}

	return &config
}

func GetRandomReply() string {
	if len(defaultReplies) > 0 {
		return defaultReplies[rand.Intn(len(defaultReplies))]
	}
	return "DEMENZA MI COLSE"
}
