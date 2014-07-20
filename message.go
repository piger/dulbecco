package dulbecco

import (
	"strings"
	"time"
	"fmt"
)


// Struct containing the incoming data.
// Src => "irc.example.com" or "nick!ident@host"
// Raw => "nick!ident@host PRIVMSG #channel :hello world"
type Message struct {
	Ident, Nick, Host, Src string

	Cmd, Raw string
	Args []string
	Time time.Time
}

func parseMessage(s string) *Message {
	message := &Message{Raw: s}

	// line begins with a source:
	// :ident!nick@host PRIVMSG #test :ciaone
	if s[0] == ':' {
		splitted := strings.SplitN(s[1:], " ", 2)
		if len(splitted) != 2 {
			fmt.Printf("Invalid line: %q\n", s)
			return nil
		}
		message.Src, s = splitted[0], splitted[1]

		// message.Src can be a simple hostname or a "IRC" mask
		message.Host = message.Src

		nidx, iidx := strings.Index(message.Src, "!"), strings.Index(message.Src, "@")
		if nidx != -1 && iidx != -1 {
			message.Nick = message.Src[:nidx]
			message.Ident = message.Src[nidx + 1:iidx]
			message.Host = message.Src[iidx + 1:]
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
	if len(args) > 1 {
		message.Args = args[1:]
	}

	// special handling for CTCPs
	// :sand!~sand@localhost PRIVMSG nickname :PING 1405848291 393196
	// Args[0] = nickname
	// Args[1] = \001PING 1405848291 393196\001
	if (message.Cmd == "PRIVMSG" || message.Cmd == "NOTICE") &&
		strings.HasPrefix(message.Args[1], "\001") &&
		strings.HasSuffix(message.Args[1], "\001") {
		t := strings.SplitN(strings.Trim(message.Args[1], "\001"), " ", 2)
		if len(t) > 1 {
			message.Args[1] = t[1]
		}
		// now t[] contains: ["PING", "1405848291 393196"]

		// now Args[1] is: "1405848291 393196" without CTCP characters (\001)
		if c:= strings.ToUpper(t[0]); c == "ACTION" && message.Cmd == "PRIVMSG" {
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

	return message
}
