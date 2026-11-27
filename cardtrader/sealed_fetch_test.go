package cardtrader

import (
	"context"
	"io"
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// catalogTransport serves a Gundam catalog of two expansions: the first
// holds a sealed blueprint for each way one settles and a single, the
// second answers as failed when broken is set.
func catalogTransport(broken bool) http.RoundTripper {
	return transportFunc(func(req *http.Request) (*http.Response, error) {
		body := `[{"id":1,"game_id":23,"name":"Freedom Ascension"},` +
			`{"id":2,"game_id":23,"name":"Generation Pulse"},` +
			`{"id":9,"game_id":1,"name":"Magic"}]`
		switch req.URL.Query().Get("expansion_id") {
		case "1":
			body = `[{"id":100,"name":"Mystery Assortment","expansion_id":1},` +
				`{"id":200,"name":"Something Else Entirely","tcg_player_id":555,"expansion_id":1},` +
				`{"id":800,"name":"Nothing The Resolver Knows","expansion_id":1},` +
				`{"id":900,"name":"Freedom Ascension Booster Box","category_id":272,"expansion_id":1}]`
		case "2":
			body = `[{"id":300,"name":"Starter Deck 10: Generation Pulse","expansion_id":2}]`
			if broken {
				body = `{"extra":{"message":"boom"}}`
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
}

// TestMatchSealedCatalogFetches drives the export's fetch and match over the
// catalog: the sealed blueprints of the game's expansions, the single left
// out by FormatBlueprints, one match per blueprint.
func TestMatchSealedCatalogFetches(t *testing.T) {
	ct, err := newSealedMatcher(sealedMatchBackend())
	if err != nil {
		t.Fatal(err)
	}
	client := &CTAuthClient{client: &http.Client{Transport: catalogTransport(false)}}
	got, err := ct.matchSealedCatalog(context.Background(), client)
	if err != nil {
		t.Fatalf("matchSealedCatalog(): %v", err)
	}
	want := []mtgban.SealedMatch{
		{StoreID: "100", Name: "Mystery Assortment", UUID: "gd05-box", Via: mtgban.SealedByID},
		{StoreID: "200", Name: "Something Else Entirely", UUID: "gd05-pack", Via: mtgban.SealedByBridge},
		{StoreID: "300", Name: "Starter Deck 10: Generation Pulse", UUID: "st10-deck", Via: mtgban.SealedByName},
		{StoreID: "800", Name: "Nothing The Resolver Knows", Via: mtgban.SealedRefused, Reason: matcher.ErrCardDoesNotExist.Error()},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMatchSealedCatalogSkippedExpansion pins an expansion failing to fetch
// failing the export rather than leaving its blueprints out.
func TestMatchSealedCatalogSkippedExpansion(t *testing.T) {
	ct, err := newSealedMatcher(sealedMatchBackend())
	if err != nil {
		t.Fatal(err)
	}
	client := &CTAuthClient{client: &http.Client{Transport: catalogTransport(true)}}
	got, err := ct.matchSealedCatalog(context.Background(), client)
	if err == nil {
		t.Fatalf("got %d matches, want an error", len(got))
	}
	if !strings.Contains(err.Error(), "fetch: 1,") || !strings.Contains(err.Error(), "2 Generation Pulse") {
		t.Errorf("error %q does not name the count and the first skip", err)
	}
}

// TestSealedBridgeStrict pins the strict bridge: the whole map when every
// expansion fetches, an error naming the skip when one does not.
func TestSealedBridgeStrict(t *testing.T) {
	client := &CTAuthClient{client: &http.Client{Transport: catalogTransport(false)}}
	bridge, err := strictSealedBridge(context.Background(), client, matcher.GameGundam)
	if err != nil {
		t.Fatalf("strictSealedBridge(): %v", err)
	}
	if !maps.Equal(bridge, map[int]int{}) {
		t.Errorf("got %v, want no links: the fixture sells under no Cardmarket id", bridge)
	}

	client = &CTAuthClient{client: &http.Client{Transport: catalogTransport(true)}}
	bridge, err = strictSealedBridge(context.Background(), client, matcher.GameGundam)
	if err == nil {
		t.Fatalf("got %d links, want an error", len(bridge))
	}
	if !strings.Contains(err.Error(), "fetch: 1,") || !strings.Contains(err.Error(), "2 Generation Pulse") {
		t.Errorf("error %q does not name the count and the first skip", err)
	}
}

// TestSealedBridgeLoggedTolerant pins the logged bridge keeping what fetched
// when an expansion is skipped, as bantool reads it.
func TestSealedBridgeLoggedTolerant(t *testing.T) {
	client := &CTAuthClient{client: &http.Client{Transport: catalogTransport(true)}}
	var logs []string
	_, err := sealedBridge(context.Background(), client, matcher.GameGundam, func(string, ...any) {
		logs = append(logs, "skip")
	})
	if err != nil {
		t.Fatalf("sealedBridge(): %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("logged %d skips, want 1", len(logs))
	}
}

// emptyCatalogTransport serves a Gundam catalog with nothing in it: no
// expansions at all, or with listed set, the two expansions and no blueprint
// in either.
func emptyCatalogTransport(listed bool) http.RoundTripper {
	return transportFunc(func(req *http.Request) (*http.Response, error) {
		body := `[]`
		if listed && req.URL.Query().Get("expansion_id") == "" {
			body = `[{"id":1,"game_id":23,"name":"Freedom Ascension"},` +
				`{"id":2,"game_id":23,"name":"Generation Pulse"}]`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
}

// TestMatchSealedCatalogEmptyPull pins an empty catalog failing the export
// and the strict bridge, where an empty answer would read as every id
// withdrawn, and the logged bridge still answering with nothing.
func TestMatchSealedCatalogEmptyPull(t *testing.T) {
	for _, listed := range []bool{false, true} {
		ct, err := newSealedMatcher(sealedMatchBackend())
		if err != nil {
			t.Fatal(err)
		}
		client := &CTAuthClient{client: &http.Client{Transport: emptyCatalogTransport(listed)}}
		got, err := ct.matchSealedCatalog(context.Background(), client)
		if err == nil || !strings.Contains(err.Error(), "no sealed blueprints") {
			t.Errorf("expansions listed %v: got %d matches and %v, want the empty-pull error", listed, len(got), err)
		}

		bridge, err := strictSealedBridge(context.Background(), client, matcher.GameGundam)
		if err == nil || !strings.Contains(err.Error(), "no blueprints") {
			t.Errorf("expansions listed %v: got %d links and %v, want the empty-pull error", listed, len(bridge), err)
		}

		bridge, err = sealedBridge(context.Background(), client, matcher.GameGundam, nil)
		if err != nil || len(bridge) != 0 {
			t.Errorf("expansions listed %v: the logged bridge gave %v and %v, want no links and no error", listed, bridge, err)
		}
	}
}

// TestMatchSealedCatalogOnlySingles pins a catalog holding singles and no
// sealed blueprint failing the export, while the bridge, which reads both,
// answers.
func TestMatchSealedCatalogOnlySingles(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		body := `[{"id":1,"game_id":23,"name":"Freedom Ascension"}]`
		if req.URL.Query().Get("expansion_id") == "1" {
			body = `[{"id":900,"name":"Freedom Ascension Booster Box","category_id":272,"expansion_id":1}]`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	ct, err := newSealedMatcher(sealedMatchBackend())
	if err != nil {
		t.Fatal(err)
	}
	client := &CTAuthClient{client: &http.Client{Transport: transport}}
	got, err := ct.matchSealedCatalog(context.Background(), client)
	if err == nil || !strings.Contains(err.Error(), "no sealed blueprints") {
		t.Errorf("got %d matches and %v, want the empty-pull error", len(got), err)
	}
	_, err = strictSealedBridge(context.Background(), client, matcher.GameGundam)
	if err != nil {
		t.Errorf("strictSealedBridge(): %v, want the singles to answer", err)
	}
}
