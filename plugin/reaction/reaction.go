package reaction

import (
	"../"
	"../../utils"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"math/rand"
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
		// allow voidbot to be addressed directly, and modify the response
		prefix := ""
		if strings.HasPrefix(text, fmt.Sprintf("%s: ", conn.Me().Nick)) {
			text = text[len(conn.Me().Nick)+2:]
			prefix = line.Src.Nick + ": "
		}
		// implement super awesome reaction logic here
		if strings.ToLower(text) == "herp" {
			plugin.Conn(conn).Privmsg(reply, prefix+utils.MatchCase("derp", text))
		} else if strings.ToLower(text) == "is me1000 drunk?" || (line.Src.Nick == "Me1000" && strings.ToLower(text) == "am i drunk?") {
			plugin.Conn(conn).Privmsg(reply, prefix+randResponse([]string{"yes", "always"}))
		}
	})
	return nil
}

func randResponse(opts []string) string {
	return opts[rand.Intn(len(opts))]
}
