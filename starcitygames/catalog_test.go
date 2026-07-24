package starcitygames

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestResolveProductForeignSets covers inherently foreign sets: their canonical
// mtgban card is the foreign printing, so a foreign-language product resolves to
// it directly (Match's English-only language gate is bypassed). This includes
// Foreign Black Border (4BB, and SCG's 3BB -> FBB) and the Italian Legends and
// The Dark (LEGITA/DRKITA). A foreign card of an English-primary set with no
// distinct foreign set (Portal Three Kingdoms) must stay unmatched rather than
// collapse onto the English printing.
func TestResolveProductForeignSets(t *testing.T) {
	a1 := []struct {
		name, sku, lang, num, wantSet, wantNum string
	}{
		{"Mishra's Factory", "SGL-MTG-4BB-361-KON", "Korean", "361", "4BB", "361"},
		{"Vesuvan Doppelganger", "SGL-MTG-3BB-88-ITN", "Italian", "88", "FBB", "88"},
		{"Caverns of Despair", "SGL-MTG-LEG-136-ITN", "Italian", "136", "LEGITA", "136ita"},
		{"Season of the Witch", "SGL-MTG-DRK-52-ITN", "Italian", "52", "DRKITA", "52ita"},
	}
	for _, tt := range a1 {
		t.Run(tt.name, func(t *testing.T) {
			id, err := resolveProduct(GameMagic, CatalogProduct{
				SKU: tt.sku, Name: tt.name, Game: "Magic: The Gathering",
				Language: tt.lang, CollectorNumber: tt.num,
				Finish: "Non-foil", FinishGroup: "Non-foil",
			})
			if err != nil {
				t.Fatalf("resolveProduct: %v", err)
			}
			co, _ := mtgmatcher.GetUUID(id)
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("got %s #%s, want %s #%s", co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
		})
	}

	// Foreign Portal Three Kingdoms has no distinct set, so it must not collapse
	// onto the English printing.
	if id, err := resolveProduct(GameMagic, CatalogProduct{
		SKU: "SGL-MTG-PTK-137-JAN", Name: "Hua Tuo, Honored Physician", Game: "Magic: The Gathering",
		Language: "Japanese", CollectorNumber: "137", Finish: "Non-foil", FinishGroup: "Non-foil",
	}); err == nil {
		co, _ := mtgmatcher.GetUUID(id)
		t.Errorf("Portal Three Kingdoms Japanese collapsed onto %s #%s, want unmatched", co.SetCode, co.Number)
	}
}

// TestResolveProductWARJapanese covers the War of the Spark Japanese
// planeswalkers: SCG's "-WAR2-" SKU with a Japanese-language Scryfall id the
// index doesn't carry. It must resolve to the jpwalker printing WAR #NNN★,
// honoring the foil flag, rather than being rejected as non-english.
func TestResolveProductWARJapanese(t *testing.T) {
	tests := []struct {
		name     string
		sku      string
		finish   string
		wantNum  string
		wantFoil bool
	}{
		{"nonfoil", "SGL-MTG-WAR2-184-JAN", "Non-foil", "184★", false},
		{"foil", "SGL-MTG-WAR2-184-JAF", "Foil", "184★", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := resolveProduct(GameMagic, CatalogProduct{
				SKU:             tt.sku,
				Name:            "Ajani, the Greathearted",
				Game:            "Magic: The Gathering",
				Set:             "War of the Spark",
				Finish:          tt.finish,
				FinishGroup:     tt.finish,
				Language:        "Japanese",
				CollectorNumber: "184",
			})
			if err != nil {
				t.Fatalf("resolveProduct: %v", err)
			}
			co, _ := mtgmatcher.GetUUID(id)
			if co.SetCode != "WAR" || co.Number != tt.wantNum || co.Foil != tt.wantFoil {
				t.Errorf("got %s #%s foil=%v, want WAR #%s foil=%v", co.SetCode, co.Number, co.Foil, tt.wantNum, tt.wantFoil)
			}
		})
	}
}

