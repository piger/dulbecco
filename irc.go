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
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Channel struct {
	name  string
	topic string
	names []string
}

type User struct {
	nickname, username, hostname, realname string
	channels                               map[string]bool
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
	// server address+port: "irc.example.com:6667"
	address string

	username, realname, nickname, password string
	altnicknames                           []string
	channels                               []string
	chanmap                                map[string]*Channel
	usermap                                map[string]*User

	// ping frequency
	pingFreq time.Duration

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
		password:        srvConfig.Password,
		useTLS:          srvConfig.Ssl,
		sslConfig:       nil,
		pingFreq:        3 * time.Minute,
		username:        username,
		realname:        realname,
		nickname:        nickname,
		altnicknames:    altnicknames,
		chanmap:         make(map[string]*Channel),
		usermap:         make(map[string]*User),
		connected:       false,
		tryReconnect:    false,
		channels:        srvConfig.Channels,
		out:             make(chan string, 32),
		cWrite:          make(chan bool),
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
	c.connected = true

	c.io = bufio.NewReadWriter(
		bufio.NewReader(c.sock),
		bufio.NewWriter(c.sock))

	go c.writeLoop()
	go c.readLoop()
	go c.pingLoop()

	c.RunCallbacks(&Message{Cmd: "INIT"})

	return nil
}

func (c *Connection) writeLoop() {
	for {
		select {
		case line := <-c.out:
			err := c.write(line)
			if err != nil {
				log.Println("ERROR writing:", err)
				go c.shutdown()
				return
			}
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

		line = strings.TrimRight(line, "\r\n")
		// log.Println("READ: ", line)

		if message := parseMessage(line); message != nil {
			log.Println("message =", message.Dump())
			c.RunCallbacks(message)
		} else {
			log.Println("parsing failed for line:", line)
		}
	}
}

func (c *Connection) pingLoop() {
	tick := time.NewTicker(c.pingFreq)

	for {
		select {
		case <-tick.C:
			c.ServerPing()
		case <-c.cPing:
			tick.Stop()
			return
		}
	}
}

func (c *Connection) write(line string) error {
	if !c.floodProtection {
		if t := c.rateLimit(len(line)); t != 0 {
			log.Println("anti-flood: sleeping for %.2f seconds", t.Seconds())
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

func (c *Connection) shutdown() {
	c.mutex.Lock()
	log.Println("enter shutdown()")

	if c.connected {
		log.Println("shutting down connection")

		c.connected = false
		c.sock.Close()
		c.cWrite <- true
		c.cPing <- true

		c.io = nil
		c.sock = nil

		c.RunCallbacks(&Message{Cmd: "DISCONNECT"})

		if c.tryReconnect {
			go c.reconnect()
		} else {
			c.CQuit <- true
		}
		log.Println("end of shutdown")
	}
	c.mutex.Unlock()
	log.Println("exit shutdown()")
}

func (c *Connection) reconnect() {
	for {
		log.Println("Sleeping 5 minutes before trying to reconnect")
		time.Sleep(5 * time.Minute)
		if err := c.Connect(); err == nil {
			return
		}
	}
}
