package tweet

import (
	"../"
	"../../utils"
	"code.google.com/p/go.net/html"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode"
)

type Tweet struct {
	Username  string
	Fullname  string
	Tweet     string
	Timestamp string
}

func (t Tweet) String() string {
	return fmt.Sprintf("%s (%s) at %s: %s", t.Fullname, t.Username, t.Timestamp, t.Tweet)
}

func (t Tweet) Valid() bool {
	return t.Username != "" && t.Fullname != "" && t.Tweet != "" && t.Timestamp != ""
}

func init() {
	plugin.RegisterPlugin("tweet", plugin.Callbacks{Init: setupTweet})
}

func setupTweet(reg *callback.Registry) error {
	reg.AddCallback("URL", func(conn *irc.Conn, line irc.Line, dst string, url *url.URL) {
		if url.Scheme == "http" || url.Scheme == "https" {
			if url.Host == "twitter.com" || url.Host == "www.twitter.com" {
				if url.Fragment != "noquote" {
					username, tweet_id := parseTwitterURL(url)
					if username != "" && tweet_id != "" {
						go processTweetURL(plugin.Conn(conn), line, dst, username, tweet_id)
					}
				}
			}
		}
	})
	return nil
}

func parseTwitterURL(url *url.URL) (username, tweet_id string) {
	comps := strings.Split(url.Path, "/")
	if comps[0] == "" {
		comps = comps[1:]
	}
	if len(comps) >= 3 && (comps[1] == "status" || comps[1] == "statuses") &&
		len(comps[2]) > 0 && strings.IndexFunc(comps[2], func(r rune) bool { return !unicode.IsDigit(r) }) == -1 {
		username, tweet_id = comps[0], comps[2]
	}
	return
}

func processTweetURL(conn plugin.IrcConn, line irc.Line, dst, username, tweet_id string) {
	url := fmt.Sprintf("http://twitter.com/%s/status/%s", username, tweet_id)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, "tweet: unexpected response:", resp)
		return
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var tweet Tweet
	var tf func(*html.Node)
	tf = func(n *html.Node) {
		if n.Type == html.ElementNode {
			classes := utils.ClassMap(n)
			if classes["tweet-text"] {
				tweet.Tweet = utils.NodeString(n)
			} else if classes["tweet-timestamp"] {
				tweet.Timestamp = utils.NodeAttr(n, "title")
			} else if classes["original-tweet"] {
				tweet.Fullname = utils.NodeAttr(n, "data-name")
				username := utils.NodeAttr(n, "data-screen-name")
				if username != "" {
					tweet.Username = "@" + username
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			tf(c)
		}
	}
	var f func(*html.Node) bool
	f = func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			classes := utils.ClassMap(n)
			if classes["permalink-tweet"] {
				tf(n)
				return true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if f(c) {
				return true
			}
		}
		return false
	}
	f(doc)

	if tweet.Valid() {
		conn.PrivmsgN(dst, "\00310,01\002Twitter\017 | "+tweet.String(), 4)
	} else {
		fmt.Println("Could not find tweet in page", url)
	}
}
