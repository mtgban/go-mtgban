package cardmarket

import (
	"errors"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
)

// onePieceDatastore is the published One Piece datastore cut down to the
// three printings this test turns on, copied verbatim: one collector number
// sold as a base art and two alternates, which is the shape the catalog
// cannot tell apart. Each carries its own TCGplayer id, which is what the
// bridge names them by.
const onePieceDatastore = `{"data": {
 "game": "onepiece",
 "sets": {"OP05": {"name": "Awakening of the New Era", "releaseDate": "2023-11-25"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 527875}, "finish": "Foil", "id": "op05-069_527875_foil", "name": "Trafalgar Law", "number": "OP05-069", "rarity": "SR", "setCode": "OP05"},
  {"externalLinks": {"tcgPlayerId": 527662}, "finish": "Foil", "id": "op05-069_527662_foil", "name": "Trafalgar Law", "number": "OP05-069", "promoTypes": ["Alternate Art"], "rarity": "SR", "setCode": "OP05", "variant": "Alternate Art"},
  {"externalLinks": {"tcgPlayerId": 527019}, "finish": "Foil", "id": "op05-069_527019_foil", "name": "Trafalgar Law", "number": "OP05-069", "promoTypes": ["Alternate Art Manga"], "rarity": "SR", "setCode": "OP05", "variant": "Alternate Art Manga"}
 ]
}}`

// TestOnePieceBridgeNamesThePrinting pins that the bridge names the printing
// where the catalog only counts. Cardmarket sells the three arts as three
// products and tells them apart with a V-index of its own ordering, so the
// index is a guess: the run this was written against had V.2 and V.3 landing
// on each other's printing. The bridge says which outright, and where it says
// nothing the catalog still names what it can.
func TestOnePieceBridgeNamesThePrinting(t *testing.T) {
	b := datastoreBackend(t, "onepiece", onePieceDatastore)

	for _, tt := range []struct {
		desc    string
		mkmID   int
		bridge  map[int]int
		product string
		want    string
	}{
		{
			desc:  "the bridge names the alternate art the index only counts",
			mkmID: 100, bridge: map[int]int{100: 527662},
			product: "Trafalgar Law (OP05-069) (V.2)", want: "op05-069_527662_foil",
		},
		{
			desc:  "and the manga alternate, which shares its number",
			mkmID: 101, bridge: map[int]int{101: 527019},
			product: "Trafalgar Law (OP05-069) (V.3)", want: "op05-069_527019_foil",
		},
		{
			desc:  "a product the bridge does not know is still named by the catalog",
			mkmID: 102, bridge: map[int]int{},
			product: "Trafalgar Law (OP05-069)", want: "op05-069_527875_foil",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			mkm, err := NewScraperIndex(b)
			if err != nil {
				t.Fatalf("NewScraperIndex(b) = %v", err)
			}
			mkm.tcgBridge = tt.bridge
			mkm.priceGuide = map[int]cm.PriceGuide{tt.mkmID: {IDProduct: tt.mkmID, LowPrice: 1, TrendPrice: 2}}
			product := cm.Product{
				IDProduct:     tt.mkmID,
				Name:          tt.product,
				Number:        "OP05-069",
				ExpansionName: "Awakening of the New Era",
			}
			channel := make(chan responseChan, 8)
			if err := mkm.processProduct(channel, &product); err != nil {
				t.Fatalf("processProduct(%q) = %v", tt.product, err)
			}
			close(channel)
			var got string
			for res := range channel {
				if res.cardID != "" {
					got = res.cardID
					break
				}
			}
			if got != tt.want {
				t.Errorf("processProduct(%q) named %q, want %q", tt.product, got, tt.want)
			}
		})
	}
}

// onePieceReprintDatastore holds a reprint set that keeps a card under its
// original starter-deck number rather than the base set's, for pinning the
// Reprints/Demo Decks wrong-variant refusal.
const onePieceReprintDatastore = `{"data": {
 "game": "onepiece",
 "sets": {"OP-RP": {"name": "Revision Pack Cards", "releaseDate": "2023-01-01"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 451393}, "finish": "Foil", "id": "st01-001_451393_foil", "name": "Monkey.D.Luffy", "number": "ST01-001", "rarity": "L", "setCode": "OP-RP"}
 ]
}}`

