package cardtrader

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// hostRewriter sends every request to the test server, keeping the rest of
// the client's chain (retries, the bearer header) in front of it.
type hostRewriter struct {
	next   http.RoundTripper
	target *url.URL
}

func (h hostRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = h.target.Scheme
	req.URL.Host = h.target.Host
	return h.next.RoundTrip(req)
}

// fakeCardTrader serves the three export endpoints from fixed bodies and
// records the bearer tokens it saw.
type fakeCardTrader struct {
	expansions string
	blueprints map[string]string // keyed by expansion_id; a missing key answers an error body
	products   string
	mu         sync.Mutex
	tokens     []string
	status     map[string]int // per path, 0 means 200
}

func (f *fakeCardTrader) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.tokens = append(f.tokens, r.Header.Get("Authorization"))
	f.mu.Unlock()
	if code := f.status[r.URL.Path]; code != 0 {
		w.WriteHeader(code)
		fmt.Fprint(w, `{"error":"nope"}`)
		return
	}
	switch r.URL.Path {
	case "/api/v2/expansions/export":
		fmt.Fprint(w, f.expansions)
	case "/api/v2/blueprints/export":
		body, found := f.blueprints[r.URL.Query().Get("expansion_id")]
		if !found {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error_code":"not_found","errors":[],"extra":{"message":"Expansion not found"},"request_id":"x"}`)
			return
		}
		fmt.Fprint(w, body)
	case "/api/v2/products/export":
		fmt.Fprint(w, f.products)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func newFakeClient(t *testing.T, f *fakeCardTrader) *CTAuthClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(srv.Close)
	target, _ := url.Parse(srv.URL)
	ct := NewCTAuthClient("tok-123")
	ct.client.Transport = hostRewriter{next: ct.client.Transport, target: target}
	return ct
}

func fixture() *fakeCardTrader {
	return &fakeCardTrader{
		expansions: `[{"id":10,"game_id":1,"code":"lea","name":"Alpha"},{"id":20,"game_id":18,"code":"tfc","name":"The First Chapter"}]`,
		blueprints: map[string]string{
			"10": `[{"id":100,"name":"Black Lotus","game_id":1,"category_id":1,"expansion_id":10,"scryfall_id":"abc","tcg_player_id":1234,"expansion":{"name":"Alpha","code":"lea"},"fixed_properties":{"collector_number":"232"}}]`,
			"20": `[{"id":200,"name":"Elsa - Snow Queen","game_id":18,"category_id":30,"expansion_id":20,"tcg_player_id":5678,"expansion":{"name":"The First Chapter","code":"tfc"}}]`,
		},
		products: `[{"id":1,"name_en":"Black Lotus","quantity":2,"price_cents":1000000,"price_currency":"EUR","blueprint_id":100,"game_id":1,"category_id":1,"properties_hash":{"condition":"Near Mint","mtg_language":"en","mtg_foil":false}},{"id":2,"name_en":"Elsa - Snow Queen","quantity":1,"price_cents":250,"price_currency":"EUR","blueprint_id":200,"game_id":18,"category_id":30,"properties_hash":{"condition":"Played","lorcana_language":"en","lorcana_foil":true}}]`,
		status:   map[string]int{},
	}
}

func TestExportInventoryJoinsTheThreeEndpoints(t *testing.T) {
	f := fixture()
	ct := newFakeClient(t, f)

	inv, err := ct.ExportInventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Products) != 2 {
		t.Fatalf("products: got %d, want 2", len(inv.Products))
	}
	if inv.Products[0].Name != "Black Lotus" || inv.Products[0].GameID != 1 || inv.Products[0].CategoryID != 1 {
		t.Errorf("export fields not decoded: %+v", inv.Products[0])
	}
	if inv.Products[1].Properties.LorcanaFoil != true || inv.Products[1].Properties.Condition != "Played" {
		t.Errorf("properties not decoded: %+v", inv.Products[1].Properties)
	}
	if bp := inv.Blueprints[100]; bp == nil || bp.Name != "Black Lotus" || bp.Expansion.Code != "lea" {
		t.Errorf("blueprint 100: %+v", bp)
	}
	if bp := inv.Blueprints[200]; bp == nil || bp.GameID != 18 {
		t.Errorf("blueprint 200: %+v", bp)
	}
	if exp := inv.Expansions[20]; exp.Name != "The First Chapter" || exp.GameID != 18 {
		t.Errorf("expansion 20: %+v", exp)
	}
	if len(inv.Unavailable) != 0 {
		t.Errorf("unavailable: %v", inv.Unavailable)
	}
	for _, tok := range f.seen() {
		if tok != "Bearer tok-123" {
			t.Fatalf("a request went out as %q", tok)
		}
	}
	if len(f.seen()) != 4 {
		t.Errorf("requests: got %d, want 4 (expansions, two blueprint fetches, products)", len(f.seen()))
	}
}

func TestExportInventoryKeepsGoingPastAMissingCatalog(t *testing.T) {
	f := fixture()
	delete(f.blueprints, "20")
	ct := newFakeClient(t, f)

	inv, err := ct.ExportInventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, found := inv.Blueprints[100]; !found {
		t.Error("the other expansion's blueprints were dropped")
	}
	if _, found := inv.Blueprints[200]; found {
		t.Error("a blueprint appeared for the expansion that failed")
	}
	if inv.Unavailable[20] == nil || !strings.Contains(inv.Unavailable[20].Error(), "Expansion not found") {
		t.Errorf("unavailable[20] = %v", inv.Unavailable[20])
	}
	if len(inv.Products) != 2 {
		t.Errorf("the products were not fetched after a failed catalog: %d", len(inv.Products))
	}
}

func TestExportInventoryReportsARefusedToken(t *testing.T) {
	f := fixture()
	f.status["/api/v2/expansions/export"] = http.StatusUnauthorized
	ct := newFakeClient(t, f)

	_, err := ct.ExportInventory(context.Background())
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want an error naming the 401, got %v", err)
	}
	if len(f.seen()) != 1 {
		t.Errorf("requests after a refused token: %d, want 1", len(f.seen()))
	}
}

func TestExportInventoryFailsWhenProductsFail(t *testing.T) {
	f := fixture()
	f.products = `{"error":"broken"}`
	ct := newFakeClient(t, f)

	_, err := ct.ExportInventory(context.Background())
	if err == nil {
		t.Fatal("want an error when the products do not decode")
	}
}

func TestExportInventoryStopsOnACancelledContext(t *testing.T) {
	f := fixture()
	ct := newFakeClient(t, f)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ct.ExportInventory(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestGameID(t *testing.T) {
	if id, found := GameID(mtgmatcher.GameMagic); !found || id != GameMagic {
		t.Errorf("magic: %d %v", id, found)
	}
	if id, found := GameID(mtgmatcher.GameLorcana); !found || id != GameLorcana {
		t.Errorf("lorcana: %d %v", id, found)
	}
	if _, found := GameID(mtgmatcher.Game("digimon")); found {
		t.Error("a game the package has no scraper for was found")
	}
}

// seen returns a copy of the bearer tokens recorded so far.
func (f *fakeCardTrader) seen() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.tokens...)
}
