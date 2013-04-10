package utils

import (
	"regexp"
	"strconv"
	"strings"
)

var colorRegex = regexp.MustCompile(`\002|\003\d{0,2}(?:,?\d{1,2})|\017`)

func ColorToANSI(s string) string {
	bold := false
	color := false
	result := colorRegex.ReplaceAllStringFunc(s, func(sub string) string {
		switch sub[0] {
		case 2:
			if bold {
				bold = false
				return "\033[22m"
			}
			bold = true
			return "\033[1m"
		case 3:
			cols := strings.Split(sub[1:], ",")
			result := ""
			if cols[0] != "" {
				fg, _ := strconv.Atoi(cols[0]) // can't fail
				result += "38;5;" + strconv.Itoa(mircToANSI[fg])
				if len(cols) > 1 {
					result += ";"
				}
			}
			if len(cols) > 1 {
				bg, _ := strconv.Atoi(cols[1]) // can't fail
				result += "48;5;" + strconv.Itoa(mircToANSI[bg])
			}
			if result == "" {
				result = "\033[39;49m"
				color = false
			} else {
				result = "\033[" + result + "m"
				color = true
			}
			return result
		case 15:
			bold, color = false, false
			return "\033[0m"
		}
		return ""
	})
	if bold || color {
		result = result + "\033[0m"
	}
	return result
}

var mircToANSI = map[int]int{
	0:  7,  // white
	1:  0,  // black
	2:  4,  // blue
	3:  2,  // green
	4:  1,  // red
	5:  88, // maroon
	6:  5,  // purple
	7:  3,  // yellow
	8:  11, // bright yellow
	9:  10, // bright green
	10: 6,  // cyan
	11: 14, // bright cyan
	12: 12, // bright blue
	13: 13, // bright purple
	14: 8,  // dark grey
	15: 7,  // light grey
}
