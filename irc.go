// dulbecco is a IRC bot.
//
// Plugin support through external commands (bash, python, ...).
//
//   most of the code was copied from:
//   - https://github.com/fluffle/goirc
//   - https://github.com/thoj/go-ircevent
package dulbecco

import (
	"bufio"
	"crypto/tls"
	"errors"
	"github.com/piger/dulbecco/markov"
	"log"
	"net"
	"strings"
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
)

type Channel struct {
	name, topic string
	names       []string
}

type User struct {
	nickname, username, hostname, realname string

	channels map[string]bool
}

func NewUser(nickname string) *User {
	user := &User{
		nickname: nickname,
		channels: make(map[string]bool),
	}
	return user
}

// A connection to the IRC server, also the main data structure of the IRC bot.
type Connection struct {
	config ServerConfiguration

	// current nickname
	nickname string

	chanmap map[string]*Channel
	usermap map[string]*User

	// enable anti-flood protection
	floodProtection bool
	// anti-flood internal counters
	badness  time.Duration
	lastSent time.Time

	// IO
	sock         net.Conn
	io           *bufio.ReadWriter
	out          chan string
	connected    bool
	tryReconnect bool

	// lock shutdown calls
	mutex sync.Mutex

	// SSL
	useTLS bool

	// Internal channels
	inerr  chan error
	outerr chan bool
	wg     sync.WaitGroup

	// markov database
	mdb *markov.MarkovDB

	// callbacks
	events map[string]map[string]func(*Message)
}

func NewConnection(config ServerConfiguration, botConfig *Configuration, mdb *markov.MarkovDB) *Connection {
	conn := &Connection{
		config:          config,
		nickname:        config.Nickname,
		chanmap:         make(map[string]*Channel),
		usermap:         make(map[string]*User),
		floodProtection: true,
		lastSent:        time.Now(),
		out:             make(chan string, 32),
		inerr:           make(chan error, numLoops),
		outerr:          make(chan bool, numLoops),
		mdb:             mdb,
	}

	// setup internal callbacks
	conn.SetupCallbacks()
	conn.SetupPlugins(botConfig.Plugins)

	return conn
}

func (c *Connection) MainLoop() {
	for {
		err := c.Connect()
		if err != nil {
			log.Printf("Connection error: %s\n", err)
		}
		if !c.tryReconnect {
			return
		}

		c.reinit()
		time.Sleep(time.Second * 5)
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

// Connect to the server, launch all internal goroutines.
func (c *Connection) Connect() (err error) {
	if c.config.Ssl {
		// XXX should use a valid tls client configuration insted of nil (default)
		c.sock, err = tls.Dial("tcp", c.config.Address, nil)
	} else {
		c.sock, err = net.Dial("tcp", c.config.Address)
	}
	if err != nil {
		return
	}

	log.Println("Connected to:", c.config.Address)
	c.connected = true

	c.io = bufio.NewReadWriter(
		bufio.NewReader(c.sock),
		bufio.NewWriter(c.sock))

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
	defer func() {
		log.Println("exiting errLoop()")
		c.wg.Done()
	}()

	for {
		select {
		case err := <-c.inerr:
			log.Printf("Incoming error: %s\n", err)

			// ensure we have closed the socket
			c.sock.Close()

			// incoming error from a goroutine
			for i := 0; i < numLoops; i++ {
				c.outerr <- true
			}
			return
		case <-c.outerr:
			log.Printf("errLoop: alert received, bye!\n")
			return
		}
	}
}

func (c *Connection) writeLoop() {
	defer func() {
		defer log.Println("exiting writeLoop()")
		defer c.wg.Done()
	}()

	for {
		select {
		case line := <-c.out:
			err := c.write(line)
			if err != nil {
				log.Println("ERROR writing:", err)
				c.inerr <- err
				return
			}
		case <-c.outerr:
			return
		}
	}
}

func (c *Connection) readLoop() {
	defer func() {
		defer log.Println("exiting readLoop()")
		defer c.wg.Done()
	}()

	for {
		// a read() on a closed socket should always fail, so we can skip
		// the check on outerr.
		line, err := c.io.ReadString('\n')
		if err != nil {
			log.Printf("error reading from socket: %s\n", err)
			c.inerr <- err
			return
		}

		line = strings.TrimRight(line, "\r\n")

		if message, err := parseMessage(line); err == nil {
			log.Println("message =", message.Dump())
			c.RunCallbacks(message)
		} else {
			log.Printf("parsing failed (%s) for line: %q\n", err, line)
		}
	}
}

func (c *Connection) pingLoop() {
	defer func() {
		defer log.Println("exiting pingLoop()")
		defer c.wg.Done()
	}()

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
			log.Printf("anti-flood: sleeping for %.2f seconds\n", t.Seconds())
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
