package coolstuffinc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// sealedMatchBackend is a Yu-Gi-Oh set of three products: the box carries
// csiId 40, the tin and the deck carry no id and are found by name.
func sealedMatchBackend(t *testing.T) *matcher.Backend {
	t.Helper()
	b := &matcher.Backend{
		Game:           matcher.GameYuGiOh,
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"LEDU": {Name: "Legendary Duelists", Code: "LEDU"},
		},
	}
	b.AddSealed("ygo-box", "Legendary Duelists Booster Box", "LEDU", "", map[string]string{"csiId": "40"})
	b.AddSealed("ygo-tin", "Dragons of Legend Tin", "LEDU", "", nil)
	b.AddSealed("ygo-deck", "Structure Deck Albaz Strike", "LEDU", "", nil)
	b.SortSealed()
	b.IndexSets()

	for _, name := range []string{"Mystery Assortment", "Mystery Bundle", "Yu-Gi-Oh! - Dragons of Legend Tin", "Blue-Eyes White Dragon"} {
		_, err := b.ResolveSealed(name)
		if err == nil {
			t.Fatalf("the resolver placed %q, so the fixture proves nothing", name)
		}
	}
	for _, name := range []string{"Legendary Duelists Booster Box (Japanese)", "Legendary Duelists Booster Box (German)"} {
		if !matcher.SealedIsLanguageVariant(name) {
			t.Fatalf("%q reads as English, so the fixture proves nothing", name)
		}
	}
	return b
}

// sealedMatchBuylistClient answers the buylist download with the fixture.
func sealedMatchBuylistClient(shelf string) *http.Client {
	return &http.Client{Transport: fixtureTransport{
		fmt.Sprintf(csiBuylistURL, shelf): "testdata/sealed_match_buylist.json",
	}}
}

// sealedMatchExport runs the export's matching over the fixture feeds.
func sealedMatchExport(t *testing.T, b *matcher.Backend) []mtgban.SealedMatch {
	t.Helper()
	csi, err := newSealedMatcher(b)
	if err != nil {
		t.Fatal(err)
	}
	searchFixture(t, "testdata/sealed_match_search.html")
	rows, err := csi.sealedSearchRows(context.Background(), "box")
	if err != nil {
		t.Fatalf("sealedSearchRows(): %v", err)
	}
	entries, err := GetBuylist(context.Background(), sealedMatchBuylistClient(csi.shelf), csi.shelf)
	if err != nil {
		t.Fatalf("GetBuylist(): %v", err)
	}
	return csi.matchCatalog(rows, entries)
}

// TestMatchSealedCatalogAgreesWithLoad pins that what Load prices and what
// the export reports land on the same product for every id, from the same
// feeds. The out-of-stock row Load does not price, the export still places,
// and the id the two feeds place apart is pinned in the rows test.
func TestMatchSealedCatalogAgreesWithLoad(t *testing.T) {
	b := sealedMatchBackend(t)

	csi, err := NewScraperSealed(b)
	if err != nil {
		t.Fatal(err)
	}
	searchFixture(t, "testdata/sealed_match_search.html")
	channel := make(chan responseChan, 16)
	_, err = csi.processSealedSearch(context.Background(), channel, "box")
	if err != nil {
		t.Fatalf("processSealedSearch(): %v", err)
	}
	close(channel)
	retail := map[string]string{}
	for record := range channel {
		retail[record.invEntry.OriginalID] = record.cardID
	}

	csi.client = sealedMatchBuylistClient(csi.shelf)
	err = csi.parseBL(context.Background())
	if err != nil {
		t.Fatalf("parseBL(): %v", err)
	}
	pidOf := map[string]string{"1": "40", "2": "200", "3": "700", "4": "800", "5": "900", "6": "950", "7": "960"}
	buylist := map[string]string{}
	for uuid, entries := range csi.buylist {
		for _, entry := range entries {
			buylist[pidOf[entry.InstanceID]] = uuid
		}
	}

	exported := map[string]string{}
	for _, match := range sealedMatchExport(t, b) {
		exported[match.StoreID] = match.UUID
	}

	const disputed = "950"
	for source, priced := range map[string]map[string]string{"search": retail, "buylist": buylist} {
		for pid, uuid := range priced {
			if pid == disputed {
				continue
			}
			if exported[pid] != uuid {
				t.Errorf("%s: the %s prices it on %q, the export says %q", pid, source, uuid, exported[pid])
			}
		}
	}
	for pid, uuid := range exported {
		if uuid == "" || pid == disputed {
			continue
		}
		_, inRetail := retail[pid]
		_, inBuylist := buylist[pid]
		if !inRetail && !inBuylist && pid != "300" {
			t.Errorf("%s: the export places it on %q, Load prices it nowhere", pid, uuid)
		}
	}
	if retail["300"] != "" {
		t.Errorf("Load priced the out-of-stock row on %q", retail["300"])
	}
}

// TestMatchSealedCatalogRows pins every row of the fixture: one match per
// id across both feeds, in id order, a placed answer winning over the other
// feed's refusal, a search row placing nothing left out (the language
// variant, the unnamed row, the card) and every buylist box kept.
func TestMatchSealedCatalogRows(t *testing.T) {
	got := sealedMatchExport(t, sealedMatchBackend(t))
	want := []mtgban.SealedMatch{
		{StoreID: "40", Name: "Mystery Assortment", UUID: "ygo-box", Via: mtgban.SealedByID},
		{StoreID: "200", Name: "Yu-Gi-Oh! - Dragons of Legend Tin", UUID: "ygo-tin", Via: mtgban.SealedByName},
		{StoreID: "300", Name: "Structure Deck Albaz Strike", UUID: "ygo-deck", Via: mtgban.SealedByName},
		{StoreID: "700", Name: "Mystery Bundle", Via: mtgban.SealedRefused, Reason: matcher.ErrCardDoesNotExist.Error()},
		{StoreID: "900", Name: "Legendary Duelists Booster Box (German)", Via: mtgban.SealedSkipped, Reason: "language variant"},
		{StoreID: "950", Name: "Dragons of Legend Tin", Via: mtgban.SealedRefused, Reason: "the search places it on ygo-tin, the buylist on ygo-deck"},
		{StoreID: "960", Name: "Dragons of Legend Tin", UUID: "ygo-tin", Via: mtgban.SealedByName},
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
// feeds list it, sorted by the id's number.
func TestMatchSealedCatalogOnePerID(t *testing.T) {
	csi, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	search := []sealedRow{
		{pid: "100", name: "Dragons of Legend Tin"},
		{pid: "4", name: "Blue-Eyes White Dragon"},
		{pid: "100", name: "Dragons of Legend Tin"},
		{pid: "30", name: "Structure Deck Albaz Strike"},
	}
	buylist := []CSIPriceEntry{
		{PID: "4", Name: "Mystery Bundle", RarityName: "Box"},
		{PID: "30", Name: "Mystery Bundle", RarityName: "Box"},
		{PID: "100", Name: "Dragons of Legend Tin", RarityName: "Box"},
	}
	var ids []string
	for _, match := range csi.matchCatalog(search, buylist) {
		ids = append(ids, match.StoreID)
	}
	want := []string{"4", "30", "100"}
	if !slices.Equal(ids, want) {
		t.Errorf("got ids %v, want %v", ids, want)
	}
}

// TestMatchSealedCatalogRefuses pins the games the export will not match,
// refused before any request.
func TestMatchSealedCatalogRefuses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, game := range []matcher.Game{matcher.GameMagic, matcher.GameGundam, matcher.GamePalworld, matcher.GameFleshAndBlood} {
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
