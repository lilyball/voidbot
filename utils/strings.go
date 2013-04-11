package utils

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

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
		result = result + MatchCase(repl, s[idx:end])

		last = end
		idx, end = IndexFold(s, sub, end)
	}
	if result != "" {
		result = result + s[last:]
	}
	return result
}

func MatchCase(s, src string) string {
	if src == "" {
		return s
	}

	// if s is longer than src, re-use the last char from src repeatedly.
	// if src is longer than s, only use the first len(s) chars of src
	runes, runesrc := []rune(s), []rune(src)

	for si, srci := 0, 0; si < len(runes); si++ {
		c := runesrc[srci]
		if unicode.IsUpper(c) {
			runes[si] = unicode.ToUpper(runes[si])
		} else if unicode.IsLower(c) {
			runes[si] = unicode.ToLower(runes[si])
		}
		if srci < len(runesrc)-1 {
			srci++
		}
	}

	return string(runes)
}
