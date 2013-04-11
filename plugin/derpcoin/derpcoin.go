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
	conn.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		text := line.Args[len(line.Args)-1]
		dst := line.Args[0]

		if strings.HasPrefix(dst, "#") {
			comps := strings.SplitN(text, " ", 2)
			if comps[0] == "!derpcoin" {
				if len(comps) > 1 {
					if line.Nick == "Me1000" {
						plugin.Conn(conn).Privmsg(dst, "no")
						return
					}
					flag := strings.ToLower(comps[1])
					if flag == "on" {
						enabled = true
						plugin.Conn(conn).Privmsg(dst, "derpcoin enabled")
					} else if flag == "off" {
						enabled = false
						plugin.Conn(conn).Privmsg(dst, "derpcoin diasbled")
					}
				} else {
					msg := "derpcoin is: "
					if enabled {
						msg += "ON"
					} else {
						msg += "OFF"
					}
					plugin.Conn(conn).Privmsg(dst, msg)
				}
			} else if enabled {
				text = utils.ReplaceAllFold(text, "bitcoin", "derpcoin")
				if text != "" {
					plugin.Conn(conn).Privmsg(dst, text)
				} else if subs := btcRegex.FindStringSubmatch(text); subs != nil {
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
			}
		}
	})
	return nil
}
