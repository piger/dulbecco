package dulbecco

import (
	"strings"
)


// send a "raw" line to the server
func (c *Connection) Raw(s string) {
	c.out <- s
}

// NICK command
func (c *Connection) Nick(nickname string) {
	c.out <- "NICK " + nickname
}

// USER command
// http://tools.ietf.org/html/rfc2812#section-3.1.3
// Parameters: <user> <mode> <unused> <realname>
func (c *Connection) User(ident, realname string) {
	c.out <- "USER " + ident + " 12 * :" + realname
}

// JOIN command
func (c *Connection) Join(channel string) {
	c.out <- "JOIN " + channel
}

// PART command
// optional argument: the part message
func (c *Connection) Part(channel string, message ...string) {
	msg := strings.Join(message, " ")
	if msg != "" {
		msg = " :" + msg
	}
	c.out <- "PART " + channel + msg
}

// QUIT command
func (c *Connection) Quit(message ...string) {
	msg := strings.Join(message, " ")
	if msg == "" {
		msg = "Attuo il decesso gallico"
	}

	c.out <- "QUIT :" + msg
}

// PRIVMSG command
func (c *Connection) Privmsg(target, message string) {
	c.out <- "PRIVMSG " + target + " :" + message
}

// NOTICE command
func (c *Connection) Notice(target, message string) {
	c.out <- "NOTICE " + target + " :" + message
}
