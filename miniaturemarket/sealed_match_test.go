package miniaturemarket

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// sealedMatchBackend is a Flesh and Blood set of four products and the card
// one of them is named for: the box carries a miniaturemarketId, the others
// carry none and are found by name.
func sealedMatchBackend(t *testing.T) *matcher.Backend {
	t.Helper()
	b := &matcher.Backend{
		Game:           matcher.GameFleshAndBlood,
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"SEA": {Name: "High Seas", Code: "SEA"},
		},
	}
	b.AddSealed("sea-box", "High Seas Booster Box", "SEA", "", map[string]string{"miniaturemarketId": "100"})
	b.AddSealed("sea-pack", "High Seas Booster Pack", "SEA", "", nil)
	b.AddSealed("sea-blitz", "High Seas Blitz Deck", "SEA", "", nil)
	b.AddSealed("sea-blaze", "Silver Age Chapter 3 Deck - Blaze Firemind (Wizard)", "SEA", "", nil)
	b.SortSealed()
	b.IndexSets()

	card := matcher.Normalize("Blaze Firemind")
	b.UUIDs["blaze"] = &matcher.CardObject{Card: matcher.Card{UUID: "blaze", Name: "Blaze Firemind", SetCode: "SEA"}}
	b.AllNames = append(b.AllNames, card)
	b.Hashes[card] = []string{"blaze"}

	for _, name := range []string{"Mystery Assortment", "Mystery Bundle", "High Seas Blitz Deck (New Arrival)", "Silver Age Chapter 3 Deck - Blaze (Wizard)"} {
		_, err := b.ResolveSealed(name)
		if err == nil {
			t.Fatalf("the resolver placed %q, so the fixture proves nothing", name)
		}
	}
	_, err := resolveByNamedCard(b, "High Seas Blitz Deck (New Arrival)")
	if err == nil {
		t.Fatal("the named-card fallback placed the decorated deck, so the retry proves nothing")
	}
	_, err = matcher.ParsePrice("call us")
	if err == nil {
		t.Fatal("the fixture's bad price parses, so it proves nothing")
	}
	return b
}

// pageTransport answers every request with the fixture page.
type pageTransport struct{}

func (pageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	data, err := os.ReadFile("testdata/sealed_match_page.html")
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(data)),
		Request:    req,
	}, nil
}

// sealedMatchExport runs the export's matching over the fixture page.
func sealedMatchExport(t *testing.T, b *matcher.Backend) []mtgban.SealedMatch {
	t.Helper()
	mm, err := newSealedMatcher(b)
	if err != nil {
		t.Fatal(err)
	}
	mm.client = &http.Client{Transport: pageTransport{}}
	listings, err := mm.pageListings(context.Background(), 1)
	if err != nil {
		t.Fatalf("pageListings(): %v", err)
	}
	return mm.matchCatalog(listings)
}

// TestMatchSealedCatalogAgreesWithLoad pins that what Load prices and what
// the export reports land on the same product for every id. The listing
// whose price does not parse, Load does not price and the export places.
func TestMatchSealedCatalogAgreesWithLoad(t *testing.T) {
	b := sealedMatchBackend(t)

	mm, err := NewScraperSealed(b)
	if err != nil {
		t.Fatal(err)
	}
	mm.buildProductMap()
	mm.client = &http.Client{Transport: pageTransport{}}
	channel := make(chan respChan, 16)
	err = mm.processPage(context.Background(), channel, 1)
	if err != nil {
		t.Fatalf("processPage(): %v", err)
	}
	close(channel)
	priced := map[string]string{}
	for record := range channel {
		if record.drop != "" {
			continue
		}
		id := record.invEntry.URL[strings.LastIndexByte(record.invEntry.URL, '/')+1:]
		priced[id] = record.cardID
	}

	exported := map[string]string{}
	for _, match := range sealedMatchExport(t, b) {
		exported[match.StoreID] = match.UUID
	}

	for id, uuid := range priced {
		if exported[id] != uuid {
			t.Errorf("%s: Load prices it on %q, the export says %q", id, uuid, exported[id])
		}
	}
	for id, uuid := range exported {
		if uuid != "" && priced[id] != uuid && id != "800" {
			t.Errorf("%s: the export says %q, Load prices it on %q", id, uuid, priced[id])
		}
	}
	if priced["800"] != "" {
		t.Errorf("Load priced the listing with no price on %q", priced["800"])
	}
}

// TestMatchSealedCatalogRows pins every row of the fixture.
func TestMatchSealedCatalogRows(t *testing.T) {
	got := sealedMatchExport(t, sealedMatchBackend(t))
	want := []mtgban.SealedMatch{
		{StoreID: "100", Name: "Mystery Assortment", UUID: "sea-box", Via: mtgban.SealedByID},
		{StoreID: "200", Name: "Flesh & Blood TCG: High Seas - Booster Pack", UUID: "sea-pack", Via: mtgban.SealedByName},
		{StoreID: "300", Name: "Flesh & Blood TCG: High Seas Blitz Deck (New Arrival)", UUID: "sea-blitz", Via: mtgban.SealedByName},
		{StoreID: "400", Name: "Flesh & Blood TCG: Silver Age Chapter 3 Deck - Blaze (Wizard)", UUID: "sea-blaze", Via: mtgban.SealedByName},
		{StoreID: "500", Name: "Flesh & Blood TCG: High Seas Booster Box (Japanese)", Via: mtgban.SealedSkipped, Reason: "language variant"},
		{StoreID: "600", Via: mtgban.SealedSkipped, Reason: "unnamed listing"},
		{StoreID: "700", Name: "Flesh & Blood TCG: Mystery Bundle", Via: mtgban.SealedRefused, Reason: matcher.ErrCardDoesNotExist.Error()},
		{StoreID: "800", Name: "Flesh & Blood TCG: High Seas Blitz Deck", UUID: "sea-blitz", Via: mtgban.SealedByName},
	}
	if len(got) != len(want) {
		t.Errorf("got %d matches, want %d: %+v", len(got), len(want), got)
	}
	for i := range min(len(got), len(want)) {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMatchSealedCatalogOnePerID pins one match per id however often the
// walk sees it, a placed answer kept over one placing nothing, sorted by
// the id's number.
func TestMatchSealedCatalogOnePerID(t *testing.T) {
	mm, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	listings := []listing{
		{id: "1000", name: "High Seas Blitz Deck"},
		{id: "4", name: "Mystery Bundle"},
		{id: "1000", name: "Mystery Bundle"},
		{id: "30", name: "Mystery Bundle"},
	}
	var ids []string
	matches := mm.matchCatalog(listings)
	for _, match := range matches {
		ids = append(ids, match.StoreID)
	}
	if !slices.Equal(ids, []string{"4", "30", "1000"}) {
		t.Errorf("got ids %v, want 4, 30, 1000", ids)
	}
	if len(matches) == 3 && matches[2].UUID != "sea-blitz" {
		t.Errorf("id 1000 says %+v, want the placed answer", matches[2])
	}
}

// TestMatchSealedCatalogRefuses pins the games the export will not match,
// refused before any request.
func TestMatchSealedCatalogRefuses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, game := range []matcher.Game{matcher.GameMagic, matcher.GamePokemon} {
		matches, err := MatchSealedCatalog(ctx, &matcher.Backend{Game: game})
		if err == nil {
			t.Errorf("%s: got %d matches, want an error", game, len(matches))
			continue
		}
		if errors.Is(err, context.Canceled) {
			t.Errorf("%s: reached the network", game)
		}
	}
}
