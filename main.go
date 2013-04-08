package main

import (
	"./plugin"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"os"
	"os/signal"
)

func main() {
	conn := irc.SimpleClient("voidbot", "voidbot", "#voidptr bot")
	conn.EnableStateTracking()

	plugin.InvokeSetup(conn)

	fmt.Println("Bot started")
	conn.AddHandler(irc.CONNECTED, func(conn *irc.Conn, line *irc.Line) {
		fmt.Println("Connected")
		conn.Join("#voidptr")
	})

	quit := make(chan bool, 1)
	conn.AddHandler(irc.DISCONNECTED, func(conn *irc.Conn, line *irc.Line) {
		quit <- true
	})

	conn.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		dst := line.Args[0]

		if dst == conn.Me.Nick {
			fmt.Println(line.Raw)
		}
	})

	conn.AddHandler("NOTICE", func(conn *irc.Conn, line *irc.Line) {
		dst := line.Args[0]

		if dst == conn.Me.Nick {
			fmt.Println(line.Raw)
		}
	})

	conn.AddHandler("JOIN", func(conn *irc.Conn, line *irc.Line) {
		if line.Nick == conn.Me.Nick {
			fmt.Printf("! Channel %s joined\n", line.Args[0])
		}
	})

	conn.AddHandler("PART", func(conn *irc.Conn, line *irc.Line) {
		if line.Nick == conn.Me.Nick {
			fmt.Printf("! Channel %s left\n", line.Args[0])
		}
	})

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	go func() {
		for {
			sig := <-signals
			if sig == os.Interrupt {
				quit <- false
			}
		}
	}()

	fmt.Println("Connecting...")
	conn.SSL = true
	err := conn.Connect("chat.freenode.net", "voidbot:redacted")
	if err != nil {
		fmt.Println("error:", err)
		quit <- true
	}

	go handleStdin(conn, quit)

	dcsent := false
	for {
		flag := <-quit
		if !flag && !dcsent && conn.Connected {
			fmt.Println("Quitting...")
			conn.Quit("Quitting...")
			dcsent = true
		} else {
			break
		}
	}

	plugin.InvokeTeardown()

	fmt.Println("Goodbye")
}
