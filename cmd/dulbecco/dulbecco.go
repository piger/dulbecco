package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/piger/dulbecco"
)

var (
	configFile = flag.String("config", "./config.json", "Path to the configuration file")
)

func main() {
	flag.Parse()

	config := dulbecco.ReadConfig(*configFile)
	// fmt.Printf("%+v\n", config)
	var connections []dulbecco.Connection

	for _, server := range config.Servers {
		fmt.Printf("server config: %+v\n", server)

		conn := dulbecco.NewConnection(&server, config)
		if err := conn.Connect(); err != nil {
			fmt.Printf("error connecting to server: %s\n", err)
			return
		}
		connections = append(connections, *conn)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		fmt.Printf("C-c received, exiting (%v)", sig)
		return
	}

	fmt.Printf("ciaone\n")
}
