package vegassingles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// serveBuylist answers like Hippolink's product listing from a fixture,
// serving at most perPage products whatever the request asks for.
func serveBuylist(t *testing.T, game string, products []HLProduct, perPage int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("game") != game {
			t.Errorf("asked for game %q, want %q", q.Get("game"), game)
		}
		offset, err := strconv.Atoi(q.Get("offset"))
		if err != nil {
			t.Errorf("offset %q: %v", q.Get("offset"), err)
		}
		end := min(offset+perPage, len(products))
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(HLResponse{
			Products: products[offset:end],
			Total:    len(products),
			Limit:    perPage,
			Offset:   offset,
		})
		if err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func readFixture(t *testing.T, name string) []HLProduct {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var products []HLProduct
	err = json.Unmarshal(data, &products)
	if err != nil {
		t.Fatal(err)
	}
	return products
}

// TestMovedHostIsNamed pins the failure that left the buylist stale: the old
// host redirected to a web page, and the run reported only that "<" was not
// JSON.
func TestMovedHostIsNamed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html;charset=UTF-8")
		_, _ = w.Write([]byte("<!DOCTYPE html><html></html>"))
	}))
	defer server.Close()

	vs, err := NewScraper(&matcher.Backend{Game: matcher.GamePokemon})
	if err != nil {
		t.Fatal(err)
	}
	vs.client.buylistURL = server.URL

	err = vs.Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), `"text/html", not JSON`) {
		t.Errorf("Load answered %v, want it to name the HTML page", err)
	}
}

// newPagedScraper returns a scraper reading a buylist of four one-product
// pages, with answer deciding the status of each page past the first.
func newPagedScraper(t *testing.T, answer func(offset string) int) *Vegassingles {
	t.Helper()
	pages := serveBuylist(t, GamePokemon, make([]HLProduct, 4), 1).Config.Handler
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		if offset != "0" {
			status := answer(offset)
			if status != http.StatusOK {
				w.WriteHeader(status)
				return
			}
		}
		pages.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	vs, err := NewScraper(&matcher.Backend{Game: matcher.GamePokemon})
	if err != nil {
		t.Fatal(err)
	}
	vs.client.buylistURL = server.URL
	vs.maxConcurrency = 1
	return vs
}

// TestLostPageFailsLoad refuses a buylist missing one of its later pages.
func TestLostPageFailsLoad(t *testing.T) {
	// A 404 is not retried
	vs := newPagedScraper(t, func(offset string) int {
		if offset == "2" {
			return http.StatusNotFound
		}
		return http.StatusOK
	})

	err := vs.Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "1 of 3 later pages lost") {
		t.Errorf("Load answered %v, want it to count the lost page", err)
	}
	if len(vs.Buylist()) != 0 || !vs.buylistDate.IsZero() {
		t.Errorf("filed %d cards dated %v from a short crawl", len(vs.Buylist()), vs.buylistDate)
	}
}

// TestCancelledCrawlFailsLoad refuses a buylist whose crawl was cancelled
// past its first page, counting the pages never asked for as lost.
func TestCancelledCrawlFailsLoad(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vs := newPagedScraper(t, func(string) int {
		cancel()
		return http.StatusNotFound
	})

	err := vs.Load(ctx)
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "3 of 3 later pages lost") {
		t.Errorf("Load answered %v, want the cancellation and the lost pages", err)
	}
	if len(vs.Buylist()) != 0 || !vs.buylistDate.IsZero() {
		t.Errorf("filed %d cards dated %v from a cancelled crawl", len(vs.Buylist()), vs.buylistDate)
	}
}

// TestBuylistPricesEveryOffer walks a buylist the host serves fewer products
// per page of than asked for, and files every offer on the printing its
// product and finish name: the reverse holo and plain copies of one product
// are two printings.
func TestBuylistPricesEveryOffer(t *testing.T) {
	b := withPokemon(t)
	products := readFixture(t, "pokemon_buylist.json")
	server := serveBuylist(t, GamePokemon, products, 1)

	vs, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}
	vs.client.buylistURL = server.URL
	var logs []string
	vs.logCallback = func(format string, a ...any) {
		logs = append(logs, format)
	}

	err = vs.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Errorf("logged %d lines, want only the product count: %q", len(logs), logs)
	}

	type key struct {
		product int64
		finish  string
	}
	finishes := map[key]string{}
	var offers int
	for _, product := range products {
		for _, offer := range product.Conditions {
			offers++
			sku := strconv.FormatInt(offer.TCGplayerSkuID, 10)
			var found bool
			for cardID, entries := range vs.Buylist() {
				for _, entry := range entries {
					if entry.InstanceID != sku {
						continue
					}
					found = true
					if entry.BuyPrice != offer.Cash {
						t.Errorf("sku %s bought at %.2f, want the cash offer %.2f", sku, entry.BuyPrice, offer.Cash)
					}
					if entry.OriginalID != strconv.FormatInt(product.TCGplayerProductID, 10) {
						t.Errorf("sku %s filed under product %s", sku, entry.OriginalID)
					}
					co, err := b.GetUUID(cardID)
					if err != nil {
						t.Fatal(err)
					}
					finishes[key{product.TCGplayerProductID, product.Finish}] = co.Finish
				}
			}
			if !found {
				t.Errorf("sku %s (%s %s %s) not priced", sku, product.ProductName, product.Finish, offer.Condition)
			}
		}
	}

	var entries int
	for _, cardEntries := range vs.Buylist() {
		entries += len(cardEntries)
	}
	if entries != offers {
		t.Errorf("buylist holds %d entries, want the fixture's %d offers", entries, offers)
	}
	for k, finish := range finishes {
		want := matcher.FinishSlug(k.finish)
		if finish != want {
			t.Errorf("product %d %s landed on a %s printing, want %s", k.product, k.finish, finish, want)
		}
	}
}

