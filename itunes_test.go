package itunes_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/deepilla/itunes"
)

func TestToRSS(t *testing.T) {

	data := map[string]struct {
		Paths []string
		Feed  string
		Err   error
	}{
		"Go Time": {
			Paths: []string{
				"podcasts/go-time/itunes-page",
			},
			Feed: "https://changelog.com/gotime/feed",
		},
		"Homecoming": {
			Paths: []string{
				"podcasts/homecoming/itunes-page",
				"podcasts/homecoming/plist",
			},
			Feed: "http://feeds.gimletmedia.com/homecomingshow",
		},
		"Linux Voice": {
			Paths: []string{
				"podcasts/linux-voice/itunes-page",
			},
			Feed: "https://www.linuxvoice.com/podcast_mp3.rss",
		},
		"Longform": {
			Paths: []string{
				"podcasts/longform/itunes-page",
			},
			Feed: "http://longform.libsyn.com/rss",
		},
		"No Such Thing As A Fish": {
			Paths: []string{
				"podcasts/no-such-thing-as-a-fish/itunes-page",
				"podcasts/no-such-thing-as-a-fish/plist",
			},
			Feed: "https://audioboom.com/channels/2399216.rss",
		},
		"Pod Save America": {
			Paths: []string{
				"podcasts/pod-save-america/itunes-page",
				"podcasts/pod-save-america/plist",
			},
			Feed: "http://feeds.feedburner.com/pod-save-america",
		},
		"Revisionist History": {
			Paths: []string{
				"podcasts/revisionist-history/itunes-page",
			},
			Feed: "http://feeds.feedburner.com/RevisionistHistory",
		},
		"S-Town": {
			Paths: []string{
				"podcasts/s-town/itunes-page",
				"podcasts/s-town/plist-1",
				"podcasts/s-town/plist-2",
				"podcasts/s-town/plist-3",
			},
			Feed: "http://feeds.stownpodcast.org/stownpodcast",
		},
		"Serial": {
			Paths: []string{
				"podcasts/serial/itunes-page",
				"podcasts/serial/plist",
			},
			Feed: "http://feeds.serialpodcast.org/serialpodcast",
		},
		"The /Filmcast": {
			Paths: []string{
				"podcasts/filmcast/itunes-page",
			},
			Feed: "http://feeds.feedburner.com/filmcast",
		},
		"Wittertainment": {
			Paths: []string{
				"podcasts/wittertainment/itunes-page",
				"podcasts/wittertainment/plist",
			},
			Feed: "https://podcasts.files.bbci.co.uk/b00lvdrj.rss",
		},

		"No Feed (HTML)": {
			Paths: []string{
				"errors/no-feed/itunes-missing-user-agent",
				"errors/no-feed/itunes-no-episodes",
				"errors/no-feed/itunes-itunesu",
			},
			Err: itunes.ErrNoFeed,
		},
		"No Feed (XML)": {
			Paths: []string{
				"errors/no-feed/plist-item-not-available",
				"errors/no-feed/plist-incomplete",
				"errors/no-feed/plist-blank-url",
			},
			Err: itunes.ErrNoFeed,
		},
		"Too Many Redirects": {
			Paths: []string{
				"errors/too-many-redirects/plist-4",
				"errors/too-many-redirects/plist-recursive",
			},
			Err: errors.New("too many redirects"),
		},
	}

	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	client := validateRequests(t, redirectRequests(ts, http.DefaultClient))

	for name, test := range data {
		for i, url := range test.Paths {

			feed, err := itunes.ToRSSClient(client, url)

			if !equalErrors(err, test.Err) {
				t.Errorf("%s [%d/%d]: expected error %s, got %s", name, i+1, len(test.Paths), formatError(test.Err), formatError(err))
			}

			if feed != test.Feed {
				t.Errorf("%s [%d/%d]: expected feed %q, got %q", name, i+1, len(test.Paths), test.Feed, feed)
			}
		}
	}
}

func TestBadRequest(t *testing.T) {

	data := []struct {
		URL string
		Err error
	}{
		{
			URL: "",
			Err: errors.New(`fetch error: Get : unsupported protocol scheme ""`),
		},
		{
			URL: "pcast://itunes.apple.com/podcasts/123456789",
			Err: errors.New(`fetch error: Get pcast://itunes.apple.com/podcasts/123456789: unsupported protocol scheme "pcast"`),
		},
		{
			URL: "://itunes.apple.com/podcasts/123456789",
			Err: errors.New("bad request: missing protocol scheme"),
		},
		{
			URL: "http://itunes.apple.com/podcasts/123456789#bad%%20escaping",
			Err: errors.New(`bad request: invalid URL escape "%%2"`),
		},
	}

	for _, test := range data {

		_, err := itunes.ToRSS(test.URL)

		if !equalErrors(err, test.Err) {
			t.Errorf("URL %s: expected error %s, got %s", test.URL, formatError(test.Err), formatError(err))
		}
	}
}

