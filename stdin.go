package main

import (
	"bufio"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"io"
	"os"
	"strings"
)

func handleStdin(conn *irc.Conn, quit chan<- bool) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.TrimSpace(text) != "" {
			handleInputLine(conn, text)
		}
	}
	if err := scanner.Err(); err != nil {
		if err == io.EOF {
			quit <- false
		} else {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
	}
}

func handleInputLine(conn *irc.Conn, text string) {
	words := strings.SplitN(text, " ", 2)
	cmd := words[0]
	if !strings.HasPrefix(cmd, "/") {
		return
	}
	cmd = cmd[1:]

	if f, ok := inputCommands[cmd]; ok {
		f(conn, words[1])
	}
}

var inputCommands = map[string]func(*irc.Conn, string){
	"raw": func(conn *irc.Conn, text string) {
		fmt.Println(text)
		conn.Raw(text)
	},
	"msg": func(conn *irc.Conn, text string) {
		words := strings.SplitN(text, " ", 2)
		if len(words) != 2 || words[0] == "" || words[1] == "" {
			fmt.Fprintln(os.Stderr, "usage: /msg target text")
			return
		}
		fmt.Printf("--> %s: %s\n", words[0], words[1])
		conn.Privmsg(words[0], words[1])
	},
	"notice": func(conn *irc.Conn, text string) {
		words := strings.SplitN(text, " ", 2)
		if len(words) != 2 || words[0] == "" || words[1] == "" {
			fmt.Fprintln(os.Stderr, "usage: /notice target text")
			return
		}
		fmt.Printf("--> NOTICE[%s]: %s\n", words[0], words[1])
		conn.Notice(words[0], words[1])
	},
	"me": func(conn *irc.Conn, text string) {
		words := strings.SplitN(text, " ", 2)
		if len(words) != 2 || words[0] == "" || words[1] == "" {
			fmt.Fprintln(os.Stderr, "usage: /me target text")
			return
		}
		fmt.Printf("--> %s ACTION: %s %s\n", words[0], conn.Me.Nick, words[1])
		conn.Action(words[0], words[1])
	},
}
