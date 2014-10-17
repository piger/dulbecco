package dulbecco

import (
	"encoding/json"
	"errors"
	"io/ioutil"
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

func ReadConfig(filename string) (*Configuration, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Configuration
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, err
	} else if len(config.Servers) < 1 {
		return nil, errors.New("no servers defined")
	}

	copy(defaultReplies, config.Replies)

	return &config, nil
}

func GetRandomReply() string {
	if len(defaultReplies) > 0 {
		return defaultReplies[rand.Intn(len(defaultReplies))]
	}
	return "DEMENZA MI COLSE"
}
