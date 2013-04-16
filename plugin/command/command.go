package command

import (
	"../"
	"fmt"
	"github.com/fluffle/goevent/event"
	irc "github.com/fluffle/goirc/client"
	"strings"
	"unicode"
	"unicode/utf8"
)

func init() {
	plugin.RegisterSetup(setup)
}

const commandPrefix = "!"

func setup(conn *irc.Conn, er event.EventRegistry) error {
	conn.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		if len(line.Args) != 2 {
			// malformed line?
			return
		}
		dst := line.Args[0]
		text := line.Args[1]

		if strings.HasPrefix(text, commandPrefix) &&
			len(text) > len(commandPrefix) &&
			func() bool { r, _ := utf8.DecodeRuneInString(text[len(commandPrefix):]); return unicode.IsLetter(r) }() {
			// this is a command
			words := strings.SplitN(text, " ", 2)
			cmd, arg := words[0][len(commandPrefix):], ""
			if len(words) > 1 {
				arg = words[1]
			}
			reply, isPrivate := dst, false
			if !strings.HasPrefix(reply, "#") {
				reply, isPrivate = line.Nick, true
			}
			er.Dispatch("COMMAND", conn, line, cmd, arg, reply, isPrivate)
		} else if strings.HasPrefix(dst, "#") {
			er.Dispatch("PRIVMSG", conn, line, dst, text)
		} else if dst == conn.Me.Nick {
			er.Dispatch("WHISPER", conn, line, text)
		} else {
			fmt.Println("Unknown destination on PRIVMSG:", line.Raw)
		}
	})
	return nil
}
