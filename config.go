package dulbecco

import (
	"encoding/json"
	"errors"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"math/rand"
	"path/filepath"
)

var defaultReplies []string

type Configuration struct {
	Servers []ServerConfiguration `toml:"server"`
	Plugins []PluginConfiguration `toml:"plugin"`
	Replies []string              `toml:"replies"`
	Hipchat HipchatConfiguration  `toml:"hipchat"`
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

type HipchatConfiguration struct {
	Address string
}

func ReadConfig(filename string) (*Configuration, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config *Configuration
	if filepath.Ext(filename) == ".json" {
		config, err = readJsonConfig(data)
	} else {
		config, err = readTomlConfig(data)
	}
	if err != nil {
		return nil, err
	}

	if len(config.Servers) < 1 {
		return nil, errors.New("no servers defined")
	}

	defaultReplies = append(defaultReplies, config.Replies...)

	return config, nil
}

func readJsonConfig(data []byte) (*Configuration, error) {
	var config Configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func readTomlConfig(data []byte) (*Configuration, error) {
	var config Configuration
	if _, err := toml.Decode(string(data), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func GetRandomReply() string {
	if len(defaultReplies) > 0 {
		return defaultReplies[rand.Intn(len(defaultReplies))]
	}
	return "DEMENZA MI COLSE"
}
