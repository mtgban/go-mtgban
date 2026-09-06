package cardtrader

import "testing"

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
// Trader spells its own way: a misspelt name, a shelf whose numbers are its
// own count, and a promo shelved under the booster it came with.
func TestYgoShelves(t *testing.T) {
	bp := Blueprint{Name: "Cyber Repair ant", Version: "Ultra Rare"}
	if got := gameName(GameYuGiOh, &bp); got != "Cyber Repair Plant" {
		t.Errorf("gameName = %q, want the catalog's spelling", got)
	}
	bp = Blueprint{Name: "Fabled Ashenveil", Version: "Common"}
	bp.Expansion.Name = "2-Player Starter Deck Yuya & Declan"
	if got := gameVariation(GameYuGiOh, &bp, "007"); got != "Common" {
		t.Errorf("gameVariation(Yuya & Declan) = %q, want the rarity alone", got)
	}
	bp = Blueprint{ID: 70125, Name: "Level Retuner", Version: "Super Rare"}
	bp.Expansion.Name = "Raging Battle"
	if got := gameEdition(GameYuGiOh, &bp); got != "Duelist Pack Collection Tin" {
		t.Errorf("gameEdition(RGBT-ENPP6) = %q, want the tin", got)
	}
	if got := gameVariation(GameYuGiOh, &bp, "006"); got != "RGBT-ENPP6 Super Rare" {
		t.Errorf("gameVariation(RGBT-ENPP6) = %q", got)
	}
	bp = Blueprint{ID: 1, Name: "Harpie Lady", Version: "Common"}
	bp.Expansion.Name = "Metal Raiders"
	if got := gameEdition(GameYuGiOh, &bp); got != "Metal Raiders" {
		t.Errorf("gameEdition(Metal Raiders) = %q", got)
	}
}
