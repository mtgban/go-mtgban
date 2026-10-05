package cardtrader

import "testing"

// TestGameFinishFleshAndBlood pins what a Flesh and Blood listing names as its
// finish. The print run and the treatment cross, and the datastore gives every
// crossing its own printing, so a listing that names only one of the two can
// settle for a printing of the other run.
func TestGameFinishFleshAndBlood(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		shelf        string
		treatment    string
		firstEdition bool
		want         string
	}{
		// The first run is named outright, whatever shelf it sits on.
		{"first edition, plain", "Monarch - First", "Regular", true, "1st Edition Normal"},
		{"first edition, foil", "Monarch - First", "Rainbow Foil", true, "1st Edition Rainbow Foil"},
		// A shelf that sells the unlimited run says so, where the flag only
		// says "not first" - which names a printing wherever the unlimited
		// run exists and names nothing where it does not.
		{"unlimited shelf, plain", "Monarch - Unlimited", "Regular", false, "Unlimited Edition Normal"},
		{"unlimited shelf, unset treatment", "Welcome to Rathe - Unlimited", "", false, "Unlimited Edition Normal"},
		{"unlimited shelf, stringly false", "Monarch - Unlimited", "false", false, "Unlimited Edition Normal"},
		// A shelf naming no run leaves the finish to the treatment, which is
		// what the promo and deck shelves have always relied on: they sell one
		// printing and the datastore labels it first edition.
		{"run-less shelf stays empty", "Ira Welcome Deck", "Regular", false, ""},
		{"run-less shelf, promo", "LGS Armory Events", "", false, ""},
		// The treatment still speaks for itself when it is not the plain one.
		{"foil needs no run", "Ira Welcome Deck", "Cold Foil", false, "Cold Foil"},
		{"foil on an unlimited shelf", "Monarch - Unlimited", "Rainbow Foil", false, "Rainbow Foil"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			bp := shelfBlueprint(tt.shelf)
			var product Product
			product.Properties.FabFoilNew = tt.treatment
			product.Properties.FirstEdition = tt.firstEdition
			if got := gameFinish(GameFleshAndBlood, bp, product); got != tt.want {
				t.Errorf("gameFinish = %q, want %q", got, tt.want)
			}
		})
	}
}

// shelfBlueprint names a blueprint's expansion, the only field the finish reads.
func shelfBlueprint(name string) *Blueprint {
	var bp Blueprint
	bp.Expansion.Name = name
	return &bp
}

// TestGameFinishPokemon pins that a Pokemon listing's finish follows the
// blueprint's version as well as its flags: a holo rare names its treatment,
// a first-edition one crosses it with the run, and a non-holo or reverse
// version never reads as a holo.
func TestGameFinishPokemon(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		version      string
		reverse      bool
		firstEdition bool
		want         string
	}{
		{"holo rare", "Holo Rare | 8/64", false, false, "Holofoil"},
		{"first edition holo rare", "Holo Rare | 8/64", false, true, "1st Edition Holofoil"},
		{"cosmos holo", "Cosmos Holo | 144/172", false, false, "Holofoil"},
		{"first edition, no treatment", "Rare | 8/64", false, true, "1st Edition"},
		{"non-holo", "Non-Holo | 053/167", false, false, ""},
		{"reverse flag wins", "Holo Rare | 8/64", true, false, "Reverse Holofoil"},
		{"reverse version", "Reverse Holo | 8/64", false, false, ""},
		{"no version", "", false, false, ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			var bp Blueprint
			bp.Version = tt.version
			var product Product
			product.Properties.PokemonReverse = tt.reverse
			product.Properties.FirstEdition = tt.firstEdition
			got := gameFinish(GamePokemon, &bp, product)
			if got != tt.want {
				t.Errorf("gameFinish = %q, want %q", got, tt.want)
			}
		})
	}
}
