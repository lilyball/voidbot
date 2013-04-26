package vimeo

import (
	"../"
	"encoding/xml"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

func init() {
	plugin.RegisterSetup(setup)
}

type Video struct {
	Title    string `xml:"title"`
	URL      string `xml:"url"`
	Duration int    `xml:"duration"`
}

func (v Video) String() string {
	dur := time.Second * time.Duration(v.Duration)
	durs := fmt.Sprintf("%d:%02d", int(dur.Minutes()), int(dur.Seconds())%60)
	return fmt.Sprintf("%s | %s | %s", v.Title, durs, v.URL)
}

func setup(hreg irc.HandlerRegistry, reg *callback.Registry) error {
	reg.AddCallback("URL", func(conn *irc.Conn, line irc.Line, dst string, url *url.URL) {
		if url.Scheme == "http" || url.Scheme == "https" {
			if url.Host == "vimeo.com" || url.Host == "www.vimeo.com" {
				path := url.Path
				if strings.HasPrefix(path, "/") {
					path = path[1:]
				}
				if len(path) > 0 && strings.IndexFunc(path, func(r rune) bool { return !unicode.IsDigit(r) }) == -1 {
					go handleVimeo(plugin.Conn(conn), line, dst, path)
				}
			}
		}
	})
	return nil
}

func handleVimeo(conn plugin.IrcConn, line irc.Line, dst, video_id string) {
	resp, err := http.Get(fmt.Sprintf("http://vimeo.com/api/v2/video/%s.xml", video_id))
	if err != nil {
		fmt.Println("vimeo:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("vimeo: unexpected status code", resp.StatusCode)
		return
	}
	var videos struct {
		Video Video `xml:"video"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&videos); err != nil {
		fmt.Println("vimeo:", err)
		return
	}
	conn.Privmsg(dst, "\0030,11vimeo\017 | "+videos.Video.String())
}
