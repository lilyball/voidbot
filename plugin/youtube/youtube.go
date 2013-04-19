package youtube

import (
	"../"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Video struct {
	Title    string
	Duration time.Duration
	Key      string
	Fragment string
}

func (v Video) String() string {
	dur := fmt.Sprintf("%d:%02d", int(v.Duration.Minutes()), int(v.Duration.Seconds())%60)
	frag := ""
	if v.Fragment != "" {
		frag = "#" + v.Fragment
	}
	return fmt.Sprintf("%s | %s | https://youtu.be/%s%s", v.Title, dur, v.Key, frag)
}

func init() {
	plugin.RegisterSetup(setup)
}

func setup(hreg irc.HandlerRegistry, reg *callback.Registry) error {
	reg.AddCallback("URL", func(conn *irc.Conn, line irc.Line, dst string, url *url.URL) {
		if url.Scheme == "http" || url.Scheme == "https" {
			if url.Host == "youtube.com" || url.Host == "www.youtube.com" {
				if url.Path == "/watch" {
					if key, ok := url.Query()["v"]; ok && key != nil {
						go handleYoutubeVideo(plugin.Conn(conn), line, dst, key[0], url.Fragment)
					}
				}
			} else if url.Host == "youtu.be" {
				go handleYoutubeVideo(plugin.Conn(conn), line, dst, strings.TrimLeft(url.Path, "/"), url.Fragment)
			}
		}
	})
	return nil
}

func handleYoutubeVideo(conn plugin.IrcConn, line irc.Line, dst, key, fragment string) {
	url := fmt.Sprintf("http://gdata.youtube.com/feeds/api/videos/%s", key)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("youtube:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("youtube: unexpected response:", resp)
		return
	}

	d := xml.NewDecoder(resp.Body)

	v, err := parseFeed(d)
	if err != nil {
		fmt.Println("youtube:", err)
		return
	}

	v.Key = key
	v.Fragment = fragment

	conn.Privmsg(dst, "\0031,15You\0030,5Tube\017 | "+v.String())
}

type Feed struct {
	Groups []struct {
		Titles []struct {
			Type  string `xml:"type,attr"`
			Value string `xml:",chardata"`
		} `xml:"http://search.yahoo.com/mrss/ title"`
		Duration struct {
			Value int `xml:"seconds,attr"`
		} `xml:"http://gdata.youtube.com/schemas/2007 duration"`
	} `xml:"http://search.yahoo.com/mrss/ group"`
}

func parseFeed(d *xml.Decoder) (v Video, err error) {
	var feed Feed
	err = d.Decode(&feed)
	if err != nil {
		return
	}

	if len(feed.Groups) == 0 || len(feed.Groups[0].Titles) == 0 {
		err = errors.New("invalid feed")
		return
	}

	group := feed.Groups[0]
	found := false
	for _, title := range group.Titles {
		if title.Type == "plain" {
			v.Title = title.Value
			found = true
			break
		}
	}
	if !found {
		v.Title = group.Titles[0].Value
	}

	v.Duration = time.Duration(group.Duration.Value) * time.Second

	return
}
