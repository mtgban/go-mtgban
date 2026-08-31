package vegassingles

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestProductSlug pins the shape of the storefront's product path. The
// display name carries the collector number, and a number written over its
// set total carries a slash, so a slash left in place stops being a
// character in the path and becomes a step down it - the link led to a page
// the storefront does not serve. Every name below is one the feed publishes.
func TestProductSlug(t *testing.T) {
	for _, tt := range []struct {
		desc, name, want string
	}{
		{
			desc: "a number written over its set total keeps both halves",
			name: "Yasuo, Windrider (Overnumbered 235/221) - Spiritforged Foil",
			want: "yasuo-windrider-overnumbered-235-221-spiritforged-foil",
		},
		{
			desc: "an apostrophe closes up rather than parting the word",
			name: "AZ's Tranquility 120/086  - Holofoil ME04 Chaos Rising - Special Illustration Rare",
			want: "azs-tranquility-120-086-holofoil-me04-chaos-rising-special-illustration-rare",
		},
		{
			desc: "a name with no punctuation at all is unchanged",
			name: "Acro Bike (Secret) 178  - Holofoil SM  Celestial Storm - Secret Rare",
			want: "acro-bike-secret-178-holofoil-sm-celestial-storm-secret-rare",
		},
		{
			desc: "the marker a signature card carries is not a separator of its own",
			name: "Yasuo - Windrider (Signature) (235*/221) - Spiritforged Foil",
			want: "yasuo-windrider-signature-235-221-spiritforged-foil",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := buildProductSlug(tt.name); got != tt.want {
				t.Errorf("buildProductSlug(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestCrawlAsksForStock pins that the crawl asks the storefront for what it
// holds. Nearly three products in four the endpoint answers with are stocked
// in no condition at all, and none of them is priced either way.
func TestCrawlAsksForStock(t *testing.T) {
	var asked string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.Query().Get("in_stock")
		_, _ = w.Write([]byte(`{"products":[]}`))
	}))
	defer srv.Close()

	vs := NewScraper(GameRiftbound)
	client := *vs.client
	client.baseURL = srv.URL
	vs.client = &client
	_, _ = vs.client.getPage(context.Background(), 1, sortForward, "")

	if asked != "true" {
		t.Errorf("in_stock = %q, want %q", asked, "true")
	}
}
