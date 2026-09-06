package cardkingdom

import (
	"testing"

	"github.com/mtgban/go-cardkingdom"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestBackFace pins how the back of a double-faced token is read off the
// listing: the number from the variation when both are written there, from
// the name when bracketed into it, and left blank otherwise; and that a
// single card or a three-faced sheet stays out.
func TestBackFace(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		product   cardkingdom.Product
		ok        bool
		sku       string
		name      string
		variation string
	}{
		{
			desc: "both numbers in the variation",
			product: cardkingdom.Product{
				SKU:       "RFTM3C-0016",
				Name:      "Beast Token // Elephant Token",
				Variation: "0016 // 0018 - Ripple Foil",
			},
			ok:        true,
			sku:       "RFTM3C-0018",
			name:      "Elephant Token // Beast Token",
			variation: "0018 // 0016 - Ripple Foil",
		},
		{
			desc: "the older dash spelling",
			product: cardkingdom.Product{
				SKU:       "TC14-013",
				Name:      "Demon Token - Zombie Token",
				Variation: "Walker",
			},
			ok:        true,
			sku:       "TC14-",
			name:      "Zombie Token // Demon Token",
			variation: "Walker",
		},
		{
			desc: "the back number bracketed into the name",
			product: cardkingdom.Product{
				SKU:  "TC20-009",
				Name: "Zombie Token // Human Soldier Token (003)",
			},
			ok:   true,
			sku:  "TC20-003",
			name: "Human Soldier Token // Zombie Token",
		},
		{
			desc: "the front number bracketed into the name",
			product: cardkingdom.Product{
				SKU:  "TPIP-0018",
				Name: "Treasure Token (0018) // Wasteland Survival Guide Token",
			},
			ok:   true,
			sku:  "TPIP-",
			name: "Wasteland Survival Guide Token // Treasure Token",
		},
		{
			desc:    "a single-faced token",
			product: cardkingdom.Product{SKU: "TBLB-0007", Name: "Fish Token"},
		},
		{
			desc: "a split card the sheet sweeps in",
			product: cardkingdom.Product{
				SKU:  "TSB-106",
				Name: "Assault // Battery",
			},
		},
		{
			desc: "three faces",
			product: cardkingdom.Product{
				SKU:       "MTWOE-0017",
				Name:      "Wicked Role Token // Cursed Role Token // Treasure Token",
				Variation: "0017 // 0034",
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			back, ok := backFace(tt.product)
			if ok != tt.ok {
				t.Fatalf("backFace(%v) ok = %v, want %v", tt.product, ok, tt.ok)
			}
			if !ok {
				return
			}
			if back.SKU != tt.sku || back.Name != tt.name || back.Variation != tt.variation {
				t.Errorf("backFace(%v) = %q %q %q, want %q %q %q",
					tt.product, back.SKU, back.Name, back.Variation, tt.sku, tt.name, tt.variation)
			}
		})
	}
}

// TestMatchBack pins where the back face lands: on its own sheet by number
// when the listing writes one, by name when the sheet holds the name once,
// and nowhere when the sheet holds it more than once.
func TestMatchBack(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		product cardkingdom.Product
		landed  string
	}{
		{
			desc: "by number",
			product: cardkingdom.Product{
				SKU:       "TOTC-0018",
				Name:      "Insect Token // Elemental Token",
				Variation: "0018 // 0017",
				Edition:   "Outlaws of Thunder Junction Commander Decks",
			},
			landed: "Elemental|TOTC|17",
		},
		{
			desc: "by name",
			product: cardkingdom.Product{
				SKU:     "TSCD-023",
				Name:    "Saproling Token // Soldier Token",
				Edition: "Starter Commander Decks",
			},
			landed: "Soldier|TSCD|8",
		},
		{
			desc: "a name the sheet holds twice",
			product: cardkingdom.Product{
				SKU:     "TC15-002",
				Name:    "Angel Token - Knight Token (Stewart)",
				Edition: "Commander 2015",
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			back, ok := backFace(tt.product)
			if !ok {
				t.Fatalf("backFace(%v) found no back face", tt.product)
			}
			cardID, err := matchBack(back)
			if tt.landed == "" {
				if err == nil {
					t.Fatalf("matchBack(%v) = %s, want an error", back, cardID)
				}
				return
			}
			if err != nil {
				t.Fatalf("matchBack(%v) = %v", back, err)
			}
			co, _ := mtgmatcher.GetUUID(cardID)
			if got := co.Name + "|" + co.SetCode + "|" + co.Number; got != tt.landed {
				t.Errorf("matchBack(%v) = %s, want %s", back, got, tt.landed)
			}
		})
	}
}
