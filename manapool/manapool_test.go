package manapool

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestMain loads the datastore when one is configured; the unit test below
// reads no cards, so a checkout without it still runs that.
var (
	datastoreOnce sync.Once
	datastoreErr  error
	datastoreB    *mtgmatcher.Backend
)

// withMagic loads the Magic datastore the first time a test asks for it, and
// skips where the run carries none.
func withMagic(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		datastoreB, datastoreErr = datastore.Read("magic", path)
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if datastoreB == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return datastoreB
}

// TestAddCheapestKeepsTheLowerPrice pins that a printing the store files under
// more than one product is priced once per grade, at the lower of the prices
// it arrives with, whichever product arrives first.
func TestAddCheapestKeepsTheLowerPrice(t *testing.T) {
	mp := NewScraper(&mtgmatcher.Backend{})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: "NM", Price: 25, URL: "first"})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: "NM", Price: 15, URL: "cheaper"})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: "NM", Price: 40, URL: "dearer"})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: "SP", Price: 9, URL: "other-grade"})

	got := map[string]mtgban.InventoryEntry{}
	for _, e := range mp.Inventory()["card"] {
		got[e.Conditions] = e
	}
	if len(got) != 2 {
		t.Fatalf("published %d grades, want NM and SP: %+v", len(got), got)
	}
	if got["NM"].Price != 15 || got["NM"].URL != "cheaper" {
		t.Errorf("NM is %.0f from %q, want 15 from the cheaper product", got["NM"].Price, got["NM"].URL)
	}
	if got["SP"].Price != 9 {
		t.Errorf("SP is %.0f, want 9", got["SP"].Price)
	}
}

// TestPriceResolvesTokenPairing pins a real listing resolving to the
// combined entity mtgmatcher/magic derives for it (a real, usable
// TCGplayer id in mtgjson's own tokenProducts feed), not the single face
// Mana Pool's own scryfall_id names on its own: Squirrel // Starscape
// Cleric (The Lost Caverns of Ixalan Commander) ships as one physical
// card, but Mana Pool's scryfall_id for this listing is Scryfall's own id
// for the Squirrel face alone.
func TestPriceResolvesTokenPairing(t *testing.T) {
	b := withMagic(t)

	card := Product{
		URL:       "https://manapool.com/card/tblb/15-23/squirrel-starscape-cleric",
		ProductID: "000c21ee-d464-4e49-a8f0-a31280e8c453", SetCode: "TBLB", Number: "15-23",
		Name:       "Squirrel // Starscape Cleric",
		ScryfallID: "5a6ec62e-0e9b-4312-bfe8-cc85d76fd9e0", TcgplayerProductID: 561444,
		LanguageID: "EN", ConditionID: "NM", FinishID: "NF", LowPrice: 35, AvailableQuantity: 1,
	}
	mp := NewScraper(b)
	mp.price([]Product{card})

	found := false
	for id, entries := range mp.Inventory() {
		if len(entries) == 0 || entries[0].URL != "https://manapool.com/card/tblb/15-23/squirrel-starscape-cleric?conditions=NM&finish=nonfoil" {
			continue
		}
		found = true
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatalf("GetUUID(%s) = %v", id, err)
		}
		if co.Identifiers["derivedTokenPair"] != "true" {
			t.Errorf("resolved to %q, want a derived token pairing", co.Name)
		}
	}
	if !found {
		t.Fatal("listing produced no inventory entry, want the combined pairing to resolve")
	}
}

