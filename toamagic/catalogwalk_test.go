package toamagic

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// TestCatalogReadsEachPageOnce pins that a walk reads no page twice: a page
// whose next link names itself ends its category, and a set listed under two
// parents is read under the first. Recursing, master overflowed the stack on
// the first and read the set again under the second.
func TestCatalogReadsEachPageOnce(t *testing.T) {
	const listing = `<h1 class="page-title">Unhinged</h1><ul class="products"></ul>`
	pages := map[string]string{
		"/sets":   `<h1 class="page-title">Core Sets</h1><a class="clearfix" href="/core">c</a><a class="clearfix" href="/recent">c</a>`,
		"/core":   `<h1 class="page-title">Core Sets</h1><a class="clearfix" href="/set/1">c</a>`,
		"/recent": `<h1 class="page-title">Recent Sets</h1><a class="clearfix" href="/set/1">c</a>`,
		"/set/1":  listing + `<a class="next_page" href="/set/2">n</a>`,
		"/set/2":  listing + `<a class="next_page" href="/set/2">n</a>`,
	}
	var asked []string
	toa := &TOAMagic{client: &http.Client{Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
		asked = append(asked, req.URL.Path)
		body := io.NopCloser(strings.NewReader("<html>" + pages[req.URL.Path] + "</html>"))
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: body, Request: req}, nil
	})}}
	err := toa.processProduct(context.Background(), make(chan responseChan, 1), "/sets", map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/sets", "/core", "/set/1", "/set/2", "/recent"}
	if !slices.Equal(asked, want) {
		t.Errorf("asked for %v, want %v", asked, want)
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
