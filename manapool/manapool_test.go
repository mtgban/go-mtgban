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
// it arrives with, whichever product arrives first, holding every product's
// copies.
func TestAddCheapestKeepsTheLowerPrice(t *testing.T) {
	mp := NewScraper(&mtgmatcher.Backend{})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: mtgban.NM, Price: 25, Available: 3, URL: "first"})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: mtgban.NM, Price: 15, Available: 1, URL: "cheaper"})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: mtgban.NM, Price: 40, Available: 2, URL: "dearer"})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: mtgban.SP, Price: 9, URL: "other-grade"})

	got := map[mtgban.Condition]mtgban.InventoryEntry{}
	for _, e := range mp.Inventory()["card"] {
		got[e.Conditions] = e
	}
	if len(got) != 2 {
		t.Fatalf("published %d grades, want NM and SP: %+v", len(got), got)
	}
	if got[mtgban.NM].Price != 15 || got[mtgban.NM].URL != "cheaper" {
		t.Errorf("NM is %.0f from %q, want 15 from the cheaper product", got[mtgban.NM].Price, got[mtgban.NM].URL)
	}
	if got[mtgban.NM].Available != 6 || got[mtgban.NM].Quantity != 1 {
		t.Errorf("NM has %d available and quantity %d, want the 6 of all three products and one", got[mtgban.NM].Available, got[mtgban.NM].Quantity)
	}
	if got[mtgban.SP].Price != 9 {
		t.Errorf("SP is %.0f, want 9", got[mtgban.SP].Price)
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
		t.Fatal("Human TELD #2 not present in this datastore")
	}
	if co, _ := b.GetUUID(single); co == nil || co.Name != "Human" {
		t.Fatal("scryfall_id 94057dc6... no longer names a bare Human face in this datastore")
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
		t.Fatal("Bounty: The Outsider // Wanted! TOTC #36 not present in this datastore")
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
	mp.logCallback = func(format string, a ...any) { logged = append(logged, fmt.Sprintf(format, a...)) }
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

// TestIndexFoilMarketNeedsAFoilFinish pins that a foil market is recorded on
// the etched copy of a printing sold nonfoil and etched, and not at all on one
// sold nonfoil only, rather than as a second price beside the nonfoil one. A
// printing sold etched only keeps its foil market on its own etched copy, not
// beside the foil one of the number it shares a name with.
func TestIndexFoilMarketNeedsAFoilFinish(t *testing.T) {
	b := withMagic(t)

	rows := []Product{
		{Name: "Carrion Feeder", SetCode: "SLD", Number: "1114", ScryfallID: "c0e175cb-ffea-473f-8422-53273f263016",
			PriceMarket: 4488, PriceMarketFoil: 2546, URL: "https://manapool.com/card/sld/1114/carrion-feeder"},
		{Name: "Arcane Signet", SetCode: "FDC", Number: "245", ScryfallID: "ee7710cf-e73d-479f-bc8d-0e78a0e324d9",
			PriceMarket: 38, PriceMarketFoil: 58, URL: "https://manapool.com/card/fdc/245/arcane-signet"},
		{Name: "Demonlord Belzenlok", SetCode: "SLD", Number: "159", ScryfallID: "27197660-8489-419b-9ad6-29a8713e4673",
			PriceMarket: 155, PriceMarketFoil: 193, URL: "https://manapool.com/card/sld/159/demonlord-belzenlok"},
		{Name: "Demonlord Belzenlok", SetCode: "SLD", Number: "159★", ScryfallID: "5d58cb4d-2091-40c8-b97c-09bf9c022a8b",
			PriceMarketFoil: 241, URL: "https://manapool.com/card/sld/159%E2%98%85/demonlord-belzenlok"},
	}
	mp := NewScraperIndex(b)
	mp.price(rows)

	prices := map[string][]float64{}
	for id, entries := range mp.Inventory() {
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatalf("GetUUID(%s) = %v", id, err)
		}
		for _, e := range entries {
			key := co.SetCode + " " + co.Number
			if co.Foil {
				key += " foil"
			}
			if co.Etched {
				key += " etched"
			}
			prices[key] = append(prices[key], e.Price)
		}
	}
	if got := prices["SLD 1114"]; len(got) != 1 || got[0] != 44.88 {
		t.Errorf("nonfoil Carrion Feeder is %v, want its market alone", got)
	}
	if got := prices["SLD 1114 etched"]; len(got) != 1 || got[0] != 25.46 {
		t.Errorf("etched Carrion Feeder is %v, want the foil market", got)
	}
	if got := prices["FDC 245"]; len(got) != 1 || got[0] != 0.38 {
		t.Errorf("Arcane Signet is %v, want its nonfoil market alone", got)
	}
	if got := prices["SLD 159"]; len(got) != 1 || got[0] != 1.55 {
		t.Errorf("nonfoil Demonlord Belzenlok is %v, want its market alone", got)
	}
	if got := prices["SLD 159 foil"]; len(got) != 1 || got[0] != 1.93 {
		t.Errorf("foil Demonlord Belzenlok 159 is %v, want its own foil market alone", got)
	}
	if got := prices["SLD 159★ etched"]; len(got) != 1 || got[0] != 2.41 {
		t.Errorf("etched-only Demonlord Belzenlok 159★ is %v, want its foil market", got)
	}
}

