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
	plugin.RegisterPlugin("alpha", plugin.Callbacks{Init: setup})
}

func setup(reg *callback.Registry) error {
	reg.AddCallback("COMMAND", func(conn *irc.Conn, line irc.Line, cmd, arg, reply string, isPrivate bool) {
		if cmd == "alpha" {
			arg = strings.TrimSpace(arg)
			if arg == "" {
				plugin.Conn(conn).Notice(reply, "!alpha requires an argument")
			} else {
				go runQuery(plugin.Conn(conn), arg, reply)
			}
		}
	})
	return nil
}

type QueryResult struct {
	IsSuccess     bool         `xml:"success,attr"`
	IsError       bool         `xml:"error,attr"`
	ParseTimedOut bool         `xml:"parsetimedout,attr"`
	TimedOut      string       `xml:"timedout,attr"`
	Recalculate   string       `xml:"recalculate,attr"`
	Pods          []Pod        `xml:"pod"`
	Error         QueryError   `xml:"error"`
	DidYouMeans   []DidYouMean `xml:"didyoumeans>didyoumean"`
	Tips          []Tip        `xml:"tips>tip"`
	FutureTopic   *FutureTopic `xml:"futuretopic"`
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
	Title     string `xml:"title,attr"`
	Primary   bool   `xml:"primary,attr"`
	Plaintext string `xml:"plaintext"`
	MathML    string `xml:"mathml"`
}

type DidYouMean struct {
	Score float32 `xml:"score,attr"`
	Level string  `xml:"level,attr"`
	Text  string  `xml:",chardata"`
}

type Tip struct {
	Text string `xml:"text,attr"`
}

type FutureTopic struct {
	Topic string `xml:"topic,attr"`
	Msg   string `xml:"msg,attr"`
}

var header = "\00304Wolfram\017|\00307Alpha\017"

func runQuery(conn plugin.IrcConn, arg, reply string) {
	runAPICall(conn, reply, arg, constructURL(arg), true, true)
}

func runAPICall(conn plugin.IrcConn, reply, query, url string, reinterpret, recalculate bool) {
	resp, err := http.Get(url)
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

	if !reinterpret {
		conn.Notice(reply, header+" Using closest Wolfram|Alpha interpretation: "+query)
	}
	if !result.IsSuccess {
		if result.ParseTimedOut {
			conn.Notice(reply, header+" error: parse timed out")
		} else if !result.IsError {
			if len(result.DidYouMeans) > 0 && recalculate {
				text := result.DidYouMeans[0].Text
				runAPICall(conn, reply, text, constructURL(text), false, true)
			} else if result.FutureTopic != nil {
				conn.Notice(reply, fmt.Sprintf("%s %s: %s", header, result.FutureTopic.Topic, result.FutureTopic.Msg))
			} else {
				msg := header + " Wolfram|Alpha doesn't know how to interpret your query"
				if len(result.Tips) > 0 {
					msg += ". " + result.Tips[0].Text
				}
				conn.Notice(reply, msg)
			}
		} else {
			conn.NoticeN(reply, header+" error: "+result.Error.Msg, 5)
		}
	} else if len(result.Pods) == 0 {
		if result.Recalculate != "" && recalculate {
			runAPICall(conn, reply, query, result.Recalculate, reinterpret, false)
		} else if result.TimedOut != "" {
			conn.Notice(reply, header+" timed out: "+result.TimedOut)
		} else {
			conn.Notice(reply, header+" Malformed results from API")
		}
	} else {
		pod := result.Pods[0]
		if pod.Id == "Input" && len(result.Pods) > 1 {
			pod = result.Pods[1]
		}
		// use the primary pod if there is one
		// otherwise use one with the title of "Result"
		for _, p := range result.Pods {
			if p.Primary {
				pod = p
				break
			} else if p.Title == "Result" {
				pod = p
			}
		}
		if pod.IsError {
			conn.NoticeN(reply, header+" error: "+pod.Error.Msg, 5)
		} else {
			// use the primary subpod, if it has a non-empty plaintext.
			// otherwise, use the first subpod with a non-empty plaintext
			var subpod *Subpod
			for _, sp := range pod.Subpods {
				if sp.Plaintext != "" {
					if sp.Primary {
						subpod = &sp
						break
					} else if subpod == nil {
						subpod = &sp
					}
				}
			}
			if subpod == nil {
				conn.Notice(reply, header+" Couldn't find plain text representation of answer")
			} else {
				conn.NoticeN(reply, fmt.Sprintf("%s %s: %s", header, pod.Title, subpod.Plaintext), 5)
			}
		}
	}
}

func constructURL(query string) string {
	query = url.QueryEscape(query)
	location := url.QueryEscape("San Francisco, CA")
	appid := url.QueryEscape("P9KHX4-E8QPJ45UTA")
	return fmt.Sprintf("http://api.wolframalpha.com/v2/query?input=%s&appid=%s&format=plaintext&location=%s&podindex=1,2", query, appid, location)
}
