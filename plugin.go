package main

import (
	"fmt"
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

func RegisterPluginSetup(f func(*irc.Conn) error) {
	pluginSetup.Lock()
	defer pluginSetup.Unlock()
	if pluginSetup.Done != nil {
		panic("setup was already invoked")
	}
	pluginSetup.Funcs = append(pluginSetup.Funcs, func(args ...interface{}) error {
		return f(args[0].(*irc.Conn))
	})
}

func RegisterPluginTeardown(f func() error) {
	pluginTeardown.Lock()
	defer pluginTeardown.Unlock()
	if pluginTeardown.Done != nil {
		panic("teardown was already invoked")
	}
	pluginTeardown.Funcs = append(pluginTeardown.Funcs, func(args ...interface{}) error {
		return f()
	})
}

func invokePluginSetup(conn *irc.Conn) {
	invoke(&pluginSetup, "setup", func() {
		invokePluginTeardown()
		os.Exit(1)
	}, conn)
}

func invokePluginTeardown() {
	invoke(&pluginTeardown, "teardown", nil)
}

func invoke(info *setupInfo, name string, onErr func(), args ...interface{}) {
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
		return true
	}() {
		for _, f := range funcs {
			if err := f(args...); err != nil {
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
