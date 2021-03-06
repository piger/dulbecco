package dulbecco

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/piger/dulbecco/markov"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var (
	pingFrequency = 3 * time.Minute
)

// the number of *Loop() methods on Connection; it's used for synchronization
// and must be updated accordingly.
const (
	numLoops = 4

	NickservName = "nickserv"

	// IRC defines a maximum "line" length of 512 characters, including \r\n,
	// the command and all parameters. On Azzurra the limit seems to be lower...
	MaximumCommandLength = 460

	SleepBetweenReconnects = time.Minute * 5
)

// A connection to the IRC server, also the main data structure of the IRC bot.
type Connection struct {
	config ServerConfiguration

	// current nickname
	nickname string

	// enable anti-flood protection
	floodProtection bool
	// anti-flood internal counters
	badness  time.Duration
	lastSent time.Time

	// IO
	sock         net.Conn
	io           *bufio.ReadWriter
	out          chan string
	tryReconnect bool

	// Internal channels
	inerr  chan error
	outerr chan bool
	wg     sync.WaitGroup

	// markov database
	mdb *markov.MarkovDB

	// callbacks
	events CallbackMap
}

func NewConnection(config ServerConfiguration, botConfig *Configuration, mdb *markov.MarkovDB) *Connection {
	conn := &Connection{
		config:          config,
		nickname:        config.Nickname,
		floodProtection: true,
		lastSent:        time.Now(),
		out:             make(chan string, 32),
		inerr:           make(chan error, numLoops),
		outerr:          make(chan bool, numLoops),
		mdb:             mdb,
		tryReconnect:    true,
		events:          make(CallbackMap),
	}

	// setup internal callbacks
	conn.SetupCallbacks(botConfig.Plugins)

	return conn
}

func (c *Connection) MainLoop() {
	for {
		if err := c.Connect(); err != nil {
			log.Print("Connection error: ", err)
		}
		if !c.tryReconnect {
			return
		}

		c.reinit()
		log.Printf("Sleeping %v before attempting a reconnection", SleepBetweenReconnects)
		time.Sleep(SleepBetweenReconnects)
	}
}

func (c *Connection) reinit() {
	close(c.out)
	close(c.inerr)
	close(c.outerr)
	c.out = make(chan string, 32)
	c.inerr = make(chan error, numLoops)
	c.outerr = make(chan bool, numLoops)
}

func readTLSCertificate(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

// Connect to the server, launch all internal goroutines.
func (c *Connection) Connect() (err error) {
	if c.config.Ssl {
		tlsConfig := &tls.Config{
			ServerName: c.config.GetHostname(),
		}

		if c.config.SslInsecure {
			log.Print("Using insecure TLS for ", c.config.Name)
			tlsConfig.InsecureSkipVerify = true
		}

		if c.config.SslCertificate != "" {
			roots := x509.NewCertPool()
			tlsCert, err := readTLSCertificate(c.config.SslCertificate)
			if err != nil {
				log.Fatalf("Cannot read TLS certificate %s: %s", c.config.SslCertificate, err)
			}
			if ok := roots.AppendCertsFromPEM(tlsCert); !ok {
				log.Fatalf("Cannot use TLS certificate %s: %s", c.config.SslCertificate, err)
			}
			tlsConfig.RootCAs = roots
		}

		c.sock, err = tls.Dial("tcp", c.config.Address, tlsConfig)
	} else {
		c.sock, err = net.Dial("tcp", c.config.Address)
	}
	if err != nil {
		return
	}

	log.Print("Connected to: ", c.config.Name)
	c.io = bufio.NewReadWriter(bufio.NewReader(c.sock), bufio.NewWriter(c.sock))

	// remember to update numLoops if you add or remove loop methods!
	c.wg.Add(numLoops)
	go c.writeLoop()
	go c.readLoop()
	go c.pingLoop()
	go c.errLoop()

	c.RunCallbacks(&Message{Cmd: "INIT"})

	c.wg.Wait()

	return nil
}

func (c *Connection) errLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.inerr:
			// ensure we have closed the socket
			if err := c.sock.Close(); err != nil {
				log.Print("error closing socket: ", err)
			}

			// incoming error from a goroutine
			for i := 0; i < numLoops; i++ {
				c.outerr <- true
			}
			return
		case <-c.outerr:
			return
		}
	}
}

func (c *Connection) writeLoop() {
	defer c.wg.Done()

	for {
		select {
		case line := <-c.out:
			err := c.write(line)
			if err != nil {
				log.Print("socket write error: ", err)
				c.inerr <- err
				return
			}
		case <-c.outerr:
			return
		}
	}
}

func (c *Connection) readLoop() {
	defer c.wg.Done()

	for {
		// a read() on a closed socket should always fail, so we can skip
		// the check on outerr.
		line, err := c.io.ReadString('\n')
		if err != nil {
			log.Print("socket read error: ", err)
			c.inerr <- err
			return
		}

		if message, err := parseMessage(line); err != nil {
			log.Printf("parsing failed (%s) for line: %q", err, line)
		} else {
			c.RunCallbacks(message)
		}
	}
}

func (c *Connection) pingLoop() {
	defer c.wg.Done()

	tick := time.NewTicker(pingFrequency)
	for {
		select {
		case <-tick.C:
			c.ServerPing()
		case <-c.outerr:
			tick.Stop()
			return
		}
	}
}

func (c *Connection) write(line string) error {
	if !c.floodProtection {
		if t := c.rateLimit(len(line)); t != 0 {
			log.Printf("anti-flood: sleeping for %.2f seconds", t.Seconds())
			<-time.After(t)
		}
	}

	if _, err := c.io.WriteString(line); err != nil {
		return err
	}

	if err := c.io.Flush(); err != nil {
		return err
	}
	return nil
}

func (c *Connection) rateLimit(chars int) time.Duration {
	lineTime := 2*time.Second + time.Duration(chars)*time.Second/120
	elapsed := time.Now().Sub(c.lastSent)
	if c.badness += lineTime - elapsed; c.badness < 0 {
		c.badness = 0
	}
	c.lastSent = time.Now()

	if c.badness > 10*time.Second {
		return lineTime
	}

	return 0
}

func (c *Connection) Shutdown() {
	c.tryReconnect = false
	c.inerr <- errors.New("shutdown requested")
}
