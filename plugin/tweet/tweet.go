package tweet

import (
	"../"
	"code.google.com/p/go.net/html"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	plugin.RegisterSetup(setupTweet)
}

func setupTweet(hreg irc.HandlerRegistry, reg *callback.Registry) error {
	reg.AddCallback("URL", func(conn *irc.Conn, line irc.Line, dst string, url *url.URL) {
		if url.Scheme == "http" || url.Scheme == "https" {
			if url.Host == "twitter.com" || url.Host == "www.twitter.com" {
				if strings.Contains(url.Path, "/status/") && url.Fragment != "noquote" {
					go processTweetURL(plugin.Conn(conn), line, dst, url.String())
				}
			}
		}
	})
	return nil
}

func processTweetURL(conn plugin.IrcConn, line irc.Line, dst, url string) {
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
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			classes := classMap(n)
			if classes["tweet-text"] {
				tweet.Tweet = nodeString(n)
			} else if classes["tweet-timestamp"] {
				tweet.Timestamp = nodeAttr(n, "title")
			} else if classes["original-tweet"] {
				tweet.Fullname = nodeAttr(n, "data-name")
				username := nodeAttr(n, "data-screen-name")
				if username != "" {
					tweet.Username = "@" + username
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if tweet.Valid() {
		conn.PrivmsgN(dst, "\00310,01\002Twitter\017 | "+tweet.String(), 4)
	} else {
		fmt.Println("Could not find tweet in page", url)
	}
}

func nodeAttr(node *html.Node, attr string) string {
	if node.Type == html.ElementNode {
		for _, at := range node.Attr {
			if at.Namespace == "" && at.Key == attr {
				return at.Val
			}
		}
	}
	return ""
}

func classMap(node *html.Node) map[string]bool {
	if node.Type == html.ElementNode {
		classes := strings.Split(nodeAttr(node, "class"), " ")
		results := make(map[string]bool)
		for _, class := range classes {
			if class != "" {
				results[class] = true
			}
		}
		return results
	}
	return nil
}

func nodeString(node *html.Node) string {
	switch node.Type {
	case html.TextNode:
		return node.Data
	case html.DocumentNode:
		fallthrough
	case html.ElementNode:
		var result string
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			result = result + nodeString(c)
		}
		return result
	}
	return ""
}
