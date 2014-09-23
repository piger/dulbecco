package dulbecco

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

func (c *Connection) AddCallback(name string, callback func(*Message)) string {
	name = strings.ToUpper(name)

	if _, ok := c.events[name]; !ok {
		c.events[name] = make(map[string]func(*Message))
	}

	hash := sha1.New()
	rawId := []byte(fmt.Sprintf("%v%d", reflect.ValueOf(callback).Pointer(), rand.Int63()))
	hash.Write(rawId)
	id := fmt.Sprintf("%x", hash.Sum(nil))
	c.events[name][id] = callback
	// log.Println("Registered callback:", id)
	return id
}

func (c *Connection) RemoveCallback(name string, id string) bool {
	name = strings.ToUpper(name)

	if event, ok := c.events[name]; ok {
		if _, ok := event[id]; ok {
			delete(c.events[name], id)
			return true
		}
		fmt.Printf("No callback found in %s with id %s\n", name, id)
		return false
	}

	fmt.Printf("Event not found: %s\n", name)
	return false
}

// Execute registered callbacks for message
func (c *Connection) RunCallbacks(message *Message) {
	if callbacks, ok := c.events[message.Cmd]; ok {
		for _, callback := range callbacks {
			go callback(message)
		}
	}

	// catch-all handlers
	if callbacks, ok := c.events["*"]; ok {
		for _, callback := range callbacks {
			go callback(message)
		}
	}
}

// Add internal callbacks.
func (c *Connection) SetupCallbacks() {
	c.events = make(map[string]map[string]func(*Message))

	c.AddCallback("INIT", c.h_INIT)
	c.AddCallback("001", c.h_001)
	c.AddCallback("433", c.h_433)
	c.AddCallback("PRIVMSG", c.h_PRIVMSG)
	c.AddCallback("NOTICE", c.h_NOTICE)
	c.AddCallback("PING", c.h_PING)
	c.AddCallback("PONG", c.h_PONG)
	c.AddCallback("CTCP", c.h_CTCP)
	c.AddCallback("JOIN", c.h_JOIN)
	c.AddCallback("332", c.h_332)
	c.AddCallback("352", c.h_352)
	c.AddCallback("353", c.h_353)
	c.AddCallback("KICK", c.h_KICK)
	c.AddCallback("PART", c.h_PART)
}

// Add callbacks for every configured plugin.
func (c *Connection) SetupPlugins(plugins []PluginConfiguration) {
	for _, plugin := range plugins {
		log.Println("Adding callback for plugin:", plugin.Name)
		c.addPluginCallback(plugin)
	}
}

// Add a callback for a plugin.
// We use a separate method because we need a "copy" of the "plugin" variable,
// since it will be bound inside the closure.
func (c *Connection) addPluginCallback(plugin PluginConfiguration) {

	// this is the actual plugin callback
	c.AddCallback("PRIVMSG", func(message *Message) {
		// "trigger" contains a regular expression with optional capture groups
		// command is a text/template that can contain captures from the trigger
		// regexp.
		re := regexp.MustCompile(plugin.Trigger)
		match := re.FindStringSubmatch(message.Args[1])
		if match == nil {
			return
		}
		captures := make(map[string]string)
		for i, name := range re.SubexpNames() {
			if i == 0 || name == "" {
				continue
			}
			captures[name] = match[i]
		}
		cmdtpl, err := template.New("cmd").Parse(plugin.Command)
		if err != nil {
			log.Printf("Cannot parse template: %s\n", err)
			return
		}
		var cmdbuf bytes.Buffer
		if err := cmdtpl.Execute(&cmdbuf, captures); err != nil {
			log.Printf("Cannot execute template: %s\n", err)
			return
		}
		cmdline := cmdbuf.String()

		log.Printf("Running plugin %s: %s", plugin.Name, cmdline)

		cmds := strings.Fields(cmdline)
		cmd := exec.Command(cmds[0], cmds[1:]...)
		env := []string{
			"IRC_NICKNAME=" + message.Nick,
			"IRC_HOST=" + message.Host,
			"IRC_IDENT=" + message.Ident,
			"IRC_ARGS=" + strings.Join(message.Args, " "),
			"IRC_COMMAND=" + message.Cmd,
			"IRC_TIMESTAMP=" + message.Time.String(),
			"IRC_RAW=" + message.Raw,
		}
		cmd.Env = append(os.Environ(), env...)

		if out, err := cmd.Output(); err == nil {
			lines := strings.Trim(string(out), "\n")
			target := message.ReplyTarget()
			for _, line := range strings.Split(lines, "\n") {
				c.Privmsg(target, line)
			}
		} else {
			log.Printf("Failed exec for plugin '%s': %s\n", plugin.Name, err)
		}
	})
}

// callbacks

// helper function to call when a user leave a channel
func (c *Connection) removeUserChannel(nickname, channelname string) {
	_, exists := c.chanmap[channelname]
	if !exists {
		log.Println("removeUserChannel: channel not found:", channelname)
		return
	}
	user, exists := c.usermap[nickname]
	if !exists {
		log.Println("removeUserChannel: nickname not found:", nickname)
		return
	}
	delete(user.channels, channelname)
	if len(user.channels) == 0 {
		delete(c.usermap, nickname)
	}
}

// The INIT pseudo-event is fired when the TCP connection to the IRC
// server is established successfully.
func (c *Connection) h_INIT(message *Message) {
	if len(c.config.Password) > 0 {
		c.Pass(c.config.Password)
	}
	c.Nick(c.nickname)
	c.User(c.config.Username, c.config.Realname)
}

