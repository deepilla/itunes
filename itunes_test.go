package itunes_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
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
		"No Feed": {
			Paths: []string{
				"errors/no-feed/itunes-missing-user-agent",
				"errors/no-feed/itunes-no-episodes",
				"errors/no-feed/itunes-itunesu",
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

func TestBadURL(t *testing.T) {

	urls := []string{
		"",
		"https://",
		"://itunes.apple.com/podcasts/123456789",
		"1ttps://itunes.apple.com/podcasts/123456789",
		"http://itunes.apple.com/podcasts/123456789#bad%%20escaping",
	}

	skipped := 0

	for _, u := range urls {

		_, err := url.Parse(u)
		if err == nil {
			skipped++
			t.Logf("Warning: Parse(%q) didn't return an error", u)
			continue
		}

		if e, ok := err.(*url.Error); ok {
			err = e.Err
		}

		exp := fmt.Errorf("fetch error: bad URL: %s", err)
		_, got := itunes.ToRSS(u)

		if !equalErrors(got, exp) {
			t.Errorf("URL %q: expected error %s, got %s", u, formatError(exp), formatError(got))
		}
	}

	if skipped == len(urls) {
		t.Errorf("No requests tested")
	}
}

func TestClientError(t *testing.T) {

	msgs := []string{
		"it was nearly eleven when I started to return",
		"the night was unexpectedly dark",
		"to me, walking out of the lighted passage of my cousin's house",
		"it seemed indeed black",
	}

	for _, s := range msgs {

		client := clientFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New(s)
		})

		exp := fmt.Errorf("fetch error: %s", s)
		_, got := itunes.ToRSSClient(client, "")

		if !equalErrors(got, exp) {
			t.Errorf("expected error %s, got %s", formatError(exp), formatError(got))
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
		client := redirectRequests(ts, http.DefaultClient)

		msg := http.StatusText(code)
		if msg == "" {
			msg = fmt.Sprintf("status code %d", code) // Go's default status for unrecognised error codes
		}

		exp := fmt.Errorf("fetch error: HTTP Status %d %s", code, msg)
		_, got := itunes.ToRSSClient(client, "")

		if !equalErrors(got, exp) {
			t.Errorf("Status %d: expected error %s, got %s", code, formatError(exp), formatError(got))
		}

		ts.Close()
	}
}

func TestBadContentType(t *testing.T) {

	types := []string{
		"",
		"text/",
		"text/xml; =",
	}

	skipped := 0

	for _, ctype := range types {

		_, _, err := mime.ParseMediaType(ctype)
		if err == nil {
			skipped++
			t.Logf("Warning: ParseMediaType(%q) didn't return an error", ctype)
			continue
		}

		ts := httptest.NewServer(contentTypeHandler(ctype))
		client := redirectRequests(ts, http.DefaultClient)

		exp := fmt.Errorf("bad Content Type %q: %s", ctype, err)
		_, got := itunes.ToRSSClient(client, "")

		if !equalErrors(got, exp) {
			t.Errorf("Content Type %q: expected error %s, got %s", ctype, formatError(exp), formatError(got))
		}

		ts.Close()
	}

	if skipped == len(types) {
		t.Errorf("No content types tested")
	}
}

func TestUnexpectedContentType(t *testing.T) {

	types := []string{
		"image/png",
		"text/plain; charset=utf-8",
		"audio/mpeg",
	}

	for _, ctype := range types {

		ts := httptest.NewServer(contentTypeHandler(ctype))
		client := redirectRequests(ts, http.DefaultClient)

		exp := fmt.Errorf("unexpected Content Type %q", ctype)
		_, got := itunes.ToRSSClient(client, "")

		if !equalErrors(got, exp) {
			t.Errorf("Content Type %q: expected error %s, got %s", ctype, formatError(exp), formatError(got))
		}

		ts.Close()
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
