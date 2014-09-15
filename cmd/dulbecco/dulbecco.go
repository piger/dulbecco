package main

import (
	"flag"
	"github.com/piger/dulbecco"
	"log"
	"os"
	"os/signal"
	"sync"
)

var (
	configFile = flag.String("config", "./config.json", "Path to the configuration file")
)

func main() {
	flag.Parse()

	config := dulbecco.ReadConfig(*configFile)
	var connections []*dulbecco.Connection
	var wg sync.WaitGroup

	for _, server := range config.Servers {
		log.Printf("Connecting to: %s\n", server.Address)

		wg.Add(1)
		conn := dulbecco.NewConnection(&server, config)
		connections = append(connections, conn)
		go func() {
			conn.MainLoop()
			wg.Done()
		}()
	}

	cExit := make(chan bool)
	go func() {
		wg.Wait()
		cExit <- true
	}()

	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)
	select {
	case <-csig:
		log.Printf("ctrl-c received\n")
		for _, conn := range connections {
			conn.Shutdown()
		}
	case <-cExit:
		return
	}
}
