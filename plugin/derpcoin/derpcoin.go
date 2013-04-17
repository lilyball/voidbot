package derpcoin

import (
	"../"
	"../../utils"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	plugin.RegisterSetup(setup)
}

var enabled = true

var btcRegex = regexp.MustCompile("(?i)(\\d+(?:\\.\\d*)?|\\.\\d+) ?btcs?\\b")

func setup(hreg irc.HandlerRegistry, reg *callback.Registry) error {
	reg.AddCallback("COMMAND", func(conn *irc.Conn, line irc.Line, cmd string, arg string, reply string, isPrivate bool) {
		if cmd == "derpcoin" && !isPrivate {
			arg = strings.ToLower(strings.TrimSpace(arg))
			if arg == "" {
				msg := "derpcoin is: "
				if enabled {
					msg += "ON"
				} else {
					msg += "OFF"
				}
				plugin.Conn(conn).Privmsg(reply, msg)
			} else if arg == "on" || arg == "off" {
				if line.Src.Nick == "Me1000" {
					plugin.Conn(conn).Privmsg(reply, "no")
				} else if arg == "on" {
					enabled = true
					plugin.Conn(conn).Privmsg(reply, "derpcoin enabled")
				} else if arg == "off" {
					enabled = false
					plugin.Conn(conn).Privmsg(reply, "derpcoin disabled")
				}
			} else {
				plugin.Conn(conn).Privmsg(reply, "derp?")
			}
		}
	})
	reg.AddCallback("PRIVMSG", func(conn *irc.Conn, line irc.Line, dst, text string) {
		if enabled {
			derptext := utils.ReplaceAllFold(text, "bitcoin", "derpcoin")
			if derptext != "" {
				plugin.Conn(conn).Privmsg(dst, derptext)
				return
			}
		}
		if subs := btcRegex.FindStringSubmatch(text); subs != nil {
			// we found a construct like "0.01 BTC"
			if val, err := strconv.ParseFloat(subs[1], 64); err == nil {
				const kUpperLimit = 1000
				exchangeRate := float64(rand.Intn(kUpperLimit*100)) / 100.0
				val *= exchangeRate
				val = math.Floor(val*100.0) / 100.0
				msg := fmt.Sprintf("%s: that's almost $%.2f!", line.Src.Nick, val)
				plugin.Conn(conn).Privmsg(dst, msg)
			}
		}
	})
	return nil
}
