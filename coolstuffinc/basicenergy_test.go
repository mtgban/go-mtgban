package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPokemonBasicEnergyName pins the Scarlet & Violet / Mega Evolution era
// basic energies to the catalog's "Basic <Type> Energy" spelling. The
// storefront's own product code ("SVE019") glued onto the number is what
// says which of a set's several basic-energy printings this one is, and a
// shelf that prints its own bare numbers (Shrouded Fable's 1-8) is read off
// the shelf itself rather than redirected to the code's own set.
func TestPokemonBasicEnergyName(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []struct {
		name, edition, wantID string
	}{
		{"Water Energy - SVE019 (Reverse Foil)", "SV Black Bolt", "019_645288_reverseholofoil"},
		{"Water Energy - MEE003 (Reverse Foil)", "ME Ascended Heroes", "3_656265_reverseholofoil"},
		{"Grass Energy - SVE001 (Reverse Foil)", "SV Shrouded Fable", "1_562155_reverseholofoil"},
		{"Grass Energy (Secret Rare) - 278/193", "SV Paldea Evolved", "278-193_497700_holofoil"},
		// Another shelf (World Championship Decks, Prize Pack Series)
		// prints its own bare-numbered energy at this number too; the
		// SVE/MEE code must still redirect to its own set.
		{"Darkness Energy - SVE015", "SV Stellar Crown", "015_578868"},
		{"Fire Energy - MEE002", "Mega Evolution", "2_656264"},
		{"Water Energy - MEE003", "Mega Evolution", "3_656265"},
		{"Lightning Energy - MEE004", "Mega Evolution", "4_656266"},
		// CSI's "(Cosmo Holo)" bracket (catalog: "Cosmos Holo") must reach
		// the SVE holofoil row it names, not the plain nonfoil.
		{"Metal Energy (Cosmo Holo) - SVE008", "SV 151", "008_517184_holofoil"},
		{"Darkness Energy (Cosmo Holo) - SVE007", "SV 151", "007_517183_holofoil"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := pokemonListing(b, tt.name, tt.edition, "", false)
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("match(%q) = %v", card, err)
			}
			if id != tt.wantID {
				co, _ := b.GetUUID(id)
				t.Errorf("match(%q) = %q (%s/%s/%s), want %q", card, id, co.SetCode, co.Number, co.Finish, tt.wantID)
			}
		})
	}
}

// TestPokemonBasicEnergyRefusesUnnumbered pins that a bare "<Type> Energy"
// carrying only a finish word or a retail note - no numbered tail - is
// refused rather than priced as the set's lone Hyper Rare of that name.
func TestPokemonBasicEnergyRefusesUnnumbered(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []struct {
		energyType, bracket, edition, numbered, variation string
	}{
		{"Water", "Non-Holo", "SV Paldea Evolved", "", ""},
		{"Water", "Reverse Foil", "SV Paldea Evolved", "", ""},
		{"Water", "", "SV Paldea Evolved", "", "Reverse Foil"},
		{"Psychic", "", "SV 151", "", "Cosmos Holo"},
		{"Fire", "", "SV Obsidian Flames", "", "Reverse Foil"},
		// A year is not a collector number; the plain path must not
		// land on the set's lone Hyper Rare of that name.
		{"Fire", "", "SV Obsidian Flames", "2023", ""},
		{"Water", "", "SV Paldea Evolved", "2023", ""},
	} {
		t.Run(tt.energyType+"/"+tt.edition, func(t *testing.T) {
			card := pokemonBasicEnergy(b, tt.energyType, tt.bracket, tt.edition, tt.numbered, tt.variation, false)
			if card != nil {
				t.Errorf("pokemonBasicEnergy(%q, %q) = %v, want nil", tt.energyType, tt.edition, card)
			}
		})
	}
}

// TestPokemonBasicEnergyPlainPathChecksSet pins that a plain "<Type> Energy"
// listing - a bare number, no SVE/MEE code - refuses when the shelf it
// arrived on carries no Basic <Type> Energy at that number, rather than
// accepting whichever other set's printing happens to share the number: SV
// Twilight Masquerade carries neither "8" nor "7", but Mega Evolution's
// Metal and Darkness energies do, and the plain path must refuse both
// rather than land on either.
func TestPokemonBasicEnergyPlainPathChecksSet(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []string{
		"Metal Energy - 8",
		"Darkness Energy - 7",
	} {
		t.Run(tt, func(t *testing.T) {
			card := pokemonListing(b, tt, "SV Twilight Masquerade", "", false)
			id, err := b.Match(card)
			if err == nil {
				co, _ := b.GetUUID(id)
				t.Errorf("match(%q) = %q (%s/%s), want a refusal", card, id, co.SetCode, co.Number)
			}
		})
	}
}
