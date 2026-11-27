package coolstuffinc

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// mysteryName is the fixtures' row the resolver cannot place: only its id
// says which product it is.
const mysteryName = "Mystery Assortment"

// sealedIDsBackend is a Riftbound set of two products. The collection
// carries two csiIds when ids is true, each with a storefront row of its
// own; the box carries none and is found by name.
func sealedIDsBackend(t *testing.T, ids bool) *matcher.Backend {
	t.Helper()
	b := &matcher.Backend{
		Game:           matcher.GameRiftbound,
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"OGN": {Name: "Origins", Code: "OGN"},
		},
	}
	var collectionIDs matcher.SealedIDs
	if ids {
		collectionIDs = matcher.SealedIDs{"csiId": {"800000", "900001"}}
	}
	b.AddSealed("ogn-box", "Origins Booster Box", "OGN", "", nil)
	b.AddSealed("ogn-collection", "Origins Arena Collection", "OGN", "", collectionIDs)
	b.SortSealed()
	b.IndexSets()

	_, err := b.ResolveSealed(mysteryName)
	if err == nil {
		t.Fatalf("the resolver placed %q, so the fixture proves nothing", mysteryName)
	}
	return b
}

// sealedSearchOffers runs one sealed search over the id fixture and answers
// the uuid each priced row landed on, by the row's id.
func sealedSearchOffers(t *testing.T, b *matcher.Backend) map[string]string {
	t.Helper()
	csi, err := NewScraperSealed(b)
	if err != nil {
		t.Fatal(err)
	}
	searchFixture(t, "testdata/sealed_search_ids.html")
	channel := make(chan responseChan, 16)
	_, err = csi.processSealedSearch(context.Background(), channel, "box")
	if err != nil {
		t.Fatalf("processSealedSearch(): %v", err)
	}
	close(channel)
	got := map[string]string{}
	for record := range channel {
		got[record.invEntry.OriginalID] = record.cardID
	}
	return got
}

// TestSealedSearchByID pins that a row whose id the datastore carries is
// priced by it, even where its name places nothing. The row with no known id
// still resolves by name, and the card row still drops out. A row whose name
// marks another language is priced all the same when its id is known: the id
// says which product it is, and the language check guards only the name.
func TestSealedSearchByID(t *testing.T) {
	foreign := "Origins Arena Collection (Japanese)"
	if !matcher.SealedIsLanguageVariant(foreign) {
		t.Fatalf("%q reads as English, so the fixture proves nothing", foreign)
	}
	got := sealedSearchOffers(t, sealedIDsBackend(t, true))
	want := map[string]string{"900001": "ogn-collection", "900002": "ogn-box", "800000": "ogn-collection"}
	if len(got) != len(want) {
		t.Errorf("priced %v, want %v", got, want)
	}
	for pid, uuid := range want {
		if got[pid] != uuid {
			t.Errorf("row %s landed on %q, want %q", pid, got[pid], uuid)
		}
	}
}

// TestSealedSearchWithoutIDs pins that a datastore carrying no csiId prices
// the search by name alone, as it did before ids were read.
func TestSealedSearchWithoutIDs(t *testing.T) {
	got := sealedSearchOffers(t, sealedIDsBackend(t, false))
	if len(got) != 1 || got["900002"] != "ogn-box" {
		t.Errorf("priced %v, want only row 900002 on ogn-box", got)
	}
}

// TestSealedBuylistByID pins that the buylist's PID reaches a product by
// any of its csiIds outside Magic, and a PID it does not carry by name.
func TestSealedBuylistByID(t *testing.T) {
	csi, err := NewScraperSealed(sealedIDsBackend(t, true))
	if err != nil {
		t.Fatal(err)
	}
	csi.client = &http.Client{Transport: fixtureTransport{
		fmt.Sprintf(csiBuylistURL, csi.shelf): "testdata/sealed_buylist_ids.json",
	}}
	err = csi.parseBL(context.Background())
	if err != nil {
		t.Fatalf("parseBL(): %v", err)
	}
	for uuid, instance := range map[string]string{"ogn-collection": "1", "ogn-box": "2"} {
		entries := csi.buylist[uuid]
		if len(entries) != 1 || entries[0].InstanceID != instance {
			t.Errorf("%s holds %v, want the row with PPQID %s", uuid, entries, instance)
		}
	}
}

// TestSealedIDSharedByTwoProducts pins which product an id two products
// carry prices: the last by name, the Foil one, as Magic's map keeps the last.
func TestSealedIDSharedByTwoProducts(t *testing.T) {
	b := sealedIDsBackend(t, false)
	// Filed foil first, so the answer is the name order and not the filing
	b.AddSealed("ogn-b", "Origins Arena Bundle Foil", "OGN", "", matcher.SealedIDs{"csiId": {"700000"}})
	b.AddSealed("ogn-a", "Origins Arena Bundle", "OGN", "", matcher.SealedIDs{"csiId": {"700000"}})
	b.SortSealed()

	csi, err := NewScraperSealed(b)
	if err != nil {
		t.Fatal(err)
	}
	got := csi.productMap["700000"]
	if got != "ogn-b" {
		t.Errorf("the id prices %q, want ogn-b", got)
	}
}
