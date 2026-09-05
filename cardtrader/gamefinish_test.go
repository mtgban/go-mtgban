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
