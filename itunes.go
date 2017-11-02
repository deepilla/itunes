// Package itunes extracts the underlying RSS feed from an iTunes page.
package itunes

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"

	"golang.org/x/net/html"
)

const iTunesUA = "iTunes/10.1"
const maxRedirects = 3

// ErrNoFeed is returned by the ToRSS functions when they
// fail to find an RSS feed in the given iTunes page. This
// usually means that the function has been called on an
// unsupported page type, such as a non-podcast iTunes page
// or an iTunesU page.
var ErrNoFeed = errors.New("no feed found")

// A Client is responsible for executing HTTP requests.
// Its interface is satisfied by http.Client. Provide your
// own implementation to intercept requests and responses.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// ToRSS returns the underlying RSS feed from an iTunes URL
// using the default HTTP client.
func ToRSS(url string) (string, error) {
	return ToRSSClient(nil, url)
}

// ToRSSClient returns the underlying RSS feed from an
// iTunes URL using the provided Client.
func ToRSSClient(c Client, url string) (string, error) {

	if c == nil {
		c = http.DefaultClient
	}

	feed, err := processURL(c, url, 0)
	if err == io.EOF {
		err = ErrNoFeed
	}

	return feed, err
}

func processURL(c Client, url string, redirects int) (string, error) {

	resp, err := fetch(c, url)
	if err != nil {
		return "", fmt.Errorf("fetch error: %s", err)
	}
	defer resp.Body.Close()

	ctype := resp.Header.Get("Content-Type")
	media, _, err := mime.ParseMediaType(ctype)
	if err != nil {
		return "", fmt.Errorf("bad Content Type %q: %s", ctype, err)
	}

	switch media {
	case "text/html":
		return processHTML(resp.Body)

	case "text/xml", "application/xml":
		next, err := processXML(resp.Body)
		if err != nil {
			return "", err
		}
		redirects++
		if redirects > maxRedirects {
			return "", errors.New("too many redirects")
		}

		return processURL(c, next, redirects)

	default:
		return "", fmt.Errorf("unexpected Content Type %q", ctype)
	}
}

func processHTML(r io.Reader) (string, error) {

	var attr, val []byte

	tagButton := []byte("button")
	attrFeed := []byte("feed-url")

	z := html.NewTokenizer(r)

	for {
		tt := z.Next()

		if tt == html.ErrorToken {
			break
		}

		if tt != html.StartTagToken {
			continue
		}

		tag, hasAttrs := z.TagName()
		if !bytes.Equal(tag, tagButton) {
			continue
		}

		for hasAttrs {
			attr, val, hasAttrs = z.TagAttr()
			if bytes.Equal(attr, attrFeed) && len(val) > 0 {
				return string(val), nil
			}
		}
	}

	return "", z.Err()
}

// Matches <key>url</key><string>path/to/itunes-page</string>
var reGoto = regexp.MustCompile(`^<key>url</key><string>(\S+)</string>$`)

func processXML(r io.Reader) (string, error) {

	prevLine := []byte("<key>kind</key><string>Goto</string>")

	scanner := bufio.NewScanner(r)

	for scanner.Scan() {

		if !bytes.Equal(scanner.Bytes(), prevLine) {
			continue
		}

		if !scanner.Scan() {
			break
		}

		matches := reGoto.FindSubmatch(scanner.Bytes())
		if len(matches) != 2 {
			continue
		}

		// Unescape URL.
		// e.g. https://itunes.apple.com/WebObjects/DZR.woa/wa/viewPodcast?urlDesc=&amp;id=1234567890
		// becomes https://itunes.apple.com/WebObjects/DZR.woa/wa/viewPodcast?urlDesc=&id=1234567890
		return html.UnescapeString(string(matches[1])), nil
	}

	err := scanner.Err()
	if err == nil {
		// If Scan() returns false but Err() is nil,
		// we've reached the end of the input.
		err = io.EOF
	}

	return "", err
}

func newRequest(u string) (*http.Request, error) {

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		if e, ok := err.(*url.Error); ok {
			err = e.Err
		}
		return nil, err
	}

	// Make requests look like they come from iTunes.
	req.Header.Set("User-Agent", iTunesUA)

	return req, nil
}

func fetch(c Client, url string) (*http.Response, error) {

	req, err := newRequest(url)
	if err != nil {
		return nil, fmt.Errorf("bad URL: %s", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP Status %s", resp.Status)
	}

	return resp, nil
}
