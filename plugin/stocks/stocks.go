package stocks

import (
	"../"
	"encoding/xml"
	"fmt"
	"github.com/kballard/gocallback/callback"
	"github.com/kballard/goirc/irc"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func init() {
	plugin.RegisterSetup(setup)
}

var stockRegex = regexp.MustCompile("\\$[A-Z]{1,4}\\b")

type QueryResult struct {
	Quotes []Quote `xml:"results>quote"`
}

type Quote struct {
	Symbol             string `xml:"symbol,attr"`
	Name               string
	Ask                string
	Bid                string // monetary
	AskRealtime        string // monetary
	BidRealtime        string // monetary
	Change             string
	Open               string // monetary
	PreviousClose      string // monetary
	DaysLow            string // monetary
	DaysHigh           string // monetary
	PercentChange      string
	LastTradePriceOnly string // monetary
}

func setup(hreg irc.HandlerRegistry, reg *callback.Registry) error {
	reg.AddCallback("PRIVMSG", func(conn *irc.Conn, line irc.Line, dst, text string) {
		if matches := stockRegex.FindAllString(text, -1); matches != nil {
			for i, match := range matches {
				matches[i] = match[1:] // trim off the $
			}
			if dst == conn.Me().Nick {
				dst = line.Src.Nick
			}
			if dst == "" {
				// wtf?
				return
			}
			go queryStocks(matches, plugin.Conn(conn), dst)
		}
	})
	return nil
}

func queryStocks(stocks []string, conn plugin.IrcConn, reply string) {
	query := buildQuery(stocks)
	req := fmt.Sprintf("http://query.yahooapis.com/v1/public/yql?q=%s&env=%s", url.QueryEscape(query), url.QueryEscape("store://datatables.org/alltableswithkeys"))
	resp, err := http.Get(req)
	if err != nil {
		fmt.Println("stocks:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("stocks: unexpected code %d for query %s", resp.StatusCode, req)
		return
	}
	var result QueryResult
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("stocks:", err)
		return
	}
	if len(result.Quotes) == 0 {
		fmt.Println("stocks: Got no quotes back for query", req)
		return
	}
	quotes := formatQuotes(result.Quotes)
	if len(quotes) == 0 {
		return
	}
	// return multiple quotes on one line.
	maxLength := plugin.AllowedPrivmsgTextLength(reply)
	lines := make([]string, 1)
	const sep = "  |  "
	for _, quote := range quotes {
		lenq := len(quote)
		line := lines[len(lines)-1]
		if line != "" {
			lenq += len(sep)
		}
		if len(line)+lenq > maxLength {
			line = quote
			lines = append(lines, line)
		} else if line == "" {
			lines[len(lines)-1] = quote
		} else {
			lines[len(lines)-1] = line + sep + quote
		}
	}
	for _, line := range lines {
		conn.Privmsg(reply, line)
	}
}

func buildQuery(stocks []string) string {
	quoted := make([]string, len(stocks))
	for i, stock := range stocks {
		quoted[i] = "\"" + stock + "\""
	}
	return fmt.Sprintf("select * from yahoo.finance.quotes where symbol in (%s)", strings.Join(quoted, ","))
}

// formatQuotes formats the quotes into strings, and removes any invalid quotes
func formatQuotes(quotes []Quote) []string {
	results := make([]string, 0, len(quotes))
	for _, quote := range quotes {
		if validateQuote(quote) {
			results = append(results, formatQuote(quote))
		}
	}
	return results
}

func validateQuote(quote Quote) bool {
	if quote.LastTradePriceOnly == "0.00" {
		return false
	}
	return true
}

func formatQuote(quote Quote) string {
	resp := fmt.Sprintf("%s (%s):", quote.Symbol, quote.Name)
	// TODO: once I figure out how to distinguish regular hours from after hours
	// then we can display the two. Until then, just display regular hours
	/*var col string
	if strings.HasPrefix(quote.Change, "+") {
		col = "\00303"
	} else {
		col = "\00304"
	}
	resp = fmt.Sprintf("%s %s%s (%s)\017", resp, col, quote.Change, quote.PercentChange)*/
	if quote.LastTradePriceOnly == "" {
		fmt.Println("stocks: LastTradePriceOnly is missing")
		resp = resp + " error"
	} else {
		resp = fmt.Sprintf("%s $%s", resp, quote.LastTradePriceOnly)
	}
	return resp
}
