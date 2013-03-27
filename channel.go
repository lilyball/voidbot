package main

import (
	"fmt"
	irc "github.com/fluffle/goirc/client"
	state "github.com/fluffle/goirc/state"
)

type Channel struct {
	State *state.Channel
	Conn  *irc.Conn
}
