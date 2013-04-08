package derpcoin

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

func setup(conn *irc.Conn, er event.EventRegistry) error {
	conn.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		text := line.Args[len(line.Args)-1]
		dst := line.Args[0]

		if strings.HasPrefix(dst, "#") {
			text = ReplaceAllFold(text, "bitcoin", "derpcoin")
			if text != "" {
				fmt.Printf("--> NOTICE[%s]: %s\n", dst, text)
				conn.Notice(dst, text)
			}
		}
	})
	return nil
}

func IndexFold(s, sub string, offset int) (int, int) {
	sub = strings.ToLower(sub)
	n := len(sub)
	if n == 0 {
		return 0, 0
	}
	sc, _ := utf8.DecodeRuneInString(sub)
	for i := offset; i+n <= len(s); {
		c, size := utf8.DecodeRuneInString(s[i:])
		if unicode.ToLower(c) == sc {
			fail := false
			var i_, j int
			for i_, j = i, 0; j < n; {
				c, size := utf8.DecodeRuneInString(s[i_:])
				c2, size2 := utf8.DecodeRuneInString(sub[j:])
				if unicode.ToLower(c) != c2 {
					fail = true
					break
				}
				i_ += size
				j += size2
			}
			if !fail {
				return i, i_
			}
		}
		i += size
	}
	return -1, -1
}

// returns "" if there was no replacement
func ReplaceAllFold(s, sub, repl string) string {
	result := ""
	idx, end := IndexFold(s, sub, 0)
	last := 0
	for idx != -1 {
		if last != idx {
			result = result + s[last:idx]
		}
		result = result + MatchCase(repl, result[idx:end])

		last = end
		idx, end = IndexFold(s, sub, end)
	}
	if result != "" {
		result = result + s[last:]
	}
	return result
}

func MatchCase(s, src string) string {
	// if s is longer than src, re-use the first char from src repeatedly.
	// if src is longer than s, only use the first len(s) chars of src
	runes, runesrc := []rune(s), []rune(src)

	si, srci := 0, 0
	for len(runes)-si > len(runesrc)-srci {
		c := runesrc[srci]
		if unicode.IsUpper(c) {
			runes[si] = unicode.ToUpper(runes[si])
		} else if unicode.IsLower(c) {
			runes[si] = unicode.ToLower(runes[si])
		}
		si++
	}
	for ; si < len(runes); si, srci = si+1, srci+1 {
		c := runesrc[srci]
		if unicode.IsUpper(c) {
			runes[si] = unicode.ToUpper(runes[si])
		} else if unicode.IsLower(c) {
			runes[si] = unicode.ToLower(runes[si])
		}
	}

	return string(runes)
}
