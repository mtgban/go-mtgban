package strikezone

import (
	"errors"
	"testing"
)

func TestParseDetails(t *testing.T) {
	tests := []struct {
		details   string
		treatment string
		run       string
		language  string
	}{
		{"Near Mint Normal 1st Edition English", "Normal", "1st Edition", "English"},
		{"Light Play Normal Unlimited English", "Normal", "Unlimited", "English"},
		{"Near Mint Normal English", "Normal", "", "English"},
		{"Near Mint Parallel Foil 1st Edition English", "Parallel Foil", "1st Edition", "English"},
		{"Near Mint Cold Foil 1st Edition English", "Cold Foil", "1st Edition", "English"},
		{"Near Mint Rainbow Foil Unlimited English", "Rainbow Foil", "Unlimited", "English"},
		{"Medium Play Normal Unlimited Chinese", "Normal", "Unlimited", "Chinese"},
		{"Near Mint Rainbow Foil 1st Edition Japanese", "Rainbow Foil", "1st Edition", "Japanese"},
		{"Near Mint Foil English", "Foil", "", "English"},
	}
	for _, tt := range tests {
		treatment, run, language := parseDetails(tt.details)
		if treatment != tt.treatment || run != tt.run || language != tt.language {
			t.Errorf("parseDetails(%q) = %q, %q, %q; want %q, %q, %q",
				tt.details, treatment, run, language, tt.treatment, tt.run, tt.language)
		}
	}
}

