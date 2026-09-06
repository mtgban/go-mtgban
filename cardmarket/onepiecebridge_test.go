package cardmarket

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
)

// onePieceDatastore is the published One Piece datastore cut down to the
// three printings this test turns on, copied verbatim: one collector number
// sold as a base art and two alternates, which is the shape the catalog
// cannot tell apart. Each carries its own TCGplayer id, which is what the
// bridge names them by.
const onePieceDatastore = `{
 "game": "onepiece",
 "sets": {"OP05": {"name": "Awakening of the New Era", "releaseDate": "2023-11-25"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 527875}, "finish": "Foil", "id": "op05-069_527875_foil", "name": "Trafalgar Law", "number": "OP05-069", "rarity": "SR", "setCode": "OP05"},
  {"externalLinks": {"tcgPlayerId": 527662}, "finish": "Foil", "id": "op05-069_527662_foil", "name": "Trafalgar Law", "number": "OP05-069", "rarity": "SR", "setCode": "OP05", "variant": "Alternate Art"},
  {"externalLinks": {"tcgPlayerId": 527019}, "finish": "Foil", "id": "op05-069_527019_foil", "name": "Trafalgar Law", "number": "OP05-069", "rarity": "SR", "setCode": "OP05", "variant": "Alternate Art Manga"}
 ]
}`

// TestOnePieceBridgeNamesThePrinting pins that the bridge names the printing
// where the catalog only counts. Cardmarket sells the three arts as three
// products and tells them apart with a V-index of its own ordering, so the
// index is a guess: the run this was written against had V.2 and V.3 landing
// on each other's printing. The bridge says which outright, and where it says
// nothing the catalog still names what it can.
func TestOnePieceBridgeNamesThePrinting(t *testing.T) {
	if err := mtgmatcher.LoadDatastore(strings.NewReader(onePieceDatastore)); err != nil {
		t.Fatal(err)
	}

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
			mkm := &Index{
				gameID:     GameOnePiece,
				TCGBridge:  tt.bridge,
				priceGuide: map[int]PriceGuide{tt.mkmID: {IDProduct: tt.mkmID, LowPrice: 1, TrendPrice: 2}},
			}
			product := MKMProduct{
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
