package coolstuffinc

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// mysteryName is the fixtures' row the resolver cannot place: only its id
// says which product it is.
const mysteryName = "Mystery Assortment"

// sealedIDsBackend is a Riftbound set of three products. The collection and
// the display carry a csiId when ids is true, each with a storefront row of
// its own; the box carries none and is found by name.
func sealedIDsBackend(t *testing.T, ids bool) *matcher.Backend {
	t.Helper()
	return sealedIDsBackendWith(t, ids, nil)
}

// sealedIDsBackendWith is sealedIDsBackend with the box carrying boxIDs.
func sealedIDsBackendWith(t *testing.T, ids bool, boxIDs map[string]string) *matcher.Backend {
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
	var collectionIDs, displayIDs map[string]string
	if ids {
		collectionIDs = map[string]string{"csiId": "900001"}
		displayIDs = map[string]string{"csiId": "800000"}
	}
	b.AddSealed("ogn-box", "Origins Booster Box", "OGN", "", boxIDs)
	b.AddSealed("ogn-collection", "Origins Arena Collection", "OGN", "", collectionIDs)
	b.AddSealed("ogn-display", "Origins Display", "OGN", "", displayIDs)
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

// TestSealedSearchByID pins that a known id prices its row whatever the name
// says, another language included, while a row with no known id resolves by
// name and the card row drops out.
func TestSealedSearchByID(t *testing.T) {
	foreign := "Origins Arena Collection (Japanese)"
	if !matcher.SealedIsLanguageVariant(foreign) {
		t.Fatalf("%q reads as English, so the fixture proves nothing", foreign)
	}
	got := sealedSearchOffers(t, sealedIDsBackend(t, true))
	want := map[string]string{"900001": "ogn-collection", "900002": "ogn-box", "800000": "ogn-display"}
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
// the search by name alone.
func TestSealedSearchWithoutIDs(t *testing.T) {
	got := sealedSearchOffers(t, sealedIDsBackend(t, false))
	if len(got) != 1 || got["900002"] != "ogn-box" {
		t.Errorf("priced %v, want only row 900002 on ogn-box", got)
	}
}

// TestSealedBuylistByID pins that the buylist's PID reaches a product by its
// csiId outside Magic, and a PID it does not carry by name.
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

// TestSealedNameKeepsOffAnIDsProduct pins that once the box holds a csiId,
// the retail row and the buylist entry whose name reaches it are left out.
func TestSealedNameKeepsOffAnIDsProduct(t *testing.T) {
	b := sealedIDsBackendWith(t, true, map[string]string{"csiId": "555555"})

	got := sealedSearchOffers(t, b)
	want := map[string]string{"900001": "ogn-collection", "800000": "ogn-display"}
	if len(got) != len(want) {
		t.Errorf("priced %v, want %v", got, want)
	}
	for pid, uuid := range want {
		if got[pid] != uuid {
			t.Errorf("row %s landed on %q, want %q", pid, got[pid], uuid)
		}
	}

	csi, err := NewScraperSealed(b)
	if err != nil {
		t.Fatal(err)
	}
	var logs []string
	csi.logCallback = func(format string, a ...any) { logs = append(logs, fmt.Sprintf(format, a...)) }
	csi.client = &http.Client{Transport: fixtureTransport{
		fmt.Sprintf(csiBuylistURL, csi.shelf): "testdata/sealed_buylist_ids.json",
	}}
	err = csi.parseBL(context.Background())
	if err != nil {
		t.Fatalf("parseBL(): %v", err)
	}
	if len(csi.buylist["ogn-box"]) != 0 {
		t.Errorf("the box holds %v, want no buylist entry", csi.buylist["ogn-box"])
	}
	if len(csi.buylist["ogn-collection"]) != 1 {
		t.Errorf("the collection holds %v, want one buylist entry", csi.buylist["ogn-collection"])
	}
	refused := 0
	for _, line := range logs {
		if strings.HasSuffix(line, sealedHoldsOwnID) {
			refused++
		}
	}
	if refused != 1 {
		t.Errorf("logged %q, want one line ending %q", logs, sealedHoldsOwnID)
	}
}
