package plugin

import (
	"fmt"
	"github.com/fluffle/goevent/event"
	irc "github.com/fluffle/goirc/client"
	"os"
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
