package derpcoin

import (
	"../"
	"../../utils"
	"fmt"
	"github.com/fluffle/goevent/event"
	irc "github.com/fluffle/goirc/client"
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

func setup(conn *irc.Conn, er event.EventRegistry) error {
	er.AddHandler(event.NewHandler(func(args ...interface{}) {
		conn, line, cmd := args[0].(*irc.Conn), args[1].(*irc.Line), args[2].(string)
		isPrivate := args[5].(bool)

		if cmd == "derpcoin" && !isPrivate {
			arg, reply := args[3].(string), args[4].(string)
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
				if line.Nick == "Me1000" {
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
	}), "COMMAND")
	er.AddHandler(event.NewHandler(func(args ...interface{}) {
		conn, line, dst, text := args[0].(*irc.Conn), args[1].(*irc.Line), args[2].(string), args[3].(string)

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
				msg := fmt.Sprintf("%s: that's almost $%.2f!", line.Nick, val)
				plugin.Conn(conn).Privmsg(dst, msg)
			}
		}
	}), "PRIVMSG")
	return nil
}
