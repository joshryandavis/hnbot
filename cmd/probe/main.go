package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

var urls = []struct{ name, url string }{
	{"HN /rss", "https://news.ycombinator.com/rss"},
	{"HN /rss?p=2", "https://news.ycombinator.com/rss?p=2"},
	{"HN / HTML", "https://news.ycombinator.com/"},
	{"hnrss.org/frontpage", "https://hnrss.org/frontpage"},
	{"hnrss.org/newest", "https://hnrss.org/newest"},
	{"Algolia front_page", "https://hn.algolia.com/api/v1/search?tags=front_page"},
	{"Firebase topstories", "https://hacker-news.firebaseio.com/v0/topstories.json"},
}

func fetch(url string, h1 bool, ua string) (int, string, string) {
	tr := &http.Transport{}
	if h1 {
		tr.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	}
	c := &http.Client{Timeout: 20 * time.Second, Transport: tr}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	resp, err := c.Do(req)
	if err != nil {
		return -1, "", err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 120))
	return resp.StatusCode, resp.Proto, string(b)
}

func main() {
	for _, u := range urls {
		for _, mode := range []struct {
			label string
			h1    bool
			ua    string
		}{
			{"go-h2   no-ua", false, ""},
			{"go-h1.1 no-ua", true, ""},
			{"go-h1.1 browser-ua", true, "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"},
		} {
			code, proto, body := fetch(u.url, mode.h1, mode.ua)
			fmt.Printf("%-22s %-20s status=%-4d proto=%-9s body=%q\n", u.name, mode.label, code, proto, body)
		}
	}
}