// TestOnePieceReprintWrongVariantIsSilent pins that a Reprints product named
// with the base set's own number, which the reprint set does not carry at
// that number, refuses silently instead of surfacing a matcher error.
func TestOnePieceReprintWrongVariantIsSilent(t *testing.T) {
	b := datastoreBackend(t, "onepiece", onePieceReprintDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	mkm.priceGuide = map[int]cm.PriceGuide{765980: {IDProduct: 765980, LowPrice: 1, TrendPrice: 2}}
	product := cm.Product{
		IDProduct:     765980,
		Name:          "Monkey.D.Luffy (OP02-041)",
		Number:        "OP02-041",
		ExpansionName: "Reprints",
	}
	channel := make(chan responseChan, 8)
	err = mkm.processProduct(channel, &product)
	if !errors.Is(err, errNoPrinting) {
		t.Errorf("processProduct(765980) = %v, want errNoPrinting", err)
	}
}

// onePieceEventDatastore holds a Winner Pack copy the promo set labels with
// the event Cardmarket's Winner Cards shelf carries only in its own name.
const onePieceEventDatastore = `{"data": {
 "game": "onepiece",
 "sets": {"OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-09-30"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 708453}, "finish": "Foil", "id": "op14-087_708453_foil", "name": "Miss.Valentine (Mikita)", "number": "OP14-087", "promoTypes": ["winnerpack"], "rarity": "R", "setCode": "OP-PR", "variant": "Winner Pack 2026 Vol. 3"}
 ]
}}`

// TestOnePieceEventLabelReachesThePrinting pins that a Winner Cards product
// reaches its Winner Pack copy once the shelf's own event label is appended
// to its number, which the product's own wording never carries.
func TestOnePieceEventLabelReachesThePrinting(t *testing.T) {
	b := datastoreBackend(t, "onepiece", onePieceEventDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	mkm.priceGuide = map[int]cm.PriceGuide{896411: {IDProduct: 896411, LowPrice: 1, TrendPrice: 2}}
	product := cm.Product{
		IDProduct:     896411,
		Name:          "Miss.Valentine(Mikita) (OP14-087)",
		Number:        "OP14-087",
		ExpansionName: "Winner Cards",
	}
	channel := make(chan responseChan, 8)
	err = mkm.processProduct(channel, &product)
	if err != nil {
		t.Fatalf("processProduct(896411) = %v", err)
	}
	close(channel)
	var got string
	for res := range channel {
		if res.cardID != "" {
			got = res.cardID
			break
		}
	}
	want := "op14-087_708453_foil"
	if got != want {
		t.Errorf("processProduct(896411) named %q, want %q", got, want)
	}
}

// onePieceOffCodeDatastore adds a second printing at a different number
// beside the base OP05-069 art, to pin offCode's guard.
const onePieceOffCodeDatastore = `{"data": {
 "game": "onepiece",
 "sets": {"OP05": {"name": "Awakening of the New Era", "releaseDate": "2023-11-25"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 527875}, "finish": "Foil", "id": "op05-069_527875_foil", "name": "Trafalgar Law", "number": "OP05-069", "rarity": "SR", "setCode": "OP05"},
  {"externalLinks": {"tcgPlayerId": 599999}, "finish": "Foil", "id": "op05-070_599999_foil", "name": "Someone Else", "number": "OP05-070", "rarity": "SR", "setCode": "OP05"}
 ]
}}`

// TestOnePieceOffCodeFallsToWording pins offCode: a bridge link is another
// marketplace's, and one whose printing's number disagrees with the code
// the product's own name carries must not be trusted - the product falls to
// the wording path instead.
func TestOnePieceOffCodeFallsToWording(t *testing.T) {
	b := datastoreBackend(t, "onepiece", onePieceOffCodeDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	// The bridge points at OP05-070, but the name carries OP05-069.
	mkm.tcgBridge = map[int]int{200: 599999}
	mkm.priceGuide = map[int]cm.PriceGuide{200: {IDProduct: 200, LowPrice: 1, TrendPrice: 2}}
	product := cm.Product{
		IDProduct:     200,
		Name:          "Trafalgar Law (OP05-069)",
		Number:        "OP05-069",
		ExpansionName: "Awakening of the New Era",
	}
	channel := make(chan responseChan, 8)
	err = mkm.processProduct(channel, &product)
	if err != nil {
		t.Fatalf("processProduct(%q) = %v", product.Name, err)
	}
	close(channel)
	var got string
	for res := range channel {
		if res.cardID != "" {
			got = res.cardID
			break
		}
	}
	want := "op05-069_527875_foil"
	if got != want {
		t.Errorf("processProduct(%q) named %q, want %q", product.Name, got, want)
	}
}

// onePieceShelfCodeDatastore is a synthetic Demo Decks printing pair, to pin
// the off-code fallback against an aliased shelf: the product's own Number
// field carries another product's code, while its name still carries the
// right one. It makes no claim about any real Cardmarket product's data.
const onePieceShelfCodeDatastore = `{"data": {
 "game": "onepiece",
 "sets": {"OP-DD": {"name": "One Piece Demo Deck Cards", "releaseDate": "2022-07-08"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 602795}, "finish": "Normal", "id": "op01-013_602795", "name": "Sanji", "number": "OP01-013", "rarity": "R", "setCode": "OP-DD"},
  {"externalLinks": {"tcgPlayerId": 599999}, "finish": "Normal", "id": "op01-999_599999", "name": "Someone Else", "number": "OP01-999", "rarity": "R", "setCode": "OP-DD"}
 ]
}}`

// TestOnePieceShelfCodeFallsToWording pins that a Demo Decks product whose
// bridge is off-code still lands, on the wording path, off the code its own
// name carries rather than the wrong number Cardmarket filed it under. The
// product id and its data are synthetic.
func TestOnePieceShelfCodeFallsToWording(t *testing.T) {
	b := datastoreBackend(t, "onepiece", onePieceShelfCodeDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	// The bridge points at a printing the name's own code disagrees with.
	mkm.tcgBridge = map[int]int{900001: 599999}
	mkm.priceGuide = map[int]cm.PriceGuide{900001: {IDProduct: 900001, LowPrice: 1, TrendPrice: 2}}
	product := cm.Product{
		IDProduct:     900001,
		Name:          "Sanji (OP01-013)",
		Number:        "ST13-016",
		ExpansionName: "Demo Decks",
	}
	channel := make(chan responseChan, 8)
	err = mkm.processProduct(channel, &product)
	if err != nil {
		t.Fatalf("processProduct(%q) = %v", product.Name, err)
	}
	close(channel)
	var got string
	for res := range channel {
		if res.cardID != "" {
			got = res.cardID
			break
		}
	}
	want := "op01-013_602795"
	if got != want {
		t.Errorf("processProduct(%q) named %q, want %q", product.Name, got, want)
	}
}
