package starcitygames

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

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
