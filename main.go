package main

import (
	"database/sql"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"os/signal"
	"regexp"
	"time"
)

var URLRegex = regexp.MustCompile("(?i)\\b((?:[a-z][\\w-]+:(?:/{1,3}|[a-z0-9%])|www\\d{0,3}[.]|[a-z0-9.\\-]+[.][a-z]{2,4}/)(?:[^\\s()<>]+|\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\))+(?:\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\)|[^\\s`!()\\[\\]{};:'\".,<>?«»“”‘’]))")

func main() {
	// load up the sqlite3 database

	db, err := sql.Open("sqlite3", "./history.db")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	sqls := []string{
		"CREATE TABLE IF NOT EXISTS seen (id integer not null primary key, url text not null, nick text, src text not null, dst text not null, timestamp datetime not null)",
		"CREATE INDEX IF NOT EXISTS url_idx ON seen (url, dst)",
	}
	for _, sqlstr := range sqls {
		_, err = db.Exec(sqlstr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q: %s\n", err, sqlstr)
			os.Exit(1)
		}
	}

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

	conn.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		text := line.Args[len(line.Args)-1]
		dst := line.Args[0]

		if dst == "#voidptr" {
			matches := URLRegex.FindAllStringSubmatch(text, -1)
			if matches != nil {
				for _, submatches := range matches {
					url := submatches[1]
					handleURL(conn, db, line, dst, url)
				}
			}
		}
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
	err = conn.Connect("chat.freenode.net")
	if err != nil {
		fmt.Println("error:", err)
		quit <- struct{}{}
	}

	<-quit
	fmt.Println("Goodbye")
}

func handleURL(conn *irc.Conn, db *sql.DB, line *irc.Line, dst string, url string) {
	tx, err := db.Begin()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer func() {
		sqlstr := "INSERT INTO seen (url, nick, src, dst, timestamp) VALUES (?, ?, ?, ?, ?)"
		_, err := tx.Exec(sqlstr, url, line.Nick, line.Src, dst, time.Now())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q: %s\n", err, sqlstr)
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}()

	sqlstr := "SELECT nick, src, timestamp FROM seen WHERE url = ? AND dst = ? ORDER BY id DESC LIMIT 1"
	row := tx.QueryRow(sqlstr, url, dst)

	var nick, src string
	var timestamp time.Time
	err = row.Scan(&nick, &src, &timestamp)
	if err != sql.ErrNoRows {
		if err != nil && err != sql.ErrNoRows {
			fmt.Fprintf(os.Stderr, "%q: %s\n", err, sqlstr)
			return
		}

		if nick == "" {
			nick = src
		}

		sqlstr = "SELECT COUNT(*) FROM seen WHERE url = ? AND dst = ?"
		row = tx.QueryRow(sqlstr, url, dst)

		var count int
		err = row.Scan(&count)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		now := time.Now()
		delta := now.Sub(timestamp)
		lastSeen := formatDuration(delta)

		conn.Notice(dst, fmt.Sprintf("URL '%s' was last seen %s ago by %s (%d total)", url, lastSeen, nick, count))
	}
}

func formatDuration(d time.Duration) string {
	h := int64(d.Hours())
	if h >= 24 {
		days := h / 24
		return pluralize(days, "day")
	} else if h >= 1 {
		return pluralize(h, "hour")
	}
	m := int64(d.Minutes())
	if m >= 1 {
		return pluralize(m, "minute")
	}
	s := int64(d.Seconds())
	return pluralize(s, "second")
}

func pluralize(count int64, text string) string {
	if count > 1 {
		return fmt.Sprintf("%d %ss", count, text)
	}
	return fmt.Sprintf("%d %s", count, text)
}
