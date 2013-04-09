package plugin

import (
	"fmt"
	"github.com/fluffle/goevent/event"
	irc "github.com/fluffle/goirc/client"
	"os"
	"strings"
	"sync"
)

type setupInfo struct {
	sync.Mutex
	Funcs []func(...interface{}) error
	Done  chan struct{}
}

var (
	pluginSetup    setupInfo
	pluginTeardown setupInfo
)

func RegisterSetup(f func(*irc.Conn, event.EventRegistry) error) {
	pluginSetup.Lock()
	defer pluginSetup.Unlock()
	if pluginSetup.Done != nil {
		panic("setup was already invoked")
	}
	pluginSetup.Funcs = append(pluginSetup.Funcs, func(args ...interface{}) error {
		return f(args[0].(*irc.Conn), args[1].(event.EventRegistry))
	})
}

func RegisterTeardown(f func() error) {
	pluginTeardown.Lock()
	defer pluginTeardown.Unlock()
	if pluginTeardown.Done != nil {
		panic("teardown was already invoked")
	}
	pluginTeardown.Funcs = append(pluginTeardown.Funcs, func(args ...interface{}) error {
		return f()
	})
}

// Some utility functions for connections
type IrcConn irc.Conn

func Conn(conn *irc.Conn) *IrcConn {
	return (*IrcConn)(conn)
}

func msgToLines(msg string) []string {
	f := func(r rune) bool {
		return r == '\n' || r == '\r'
	}
	return strings.FieldsFunc(strings.Trim(msg, "\n\r"), f)
}

func msgToLinesN(msg string, n int) []string {
	lines := msgToLines(msg)
	if n >= 0 && len(lines) > n {
		line := fmt.Sprintf("...%d lines omitted...", len(lines)-n+1)
		lines = lines[:n]
		lines[n-1] = line
	}
	return lines
}

func (c *IrcConn) Privmsg(dst, msg string) {
	c.PrivmsgN(dst, msg, -1)
}

func (c *IrcConn) PrivmsgN(dst, msg string, n int) {
	lines := msgToLinesN(msg, n)
	for _, line := range lines {
		fmt.Printf("--> %s: %s\n", dst, line)
		(*irc.Conn)(c).Privmsg(dst, line)
	}
}

func (c *IrcConn) Notice(dst, msg string) {
	c.NoticeN(dst, msg, -1)
}

func (c *IrcConn) NoticeN(dst, msg string, n int) {
	lines := msgToLinesN(msg, n)
	for _, line := range lines {
		fmt.Printf("--> NOTICE[%s]: %s\n", dst, line)
		(*irc.Conn)(c).Notice(dst, line)
	}
}

func (c *IrcConn) Action(dst, msg string) {
	c.ActionN(dst, msg, -1)
}

func (c *IrcConn) ActionN(dst, msg string, n int) {
	lines := msgToLinesN(msg, n)
	for _, line := range lines {
		fmt.Printf("--> ACTION[%s]: %s %s\n", dst, c.Me.Nick, line)
		(*irc.Conn)(c).Action(dst, line)
	}
}

var pluginER event.EventRegistry

func InvokeSetup(conn *irc.Conn) {
	invoke(&pluginSetup, "setup", func() {
		pluginER = event.NewRegistry()
	}, func() {
		InvokeTeardown()
		os.Exit(1)
	}, func(f func(...interface{}) error) error {
		return f(conn, pluginER)
	})
}

func InvokeTeardown() {
	invoke(&pluginTeardown, "teardown", nil, nil, func(f func(...interface{}) error) error {
		return f()
	})
}

func invoke(info *setupInfo, name string, onInit func(), onErr func(), call func(func(...interface{}) error) error) {
	var funcs []func(...interface{}) error
	var done chan struct{}
	if func() bool {
		info.Lock()
		defer info.Unlock()
		if info.Done != nil {
			done = info.Done
			return false
		}
		funcs = make([]func(...interface{}) error, len(info.Funcs))
		copy(funcs, info.Funcs)
		done = make(chan struct{}, 1)
		info.Done = done
		if onInit != nil {
			onInit()
		}
		return true
	}() {
		for _, f := range funcs {
			if err := call(f); err != nil {
				fmt.Fprintln(os.Stderr, err)
				if onErr != nil {
					onErr()
				}
			}
		}
		done <- struct{}{}
	} else {
		done <- <-done
	}
}
