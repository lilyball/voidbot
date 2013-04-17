package main

import (
	"./plugin"
	"fmt"
	"github.com/kballard/goirc/irc"
	"os"
	"os/signal"
)

func main() {
	quit := make(chan struct{}, 1)
	config := irc.Config{
		Host: "chat.freenode.net",
		SSL:  true,

		Nick:     "voidbot",
		User:     "voidbot",
		RealName: "#voidptr bot",
		Password: "voidbot:redacted",

		Init: func(reg irc.HandlerRegistry) {
			fmt.Println("Bot started")
			reg.AddHandler(irc.CONNECTED, func(conn *irc.Conn, line irc.Line) {
				fmt.Println("Connected")
				conn.Join([]string{"#voidptr"}, nil)
			})

			reg.AddHandler(irc.DISCONNECTED, func(conn *irc.Conn, line irc.Line) {
				quit <- struct{}{}
			})

			reg.AddHandler("PRIVMSG", func(conn *irc.Conn, line irc.Line) {
				dst := line.Args[0]

				if dst == conn.Me().Nick {
					fmt.Println(line.Raw)
				}
			})

			reg.AddHandler("NOTICE", func(conn *irc.Conn, line irc.Line) {
				dst := line.Args[0]

				if dst == conn.Me().Nick {
					fmt.Println(line.Raw)
				}
			})

			reg.AddHandler("JOIN", func(conn *irc.Conn, line irc.Line) {
				if line.SrcIsMe() {
					fmt.Printf("! Channel %s joined\n", line.Args[0])
				}
			})

			reg.AddHandler("PART", func(conn *irc.Conn, line irc.Line) {
				if line.SrcIsMe() {
					fmt.Printf("! Channel %s left\n", line.Args[0])
				}
			})

			plugin.InvokeSetup(reg)
		},
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	interrupt := make(chan struct{}, 1)
	go func() {
		for {
			sig := <-signals
			if sig == os.Interrupt {
				interrupt <- struct{}{}
			}
		}
	}()

	fmt.Println("Connecting...")
	conn, err := irc.Connect(config)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	go handleStdin(conn, interrupt)

	dcsent := false
loop:
	for {
		select {
		case <-interrupt:
			if !dcsent {
				fmt.Println("Quitting...")
				if !conn.Quit("Quitting...") {
					break loop
				}
				dcsent = true
			} else {
				break loop
			}
		case <-quit:
			break loop
		}
	}

	plugin.InvokeTeardown()

	fmt.Println("Goodbye")
}