func (c *Connection) JoinChannels() {
	for _, channel := range c.config.Channels {
		c.Join(channel)
	}
}

// 001 numeric means we are "really connected" to the server. In this callback
// is safe to do things like joining channels or identifying with IRC services.
func (c *Connection) h_001(message *Message) {
	if c.config.Nickserv != "" {
		c.LoginNickserv()
	} else {
		c.JoinChannels()
	}
}

// someone joined a channel
func (c *Connection) h_JOIN(message *Message) {
	var channame = &message.Args[0]

	// are we the one joining the channel?
	if message.Nick == c.nickname {
		delete(c.chanmap, *channame)
		channel := &Channel{name: message.Args[0]}
		c.chanmap[channel.name] = channel
		log.Printf("we have joined %s\n", message.Args[0])

		log.Println("Sending a WHO")
		c.Who(channel.name)
	} else {
		log.Printf("%s joined %s\n", message.Nick, message.Args[0])
	}
}

func (c *Connection) h_PART(message *Message) {
	if message.Nick == c.nickname {
		delete(c.chanmap, message.Args[0])
	} else {
		c.removeUserChannel(message.Nick, message.Args[0])
	}
}

// RPL_TOPIC
func (c *Connection) h_332(message *Message) {
	var channame = &message.Args[1]
	var topic = &message.Args[2]

	if channel, ok := c.chanmap[*channame]; ok {
		channel.topic = *topic
	} else {
		log.Printf("[332] Channel %s not in map??\n", *channame)
	}
}

// WHO
func (c *Connection) h_352(message *Message) {
	if len(message.Args) != 8 {
		log.Println("Invalid 352:", message)
		return
	}
	nickname := &message.Args[5]
	user, exists := c.usermap[*nickname]
	if !exists {
		user = NewUser(*nickname)
	}

	user.username = message.Args[2]
	user.hostname = message.Args[3]
	user.channels[message.Args[1]] = true

	realnameStr := strings.SplitN(message.Args[7], " ", 2)
	if len(realnameStr) > 1 {
		user.realname = realnameStr[1]
	}
}

// NAMES
func (c *Connection) h_353(message *Message) {
	var channame = message.Args[2]
	var names = strings.Split(strings.Trim(message.Args[3], " "), " ")

	for _, name := range names {
		if user, ok := c.usermap[name]; ok {
			user.channels[channame] = true
		} else {
			user := NewUser(name)
			user.channels[channame] = true
			c.usermap[name] = user
		}
	}

	if channel, ok := c.chanmap[channame]; ok {
		channel.names = append(channel.names, names...)
		log.Printf("NAMES on %s: %v\n", channame, channel.names)
	}
}

// rejoin when kicked after 5 seconds
func (c *Connection) h_KICK(message *Message) {
	var channame = message.Args[0]
	var nick = strings.Trim(message.Args[1], " ")

	if nick == c.nickname {
		cTimer := time.After(5 * time.Second)
		go func() {
			<-cTimer
			c.Join(channame)
		}()
	} else {
		c.removeUserChannel(nick, channame)
	}
}

// ERR_NICKNAMEINUSE
func (c *Connection) h_433(message *Message) {
	c.Nick(c.nickname + "_")
}

// Server PING.
func (c *Connection) h_PING(message *Message) {
	c.Raw("PONG " + message.Args[0])
}

func (c *Connection) h_PONG(message *Message) {
	theirTime, err := strconv.ParseInt(message.Args[0], 10, 64)
	if err != nil {
		return
	}
	delta := time.Duration(time.Now().UnixNano() - theirTime)
	log.Printf("Lag: %v\n", delta)
}

// generic PRIVMSG callback handling QUIT command and dummy reply.
func (c *Connection) h_PRIVMSG(message *Message) {
	if strings.HasPrefix(message.Args[1], "!quit") &&
		message.Nick == "sand" {
		c.Quit()
		return
	} else if message.Nick == NickservName && strings.Index(message.Args[1], "accepted") != -1 {
		c.JoinChannels()
		return
	} else if !strings.HasPrefix(message.Args[1], c.nickname) {
		// it's not a message directed to us, but we can still train markov from it
		c.mdb.ReadSentence(message.Args[1])
		return
	}

	// strip our own nickname from the input text
	renick := regexp.MustCompile(fmt.Sprintf("%s *: *", c.nickname))
	text := renick.ReplaceAllLiteralString(message.Args[1], "")

	// markov!
	c.mdb.ReadSentence(text)
	reply := c.mdb.Generate(text)

	// do not bother answering if the answer is the same as the input phrase
	if reply == text || len(reply) == 0 {
		c.Privmsg(message.Args[0], message.Nick+": "+"DEMENZA MI COLSE")
		return
	}

	if message.IsFromChannel() {
		c.Privmsg(message.Args[0], message.Nick+": "+reply)
		// c.Privmsg(message.Args[0], message.Nick+" ciao a te")
	} else {
		c.Privmsg(message.Nick, reply)
		// c.Privmsg(message.Nick, "ehy ciao")
	}
}

func (c *Connection) h_NOTICE(message *Message) {
	nick := strings.ToLower(message.Nick)
	if nick == NickservName && strings.Index(message.Args[1], "accepted") != -1 {
		c.JoinChannels()
		return
	}
}

// general CTCP handler
func (c *Connection) h_CTCP(message *Message) {
	if message.Args[0] == "PING" {
		c.CtcpReply(message.Nick, fmt.Sprintf("PING %s", message.Args[2]))
	}
}
