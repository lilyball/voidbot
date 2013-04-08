package main

import (
	"database/sql"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"regexp"
	"strings"
	"time"
)

var URLRegex = regexp.MustCompile("(?i)\\b((?:[a-z][\\w-]+:(?:/{1,3}|[a-z0-9%])|www\\d{0,3}[.]|[a-z0-9.\\-]+[.][a-z]{2,4}/)(?:[^\\s()<>]+|\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\))+(?:\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\)|[^\\s`!()\\[\\]{};:'\".,<>?«»“”‘’]))")

var historyDB *sql.DB

func init() {
	RegisterPluginSetup(setupURLs)
	RegisterPluginTeardown(teardownURLs)
}

func setupURLs(conn *irc.Conn) error {
	var err error
	historyDB, err = sql.Open("sqlite3", "./history.db")
	if err != nil {
		return err
	}

	sqls := []string{
		"CREATE TABLE IF NOT EXISTS seen (id integer not null primary key, url text not null, nick text, src text not null, dst text not null, timestamp datetime not null)",
		"CREATE INDEX IF NOT EXISTS url_idx ON seen (url, dst)",
	}
	for _, sqlstr := range sqls {
		_, err = historyDB.Exec(sqlstr)
		if err != nil {
			return err
		}
	}

	conn.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		text := line.Args[len(line.Args)-1]
		dst := line.Args[0]

		if strings.HasPrefix(dst, "#") {
			matches := URLRegex.FindAllStringSubmatch(text, -1)
			if matches != nil {
				for _, submatches := range matches {
					url := submatches[1]
					handleURL(conn, historyDB, line, dst, url)
				}
			}
		}
	})

	conn.AddHandler("JOIN", func(conn *irc.Conn, line *irc.Line) {
		if line.Nick == conn.Me.Nick {
			fmt.Printf("! Channel %s joined\n", line.Args[0])
		}
	})
	return nil
}

func teardownURLs() error {
	if historyDB != nil {
		return historyDB.Close()
	}
	return nil
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

		msg := fmt.Sprintf("URL '%s' was last seen %s ago by %s (%d total)", url, lastSeen, nick, count)
		fmt.Printf("--> NOTICE[%s]: %s", dst, msg)
		conn.Notice(dst, msg)
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
