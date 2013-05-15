package plugin

import (
	"../utils"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"strings"
	"sync"
	"time"
)

type Callbacks struct {
	Init          func(*callback.Registry) error
	Teardown      func() error
	NewConnection func(irc.HandlerRegistry)
	Disconnected  func()
	inited        bool
}

const (
	StatePreInit = iota
	StatePostInit
	StatePostTeardown
)

var pluginState struct {
	sync.Mutex
	Callbacks []Callbacks
	State     int
}

func RegisterCallbacks(callbacks Callbacks) {
	pluginState.Lock()
	defer pluginState.Unlock()
	if pluginState.State != StatePreInit {
		panic("setup was already invoked")
	}
	pluginState.Callbacks = append(pluginState.Callbacks, callbacks)
}

// Some utility functions for connections
type IrcConn struct {
	conn irc.SafeConn
}

func Conn(conn *irc.Conn) IrcConn {
	return IrcConn{conn.SafeConn()}
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

func (c IrcConn) Privmsg(dst, msg string) {
	c.PrivmsgN(dst, msg, -1)
}

func (c IrcConn) PrivmsgN(dst, msg string, n int) {
	lines := msgToLinesN(msg, n)
	for _, line := range lines {
		logLine("%s: %s", dst, utils.ColorToANSI(line))
		c.conn.Privmsg(dst, line)
	}
}

func (c IrcConn) Notice(dst, msg string) {
	c.NoticeN(dst, msg, -1)
}

func (c IrcConn) NoticeN(dst, msg string, n int) {
	lines := msgToLinesN(msg, n)
	for _, line := range lines {
		logLine("NOTICE[%s]: %s\n", dst, utils.ColorToANSI(line))
		c.conn.Notice(dst, line)
	}
}

func (c IrcConn) Action(dst, msg string) {
	c.ActionN(dst, msg, -1)
}

func (c IrcConn) ActionN(dst, msg string, n int) {
	lines := msgToLinesN(msg, n)
	for _, line := range lines {
		logLine("ACTION[%s]: %s %s\n", dst, c.conn.Me(), utils.ColorToANSI(line))
		c.conn.Action(dst, line)
	}
}

func (c IrcConn) CTCPReply(dst, cmd, args string) {
	c.CTCPReplyN(dst, cmd, args, -1)
}

func (c IrcConn) CTCPReplyN(dst, cmd, args string, n int) {
	lines := msgToLinesN(args, n)
	for _, line := range lines {
		logLine("[ctcp(%s)] %s %s", dst, cmd, utils.ColorToANSI(line))
		c.conn.CTCPReply(dst, cmd, line)
	}
}

var registry *callback.Registry

// InvokeInit stops at the first error
func InvokeInit() error {
	pluginState.Lock()
	defer pluginState.Unlock()
	if pluginState.State != StatePreInit {
		panic("InvokeInit called after init")
	}
	pluginState.State = StatePostInit
	registry = callback.NewRegistry(callback.DispatchSerial)
	for _, callbacks := range pluginState.Callbacks {
		if callbacks.Init != nil {
			if err := callbacks.Init(registry); err != nil {
				return err
			}
		}
		callbacks.inited = true
	}
	return nil
}

func InvokeNewConnection(reg irc.HandlerRegistry) {
	pluginState.Lock()
	defer pluginState.Unlock()
	if pluginState.State != StatePostInit {
		panic("InvokeNewConnection called in wrong state")
	}
	for _, callbacks := range pluginState.Callbacks {
		if callbacks.NewConnection != nil {
			callbacks.NewConnection(reg)
		}
	}
}

func InvokeDisconnected() {
	pluginState.Lock()
	defer pluginState.Unlock()
	if pluginState.State != StatePostInit {
		panic("InvokeDisconnected called in wrong state")
	}
	for _, callbacks := range pluginState.Callbacks {
		if callbacks.Disconnected != nil {
			callbacks.Disconnected()
		}
	}
}

func InvokeTeardown() {
	pluginState.Lock()
	defer pluginState.Unlock()
	if pluginState.State != StatePostInit {
		panic("InvokeTeardown called in wrong state")
	}
	pluginState.State = StatePostTeardown

	for _, callbacks := range pluginState.Callbacks {
		if callbacks.inited && callbacks.Teardown != nil {
			if err := callbacks.Teardown(); err != nil {
				fmt.Println("error during teardown:", err)
			}
		}
		callbacks.inited = false
	}
}

// Other miscellaneous utility functions for plugins

// AllowedPrivmsgTextLength returns the amount of text that can be safely given
// to a Privmsg command with the given destination.
func AllowedPrivmsgTextLength(dst string) int {
	return 510 - len("PRIVMSG ") - len(dst) - len(" :")
}
