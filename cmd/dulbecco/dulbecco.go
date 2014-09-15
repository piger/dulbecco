package main

import (
	"flag"
	"fmt"
	"github.com/piger/dulbecco"
	"log"
)

var (
	configFile = flag.String("config", "./config.json", "Path to the configuration file")
)

func main() {
	flag.Parse()

	config := dulbecco.ReadConfig(*configFile)
	var connections []dulbecco.Connection
	quit := make(chan bool)

	for _, server := range config.Servers {
		log.Printf("Connecting to: %s\n", server.Address)

		conn := dulbecco.NewConnection(&server, config, quit)
		if err := conn.Connect(); err != nil {
			fmt.Printf("error connecting to server: %s\n", err)
			return
		}
		connections = append(connections, *conn)
	}

	for i := 0; i < len(config.Servers); i++ {
		<-quit
	}
}
