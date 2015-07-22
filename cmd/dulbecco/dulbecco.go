package main

import (
	"flag"
	"github.com/piger/dulbecco"
	"github.com/piger/dulbecco/markov"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	config, err := dulbecco.ReadConfig(*configFile)
	if err != nil {
		log.Fatal("Error with configuration file: ", err)
	}

	mdb, err := markov.NewMarkovDB(2, *markovDb)
	if err != nil {
		log.Fatal(err)
	}

	// start Hipchat handler
	if config.Hipchat.Address != "" {
		go func() {
			if err := dulbecco.HipchatHandler(config.Hipchat.Address, mdb); err != nil {
				log.Printf("Hipchat handler error: %s\n", err)
			}
		}()
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
	signal.Notify(csig, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case sig := <-csig:
			log.Printf("%v received\n", sig)
			for _, conn := range connections {
				conn.Shutdown()
			}
			signal.Stop(csig)
		case <-cExit:
			return
		}
	}
}
