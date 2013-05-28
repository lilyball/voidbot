package command

import (
	"../"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"strings"
	"unicode"
	"unicode/utf8"
)

func init() {
	plugin.RegisterPlugin("", plugin.Callbacks{Init: pluginInit, NewConnection: setup, Teardown: pluginTeardown})
}

const commandPrefix = "!"

var pluginReg *callback.Registry

func pluginInit(reg *callback.Registry) error {
	pluginReg = reg
	return nil
}

func pluginTeardown() error {
	pluginReg = nil
	return nil
}

func setup(reg irc.HandlerRegistry) {
	reg.AddHandler("PRIVMSG", func(conn *irc.Conn, line irc.Line) {
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
			if !isChannelName(reply) {
				reply, isPrivate = line.Src.Nick, true
			}
			pluginReg.Dispatch("COMMAND", conn, line, cmd, arg, reply, isPrivate)
		} else if isChannelName(dst) {
			pluginReg.Dispatch("PRIVMSG", conn, line, dst, text)
		} else if dst == conn.Me().Nick {
			pluginReg.Dispatch("WHISPER", conn, line, text)
		} else {
			fmt.Println("Unknown destination on PRIVMSG:", line.Raw)
		}
	})
	reg.AddHandler(irc.ACTION, func(conn *irc.Conn, line irc.Line) {
		dst := line.Dst
		text := line.Args[0]
		isPrivate := !isChannelName(dst)
		pluginReg.Dispatch("ACTION", conn, line, dst, text, isPrivate)
	})
}

func isChannelName(name string) bool {
	return len(name) > 0 && (name[0] == '#' || name[0] == '&' || name[0] == '!' || name[0] == '+')
}
