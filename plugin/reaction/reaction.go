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
	plugin.RegisterPlugin("reaction", plugin.Callbacks{Init: setup})
}

func setup(reg *callback.Registry) error {
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
		isDirected := false
		if strings.HasPrefix(text, fmt.Sprintf("%s: ", conn.Me().Nick)) {
			text = text[len(conn.Me().Nick)+2:]
			prefix = line.Src.Nick + ": "
			isDirected = true
		}
		// implement super awesome reaction logic here
		if strings.ToLower(text) == "herp" {
			plugin.Conn(conn).Notice(reply, prefix+utils.MatchCase("derp", text))
		} else if strings.ToLower(text) == "is me1000 drunk?" || (line.Src.Nick == "Me1000" && strings.ToLower(text) == "am i drunk?") {
			plugin.Conn(conn).Notice(reply, prefix+randResponse([]string{"yes", "always"}))
		} else if isDirected && strings.ToLower(text) == "botsnack" {
			plugin.Conn(conn).Notice(reply, prefix+randResponse([]string{"yum", "nom nom", "om nom nom"}))
		} else if isDirected && text == "<3" {
			plugin.Conn(conn).Notice(reply, prefix+"<3")
		}
	})
	return nil
}

func randResponse(opts []string) string {
	return opts[rand.Intn(len(opts))]
}
