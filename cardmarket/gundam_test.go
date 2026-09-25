package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/gundam"
)

// gundamDatastore is the published Gundam datastore cut down to the rows
// these tests turn on, images left out.
const gundamDatastore = `{"data": {
 "game": "gundam",
 "sets": {
  "GCG-PR": {"name": "Gundam Promotional Cards", "releaseDate": "2025-03-01", "type": "promo"},
  "GD01": {"name": "Newtype Rising", "releaseDate": "2025-07-25"},
  "GD03": {"name": "Steel Requiem", "releaseDate": "2026-01-30"},
  "GD05": {"name": "Freedom Ascension", "releaseDate": "2026-07-24"},
  "ST09": {"name": "Starter Deck 09: Destiny Ignition", "releaseDate": "2026-03-27"}
 },
 "cards": [
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 643150}, "finish": "Holofoil", "id": "gd01-001_643150_holofoil", "name": "Gundam", "number": "GD01-001", "rarity": "Legend Rare", "setCode": "GD01", "type": "Unit"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 645356}, "finish": "Holofoil", "id": "gd01-001_645356_holofoil", "name": "Gundam", "number": "GD01-001", "rarity": "LR+", "setCode": "GD01", "type": "Unit"},
  {"color": "Blue", "externalLinks": {"tcgPlayerId": 645375}, "finish": "Holofoil", "id": "gd01-001_645375_holofoil", "name": "Gundam", "number": "GD01-001", "rarity": "LR++", "setCode": "GD01", "type": "Unit"},
  {"externalLinks": {"tcgPlayerId": 673538}, "finish": "Normal", "id": "t-020_673538", "name": "GFreD Token", "number": "T-020", "rarity": "Common", "setCode": "GD03", "type": "Unit"},
  {"externalLinks": {"tcgPlayerId": 680689}, "finish": "Holofoil", "id": "t-020_680689_holofoil", "name": "GFreD Token", "number": "T-020", "promoTypes": ["premiumcardcollection"], "rarity": "Common", "setCode": "GCG-PR", "type": "Unit", "variant": "Premium Card Collection Gundam Assemble"},
  {"color": "Purple", "externalLinks": {"tcgPlayerId": 684001}, "finish": "Holofoil", "id": "st09-002_684001_holofoil", "name": "Force Impulse Gundam", "number": "ST09-002", "rarity": "Legend Rare", "setCode": "ST09", "type": "Unit"},
  {"color": "Purple", "externalLinks": {"tcgPlayerId": 684026}, "finish": "Holofoil", "id": "st09-001_684026_holofoil", "name": "Impulse Gundam", "number": "ST09-001", "rarity": "LR+", "setCode": "ST09", "type": "Unit"},
  {"color": "Purple", "externalLinks": {"tcgPlayerId": 705650}, "finish": "Holofoil", "id": "gd05-114_705650_holofoil", "name": "Widespread Annihilation", "number": "GD05-114", "rarity": "Rare", "setCode": "GD05", "type": "Command"},
  {"color": "Purple", "finish": "Holofoil", "id": "gd05-114-premium-card-collection-02_holofoil", "name": "Widespread Annihilation", "number": "GD05-114", "promoTypes": ["premiumcardcollection"], "rarity": "Rare", "setCode": "GCG-PR", "type": "Command", "variant": "Premium Card Collection 02"}
 ]
}}`

// TestGundamResolve pins how a Cardmarket Gundam product finds its printing:
// the parallel by its rarity spelled the catalog's way, a CardTrader link
// held to the number the product's name writes rather than to its spelling,
// and a Premium Card Collection 02 print by its product id. The products and
// links are the catalog's and CardTrader's own.
func TestGundamResolve(t *testing.T) {
	b := datastoreBackend(t, "gundam", gundamDatastore)

	for _, tt := range []struct {
		desc    string
		id      int
		product cm.CatalogProduct
		shelf   string
		bridge  map[int]int
		want    string
	}{
		{
			"a parallel by its rarity", 905530,
			cm.CatalogProduct{Name: "Gundam (GD01-001) (V.2 - Legendary Rare +)", Number: "001", Rarity: "Legendary Rare +", Version: 2},
			"Newtype Rising", nil, "gd01-001_645356_holofoil",
		},
		{
			"the second parallel", 905531,
			cm.CatalogProduct{Name: "Gundam (GD01-001) (V.3 - Legendary Rare ++)", Number: "001", Rarity: "Legendary Rare ++", Version: 3},
			"Newtype Rising", nil, "gd01-001_645375_holofoil",
		},
		{
			"a link to another number is refused", 906571,
			cm.CatalogProduct{Name: "Force Impulse Gundam (ST09-002) (V.1 - Legendary Rare)", Number: "002", Rarity: "Legendary Rare", Version: 1},
			"Starter Deck: Destiny Ignition", map[int]int{906571: 684026}, "st09-002_684001_holofoil",
		},
		{
			"a link at the product's number stands, spelled apart", 908561,
			cm.CatalogProduct{Name: "GFreD (T-020) (V.1 - Token)", Number: "T-020", Rarity: "Token", Version: 1},
			"Premium Bandai Products", map[int]int{908561: 680689}, "t-020_680689_holofoil",
		},
		{
			"a Premium Card Collection 02 print", 908747,
			cm.CatalogProduct{Name: "Widespread Annihilation (GD05-114) (V.1 - Rare)", Number: "GD05-114", Rarity: "Rare", Version: 1},
			"Premium Bandai Products", nil, "gd05-114-premium-card-collection-02_holofoil",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			mkm, err := NewScraperIndex(b)
			if err != nil {
				t.Fatalf("NewScraperIndex(b) = %v", err)
			}
			mkm.tcgBridge = tt.bridge
			got := mkm.resolveMapped(tt.id, tt.product, cm.Expansion{Name: tt.shelf})
			if got.err != nil || got.cardID != tt.want {
				t.Errorf("%d %q: resolveMapped = (%q, %v), want %q", tt.id, tt.product.Name, got.cardID, got.err, tt.want)
			}
		})
	}
}
