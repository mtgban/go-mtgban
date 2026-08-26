package cardmarket

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
)

// offShelfDatastore is the published One Piece datastore cut down to two
// collector numbers, verbatim.
//
// OP02-063 is a booster card the tournament programme handed out again, and
// every one of those copies is labelled with the event that gave it away.
// OP12-046 is a booster card the Kuzan deck reprints, which the datastore
// does not carry - though it does carry the deck's reprint of OP12-047,
// which is how a deck's shelf is supposed to reach a card.
const offShelfDatastore = `{
 "game": "onepiece",
 "sets": {
  "OP02": {"name": "Paramount War", "releaseDate": "2023-03-10"},
  "OP02-PRE": {"name": "Paramount War Pre-Release Cards", "releaseDate": "2023-03-03"},
  "OP12": {"name": "Legacy of the Master", "releaseDate": "2025-08-22"},
  "OP12-RE": {"name": "Legacy of the Master Release Event Cards", "releaseDate": "2025-08-15"},
  "OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-09-30"},
  "ST-33": {"name": "Starter Deck 33: BLUE Kuzan", "releaseDate": "2026-07-31"}
 },
 "cards": [
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 486401}, "finish": "Normal", "id": "op02-063_486401", "name": "Mr.1 (Daz.Bonez)", "number": "OP02-063", "rarity": "UC", "setCode": "OP02", "type": "Character"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 486747}, "finish": "Normal", "id": "op02-063_486747", "name": "Mr.1 (Daz.Bonez)", "number": "OP02-063", "rarity": "UC", "setCode": "OP02-PRE", "type": "Character"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 583758}, "finish": "Foil", "id": "op02-063_583758_foil", "name": "Mr.1 [Finalist] (Daz.Bonez)", "number": "OP02-063", "rarity": "UC", "setCode": "OP-PR", "type": "Character", "variant": "Offline Regional 2024 Vol. 3"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 585982}, "finish": "Normal", "id": "op02-063_585982", "name": "Mr.1 (Daz.Bonez)", "number": "OP02-063", "rarity": "UC", "setCode": "OP-PR", "type": "Character", "variant": "Online Regional 2024 Vol. 3"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 586026}, "finish": "Normal", "id": "op02-063_586026", "name": "Mr.1 (Daz.Bonez)", "number": "OP02-063", "rarity": "UC", "setCode": "OP-PR", "type": "Character", "variant": "Offline Regional 2024 Vol. 3"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 643781}, "finish": "Normal", "id": "op12-046_643781", "name": "Zephyr (Navy)", "number": "OP12-046", "rarity": "C", "setCode": "OP12", "type": "Character"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 649276}, "finish": "Normal", "id": "op12-046_649276", "name": "Zephyr (Navy)", "number": "OP12-046", "rarity": "C", "setCode": "OP12-RE", "type": "Character"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 643782}, "finish": "Foil", "id": "op12-047_643782_foil", "name": "Sengoku", "number": "OP12-047", "rarity": "R", "setCode": "OP12", "type": "Character"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 706331}, "finish": "Normal", "id": "op12-047_706331", "name": "Sengoku", "number": "OP12-047", "rarity": "R", "setCode": "ST-33", "type": "Character"}
 ]
}`

// opShelves are the expansions of the cut-down catalog, the ones the two
// products above sit on and the ones already pricing what they reached.
var opShelves = []MKMExpansion{
	{Name: "Paramount War"},
	{Name: "Promos: Paramount War"},
	{Name: "Legacy of the Master"},
	{Name: "Special Tournament Promos"},
	{Name: "Starter Deck: Blue Kuzan"},
}