func TestPreprocessDetails(t *testing.T) {
	tests := []struct {
		game    string
		name    string
		edition string
		number  string
		details string

		outName      string
		outVariation string
		outFinish    string
		outFoil      bool
		err          error
	}{
		{
			game: GamePokemon, name: "Eevee V", edition: "Crown Zenith",
			number: "108", details: "Near Mint Normal 1st Edition English",
			outName: "Eevee V", outVariation: "108 1st Edition",
		},
		{
			game: GamePokemon, name: "Charizard", edition: "Base Set Unlimited",
			number: "004", details: "Light Play Normal Unlimited English",
			outName: "Charizard", outVariation: "004 Unlimited",
		},
		{
			game: GamePokemon, name: "Kingdra", edition: "Aquapolis",
			number: "148", details: "Near Mint Parallel Foil English",
			outName: "Kingdra", outVariation: "148", outFinish: "Reverse Holofoil",
		},
		{
			// A buylist name repeats the number after a dash.
			game: GamePokemon, name: "Amarys - 170", edition: "Scarlet and Violet Prismatic Evolutions",
			number: "170", details: "Near Mint Normal 1st Edition English",
			outName: "Amarys", outVariation: "170 1st Edition",
		},
		{
			// The name's tail spells the promo number more fully than the
			// Number column does.
			game: GamePokemon, name: "Dragapult (Prime) - SWSH132", edition: "Promos Sword and Shield",
			number: "132", details: "Near Mint Normal 1st Edition English",
			outName: "Dragapult (Prime)", outVariation: "SWSH132 1st Edition",
		},
		{
			// The gallery subsets number their cards with a prefix the bare
			// Number column drops.
			game: GamePokemon, name: "Bidoof", edition: "Crown Zenith: Galarian Gallery",
			number: "029", details: "Near Mint Normal 1st Edition English",
			outName: "Bidoof", outVariation: "GG029 1st Edition",
		},
		{
			game: GamePokemon, name: "Mewtwo", edition: "Legendary Collection",
			number: "010", details: "Light Play Normal Unlimited Chinese",
			err: errForeignListing,
		},
		{
			// Super Slam was printed once: the stamp names no run.
			game: GameFleshAndBlood, name: "Apex Bonebreaker", edition: "Super Slam",
			number: "008", details: "Near Mint Cold Foil 1st Edition English",
			outName: "Apex Bonebreaker", outVariation: "008",
			outFinish: "Cold Foil", outFoil: true,
		},
		{
			game: GameFleshAndBlood, name: "Command and Conquer", edition: "Crucible of War",
			number: "063", details: "Near Mint Rainbow Foil Unlimited English",
			outName: "Command and Conquer", outVariation: "063",
			outFinish: "Unlimited Edition Rainbow Foil", outFoil: true,
		},
		{
			// A promo's number names its printing; the treatment the
			// shelf writes beside it is not trusted.
			game: GameFleshAndBlood, name: "Briar, Warden of Thorns - HER044", edition: "Flesh and Blood Promos",
			number: "044", details: "Near Mint Cold Foil 1st Edition English",
			outName: "Briar, Warden of Thorns", outVariation: "HER044", outFoil: true,
		},
		{
			game: GameFleshAndBlood, name: "Raise an Army", edition: "Heavy Hitters",
			number: "105", details: "Near Mint Normal 1st Edition English",
			outName: "Raise an Army", outVariation: "105",
			outFinish: "Normal",
		},
		{
			game: GameFleshAndBlood, name: "Fai Rising Rebellion Hero065", edition: "Flesh and Blood Promos",
			number: "Hero065", details: "Near Mint Rainbow Foil 1st Edition English",
			outName: "Fai Rising Rebellion", outVariation: "HER065", outFoil: true,
		},
		{
			game: GameFleshAndBlood, name: "Fai Hero 61", edition: "Flesh and Blood Promos",
			number: "Hero061", details: "Near Mint Cold Foil 1st Edition English",
			outName: "Fai", outVariation: "HER061", outFoil: true,
		},
		{
			game: GameFleshAndBlood, name: "- HER084Prism, Advent of Thrones", edition: "Flesh and Blood Promos",
			number: "084", details: "Near Mint Rainbow Foil 1st Edition English",
			outName: "Prism, Advent of Thrones", outVariation: "HER084", outFoil: true,
		},
		{
			game: GameFleshAndBlood, name: "Bloodrot Pox / Frailty", edition: "Flesh and Blood Promos",
			number: "LGS125LGS126", details: "Near Mint Cold Foil 1st Edition English",
			outName: "Bloodrot Pox / Frailty", outVariation: "LGS125//LGS126", outFoil: true,
		},
		{
			game: GameFleshAndBlood, name: "Theryon, Magister of Justice", edition: "Flesh and Blood Promos",
			number: "JDC008", details: "Near Mint Rainbow Foil 1st Edition English",
			outName: "Theryon, Magister of Justice", outVariation: "JDG008", outFoil: true,
		},
		{
			game: GameFleshAndBlood, name: "Hexagore, the Death Hydra - FAB186 (Golden) - FAB186", edition: "Flesh and Blood Promos",
			number: "186", details: "Near Mint Cold Foil 1st Edition English",
			outName: "Hexagore, the Death Hydra (Golden)", outVariation: "FAB186", outFoil: true,
		},
		{
			game: GameYuGiOh, name: "Dark Magician", edition: "Yugi Reloaded",
			number: "001", details: "Near Mint Normal Unlimited English",
			outName: "Dark Magician", outVariation: "001",
		},
	}
	for _, tt := range tests {
		card, err := preprocessDetails(tt.game, tt.name, tt.edition, tt.number, tt.details)
		if tt.err != nil {
			if !errors.Is(err, tt.err) {
				t.Errorf("%s/%s: error = %v; want %v", tt.game, tt.name, err, tt.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s/%s: unexpected error %v", tt.game, tt.name, err)
			continue
		}
		if card.Name != tt.outName || card.Variation != tt.outVariation ||
			card.Finish != tt.outFinish || card.Foil != tt.outFoil ||
			card.Edition != tt.edition {
			t.Errorf("%s/%s: got %q/%q/%q/%v; want %q/%q/%q/%v",
				tt.game, tt.name, card.Name, card.Variation, card.Finish, card.Foil,
				tt.outName, tt.outVariation, tt.outFinish, tt.outFoil)
		}
	}
}