func TestBadContentType(t *testing.T) {

	data := []struct {
		ContentType string
		Err         error
	}{
		{
			ContentType: "",
			Err:         errors.New(`bad Content Type "": mime: no media type`),
		},
		{
			ContentType: "text/",
			Err:         errors.New(`bad Content Type "text/": mime: expected token after slash`),
		},
		{
			ContentType: "text/xml; =",
			Err:         errors.New(`bad Content Type "text/xml; =": mime: invalid media parameter`),
		},
		{
			ContentType: "image/png",
			Err:         errors.New(`unexpected Content Type "image/png"`),
		},
		{
			ContentType: "text/plain; charset=utf-8",
			Err:         errors.New(`unexpected Content Type "text/plain; charset=utf-8"`),
		},
	}

	for _, test := range data {

		ts := httptest.NewServer(contentTypeHandler(test.ContentType))
		defer ts.Close()

		client := redirectRequests(ts, http.DefaultClient)

		_, err := itunes.ToRSSClient(client, "")

		if !equalErrors(err, test.Err) {
			t.Errorf("Content Type %q: expected error %s, got %s", test.ContentType, formatError(test.Err), formatError(err))
		}
	}
}

func TestBadHTTPStatus(t *testing.T) {

	statusCodes := []int{
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusGone,
		http.StatusTeapot,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusInternalServerError,
		420, // Enhance Your Calm (Twitter)
		444, // No Response (nginx)
		600, // Non-existent
	}

	for _, code := range statusCodes {

		ts := httptest.NewServer(errorHandler(code))
		defer ts.Close()

		exp := fmt.Errorf("bad HTTP Status: %d status code %d", code, code)
		if msg := http.StatusText(code); msg != "" {
			exp = fmt.Errorf("bad HTTP Status: %d %s", code, msg)
		}

		client := redirectRequests(ts, http.DefaultClient)

		_, err := itunes.ToRSSClient(client, "")

		if !equalErrors(err, exp) {
			t.Errorf("Status %d: expected error %s, got %s", code, formatError(exp), formatError(err))
		}
	}
}

func TestFetchError(t *testing.T) {

	msgs := []string{
		"it was nearly eleven when I started to return",
		"the night was unexpectedly dark",
		"to me, walking out of the lighted passage of my cousin's house",
		"it seemed indeed black",
	}

	for _, s := range msgs {

		exp := errors.New("fetch error: " + s)
		client := clientFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New(s)
		})

		_, err := itunes.ToRSSClient(client, "")

		if !equalErrors(err, exp) {
			t.Errorf("expected error %s, got %s", formatError(exp), formatError(err))
		}
	}
}

func contentTypeHandler(typ string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", typ)
	})
}

func errorHandler(code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "helpful error message", code)
	})
}

type clientFunc func(req *http.Request) (*http.Response, error)

func (f clientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func redirectRequests(ts *httptest.Server, client itunes.Client) itunes.Client {
	return clientFunc(func(req *http.Request) (*http.Response, error) {

		newURL := ts.URL + "/" + req.URL.Path

		u, err := url.Parse(newURL)
		if err != nil {
			return nil, err
		}

		req.URL = u
		return client.Do(req)
	})
}

func validateRequests(t *testing.T, client itunes.Client) itunes.Client {
	return clientFunc(func(req *http.Request) (*http.Response, error) {

		if got, exp := req.Method, "GET"; got != exp {
			t.Fatalf("Bad request: expected Method %q, got %q", exp, got)
		}

		if got, exp := req.Header.Get("User-Agent"), "iTunes/10.1"; got != exp {
			t.Fatalf("Bad request: expected User Agent %q, got %q", exp, got)
		}

		if req.Body != nil {
			b, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("Bad request: Expected nil body, got something unreadable (read error: %s)", err)
			}
			t.Fatalf("Bad request: Expected nil body, got %q", string(b))
		}

		return client.Do(req)
	})
}

func formatError(err error) string {
	if err == nil {
		return "Nil"
	}
	return "\"" + err.Error() + "\""
}

func equalErrors(err1, err2 error) bool {
	switch {
	case err1 == err2:
		return true
	case err1 == nil, err2 == nil:
		return false
	default:
		return err1.Error() == err2.Error()
	}
}
