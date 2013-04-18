package reaction

import (
	"../"
	"../../utils"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"strings"
)

func init() {
	plugin.RegisterSetup(setup)
}

func setup(hreg irc.HandlerRegistry, reg *callback.Registry) error {
	reg.AddCallback("PRIVMSG", func(conn *irc.Conn, line irc.Line, dst, text string) {
		reply := dst
		if dst == conn.Me().Nick {
			reply = line.Src.Nick
		}
		if reply == "" {
			// what, the server sent us a privmsg?
			return
		}
		// implement super awesome reaction logic here
		if strings.ToLower(text) == "herp" {
			plugin.Conn(conn).Privmsg(reply, utils.MatchCase("derp", text))
		}
	})
	return nil
}
