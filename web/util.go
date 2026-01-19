package web

import (
	"fmt"
	"net/url"
	"strings"

	g "maragu.dev/gomponents"
	"mvdan.cc/xurls/v2"
)

func linkify(text string) string {
	re := xurls.Relaxed()
	return re.ReplaceAllStringFunc(text, func(match string) string {
		if strings.Contains(match, "@") {
			idxEmail := re.SubexpIndex("relaxedEmail")
			matches := re.FindStringSubmatch(match)
			if matches[idxEmail] != "" {
				// return email as is
				return matches[idxEmail]
			}
		}
		url, err := url.Parse(match)
		if err != nil {
			return match
		}
		if url.Scheme == "" {
			url.Scheme = "https"
		}
		domain, err := getDomain(match)
		return fmt.Sprintf(`<a href="%s">%s</a>`, url.String(), domain)
	})
}

// Return all the links in a string
func getLinks(text string) []string {
	re := xurls.Relaxed()
	return re.FindAllString(text, -1)
}

// return the domain from the url with leading www removed
func getDomain(link string) (string, error) {
	if !strings.HasPrefix(link, "http") {
		link = "https://" + link
	}
	url, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	host := strings.TrimLeft(url.Host, "www.")

	return host, nil
}

func linkifyNode(text string) g.Node {
	return g.Raw(linkify(text))
}
