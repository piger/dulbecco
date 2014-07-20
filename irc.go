package dulbecco

import (
	"net"
	"bufio"
	"log"
	"strings"
	"time"
)


type Connection struct {
	address string
	useTLS bool

	username, realname, nickname, altnickname string

	// IO
	sock net.Conn
	io *bufio.ReadWriter
	in chan *Message
	out chan string
	Connected bool

	// Control channels
	cWrite chan bool
	cEvent chan bool

	// callbacks
	events map[string]map[string]func(*Message)
}

func NewConnection(address, nickname string) *Connection {
	conn := &Connection{
		address: address,
		useTLS: false,
		username: "dulbecco",
		realname: "Pinot di pinolo",
		nickname: nickname,
		altnickname: "dulbecco_",
		in: make(chan *Message, 32),
		out: make(chan string, 32),
		cWrite: make(chan bool),
		cEvent: make(chan bool),
	}

	conn.SetupCallbacks()

	return conn
}

func (c *Connection) Connect() error {
	if s, err := net.Dial("tcp", c.address); err == nil {
		c.sock = s
	} else {
		return err
	}

	log.Println("Connected!\n")
	c.Connected = true
	c.RunCallbacks(&Message{Cmd: "INIT"})

	c.io = bufio.NewReadWriter(
		bufio.NewReader(c.sock),
		bufio.NewWriter(c.sock))

	go c.writeLoop()
	go c.readLoop()
	go c.eventLoop()

	return nil
}

func (c *Connection) writeLoop() {
	for {
		select {
		case line := <- c.out:
			c.write(line)
		case <- c.cWrite:
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
			c.in <- message
		} else {
			log.Println("parsing failed for line: ", line)
		}
	}
}

func (c *Connection) eventLoop() {
	for {
		select {
		case message := <- c.in:
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

		c.io = nil
		c.sock = nil
	}
}
