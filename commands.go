package dulbecco

import (
	"fmt"
	"strings"
	"time"
)

const (
	privmsgLen = len("PRIVMSG")
	noticeLen  = len("NOTICE")
)

func splitPhrase(cmd, phrase string) []string {
	cmdLen, phraseLen := len(cmd), len(phrase)

	if cmdLen+phraseLen < MaximumCommandLength {
		return []string{phrase}
	}

	var result []string
	var t int
	maxLength := MaximumCommandLength - cmdLen

	for len(phrase) > 0 {
		l := len(phrase)
		if maxLength < l {
			t = maxLength
		} else {
			t = l
		}
		result = append(result, phrase[0:t])
		phrase = phrase[t:]
	}
	return result
}

// send a "raw" line to the server
func (c *Connection) Raw(s string) {
	c.out <- fmt.Sprintf("%s\r\n", s)
}

// send a "raw" formatted line to the server
func (c *Connection) Rawf(format string, a ...interface{}) {
	c.Raw(fmt.Sprintf(format, a...))
}

// NICK command
func (c *Connection) Nick(nickname string) {
	c.Rawf("NICK %s", nickname)
}

// USER command
// http://tools.ietf.org/html/rfc2812#section-3.1.3
//   Parameters: <user> <mode> <unused> <realname>
func (c *Connection) User(ident, realname string) {
	c.Rawf("USER %s 12 * :%s", ident, realname)
}

// PASS command
func (c *Connection) Pass(password string) {
	c.Rawf("PASS %s", password)
}

// JOIN command
func (c *Connection) Join(channel string) {
	c.Rawf("JOIN %s", channel)
}

// PART command
//   optional argument: the part message
func (c *Connection) Part(channel string, message ...string) {
	msg := strings.Join(message, " ")
	if msg != "" {
		msg = " :" + msg
	}
	c.Rawf("PART %s%s", channel, msg)
}

// QUIT command
//   optional argument: quit message
func (c *Connection) Quit(message ...string) {
	msg := strings.Join(message, " ")
	if msg == "" {
		msg = "Attuo il decesso gallico"
	}

	c.Rawf("QUIT :%s", msg)
}

// PRIVMSG command
func (c *Connection) Privmsg(target, message string) {
	cmd := fmt.Sprintf("PRIVMSG %s :%%s", target)
	for _, phrase := range splitPhrase(cmd, message) {
		c.Rawf(cmd, phrase)
	}
}

// PRIVMSG with format string
func (c *Connection) Privmsgf(target, format string, a ...interface{}) {
	c.Privmsg(target, fmt.Sprintf(format, a...))
}

// NOTICE command
func (c *Connection) Notice(target, message string) {
	cmd := fmt.Sprintf("NOTICE %s :%%s", target)
	for _, phrase := range splitPhrase(cmd, message) {
		c.Rawf(cmd, phrase)
	}
}

func (c *Connection) Noticef(target, format string, a ...interface{}) {
	c.Notice(target, fmt.Sprintf(format, a...))
}

// ACTION command
func (c *Connection) Action(target, message string) {
	c.Privmsgf(target, "\001%s\001", message)
}

// INVITE command
func (c *Connection) Invite(nickname, channel string) {
	c.Rawf("INVITE %s %s", nickname, channel)
}

// KICK command
func (c *Connection) Kick(channel, nickname, reason string) {
	c.Rawf("KICK %s %s :%s", channel, nickname, reason)
}

// WHOIS
func (c *Connection) Whois(nickname string) {
	c.Rawf("WHOIS %s", nickname)
}

// WHO
func (c *Connection) Who(target string) {
	c.Rawf("WHO %s", target)
}

// NAMES
func (c *Connection) Names(target string) {
	c.Rawf("NAMES %s", target)
}

// Get or set MODE
func (c *Connection) Mode(target string, modes ...string) {
	if len(modes) > 0 {
		mode := strings.Join(modes, " ")
		c.Rawf("MODE %s %s", target, mode)
	} else {
		c.Rawf("MODE %s", target)
	}
}

func (c *Connection) GetTopic(channel string) {
	c.Rawf("TOPIC %s", channel)
}

func (c *Connection) SetTopic(channel, topic string) {
	c.Rawf("TOPIC %s :%s", channel, topic)
}

// send a PING to the server
func (c *Connection) ServerPing() {
	c.Rawf("PING :%d", time.Now().UnixNano())
}

// CTCP
func (c *Connection) Ctcp(target, args string) {
	c.Privmsgf(target, "\001%s\001", args)
}

// CTCP replies
func (c *Connection) CtcpReply(target, args string) {
	c.Noticef(target, "\001%s\001", args)
}

// IDENTIFY to NickServ
func (c *Connection) LoginNickserv() {
	c.Privmsgf(NickservName, "IDENTIFY %s", c.config.Nickserv)
}
