package appdotnet

import (
	"../"
	"encoding/json"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Post struct {
	Username  string
	Fullname  string
	Text      string
	Timestamp time.Time
}

func (p Post) String() string {
	ts := p.Timestamp.Local().Format("3:04 PM - 2 Jan 06")
	return fmt.Sprintf("%s (%s) at %s: %s", p.Fullname, p.Username, ts, p.Text)
}

func init() {
	plugin.RegisterPlugin("appdotnet", plugin.Callbacks{Init: setup})
}

func setup(reg *callback.Registry, config map[string]interface{}) error {
	reg.AddCallback("URL", func(conn *irc.Conn, line irc.Line, dst string, url *url.URL) {
		if url.Scheme == "http" || url.Scheme == "https" {
			if url.Host == "alpha.app.net" {
				comps := strings.Split(strings.TrimLeft(url.Path, "/"), "/")
				if len(comps) > 2 && comps[1] == "post" {
					id := comps[2]
					go fetchADNPost(plugin.Conn(conn), line, dst, id)
				}
			}
		}
	})
	return nil
}

type Payload struct {
	Data struct {
		Text string `json:"text"`
		User struct {
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"user"`
		Timestamp time.Time `json:"created_at"`
	} `json:"data"`
}

func fetchADNPost(conn plugin.IrcConn, line irc.Line, dst, id string) {
	url := fmt.Sprintf("https://alpha-api.app.net/stream/0/posts/%s", id)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("appdotnet:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("appdotnet: unexpected response:", resp)
		return
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("appdotnet:", err)
		return
	}

	var payload Payload
	if err = json.Unmarshal(respData, &payload); err != nil {
		fmt.Println("appdotnet:", err)
		return
	}

	var post = Post{
		Username:  payload.Data.User.Username,
		Fullname:  payload.Data.User.Name,
		Text:      payload.Data.Text,
		Timestamp: payload.Data.Timestamp,
	}

	conn.NoticeN(dst, post.String(), 4)
}
