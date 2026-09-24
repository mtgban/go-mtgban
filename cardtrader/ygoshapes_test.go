package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestYgoNumber pins the collector numbers Card Trader writes its own way:
// the blueprints filed under another card's number, the token sheets and
// the pooled promo shelves numbered by their index.
func TestYgoNumber(t *testing.T) {
	for _, tt := range []struct {
		id        int
		expansion string
		number    string
		want      string
	}{
		{70125, "Raging Battle", "006", "RGBT-ENPP6"},
		{81236, "Force of the Breaker", "001", "FOTB-ENSP1"},
		{75159, "Reshef of Destruction Promos", "001sec", "ROD-EN003"},
		{79602, "Duelist Pack: Yusei Fudo 2", "002", "DP09-EN022"},
		{1, "Token Promos 3", "003", "TKN3-EN003"},
		{1, "Token Promos 4", "024", "TKN4-EN024"},
		{1, "Duelist League Promos Upperdeck", "5-001", "DL5-EN001"},
		{1, "Duelist League Promos Upperdeck", "1-E002", "DL1-E002"},
		{1, "R Comic Book Promos", "1-001", "YR01-EN001"},
		{1, "R Comic Book Promos", "03-001", "YR03-EN001"},
		{1, "Metal Raiders", "008", "008"},
		{1, "Metal Raiders", "5-001", "5-001"},
	} {
		bp := Blueprint{ID: tt.id}
		bp.Expansion.Name = tt.expansion
		if got := ygoNumber(&bp, tt.number); got != tt.want {
			t.Errorf("ygoNumber(%d, %q, %q) = %q, want %q", tt.id, tt.expansion, tt.number, got, tt.want)
		}
	}
}

// TestYgoShelves pins what the listing path asks for on the shelves Card
// Trader spells its own way: a misspelt name and a promo shelved under the
// booster it came with.
func TestYgoShelves(t *testing.T) {
	b := &mtgmatcher.Backend{}
	bp := Blueprint{Name: "Cyber Repair ant", Version: "Ultra Rare"}
	if got := gameName(b, GameYuGiOh, &bp); got != "Cyber Repair Plant" {
		t.Errorf("gameName = %q, want the catalog's spelling", got)
	}
	bp = Blueprint{ID: 70125, Name: "Level Retuner", Version: "Super Rare"}
	bp.Expansion.Name = "Raging Battle"
	if got := gameEdition(b, GameYuGiOh, &bp); got != "Duelist Pack Collection Tin" {
		t.Errorf("gameEdition(RGBT-ENPP6) = %q, want the tin", got)
	}
	if got := gameVariation(GameYuGiOh, &bp, "006"); got != "RGBT-ENPP6 Super Rare" {
		t.Errorf("gameVariation(RGBT-ENPP6) = %q", got)
	}
	bp = Blueprint{ID: 1, Name: "Harpie Lady", Version: "Common"}
	bp.Expansion.Name = "Metal Raiders"
	if got := gameEdition(b, GameYuGiOh, &bp); got != "Metal Raiders" {
		t.Errorf("gameEdition(Metal Raiders) = %q", got)
	}
	// The blueprint Card Trader actually sells writes the copyright line
	// where the rarity would go; spelling it out is what lets the matcher
	// pick MRD-008's original-artwork half over its new-art twin.
	bp = Blueprint{ID: 77934, Name: "Harpie Lady", Version: "©1996"}
	bp.Expansion.Name = "Metal Raiders"
	if got := gameVariation(GameYuGiOh, &bp, "008"); got != "008 Original Artwork" {
		t.Errorf("gameVariation(Harpie Lady MRD-008) = %q, want the artwork spelled out", got)
	}
}

// TestYgoInserts pins the insert Card Trader sells as a Yu-Gi-Oh single that
// is not a card, the same way Lorcana's own filler card is skipped.
func TestYgoInserts(t *testing.T) {
	filler := Blueprint{Name: "Rainbow Front Filler Card"}
	if !unsupportedBlueprint(GameYuGiOh, &filler) {
		t.Error("Rainbow Front Filler Card should be unsupported for Yu-Gi-Oh")
	}
	if unsupportedBlueprint(GamePokemon, &filler) || unsupportedBlueprint(GameOnePiece, &filler) {
		t.Error("Rainbow Front Filler Card should only be unsupported for Yu-Gi-Oh")
	}
}
