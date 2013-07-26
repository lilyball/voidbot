package utils

import (
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"strings"
)

func NodeAttr(node *html.Node, attr string) string {
	if node.Type == html.ElementNode {
		for _, at := range node.Attr {
			if at.Namespace == "" && at.Key == attr {
				return at.Val
			}
		}
	}
	return ""
}

func ClassMap(node *html.Node) map[string]bool {
	if node.Type == html.ElementNode {
		classes := strings.Split(NodeAttr(node, "class"), " ")
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

func NodeString(node *html.Node) string {
	switch node.Type {
	case html.TextNode:
		return node.Data
	case html.DocumentNode:
		fallthrough
	case html.ElementNode:
		var result string
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			result = result + NodeString(c)
		}
		return result
	}
	return ""
}

// Call prefixMap on the document element
func PrefixMap(node *html.Node) map[string]string {
	if node.Type != html.DocumentNode {
		return nil
	}
	var docElt *html.Node
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			docElt = c
			break
		}
	}
	if docElt == nil {
		// can the document's FirstChild ever be nil?
		return nil
	}
	for n := docElt.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.ElementNode && n.DataAtom == atom.Head && n.Namespace == "" {
			for _, a := range n.Attr {
				if a.Namespace == "" && a.Key == "prefix" {
					prefix := make(map[string]string)
					// The format of the prefix is a whitespace-separated list of
					// NCName ':' ' '+ xsd:anyURI
					// To make parsing easier, I'm just going to split on whitespace
					// and verify that each key is of the form ^[^:]+:$
					fields := strings.Fields(a.Val)
					for i := 0; i < len(fields)-1; i += 2 {
						key := fields[i]
						url := fields[i+1]
						if len(key) > 1 && key[len(key)-1] == ':' && strings.IndexRune(key[:len(key)-1], ':') == -1 {
							prefix[key[:len(key)-1]] = url
						}
					}
					return prefix
				}
			}
			break
		}
	}
	return nil
}