// TestPriceRefusesUnresolvedTokenPairing pins the reverse: Human // Food
// (Throne of Eldraine) is a real, live Mana Pool listing whose scryfall_id
// names Scryfall's own Human face alone, and mtgjson's own tokenProducts
// feed has no entry linking Human to Food at all - the plain resolution
// below would otherwise silently price this two-sided card as if it were
// a plain Human, so it must produce no inventory entry rather than that
// wrong one.
func TestPriceRefusesUnresolvedTokenPairing(t *testing.T) {
	b := withMagic(t)

	card := Product{
		URL:       "https://manapool.com/card/teld/2-18/human-food",
		ProductID: "0013c2f9-2009-4bba-84f2-b3ce8e684e26", SetCode: "TELD", Number: "2-18",
		Name:       "Human // Food",
		ScryfallID: "94057dc6-e589-4a29-9bda-90f5bece96c4", TcgplayerProductID: 200310,
		LanguageID: "EN", ConditionID: "LP", FinishID: "NF", LowPrice: 15, AvailableQuantity: 1,
	}

	// Confirm the risk is real: the plain, unguarded resolution this test
	// exists to prevent really does succeed, silently, on the Human face
	// alone.
	single, err := b.MatchID(card.ScryfallID, false, false)
	if err != nil {
		t.Skip("Human TELD #2 not present in this datastore")
	}
	if co, _ := b.GetUUID(single); co == nil || co.Name != "Human" {
		t.Skip("scryfall_id 94057dc6... no longer names a bare Human face in this datastore")
	}

	mp := NewScraper(b)
	mp.price([]Product{card})

	for _, entries := range mp.Inventory() {
		for _, e := range entries {
			if strings.HasPrefix(e.URL, card.URL) {
				t.Fatalf("listing produced an inventory entry (%s), want none: no combined entity is on file for Human // Food", e.URL)
			}
		}
	}
}

// TestPriceResolvesNativeCombinedPrinting pins that a two-sided token
// listing mtgjson already models as one native entity of its own - not a
// derived pairing at all - is never routed through the token-pairing
// override: Bounty: The Outsider // Wanted! (Tales of Middle-earth
// Commander) is filed as a single card whose own Name already reads
// "X // Y", and Mana Pool's own scryfall_id for it is that entity's own
// real id, not a lone face's.
func TestPriceResolvesNativeCombinedPrinting(t *testing.T) {
	b := withMagic(t)

	card := Product{
		URL:       "https://manapool.com/card/totc/36/bounty-the-outsider-wanted",
		ProductID: "007fa033-cbe1-4772-a11f-5f7c282b0d01", SetCode: "TOTC", Number: "36",
		Name:       "Bounty: The Outsider // Wanted!",
		ScryfallID: "92d36a9a-c39c-41e6-9f31-4fcb5e820bd9", TcgplayerProductID: 0,
		LanguageID: "EN", ConditionID: "NM", FinishID: "NF", LowPrice: 25, AvailableQuantity: 2,
	}
	mp := NewScraper(b)
	mp.price([]Product{card})

	found := false
	for id, entries := range mp.Inventory() {
		if len(entries) == 0 || !strings.HasPrefix(entries[0].URL, card.URL) {
			continue
		}
		found = true
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatalf("GetUUID(%s) = %v", id, err)
		}
		if co.Name != "Bounty: The Outsider // Wanted!" {
			t.Errorf("resolved to %q, want the native combined printing", co.Name)
		}
		if co.Identifiers["derivedTokenPair"] == "true" {
			t.Error("resolved to a derived pairing entity, want mtgjson's own native one")
		}
	}
	if !found {
		t.Skip("Bounty: The Outsider // Wanted! TOTC #36 not present in this datastore")
	}
}

// TestReplayCapturedVariants runs a captured price list through the same
// path Load takes, and pins that no row is refused as a duplicate. It needs
// the datastore and a capture of https://manapool.com/api/v1/prices/variants
// named by MANAPOOL_VARIANTS_PATH; it reports how many grades were repriced
// to a lower product, which is the change this buys.
func TestReplayCapturedVariants(t *testing.T) {
	path := os.Getenv("MANAPOOL_VARIANTS_PATH")
	if path == "" {
		t.Skip("MANAPOOL_VARIANTS_PATH not set")
	}
	b := withMagic(t)
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var doc struct {
		Data []Product `json:"data"`
	}
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		t.Fatal(err)
	}

	var logged []string
	mp := NewScraper(b)
	mp.LogCallback = func(format string, a ...any) { logged = append(logged, fmt.Sprintf(format, a...)) }
	mp.price(doc.Data)

	var dupes int
	for _, l := range logged {
		if strings.Contains(l, "duplicate") {
			dupes++
		}
	}
	if dupes != 0 {
		t.Errorf("%d rows still refused as duplicates; first: %s", dupes, logged[0])
	}
	var rows int
	for _, entries := range mp.Inventory() {
		rows += len(entries)
	}
	t.Logf("%d list rows -> %d inventory rows, %d log lines, %d duplicate refusals", len(doc.Data), rows, len(logged), dupes)
	for _, l := range logged {
		t.Logf("log: %s", l)
	}
}
