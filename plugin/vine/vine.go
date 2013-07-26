package vine

import (
	"../"
	"../../utils"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Vine struct {
	Fullname string
	Message  string
}

func (v Vine) String() string {
	return fmt.Sprintf("%s: %s", v.Fullname, v.Message)
}

func (v Vine) Valid() bool {
	return v.Fullname != "" && v.Message != ""
}

func init() {
	plugin.RegisterPlugin("vine", plugin.Callbacks{Init: setupVine})
}

func setupVine(reg *callback.Registry) error {
	reg.AddCallback("URL", func(conn *irc.Conn, line irc.Line, dst string, url *url.URL) {
		if url.Scheme == "http" || url.Scheme == "https" {
			if url.Host == "vine.co" || url.Host == "www.vine.co" {
				if url.Fragment != "noquote" {
					go processVineURL(plugin.Conn(conn), line, dst, url)
				}
			}
		}
	})
	return nil
}

func processVineURL(conn plugin.IrcConn, line irc.Line, dst string, url *url.URL) {
	resp, err := http.Get(url.String())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, "vine: unexpected response:", resp)
		return
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var vine Vine
	prefixes := utils.PrefixMap(doc)
	if prefixes == nil {
		// Vine is actually quite buggy and doesn't validate
		// Because it has <title> before <head>, this breaks the HTML5 parser
		// and causes the prefix attribute to be lost. Assume og is mapped.
		prefixes = map[string]string{"og": "http://ogp.me/ns#"}
	}
	hf := func(n *html.Node) {
		// process meta tags
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.DataAtom == atom.Meta {
				prop := utils.NodeAttr(c, "property")
				if prop == "" {
					continue
				}
				comps := strings.SplitN(prop, ":", 2)
				if len(comps) < 2 || prefixes[comps[0]] != "http://ogp.me/ns#" {
					continue
				}
				// this is an OpenGraph property
				if comps[1] != "title" {
					continue
				}
				vine.Message = utils.NodeAttr(c, "content")
				return
			}
		}
	}
	var bf func(*html.Node) bool
	bf = func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		classes := utils.ClassMap(n)
		if classes["user"] {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.DataAtom == atom.H2 {
					vine.Fullname = utils.NodeString(c)
					return true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if bf(c) {
				return true
			}
		}
		return false
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.DataAtom == atom.Head {
			hf(n)
		} else if n.DataAtom == atom.Body {
			bf(n)
		} else {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode {
					f(c)
				}
			}
		}
	}

	f(doc)

	if vine.Valid() {
		conn.NoticeN(dst, "\00300,03\002Vine\017 | "+vine.String(), 4)
	} else {
		fmt.Println("Could not find vine in page", url)
	}
}
