package youtube

import (
	"../"
	"encoding/xml"
	"errors"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"github.com/kballard/gocallback/callback"
	"net/http"
	"net/url"
	"time"
)

type Video struct {
	Title    string
	Duration time.Duration
	Key      string
}

func (v Video) String() string {
	dur := fmt.Sprintf("%d:%02d", int(v.Duration.Minutes()), int(v.Duration.Seconds())%60)
	return fmt.Sprintf("%s | %s | https://youtu.be/%s", v.Title, dur, v.Key)
}

func init() {
	plugin.RegisterSetup(setup)
}

func setup(conn *irc.Conn, reg *callback.Registry) error {
	reg.AddCallback("URL", func(conn *irc.Conn, line *irc.Line, urlStr string) {
		u, err := url.Parse(urlStr)
		if err != nil {
			fmt.Println("youtube:", err)
			return
		}
		if u.Scheme == "http" || u.Scheme == "https" {
			if u.Host == "youtube.com" || u.Host == "www.youtube.com" {
				if u.Path == "/watch" {
					if key, ok := u.Query()["v"]; ok && key != nil {
						go handleYoutubeVideo(conn, line, key[0])
					}
				}
			}
		}
	})
	return nil
}

func handleYoutubeVideo(conn *irc.Conn, line *irc.Line, key string) {
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

	dst := line.Args[0]
	plugin.Conn(conn).Privmsg(dst, "\0031,15You\0030,5Tube\017 | "+v.String())
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
