package sed

import (
	".."
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"regexp"
)

func init() {
	plugin.RegisterSetup(setup)
}

type Line struct {
	Msg    string
	Action bool
}

var channels map[string]map[string]Line // map[channel name]map[nickname]Line

var sedRegex = regexp.MustCompile(`^s/((?:\\/|[^/])+)/((?:\\/|[^/])+)/([ig]*)$`)

func setup(hreg irc.HandlerRegistry, reg *callback.Registry) error {
	channels = make(map[string]map[string]Line)
	reg.AddCallback("PRIVMSG", func(conn *irc.Conn, line irc.Line, dst, text string) {
		if line.Src.Nick == "" {
			return
		}
		if matches := sedRegex.FindStringSubmatch(text); matches != nil {
			processMatches(conn, line, dst, matches)
		} else {
			lines := channels[dst]
			if lines == nil {
				lines = make(map[string]Line)
				channels[dst] = lines
			}
			lines[line.Src.Nick] = Line{Msg: text, Action: false}
		}
	})
	reg.AddCallback("ACTION", func(conn *irc.Conn, line irc.Line, dst, text string, isPrivate bool) {
		if line.Src.Nick == "" || isPrivate {
			return
		}
		if matches := sedRegex.FindStringSubmatch(text); matches != nil {
			processMatches(conn, line, dst, matches)
		} else {
			lines := channels[dst]
			if lines == nil {
				lines = make(map[string]Line)
				channels[dst] = lines
			}
			lines[line.Src.Nick] = Line{Msg: text, Action: true}
		}
	})
	hreg.AddHandler("PART", func(conn *irc.Conn, line irc.Line) {
		if len(line.Args) < 1 {
			return
		}
		if line.Src.Nick == "" {
			return
		}
		dst := line.Args[0]
		if lines, ok := channels[dst]; ok {
			delete(lines, line.Src.Nick)
		}
	})
	hreg.AddHandler("QUIT", func(conn *irc.Conn, line irc.Line) {
		if line.Src.Nick == "" {
			return
		}
		for _, lines := range channels {
			delete(lines, line.Src.Nick)
		}
	})
	hreg.AddHandler("KICK", func(conn *irc.Conn, line irc.Line) {
		if len(line.Args) < 2 {
			return
		}
		dst := line.Args[0]
		nick := line.Args[1]
		if lines, ok := channels[dst]; ok {
			delete(lines, nick)
		}
	})
	hreg.AddHandler("NICK", func(conn *irc.Conn, line irc.Line) {
		if len(line.Args) < 1 {
			return
		}
		src := line.Src.Nick
		nick := line.Args[0]
		for _, lines := range channels {
			if line, ok := lines[src]; ok {
				lines[nick] = line
				delete(lines, src)
			}
		}
	})
	return nil
}

func processMatches(conn *irc.Conn, line irc.Line, dst string, matches []string) {
	if lines := channels[dst]; lines != nil {
		src := line.Src.Nick
		if line, ok := lines[src]; ok {
			pat := matches[1]
			ignorecase, global := false, false
			if matches[3] != "" {
				for _, c := range matches[3] {
					switch c {
					case 'i':
						ignorecase = true
					case 'g':
						global = true
					}
				}
			}
			if ignorecase {
				pat = "(?i)" + pat
			}
			re, err := regexp.Compile(pat)
			if err != nil {
				fmt.Printf("sed: bad regexp %s: %v\n", pat, err)
				return
			}
			var result string
			if global {
				result = re.ReplaceAllString(line.Msg, matches[2])
			} else {
				indices := re.FindStringSubmatchIndex(line.Msg)
				if indices == nil {
					return
				}
				bresult := []byte(line.Msg[0:indices[0]])
				bresult = re.ExpandString(bresult, matches[2], line.Msg, indices)
				bresult = append(bresult, line.Msg[indices[1]:]...)
				result = string(bresult)
			}
			if result != line.Msg {
				line.Msg = result
				lines[src] = line
				if line.Action {
					result = src + " " + result
				}
				conn.Privmsg(dst, fmt.Sprintf("%s meant to say: %s", src, result))
			}
		}
	}
}
