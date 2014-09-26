package main

import (
	"flag"
	"github.com/piger/dulbecco"
	"github.com/piger/dulbecco/markov"
	"log"
	"os"
	"os/signal"
	"sync"
)

var (
	configFile = flag.String("config", "./config.json", "Path to the configuration file")
	markovDb   = flag.String("mdb", "./markov-db", "Path to the directory containing the Markov DB")
	importDb   = flag.Bool("import", false, "Enable import mode")
	importFile = flag.String("train", "", "Train with a IRC log file")
)

const markovOrder = 2

func main() {
	flag.Parse()

	if *importDb {
		markov.ReadStdin(*markovDb, markovOrder)
		return
	} else if *importFile != "" {
		if err := markov.ReadFile(*markovDb, *importFile, markovOrder); err != nil {
			log.Fatal(err)
		}
		return
	}

	config := dulbecco.ReadConfig(*configFile)
	if len(config.Servers) < 1 {
		log.Fatal("Error in configuration: no servers defined")
	}

	mdb, err := markov.NewMarkovDB(2, *markovDb)
	if err != nil {
		log.Fatal(err)
	}

	var connections []*dulbecco.Connection
	var wg sync.WaitGroup

	for _, server := range config.Servers {
		log.Printf("Connecting to: %s\n", server.Address)

		wg.Add(1)
		conn := dulbecco.NewConnection(server, config, mdb)
		connections = append(connections, conn)
		go func(conn *dulbecco.Connection) {
			conn.MainLoop()
			wg.Done()
		}(conn)
	}

	cExit := make(chan bool)
	go func() {
		wg.Wait()
		cExit <- true
	}()

	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)
	for {
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
}
