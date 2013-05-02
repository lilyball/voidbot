package alpha

import (
	"../"
	"encoding/xml"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"net/http"
	"net/url"
	"strings"
)

func init() {
	plugin.RegisterSetup(setup)
}

func setup(hreg irc.HandlerRegistry, reg *callback.Registry) error {
	reg.AddCallback("COMMAND", func(conn *irc.Conn, line irc.Line, cmd, arg, reply string, isPrivate bool) {
		if cmd == "alpha" {
			arg = strings.TrimSpace(arg)
			if arg == "" {
				plugin.Conn(conn).Privmsg(reply, "!alpha requires an argument")
			} else {
				go runQuery(plugin.Conn(conn), arg, reply)
			}
		}
	})
	return nil
}

type QueryResult struct {
	IsSuccess bool       `xml:"success,attr"`
	IsError   bool       `xml:"error,attr"`
	Pods      []Pod      `xml:"pod"`
	Error     QueryError `xml:"error"`
}

type QueryError struct {
	Code int    `xml:"code"`
	Msg  string `xml:"msg"`
}

type Pod struct {
	Title   string     `xml:"title,attr"`
	IsError bool       `xml:"error,attr"`
	Error   QueryError `xml:"error"`
	Id      string     `xml:"id,attr"`
	Primary bool       `xml:"primary,attr"`
	Subpods []Subpod   `xml:"subpod"`
}

type Subpod struct {
	Plaintext string `xml:"plaintext"`
	MathML    string `xml:"mathml"`
}

var header = "\00304Wolfram\017|\00307Alpha\017"

func runQuery(conn plugin.IrcConn, arg, reply string) {
	resp, err := http.Get(constructURL(arg))
	if err != nil {
		fmt.Println("alpha:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("alpha: unexpected status code:", resp.StatusCode)
		return
	}

	var result QueryResult
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("alpha:", err)
		return
	}

	if !result.IsSuccess {
		if !result.IsError {
			conn.Privmsg(reply, header+" Wolfram|Alpha doesn't know how to interpret your query")
		} else {
			conn.Privmsg(reply, header+" error: "+result.Error.Msg)
		}
	} else if len(result.Pods) == 0 {
		conn.Privmsg(reply, header+" Malformed results from API")
	} else {
		pod := result.Pods[0]
		// use the primary pod if there is one
		// otherwise use one with the title of "Results"
		for _, p := range result.Pods {
			if p.Primary {
				pod = p
				break
			} else if p.Title == "Results" {
				pod = p
			}
		}
		if pod.IsError {
			conn.Privmsg(reply, header+" error: "+pod.Error.Msg)
		} else {
			// use the first subpod with a non-empty plaintext
			var subpod Subpod
			for _, sp := range pod.Subpods {
				if sp.Plaintext != "" {
					subpod = sp
					break
				}
			}
			if subpod.Plaintext == "" {
				conn.Privmsg(reply, header+" Couldn't find plain text representation of answer")
			} else {
				conn.Privmsg(reply, fmt.Sprintf("%s %s: %s", header, pod.Title, subpod.Plaintext))
			}
		}
	}
}

func constructURL(query string) string {
	query = url.QueryEscape(query)
	location := url.QueryEscape("San Francisco, CA")
	appid := url.QueryEscape("P9KHX4-E8QPJ45UTA")
	return fmt.Sprintf("http://api.wolframalpha.com/v2/query?input=%s&appid=%s&format=plaintext&location=%s&excludepodid=Input&podindex=1", query, appid, location)
}
