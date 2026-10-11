package coolstuffinc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// searchFixture points the search at a server answering every page with the
// fixture file.
func searchFixture(t *testing.T, file string) {
	t.Helper()
	page, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(page)
	}))
	t.Cleanup(srv.Close)
	saved := csiSearchURL
	csiSearchURL = srv.URL
	t.Cleanup(func() { csiSearchURL = saved })
}

// searchResults runs processSearch over one shelf and answers what it sent.
func searchResults(t *testing.T, csi *Coolstuffinc, shelf string) []responseChan {
	t.Helper()
	results := make(chan responseChan, 16)
	err := csi.processSearch(context.Background(), results, shelf, nil)
	if err != nil {
		t.Fatalf("processSearch(): %v", err)
	}
	close(results)
	var got []responseChan
	for r := range results {
		got = append(got, r)
	}
	return got
}

// fixtureTransport answers each url it holds with a file's content.
type fixtureTransport map[string]string

func (f fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	data, err := os.ReadFile(f[req.URL.String()])
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{},
		Body:          io.NopCloser(bytes.NewReader(data)),
		ContentLength: int64(len(data)),
		Request:       req,
	}, nil
}

// TestInstanceIDs pins the id each feed keeps for one store row: the cart's
// row id on the sale listing, and the PPQID on the buylist.
func TestInstanceIDs(t *testing.T) {
	b := readGameDatastore(t, "palworld", "PALWORLD_PATH")

	csi, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}
	searchFixture(t, "testdata/palworld_search.html")
	got := searchResults(t, csi, "1st Edition Dawn of Palpagos")
	if len(got) != 1 {
		t.Fatalf("sale listing sent %d offers, want 1", len(got))
	}
	if got[0].invEntry.InstanceID != "11301966" {
		t.Errorf("sale listing InstanceID = %q, want 11301966", got[0].invEntry.InstanceID)
	}

	csi.client = &http.Client{Transport: fixtureTransport{
		csiBuylistLink + csi.shelf:            "testdata/palworld_editions.html",
		fmt.Sprintf(csiBuylistURL, csi.shelf): "testdata/palworld_buylist.json",
	}}
	err = csi.parseBL(context.Background())
	if err != nil {
		t.Fatalf("parseBL(): %v", err)
	}
	var ids []string
	for _, entries := range csi.buylist {
		for _, entry := range entries {
			ids = append(ids, entry.InstanceID)
		}
	}
	if len(ids) == 0 {
		t.Fatal("the buylist row landed nowhere")
	}
	for _, id := range ids {
		if id != "11302044" {
			t.Errorf("buylist InstanceID = %q, want 11302044", id)
		}
	}
}

// TestYugiohSKURetryOnTheSaleListing pins the image retry on the sale
// listing: a listing the wording refuses lands on the number its image names,
// and one whose note says it can be several printings is left refused.
func TestYugiohSKURetryOnTheSaleListing(t *testing.T) {
	b := readGameDatastore(t, "yugioh", "YUGIOH_PATH")

	csi, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}
	searchFixture(t, "testdata/yugioh_search.html")
	got := searchResults(t, csi, "Magic Ruler")
	if len(got) != 1 {
		t.Fatalf("sent %d offers, want only the one with no note", len(got))
	}
	if got[0].invEntry.OriginalID != "1" || got[0].cardID != "mrl-103_22981_unlimited" {
		t.Errorf("product %s landed %q, want product 1 on mrl-103_22981_unlimited", got[0].invEntry.OriginalID, got[0].cardID)
	}
}
