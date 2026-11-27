package starcitygames

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// catalogServer answers the catalog download with first on the first
// request and full on every later one.
func catalogServer(t *testing.T, first, full []byte) *SCGClient {
	t.Helper()
	var served atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if served.Add(1) == 1 {
			_, _ = w.Write(first)
			return
		}
		_, _ = w.Write(full)
	}))
	t.Cleanup(srv.Close)
	client := NewSCGClient("key")
	client.catalogURL = srv.URL
	return client
}

// TestMatchSealedCatalogStreams drives the export's stream: a first download
// broken after one product the full catalog lacks, which the reset must
// clear, then the full catalog, matched as the rows test pins it.
func TestMatchSealedCatalogStreams(t *testing.T) {
	full, err := json.Marshal(sealedMatchCatalog())
	if err != nil {
		t.Fatal(err)
	}
	stray, err := json.Marshal(CatalogProduct{
		SKU: "SLD-LOR-ZZZ-011-EN", Name: "Lorcana: Winterspell Booster Case",
		Game: "Disney Lorcana", ProductType: ProductTypeSealed, Language: "English",
	})
	if err != nil {
		t.Fatal(err)
	}
	broken := append(append([]byte("["), stray...), `,{"sku":`...)

	scg, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	got, err := scg.matchSealedCatalog(context.Background(), catalogServer(t, broken, full))
	if err != nil {
		t.Fatalf("matchSealedCatalog(): %v", err)
	}
	want := scg.matchCatalog(sealedMatchCatalog())
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMatchSealedCatalogNoneOfGame pins a catalog holding no product of the
// backend's game failing the export: the spelling of the game moved.
func TestMatchSealedCatalogNoneOfGame(t *testing.T) {
	other := CatalogProduct{SKU: "SLD-MTG-BBX-FDN-EN", Name: "Foundations Booster Box", Game: "Magic: The Gathering", ProductType: ProductTypeSealed}
	body, err := json.Marshal([]CatalogProduct{other})
	if err != nil {
		t.Fatal(err)
	}
	scg, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	got, err := scg.matchSealedCatalog(context.Background(), catalogServer(t, body, body))
	if err == nil {
		t.Fatalf("got %d matches, want an error", len(got))
	}
}

// TestMatchSealedCatalogNoSKU pins a product with no sku left out, where it
// would otherwise report a product under an empty id.
func TestMatchSealedCatalogNoSKU(t *testing.T) {
	scg, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	got := scg.matchCatalog([]CatalogProduct{
		{Name: "Lorcana: Winterspell Booster Case", Game: "Lorcana", ProductType: ProductTypeSealed},
		{SKU: "SLD-LOR-CAS-011-EN", Name: "Lorcana: Winterspell Booster Case", Game: "Lorcana", ProductType: ProductTypeSealed},
	})
	want := mtgban.SealedMatch{StoreID: "SLD-LOR-CAS-011-EN", Name: "Lorcana: Winterspell Booster Case", UUID: "wsp-case", Via: mtgban.SealedByName}
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %+v, want only %+v", got, want)
	}
}

// TestMatchSealedCatalogPanicLeftOut pins a product whose matching panics
// left out of the export rather than refused: a refusal would have the
// consumer take back an id the store never withdrew. With no backend, the
// product the sku map answers matches and the one needing the resolver
// panics.
func TestMatchSealedCatalogPanicLeftOut(t *testing.T) {
	scg := &Sealed{gameID: GameLorcana, productMap: map[string]string{"A": "wsp-box"}}
	got := scg.matchCatalog([]CatalogProduct{
		{SKU: "A", Name: "Lorcana: Mystery Assortment", Game: "Lorcana", ProductType: ProductTypeSealed},
		{SKU: "B", Name: "Lorcana: Mystery Bundle", Game: "Lorcana", ProductType: ProductTypeSealed},
	})
	want := mtgban.SealedMatch{StoreID: "A", Name: "Lorcana: Mystery Assortment", UUID: "wsp-box", Via: mtgban.SealedByID}
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %+v, want only %+v", got, want)
	}
}
