package utils

import (
	"regexp"
)

var NickRegex = regexp.MustCompile("[a-zA-Z\\x5B-\\x60\\x7B-\\x7D[\\]\\\\`_^{|}][a-zA-Z0-9\\x5B-\\x60\\x7B-\\x7D[\\]\\\\`_^{|}\\-]*")
