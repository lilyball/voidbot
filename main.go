package main

import (
	"./plugin"
	"fmt"
	"github.com/kballard/goirc/irc"
	"os"
	"os/signal"
)

func main() {
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

	discon := make(chan struct{}, 1)
	var stdin *Stdin
	for {
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
					discon <- struct{}{}
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

				reg.AddHandler(irc.CTCP, func(conn *irc.Conn, line irc.Line) {
					fmt.Printf("Received CTCP[%s] from %s [%s]: %s\n", line.Args[0], line.Src.Nick, line.Src.Ident(), append(line.Args[1:len(line.Args)], "")[0])
					if line.Args[0] == "VERSION" {
						plugin.Conn(conn).CTCPReply(line.Src.Nick, "VERSION", "voidbot powered by github.com/kballard/goirc")
					} else {
						conn.DefaultCTCPHandler(line)
					}
				})

				plugin.InvokeSetup(reg)
			},
		}

		fmt.Println("Connecting...")
		conn, err := irc.Connect(config)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		if stdin != nil {
			stdin.ReplaceConn(conn)
		} else {
			stdin = NewStdin(conn)
			go stdin.Run(interrupt)
		}

		dcsent := false
	loop:
		for {
			select {
			case <-interrupt:
				if dcsent {
					break loop
				}
				dcsent = true
				fmt.Println("Quitting...")
				if !conn.Quit("Quitting...") {
					break loop
				}
			case <-discon:
				break loop
			}
		}

		if dcsent {
			break
		}
	}

	plugin.InvokeTeardown()

	fmt.Println("Goodbye")
}