// TestMagicFinishes asks for a Magic product's finish by name where the
// matcher knows the word, refusing one the printing was never sold in, and
// reads a finish spelled in a way it has not seen off the is_foil flag. A
// product on an Art Series shelf, which the datastore does not load, is
// unsupported rather than an unknown card.
func TestMagicFinishes(t *testing.T) {
	b := withMagic(t)
	vs, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		product      HLProduct
		foil, etched bool
		name         string
		refused      error
	}{
		{HLProduct{TCGplayerProductID: 505828, Finish: "normal"}, false, false, "Spectator Seating", nil},
		{HLProduct{TCGplayerProductID: 505834, Finish: "foil", IsFoil: true}, true, false, "Spectator Seating", nil},
		{HLProduct{TCGplayerProductID: 505840, Finish: "etched", IsFoil: true}, false, true, "Spectator Seating", nil},
		{HLProduct{TCGplayerProductID: 725328, Finish: "alternative_foil", IsFoil: true}, true, false, "Heroic Intervention", nil},
		{HLProduct{TCGplayerProductID: 505834, Finish: "", IsFoil: false}, false, false, "Spectator Seating", nil},
		{HLProduct{TCGplayerProductID: 505834, Finish: "unheard_of_foil", IsFoil: true}, true, false, "Spectator Seating", nil},
		// Yidris, Maelstrom Wielder in Commander 2016 is foil only
		{HLProduct{TCGplayerProductID: 124203, Finish: "normal"}, false, false, "", matcher.ErrCardWrongFinish},
		{HLProduct{TCGplayerProductID: 244089, Finish: "normal", SetName: "Adventures in the Forgotten Realms Art Series"}, false, false, "", matcher.ErrUnsupported},
	}
	for _, test := range tests {
		p := test.product
		cardID, err := vs.resolve(p, "English")
		if test.refused != nil {
			if !errors.Is(err, test.refused) {
				t.Errorf("%d %q answered %q, %v, want %v", p.TCGplayerProductID, p.Finish, cardID, err, test.refused)
			}
			// A refused offer is filed nowhere, and only a wrong finish is
			// counted rather than logged on its own line
			p.Conditions = []HLCondition{{Condition: "Near Mint", Language: "English", Cash: 1}}
			var logs []string
			vs.logCallback = func(format string, a ...any) {
				logs = append(logs, format)
			}
			counted := vs.processProduct(p)
			if counted != errors.Is(test.refused, matcher.ErrCardWrongFinish) || len(logs) != 0 || len(vs.Buylist()) != 0 {
				t.Errorf("%d %q counted %v, logged %q, filed %d cards", p.TCGplayerProductID, p.Finish, counted, logs, len(vs.Buylist()))
			}
			continue
		}
		if err != nil {
			t.Errorf("%d %q: %v", p.TCGplayerProductID, p.Finish, err)
			continue
		}
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatal(err)
		}
		if co.Name != test.name || co.Foil != test.foil || co.Etched != test.etched {
			t.Errorf("%d %q landed on %s (foil %v, etched %v)", p.TCGplayerProductID, p.Finish, co.Name, co.Foil, co.Etched)
		}
	}
}

// TestEnglishSkuOfAJapanesePrinting reads TCGplayer's English sku of a product
// that is a Japanese printing as that printing, not as a refusal.
func TestEnglishSkuOfAJapanesePrinting(t *testing.T) {
	b := withMagic(t)
	vs, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}

	// War of the Spark's Japanese alternate art Ajani
	cardID, err := vs.resolve(HLProduct{TCGplayerProductID: 189019, Finish: "foil", IsFoil: true}, "English")
	if err != nil {
		t.Fatal(err)
	}
	co, err := b.GetUUID(cardID)
	if err != nil {
		t.Fatal(err)
	}
	if co.SetCode != "WAR" || co.Number != "184★" || co.Language != "Japanese" {
		t.Errorf("landed on %s %s #%s in %s", co.Name, co.SetCode, co.Number, co.Language)
	}
}

// TestUnloadedLanguageIsRefused keeps a Japanese offer off the English
// printing where the datastore loads no Japanese one.
func TestUnloadedLanguageIsRefused(t *testing.T) {
	b := withMagic(t)
	vs, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}

	_, err = vs.resolve(HLProduct{TCGplayerProductID: 614328, Finish: "normal"}, "Japanese")
	if !errors.Is(err, matcher.ErrUnsupported) {
		t.Errorf("Aatchik in Japanese answered %v, want it unsupported", err)
	}
}
