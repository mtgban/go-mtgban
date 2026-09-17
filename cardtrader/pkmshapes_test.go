package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPkmShelves pins what the listing path asks for on the Pokemon shelves
// Card Trader spells its own way: the SV promo shelf reaches the promo set
// unless the version says the card is a jumbo, the league shelf's number
// field carries a year or an online code rather than a collector number,
// and a marker is no card.
func TestPkmShelves(t *testing.T) {
	b := &mtgmatcher.Backend{}
	bp := Blueprint{Name: "Team Rocket's Mewtwo ex", Version: "SVP 205"}
	bp.Expansion.Name = "SV Black Star Promos"
	if got := gameEdition(b, GamePokemon, &bp); got != "SV: Scarlet & Violet Promo Cards" {
		t.Errorf("gameEdition(SV Black Star Promos) = %q", got)
	}
	bp.Version = "Jumbo Oversized | SVP 205"
	if got := gameEdition(b, GamePokemon, &bp); got != "SV Black Star Promos" {
		t.Errorf("gameEdition(jumbo) = %q, want the shelf kept", got)
	}
	for _, tt := range []struct{ version, number, want string }{
		{"FFE-9JT-SUX | 2006 Non-Holo Promo", "2006", "FFE-9JT-SUX | 2006 Non-Holo Promo"},
		{"2006 Unnumbered", "KUF-7XB-05C", "2006 Unnumbered"},
		{"Pokémon League | 15/114", "015", "015 Pokémon League | 15/114"},
		{"Regional Championships | 119/132", "034", "034 Regional Championships | 119/132"},
	} {
		bp := Blueprint{Name: "Fire Energy", Version: tt.version}
		bp.Expansion.Name = "League Promos"
		if got := gameVariation(GamePokemon, &bp, tt.number); got != tt.want {
			t.Errorf("gameVariation(League Promos, %q, %q) = %q, want %q", tt.version, tt.number, got, tt.want)
		}
	}
	marker := Blueprint{Name: "VSTAR Marker"}
	marker.Expansion.Name = "Brilliant Stars"
	if !unsupportedBlueprint(GamePokemon, &marker) {
		t.Error("a VSTAR Marker is no card")
	}
	if unsupportedBlueprint(GameYuGiOh, &marker) || unsupportedBlueprint(GamePokemon, &bp) {
		t.Error("the Pokemon insert table matched the wrong game or blueprint")
	}
}

func TestLorcanaInserts(t *testing.T) {
	filler := Blueprint{Name: "Discard Filler Card"}
	if !unsupportedBlueprint(GameLorcana, &filler) {
		t.Error("Discard Filler Card should be unsupported for Lorcana")
	}
	if unsupportedBlueprint(GamePokemon, &filler) || unsupportedBlueprint(GameOnePiece, &filler) {
		t.Error("Discard Filler Card should only be unsupported for Lorcana")
	}
}
