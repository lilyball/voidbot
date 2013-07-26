package flickr

import (
	"../"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode"
)

var flickrLogo = "\00312flick\00313r\017"

func init() {
	plugin.RegisterPlugin("flickr", plugin.Callbacks{Init: setupFlickr})
}

var api_key = ""

func setupFlickr(reg *callback.Registry, config map[string]interface{}) error {
	if key, ok := config["api_key"].(string); ok {
		api_key = key
	} else {
		// can't do much without an API key
		return nil
	}
	reg.AddCallback("URL", func(conn *irc.Conn, line irc.Line, dst string, url *url.URL) {
		if url.Host == "flickr.com" || url.Host == "www.flickr.com" {
			if photo_id, set_id, ok := parseFlickrURL(url); ok {
				if photo_id != "" {
					go processFlickrPhoto(plugin.Conn(conn), line, dst, photo_id)
				} else {
					go processFlickrSet(plugin.Conn(conn), line, dst, set_id)
				}
			}
		}
	})
	return nil
}

type PhotoResp struct {
	XMLName xml.Name `xml:"rsp"`
	Stat    string   `xml:"stat,attr"`
	Err     *RespErr `xml:"err,omitempty"`
	Photo   struct {
		ID     string `xml:"id,attr"`
		Secret string `xml:"secret,attr"`
		Media  string `xml:"media,attr"`
		Owner  struct {
			Username string `xml:"username,attr"`
			Realname string `xml:"realname,attr"`
			Location string `xml:"loaction,attr"`
			NSID     string `xml:"nsid,attr"`
		} `xml:"owner"`
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Dates       struct {
			PostedUnix     int64  `xml:"posted,attr"`
			Taken          string `xml:"taken,attr"`
			LastUpdateUnix int64  `xml:"lastupdate,attr"`
		} `xml:"dates"`
		Tags []struct {
			ID     string `xml:"id,attr"`
			Author string `xml:"author,attr"`
			Raw    string `xml:"raw,attr"`
			Name   string `xml:",chardata"`
		} `xml:"tags>tag"`
		URLs []struct {
			Type string `xml:"type,attr"`
			URL  string `xml:",charadata"`
		} `xml:"urls>url"`
	} `xml:"photo"`
}

type RespErr struct {
	Code int    `xml:"code,attr"`
	Msg  string `xml:"msg,attr"`
}

func processFlickrPhoto(conn plugin.IrcConn, line irc.Line, dst, photo_id string) {
	resp, err := callAPI("flickr.photos.getInfo", "photo_id", photo_id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer resp.Body.Close()

	var rsp PhotoResp
	if err := xml.NewDecoder(resp.Body).Decode(&rsp); err != nil {
		fmt.Fprintln(os.Stderr, "flickr:", err)
		return
	}

	if rsp.Stat != "ok" {
		if rsp.Err == nil {
			fmt.Fprintln(os.Stderr, "flickr error (unknown)")
			return
		}
		fmt.Fprintf(os.Stderr, "flickr error %d: %s\n", rsp.Err.Code, rsp.Err.Msg)
		return
	}

	msg := fmt.Sprintf("%s (%s) - %s", rsp.Photo.Owner.Realname, rsp.Photo.Owner.Username, rsp.Photo.Title)
	if rsp.Photo.Media != "photo" {
		msg += fmt.Sprintf(" [%s]", rsp.Photo.Media)
	}
	conn.Notice(dst, flickrLogo+" | "+msg)
}

type PhotosetResp struct {
	XMLName  xml.Name `xml:"rsp"`
	Stat     string   `xml:"stat,attr"`
	Err      *RespErr `xml:"err,omitempty"`
	Photoset struct {
		ID               string `xml:"id,attr"`
		Owner            string `xml:"owner,attr"`
		Username         string `xml:"username,attr"`
		Primary          string `xml:"primary,attr"`
		Secret           string `xml:"secret,attr"`
		Photos           int    `xml:"photos,attr"`
		CreationDateUnix int64  `xml:"date_create,attr"`
		UpdateDateUnix   int64  `xml:"date_update,attr"`
		Title            string `xml:"title"`
		Description      string `xml:"description"`
	} `xml:"photoset"`
}

func processFlickrSet(conn plugin.IrcConn, line irc.Line, dst, set_id string) {
	resp, err := callAPI("flickr.photosets.getInfo", "photoset_id", set_id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer resp.Body.Close()

	var rsp PhotosetResp
	if err := xml.NewDecoder(resp.Body).Decode(&rsp); err != nil {
		fmt.Fprintln(os.Stderr, "flickr:", err)
		return
	}

	if rsp.Stat != "ok" {
		if rsp.Err == nil {
			fmt.Fprintln(os.Stderr, "flickr error (unknown)")
			return
		}
		fmt.Fprintf(os.Stderr, "flickr error %d: %s", rsp.Err.Code, rsp.Err.Msg)
		return
	}

	msg := fmt.Sprintf("%s - %s", rsp.Photoset.Username, rsp.Photoset.Title)
	if rsp.Photoset.Photos == 1 {
		msg += " (1 photo)"
	} else {
		msg += fmt.Sprintf(" (%d photos)", rsp.Photoset.Photos)
	}
	conn.Notice(dst, flickrLogo+" | "+msg)
}

func callAPI(method, key, val string) (*http.Response, error) {
	url := fmt.Sprintf("http://ycpi.api.flickr.com/services/rest/?method=%s&api_key=%s&%s=%s", method, api_key, key, val)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, errors.New(fmt.Sprintf("flickr: unexpected response: %s", resp))
	}

	return resp, nil
}

func parseFlickrURL(url *url.URL) (photo_id, set_id string, ok bool) {
	comps := strings.Split(url.Path, "/")
	if comps[0] == "" {
		comps = comps[1:]
	}
	if len(comps) >= 3 && comps[0] == "photos" {
		if comps[2] == "sets" {
			if len(comps) >= 4 && comps[3] != "" && isASCIIDigitString(comps[3]) {
				return "", comps[3], true
			}
		} else if comps[2] != "" && isASCIIDigitString(comps[2]) {
			return comps[2], "", true
		}
	}
	return "", "", false
}

func isASCIIDigitString(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return !isASCIIDigit(r) }) == -1
}

func isASCIIDigit(r rune) bool {
	return r <= unicode.MaxASCII && unicode.IsDigit(r)
}