// TestPriceKeepsTheListedLanguage pins that a row is priced when its code
// names the language of the printing its scryfall id resolves to, Phyrexian
// included, and dropped when it names another one.
func TestPriceKeepsTheListedLanguage(t *testing.T) {
	b := withMagic(t)

	rows := []Product{
		{Name: "Elesh Norn, Mother of Machines", SetCode: "ONE", Number: "414",
			ScryfallID: "09705595-47c6-4f7c-9351-4004bfa39218", LanguageID: "PH",
			ConditionID: "NM", FinishID: "NF", LowPrice: 2725, AvailableQuantity: 4,
			URL: "https://manapool.com/card/one/414/elesh-norn-mother-of-machines"},
		{Name: "Marang River Regent // Coil and Catch", SetCode: "TDM", Number: "378",
			ScryfallID: "484b5580-b179-4dce-8bdf-d714eb4635e5", LanguageID: "JA",
			ConditionID: "NM", FinishID: "NF", LowPrice: 111, AvailableQuantity: 74,
			URL: "https://manapool.com/card/tdm/378/marang-river-regent-coil-and-catch"},
	}
	mp := NewScraper(b)
	mp.price(rows)

	if len(mp.Inventory()) != 1 {
		t.Fatalf("priced %d printings, want the Phyrexian one alone", len(mp.Inventory()))
	}
	for id := range mp.Inventory() {
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatalf("GetUUID(%s) = %v", id, err)
		}
		if co.Language != "Phyrexian" {
			t.Errorf("priced a %s printing, want Phyrexian", co.Language)
		}
	}
}

// TestPriceResolvesReversibleCardInTokenLookalikeSet pins that a listing named
// "X // Y" in a set of real cards is not taken for a pairing of two tokens
// because the set's code starts with a T: Marang River Regent // Coil and
// Catch (Tarkir: Dragonstorm) is a card the datastore files under its front
// face.
func TestPriceResolvesReversibleCardInTokenLookalikeSet(t *testing.T) {
	b := withMagic(t)

	card := Product{
		URL:       "https://manapool.com/card/tdm/378/marang-river-regent-coil-and-catch",
		ProductID: "71fddb8b-7d80-4a59-b10a-fa4cabb9aaef", SetCode: "TDM", Number: "378",
		Name:       "Marang River Regent // Coil and Catch",
		ScryfallID: "484b5580-b179-4dce-8bdf-d714eb4635e5", TcgplayerProductID: 623988,
		LanguageID: "EN", ConditionID: "NM", FinishID: "NF", LowPrice: 111, AvailableQuantity: 74,
	}
	mp := NewScraper(b)
	mp.price([]Product{card})

	for id := range mp.Inventory() {
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatalf("GetUUID(%s) = %v", id, err)
		}
		if co.SetCode != "TDM" || co.Number != "378" {
			t.Errorf("landed on %s %s, want TDM 378", co.SetCode, co.Number)
		}
		return
	}
	t.Fatal("listing produced no inventory entry")
}
