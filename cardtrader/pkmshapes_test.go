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

// TestPkmJapaneseShelves pins the shelves that sell only Japanese product,
// even when a seller marks a copy "en": no row in the English datastore can
// ever answer for them, so the blueprint is skipped rather than refused.
func TestPkmJapaneseShelves(t *testing.T) {
	jp := Blueprint{Name: "Dusknoir"}
	jp.Expansion.Name = "Night Wanderer"
	if !unsupportedBlueprint(GamePokemon, &jp) {
		t.Error("a Night Wanderer blueprint should be unsupported")
	}
	en := Blueprint{Name: "Dusknoir"}
	en.Expansion.Name = "Paldea Evolved"
	if unsupportedBlueprint(GamePokemon, &en) {
		t.Error("an ordinary shelf should not be unsupported")
	}
}

// TestPkmProfessorProgramShelf pins the shelf's own edition and version
// wording: the catalog's set is "Professor Program Promos" rather than the
// shelf's own "Professor Program", and every version on it leads with a
// "Professor Program Stamp" no catalog row carries as a promo type.
func TestPkmProfessorProgramShelf(t *testing.T) {
	b := &mtgmatcher.Backend{}
	bp := Blueprint{Name: "Bianca's Devotion", Version: "Professor Program Stamp | 142/162"}
	bp.Expansion.Name = "Professor Program"
	if got := gameEdition(b, GamePokemon, &bp); got != "Professor Program Promos" {
		t.Errorf("gameEdition(Professor Program) = %q", got)
	}
	if got := gameVariation(GamePokemon, &bp, "142"); got != "142 142/162" {
		t.Errorf("gameVariation(Professor Program Stamp) = %q", got)
	}
	energy := Blueprint{Name: "Basic Fire Energy", Version: "Professor Program Stamp | Cosmos Holo SVE002"}
	energy.Expansion.Name = "Professor Program"
	if got := gameName(b, GamePokemon, &energy); got != "Fire Energy" {
		t.Errorf("gameName(Basic Fire Energy) = %q, want the Basic prefix dropped", got)
	}
	elsewhere := Blueprint{Name: "Basic Fire Energy"}
	elsewhere.Expansion.Name = "Base Set"
	if got := gameName(b, GamePokemon, &elsewhere); got != "Basic Fire Energy" {
		t.Errorf("gameName(Basic Fire Energy, Base Set) = %q, want it left alone off the shelf", got)
	}
}

// TestPkmHolidayCalendarShelf pins the shelf's own stamp wording, which the
// catalog spells "Holiday Calendar" rather than Card Trader's "Holiday
// Snowflake Stamp".
func TestPkmHolidayCalendarShelf(t *testing.T) {
	bp := Blueprint{Name: "Pikachu ex", Version: "063 | Holiday Snowflake Stamp"}
	bp.Expansion.Name = "Holiday Calendar"
	if got := gameVariation(GamePokemon, &bp, "063"); got != "063 063 | Holiday Calendar" {
		t.Errorf("gameVariation(Holiday Snowflake Stamp) = %q", got)
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
