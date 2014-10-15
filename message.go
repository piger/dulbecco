package dulbecco

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidServerLine = errors.New("Invalid server line: wrong number of tokens")

	CTCPChar = "\001"

	reHostmask = regexp.MustCompile(`^([^!]+)!([^@]+)@(.*)`)
)

// Each line from the IRC server is parsed into a Message struct.
//   Src => "irc.example.com" or "nick!ident@host"
//   Raw => "nick!ident@host PRIVMSG #channel :hello world"
type Message struct {
	Ident, Nick, Host, Src string

	Cmd, Raw string
	Args     []string
	Time     time.Time
}

// Returns a channel name when the message was sent to a public channel or
// a nickname when the message was sent privately.
func (m *Message) ReplyTarget() string {
	if m.IsFromChannel() {
		return m.Args[0]
	}
	return m.Nick
}

func (m *Message) Dump() string {
	return fmt.Sprintf("%+v", m)
}

// Returns true if the Message generated inside a IRC channel
//   Channel types: https://www.alien.net.au/irc/chantypes.html
func (m *Message) IsFromChannel() bool {
	if len(m.Args) > 0 && len(m.Args[0]) > 0 {
		return strings.ContainsAny(string(m.Args[0][0]), "&#!+.~")
	}

	return false
}

// Parse a line from the IRC server into a Message struct.
func parseMessage(s string) (*Message, error) {
	message := &Message{Raw: s, Time: time.Now()}

	// line begins with a source:
	// :ident!nick@host PRIVMSG #test :ciaone
	// NOTE: if a line does not start with ":" it could be a server message like
	// PING.
	if s[0] == ':' {
		// split the line into two tokens:
		// - ident!nickname@hostname
		// - [rest of the line]
		if splitted := strings.SplitN(s[1:], " ", 2); len(splitted) == 2 {
			message.Src, s = splitted[0], splitted[1]
		} else {
			return nil, ErrInvalidServerLine
		}

		// try to parse the hostmask
		hostmatch := reHostmask.FindStringSubmatch(message.Src)
		if len(hostmatch) == 0 {
			message.Host = message.Src
		} else {
			message.Nick = hostmatch[1]
			message.Ident = hostmatch[2]
			message.Host = hostmatch[3]
		}
	}

	// now we have to parse stuff like:
	// [:aaaa!~a@localhost ]PRIVMSG aaaa :ciaone !
	// [:aaaa!~a@localhost ]JOIN :#puzza
	// strings between [] are already parsed.
	args := strings.SplitN(s, " :", 2)
	if len(args) > 1 {
		// args[0] = PRIVMSG #channel
		// args[1] = hello world!
		args = append(strings.Fields(args[0]), args[1])
	} else {
		args = strings.Fields(args[0])
	}

	// args[] = "PRIVMSG", "#channel", "hello world!"

	message.Cmd = strings.ToUpper(args[0])
	message.Args = args[1:]

	// special handling for CTCPs
	// :sand!~sand@localhost PRIVMSG nickname :PING 1405848291 393196
	// Args[0] = nickname
	// Args[1] = \001PING 1405848291 393196\001
	if (message.Cmd == "PRIVMSG" || message.Cmd == "NOTICE") &&
		strings.HasPrefix(message.Args[1], CTCPChar) &&
		strings.HasSuffix(message.Args[1], CTCPChar) {
		t := strings.SplitN(strings.Trim(message.Args[1], CTCPChar), " ", 2)
		if len(t) > 1 {
			message.Args[1] = t[1]
		}
		// now t[] contains: ["PING", "1405848291 393196"]

		// now Args[1] is: "1405848291 393196" without CTCP characters (\001)
		if c := strings.ToUpper(t[0]); c == "ACTION" && message.Cmd == "PRIVMSG" {
			message.Cmd = c
		} else {
			if message.Cmd == "PRIVMSG" {
				message.Cmd = "CTCP"
			} else {
				message.Cmd = "CTCPREPLY"
			}

			message.Args = append([]string{c}, message.Args...)
		}
	}

	return message, nil
}
