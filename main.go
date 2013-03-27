package main

import (
	_ "code.google.com/p/gosqlite/sqlite"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"os"
	"os/signal"
)

func main() {
	fmt.Println("Bot started")
	conn := irc.SimpleClient("voidbot", "voidbot", "#voidptr bot")
	conn.EnableStateTracking()
	conn.AddHandler(irc.CONNECTED, func(conn *irc.Conn, line *irc.Line) {
		fmt.Println("Connected")
		conn.Join("#voidptr")
	})

	quit := make(chan struct{}, 1)
	conn.AddHandler(irc.DISCONNECTED, func(conn *irc.Conn, line *irc.Line) {
		quit <- struct{}{}
	})

	conn.AddHandler("JOIN", func(conn *irc.Conn, line *irc.Line) {

	})

	conn.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		text := line.Args[len(line.Args)-1]
		dst := line.Args[0]

		fmt.Println(line.Raw)
	})

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	go func() {
		dcsent := 0
		for {
			sig := <-signals
			if sig == os.Interrupt {
				switch dcsent {
				case 0:
					if conn.Connected {
						fmt.Println("Quitting...")
						conn.Quit("Quitting...")
					} else {
						quit <- struct{}{}
						dcsent = dcsent + 1
					}
				case 1:
					quit <- struct{}{}
				case 2:
					os.Exit(0)
				}
				dcsent = dcsent + 1
			}
		}
	}()

	fmt.Println("Connecting...")
	err := conn.Connect("chat.freenode.net")
	if err != nil {
		fmt.Println("error:", err)
		quit <- struct{}{}
	}

	<-quit
	fmt.Println("Goodbye")
}
