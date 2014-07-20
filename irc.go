package dulbecco

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type Connection struct {
	address string

	username, realname, nickname string
	altnicknames                 []string

	// ping frequency
	pingFreq time.Duration

	// IO
	sock      net.Conn
	io        *bufio.ReadWriter
	in        chan *Message
	out       chan string
	Connected bool

	// SSL
	useTLS    bool
	sslConfig *tls.Config

	// Control channels
	cWrite chan bool
	cEvent chan bool
	cPing  chan bool

	// callbacks
	events map[string]map[string]func(*Message)
}

func NewConnection(srvConfig *ServerType, genConfig *ConfigurationType) *Connection {
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
		address:      srvConfig.Address,
		useTLS:       srvConfig.Ssl,
		sslConfig:    nil,
		pingFreq:     3 * time.Minute,
		username:     username,
		realname:     realname,
		nickname:     nickname,
		altnicknames: altnicknames,
		in:           make(chan *Message, 32),
		out:          make(chan string, 32),
		cWrite:       make(chan bool),
		cEvent:       make(chan bool),
		cPing:        make(chan bool),
	}

	conn.SetupCallbacks()

	return conn
}

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

	log.Println("Connected!\n")
	c.Connected = true
	c.RunCallbacks(&Message{Cmd: "INIT"})

	c.io = bufio.NewReadWriter(
		bufio.NewReader(c.sock),
		bufio.NewWriter(c.sock))

	go c.writeLoop()
	go c.readLoop()
	go c.pingLoop()
	go c.eventLoop()

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
			log.Println("read failed: ", err)
			c.shutdown()
			return
		}

		line = strings.Trim(line, "\r\n")
		log.Println("READ: ", line)

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
	if _, err := c.io.WriteString(line + "\r\n"); err != nil {
		log.Println("write failed: ", err)
		c.shutdown()
		return
	}

	if err := c.io.Flush(); err != nil {
		log.Println("flush failed: ", err)
		c.shutdown()
		return
	}

	log.Println("wrote line: ", line)
}

func (c *Connection) shutdown() {
	if c.Connected {

		log.Println("shutting down connection")
		c.Connected = false
		c.sock.Close()
		c.cWrite <- true
		c.cEvent <- true
		c.cPing <- true

		c.io = nil
		c.sock = nil
	}
}
