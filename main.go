package main

import (
	"./plugin"
	"fmt"
	"github.com/kballard/goirc/irc"
	"io"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"os/signal"
	"syscall"
)

type Config struct {
	Server     string `yaml:"server"`
	Port       uint   `yaml:"port"`
	UseSSL     bool   `yaml:"ssl"`
	ServerPass string `yaml:"serverpass"`

	Nick     string `yaml:"nick"`
	User     string `yaml:"user"`
	RealName string `yaml:"realname"`

	Plugins []string `yaml:"plugins"`
}

func main() {
	config := checkConfig()

	if config.Server == "" {
		fmt.Fprintln(os.Stderr, "error: No valid server found in config.yaml")
		os.Exit(1)
	} else if config.Nick == "" || config.User == "" || config.RealName == "" {
		fmt.Fprintln(os.Stderr, "error: No valid user data found in config.yaml")
		os.Exit(1)
	} else if config.Plugins != nil && len(config.Plugins) == 0 {
		fmt.Fprintln(os.Stderr, "warning: You have no plugins enabled. This bot will do nothing.")
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

	if err := plugin.InvokeInit(config.Plugins); err != nil {
		fmt.Println("error in plugin init:", err)
		plugin.InvokeTeardown()
		return
	}

	discon := make(chan struct{}, 1)
	var stdin *Stdin
	for {
		config := irc.Config{
			Host: config.Server,
			Port: config.Port,
			SSL:  config.UseSSL,

			Nick:     config.Nick,
			User:     config.User,
			RealName: config.RealName,
			Password: config.ServerPass,

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

				plugin.InvokeNewConnection(reg)
			},
		}

		fmt.Println("Connecting...")
		conn, err := irc.Connect(config)
		if err != nil {
			fmt.Println("error:", err)
			plugin.InvokeTeardown()
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

		plugin.InvokeDisconnected()

		if dcsent {
			break
		}
	}

	plugin.InvokeTeardown()

	fmt.Println("Goodbye")
}

func checkConfig() Config {
	bytes, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		if perr, ok := err.(*os.PathError); ok && (perr.Err == os.ErrNotExist || perr.Err == syscall.ENOENT) {
			if err := writeSampleConfig(); err != nil {
				fmt.Fprintf(os.Stderr, "Config file not found. An error occurred while trying to write the sample config: %s\n", err)
				os.Exit(1)
			} else {
				fmt.Fprintln(os.Stderr, "Config file not found. A sample config has been written out as config.yaml")
				os.Exit(2)
			}
		} else {
			fmt.Fprintf(os.Stderr, "An error occurred while reading the config: %s\n", err)
			os.Exit(1)
		}
	}
	var config Config
	if err := goyaml.Unmarshal(bytes, &config); err != nil {
		fmt.Fprintf(os.Stderr, "An error occurred while reading config.yaml: %s", err)
		os.Exit(1)
	}
	return config
}

func writeSampleConfig() error {
	in, err := os.Open("config.yaml.tmpl")
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create("config.yaml")
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return err
}
