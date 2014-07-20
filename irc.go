package dulbecco

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// most of the code was copied from: https://github.com/fluffle/goirc/blob/go1/client/connection.go

type Connection struct {
	// server address+port: "irc.example.com:6667"
	address string

	username, realname, nickname string
	altnicknames                 []string

	// ping frequency
	pingFreq time.Duration

	// anti-flood protection
	floodProtection bool
	badness         time.Duration
	lastSent        time.Time

	// IO
	sock      net.Conn
	io        *bufio.ReadWriter
	in        chan *Message
	out       chan string
	Connected bool

	// lock shutdown calls
	mutex sync.Mutex

	// SSL
	useTLS    bool
	sslConfig *tls.Config

	// Control channels
	cWrite chan bool
	cEvent chan bool
	cPing  chan bool
	CQuit  chan bool

	// callbacks
	events map[string]map[string]func(*Message)
}

func NewConnection(srvConfig *ServerType, genConfig *ConfigurationType, quit chan bool) *Connection {
	// get configuration values from the server config object, falling back
	// to the global config.
	var nickname, username, realname string
	var altnicknames []string

	if nickname = srvConfig.Nickname; len(nickname) == 0 {
		nickname = genConfig.Nickname
	}
	if username = srvConfig.Username; len(username) == 0 {
		username = genConfig.Username
	}
	if realname = srvConfig.Realname; len(realname) == 0 {
		realname = genConfig.Realname
	}

	if len(srvConfig.Altnicknames) > 0 {
		altnicknames = append(altnicknames, srvConfig.Altnicknames...)
	} else {
		altnicknames = append(altnicknames, genConfig.Altnicknames...)
	}

	conn := &Connection{
		address:         srvConfig.Address,
		useTLS:          srvConfig.Ssl,
		sslConfig:       nil,
		pingFreq:        3 * time.Minute,
		username:        username,
		realname:        realname,
		nickname:        nickname,
		altnicknames:    altnicknames,
		in:              make(chan *Message, 32),
		out:             make(chan string, 32),
		cWrite:          make(chan bool),
		cEvent:          make(chan bool),
		cPing:           make(chan bool),
		floodProtection: true,
		badness:         0,
		lastSent:        time.Now(),
		CQuit:           quit,
	}

	// setup internal callbacks
	conn.SetupCallbacks()
	conn.SetupPlugins(genConfig.Plugins)

	return conn
}

// Connect to the server, launch all internal goroutines.
func (c *Connection) Connect() error {
	if c.useTLS {
		if s, err := tls.Dial("tcp", c.address, c.sslConfig); err == nil {
			c.sock = s
		} else {
			return err
		}
	} else {
		if s, err := net.Dial("tcp", c.address); err == nil {
			c.sock = s
		} else {
			return err
		}
	}

	log.Println("Connected to:", c.address)
	c.Connected = true

	c.io = bufio.NewReadWriter(
		bufio.NewReader(c.sock),
		bufio.NewWriter(c.sock))

	go c.writeLoop()
	go c.readLoop()
	go c.pingLoop()
	go c.eventLoop()

	c.RunCallbacks(&Message{Cmd: "INIT"})

	return nil
}

func (c *Connection) writeLoop() {
	for {
		select {
		case line := <-c.out:
			c.write(line)
		case <-c.cWrite:
			return
		}
	}
}

func (c *Connection) readLoop() {
	for {
		line, err := c.io.ReadString('\n')
		if err != nil {
			log.Println("read failed:", err)
			go c.shutdown()
			return
		}

		line = strings.Trim(line, "\r\n")
		// log.Println("READ: ", line)

		if message := parseMessage(line); message != nil {
			message.Time = time.Now()
			message.Dump()
			c.in <- message
		} else {
			log.Println("parsing failed for line: ", line)
		}
	}
}

func (c *Connection) pingLoop() {
	tick := time.NewTicker(c.pingFreq)

	for {
		select {
		case <-tick.C:
			c.Raw(fmt.Sprintf("PING :%d", time.Now().UnixNano()))
		case <-c.cPing:
			return
		}
	}
}

func (c *Connection) eventLoop() {
	for {
		select {
		case message := <-c.in:
			c.RunCallbacks(message)
		case <-c.cEvent:
			return
		}
	}
}

func (c *Connection) write(line string) {
	if !c.floodProtection {
		if t := c.rateLimit(len(line)); t != 0 {
			log.Println("anti-flood: sleeping for %.2f seconds", t.Seconds())
			<-time.After(t)
		}
	}

	if _, err := c.io.WriteString(line + "\r\n"); err != nil {
		log.Println("write failed: ", err)
		go c.shutdown()
		return
	}

	if err := c.io.Flush(); err != nil {
		log.Println("flush failed: ", err)
		go c.shutdown()
		return
	}

	// log.Println("wrote line: ", line)
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

func (c *Connection) shutdown() {
	c.mutex.Lock()
	log.Println("enter shutdown()")

	if c.Connected {
		log.Println("shutting down connection")

		c.Connected = false
		c.sock.Close()
		c.cWrite <- true
		c.cEvent <- true
		c.cPing <- true

		c.io = nil
		c.sock = nil

		c.RunCallbacks(&Message{Cmd: "DISCONNECT"})
		c.CQuit <- true
		log.Println("end of shutdown")
	}
	c.mutex.Unlock()
	log.Println("exit shutdown()")
}