// TestOffShelf pins the two products PR #224 resolved onto a printing
// another product of the same run already prices. Cardmarket sells them on a
// tournament shelf and in a starter deck; the printing they reached is the
// plain booster card, which the booster's own shelf sells as a product of
// its own. A refusal says less than a price, and claims nothing.
func TestOffShelf(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(offShelfDatastore))
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		what    string
		product MKMProduct
		want    string
	}{
		{
			what: "the tournament copy of a booster card is not the booster card",
			product: MKMProduct{
				IDProduct: 794707, Name: "Mr.1 (Daz.Bonez) (OP02-063) (V.1)",
				Number: "OP02-063", ExpansionName: "Special Tournament Promos",
			},
		},
		{
			what: "a deck reprint the datastore does not carry is not the booster card",
			product: MKMProduct{
				IDProduct: 900308, Name: "Zephyr(Navy) (OP12-046)",
				Number: "OP12-046", ExpansionName: "Starter Deck: Blue Kuzan",
			},
		},
		{
			what: "the booster's own shelf keeps the booster card",
			product: MKMProduct{
				IDProduct: 700916, Name: "Mr.1 (Daz.Bonez) (OP02-063)",
				Number: "OP02-063", ExpansionName: "Paramount War",
			},
			want: "op02-063_486401",
		},
		{
			what: "the booster's own shelf keeps the other booster card",
			product: MKMProduct{
				IDProduct: 842945, Name: "Zephyr(Navy) (OP12-046)",
				Number: "OP12-046", ExpansionName: "Legacy of the Master",
			},
			want: "op12-046_643781",
		},
		{
			what: "a shelf naming no set of ours keeps what only it prices",
			product: MKMProduct{
				IDProduct: 701207, Name: "Mr.1 (Daz.Bonez) (OP02-063)",
				Number: "OP02-063", ExpansionName: "Promos: Paramount War",
			},
			want: "op02-063_486747",
		},
		{
			what: "a deck reprint the datastore does carry stays on the deck's shelf",
			product: MKMProduct{
				IDProduct: 900309, Name: "Sengoku (OP12-047)",
				Number: "OP12-047", ExpansionName: "Starter Deck: Blue Kuzan",
			},
			want: "op12-047_706331",
		},
	} {
		t.Run(tt.what, func(t *testing.T) {
			mkm := &Index{
				gameID:       GameOnePiece,
				exchangeRate: 1,
				shelved:      shelvedSets(opShelves),
				priceGuide: map[int]PriceGuide{
					tt.product.IDProduct: {IDProduct: tt.product.IDProduct, LowPrice: 1, TrendPrice: 2},
				},
			}
			channel := make(chan responseChan, 8)
			err := mkm.processProduct(channel, &tt.product)
			close(channel)

			if tt.want == "" {
				if !errors.Is(err, errNoPrinting) {
					t.Fatalf("product %d returned %v, want %v", tt.product.IDProduct, err, errNoPrinting)
				}
				if len(channel) != 0 {
					t.Errorf("a refused product priced %d entries, want none", len(channel))
				}
				return
			}
			if err != nil {
				t.Fatalf("product %d returned %v, want a price", tt.product.IDProduct, err)
			}
			var priced int
			for out := range channel {
				priced++
				if out.cardID != tt.want {
					t.Errorf("product %d priced %s, want %s", tt.product.IDProduct, out.cardID, tt.want)
				}
			}
			if priced == 0 {
				t.Errorf("product %d priced nothing, want %s", tt.product.IDProduct, tt.want)
			}
		})
	}
}

// TestShelvedSets pins which expansions of a catalog name a set of ours: the
// booster and the deck do, and the promo buckets Cardmarket invents do not.
func TestShelvedSets(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(offShelfDatastore))
	if err != nil {
		t.Fatal(err)
	}

	shelved := shelvedSets(opShelves)
	want := map[string]string{
		"OP02":  "Paramount War",
		"OP12":  "Legacy of the Master",
		"ST-33": "Starter Deck: Blue Kuzan",
	}
	if len(shelved) != len(want) {
		t.Fatalf("shelvedSets = %v, want %v", shelved, want)
	}
	for code, name := range want {
		if shelved[code] != name {
			t.Errorf("shelvedSets[%q] = %q, want %q", code, shelved[code], name)
		}
	}
}
