package main

import (
	"bufio"
	"fmt"
	"github.com/kballard/goirc/irc"
	"io"
	"os"
	"strings"
)

type Stdin struct {
	c chan irc.SafeConn
}

func NewStdin(conn irc.SafeConn) *Stdin {
	c := make(chan irc.SafeConn, 1)
	c <- conn
	return &Stdin{c: c}
}

func (s *Stdin) WithConn(f func(conn irc.SafeConn)) {
	conn := <-s.c
	defer func() { s.c <- conn }()
	f(conn)
}

func (s *Stdin) Run(quit chan<- struct{}) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.TrimSpace(text) != "" {
			s.WithConn(func(conn irc.SafeConn) {
				handleInputLine(conn, text)
			})
		}
	}
	if err := scanner.Err(); err != nil {
		if err == io.EOF {
			quit <- struct{}{}
		} else {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
	}
}

func (s *Stdin) ReplaceConn(conn irc.SafeConn) {
	<-s.c
	s.c <- conn
}

func handleInputLine(conn irc.SafeConn, text string) {
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

var inputCommands = map[string]func(irc.SafeConn, string){
	"raw": func(conn irc.SafeConn, text string) {
		fmt.Println(text)
		conn.Raw(text)
	},
	"msg": func(conn irc.SafeConn, text string) {
		words := strings.SplitN(text, " ", 2)
		if len(words) != 2 || words[0] == "" || words[1] == "" {
			fmt.Fprintln(os.Stderr, "usage: /msg target text")
			return
		}
		fmt.Printf("--> %s: %s\n", words[0], words[1])
		conn.Privmsg(words[0], words[1])
	},
	"notice": func(conn irc.SafeConn, text string) {
		words := strings.SplitN(text, " ", 2)
		if len(words) != 2 || words[0] == "" || words[1] == "" {
			fmt.Fprintln(os.Stderr, "usage: /notice target text")
			return
		}
		fmt.Printf("--> NOTICE[%s]: %s\n", words[0], words[1])
		conn.Notice(words[0], words[1])
	},
	"me": func(conn irc.SafeConn, text string) {
		words := strings.SplitN(text, " ", 2)
		if len(words) != 2 || words[0] == "" || words[1] == "" {
			fmt.Fprintln(os.Stderr, "usage: /me target text")
			return
		}
		fmt.Printf("--> %s ACTION: %s %s\n", words[0], conn.Me().Nick, words[1])
		conn.Action(words[0], words[1])
	},
	"nick": func(conn irc.SafeConn, text string) {
		words := strings.SplitN(text, " ", 2)
		if len(words) != 1 || words[0] == "" {
			fmt.Fprintln(os.Stderr, "usage: /nick nickname")
			return
		}
		fmt.Printf("--> NICK: %s\n", words[0])
		conn.Nick(words[0])
	},
}