// TestResolveProduct exercises the two resolution paths: the Scryfall shortcut
// (present + resolvable) and the SKU/preprocess fallback (no Scryfall id).
func TestResolveProduct(t *testing.T) {
	tests := []struct {
		name string
		in   CatalogProduct
		want string
	}{
		{
			name: "scryfall shortcut foil",
			in: CatalogProduct{
				SKU:         "SGL-MTG-TOR-90-ENF",
				ScryfallID:  "6a28dd88-db90-4f02-8aa9-39051d2c4763",
				Name:        "Accelerate",
				Game:        "Magic: The Gathering",
				Set:         "Torment",
				Finish:      "Foil",
				FinishGroup: "Foil",
				Language:    "English",
			},
			want: "63c421f3-b215-5f52-82b5-74300e2a5ac4_f",
		},
		{
			name: "sku fallback when scryfall missing",
			in: CatalogProduct{
				SKU:         "SGL-MTG-UMA2-32-ENF",
				ScryfallID:  "",
				Name:        "Cavern of Souls",
				Game:        "Magic: The Gathering",
				Set:         "Ultimate Masters",
				Finish:      "Foil",
				FinishGroup: "Foil",
				Language:    "English",
			},
			want: "2b0cfd28-e73e-5519-8aea-608854b0ef43",
		},
		{
			// Etched shares a Scryfall id with the plain foil; the "Etched"
			// finish recovers the flag so the shortcut still picks _e.
			name: "etched via shortcut + finish",
			in: CatalogProduct{
				SKU:             "SGL-MTG-STA2-087-JAF",
				ScryfallID:      "f31b98f3-acc0-4113-8f70-86bf2d36b9c1",
				Name:            "Agonizing Remorse",
				Game:            "Magic: The Gathering",
				Set:             "Strixhaven Mystical Archive",
				Finish:          "Etched Foil",
				FinishGroup:     "Alt Foil",
				Language:        "Japanese",
				CollectorNumber: "87",
			},
			want: "d50b8669-352c-58d9-8cb4-6352e1f0a5ee_e",
		},
		{
			// A non-etched alt-foil (surge/rainbow/cold) shares the plain foil's
			// printing, so the shortcut with etched=false is correct.
			name: "alt-foil non-etched via shortcut",
			in: CatalogProduct{
				SKU:         "SGL-MTG-TOR-90-ENF",
				ScryfallID:  "6a28dd88-db90-4f02-8aa9-39051d2c4763",
				Name:        "Accelerate",
				Game:        "Magic: The Gathering",
				Set:         "Torment",
				Finish:      "Surge Foil",
				FinishGroup: "Alt Foil",
				Language:    "English",
			},
			want: "63c421f3-b215-5f52-82b5-74300e2a5ac4_f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveProduct(GameMagic, tt.in)
			if err != nil {
				t.Fatalf("resolveProduct: %v", err)
			}
			if got != tt.want {
				co, _ := mtgmatcher.GetUUID(got)
				t.Errorf("got %q (%s), want %q", got, co, tt.want)
			}
		})
	}
}

// TestResolveProductTCGPlayerID round-trips a real TCGplayer id: with the
// scryfall id absent, the tcgplayer id alone must resolve to the same card.
func TestResolveProductTCGPlayerID(t *testing.T) {
	var tcgID, wantUUID string
	for _, uuid := range mtgmatcher.GetUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil || co.Foil || co.Etched {
			continue
		}
		if id := co.Identifiers["tcgplayerProductId"]; id != "" {
			tcgID, wantUUID = id, uuid
			break
		}
	}
	if tcgID == "" {
		t.Skip("no tcgplayerProductId in datastore")
	}

	got, err := resolveProduct(GameMagic, CatalogProduct{
		SKU: "SGL-MTG-NONE-1-ENN", Name: "ignored", Game: "Magic: The Gathering",
		TCGPlayerID: tcgID, Finish: "Non-foil", FinishGroup: "Non-foil",
	})
	if err != nil {
		t.Fatalf("resolveProduct: %v", err)
	}
	if got != wantUUID {
		t.Errorf("got %s, want %s", got, wantUUID)
	}
}

// TestDecodeCatalog verifies the streaming decoder over the documented example.
func TestDecodeCatalog(t *testing.T) {
	const payload = `[
      {
        "id": 434, "sku": "SGL-MTG-TOR-90-ENF",
        "scryfall_id": "6a28dd88-db90-4f02-8aa9-39051d2c4763",
        "url": "/accelerate-sgl-mtg-tor-90-enf/", "name": "Accelerate",
        "game": "Magic: The Gathering", "set": "Torment",
        "finish": "Foil", "finish_group": "Foil", "language": "English",
        "collector_number": "90",
        "variants": [
          { "id": 535, "sku": "SGL-MTG-TOR-90-ENF1", "condition": "Near Mint",
            "qty": 0, "price": "8.99", "is_on_discount": false, "sell_list_price": "4.0000" },
          { "id": 536, "sku": "SGL-MTG-TOR-90-ENF2", "condition": "Played",
            "qty": 1, "price": "5.95", "is_on_discount": false, "sell_list_price": "2.0000" }
        ]
      }
    ]`

	var got []CatalogProduct
	err := decodeCatalog(strings.NewReader(payload), func(p CatalogProduct) error {
		got = append(got, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d products, want 1", len(got))
	}
	p := got[0]
	if p.ScryfallID != "6a28dd88-db90-4f02-8aa9-39051d2c4763" {
		t.Errorf("scryfall_id = %q", p.ScryfallID)
	}
	if len(p.Variants) != 2 {
		t.Fatalf("got %d variants, want 2", len(p.Variants))
	}
	if p.Variants[1].SellListPrice != "2.0000" || p.Variants[1].Condition != "Played" {
		t.Errorf("variant[1] = %+v", p.Variants[1])
	}
}
