package plugin

import (
	"../utils"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"github.com/kballard/gocallback/callback"
	"os"
	"strings"
	"sync"
	"time"
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

func RegisterSetup(f func(*irc.Conn, *callback.Registry) error) {
	pluginSetup.Lock()
	defer pluginSetup.Unlock()
	if pluginSetup.Done != nil {
		panic("setup was already invoked")
	}
	pluginSetup.Funcs = append(pluginSetup.Funcs, func(args ...interface{}) error {
		return f(args[0].(*irc.Conn), args[1].(*callback.Registry))
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

func logLine(format string, args ...interface{}) {
	args = append([]interface{}{time.Now().Format("15:04")}, args...)
	fmt.Printf("%s --> "+format+"\n", args...)
}

func (c *IrcConn) Privmsg(dst, msg string) {
	c.PrivmsgN(dst, msg, -1)
}

func (c *IrcConn) PrivmsgN(dst, msg string, n int) {
	lines := msgToLinesN(msg, n)
	for _, line := range lines {
		logLine("%s: %s", dst, utils.ColorToANSI(line))
		(*irc.Conn)(c).Privmsg(dst, line)
	}
}

func (c *IrcConn) Notice(dst, msg string) {
	c.NoticeN(dst, msg, -1)
}

func (c *IrcConn) NoticeN(dst, msg string, n int) {
	lines := msgToLinesN(msg, n)
	for _, line := range lines {
		logLine("NOTICE[%s]: %s\n", dst, utils.ColorToANSI(line))
		(*irc.Conn)(c).Notice(dst, line)
	}
}

func (c *IrcConn) Action(dst, msg string) {
	c.ActionN(dst, msg, -1)
}

func (c *IrcConn) ActionN(dst, msg string, n int) {
	lines := msgToLinesN(msg, n)
	for _, line := range lines {
		logLine("ACTION[%s]: %s %s\n", dst, c.Me.Nick, utils.ColorToANSI(line))
		(*irc.Conn)(c).Action(dst, line)
	}
}

var registry *callback.Registry

func InvokeSetup(conn *irc.Conn) {
	invoke(&pluginSetup, "setup", func() {
		registry = callback.NewRegistry(callback.DispatchSerial)
	}, func() {
		InvokeTeardown()
		os.Exit(1)
	}, func(f func(...interface{}) error) error {
		return f(conn, registry)
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
