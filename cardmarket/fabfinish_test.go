package cardmarket

import "testing"

// TestFabFinish pins the printing a Cardmarket product names, which it does
// in two places at once: the print run in the expansion, the treatment in a
// parenthetical after the card's name. Naming only one of the two reaches no
// printing, since the datastore crosses them.
func TestFabFinish(t *testing.T) {
	for _, tt := range []struct{ expansion, name, want string }{
		{"Tales of Aria - First", "Cracker Jax (Cold Foil)", "1st Edition Cold Foil"},
		{"Tales of Aria - First", "Deep Blue (Regular)", "1st Edition Normal"},
		{"Tales of Aria - Unlimited", "Thump (Blue) (Rainbow Foil)", "Unlimited Edition Rainbow Foil"},
		{"Tales of Aria - Unlimited", "Runaways (Regular)", "Unlimited Edition Normal"},
		// Welcome to Rathe's first run is the one the catalog calls Alpha,
		// and the datastore files under every other set's 1st Edition.
		{"Welcome to Rathe - Alpha", "Last Ditch Effort (Regular)", "1st Edition Normal"},
		{"Welcome to Rathe - Alpha", "Sink Below (Red) (Rainbow Foil)", "1st Edition Rainbow Foil"},
		// No run named: the treatment alone still names a printing, the
		// plain one included.
		{"LSS Promos", "Go Bananas (Rainbow Foil)", "Rainbow Foil"},
		{"LSS Promos", "Taylor", ""},
		{"LSS Promos", "Ruu'di, Gem Keeper (Regular)", "Normal"},
		// A parenthetical that is part of the card's name, not a treatment.
		{"LSS Promos", "Sigil of Suffering (Yellow)", ""},
		// The art spelled ahead of the treatment says nothing about the
		// finish, and the treatment behind it says all of it.
		{"GEM Pack Promos", "Display Loyalty (Extended Art Rainbow Foil)", "Rainbow Foil"},
		{"GEM Pack Promos", "Fast and Furious (Extended Art Regular)", "Normal"},
		{"Crucible of War - First", "Twinning Blade (Extended Art Rainbow Foil)", "1st Edition Rainbow Foil"},
	} {
		if got := fabFinish(tt.expansion, tt.name); got != tt.want {
			t.Errorf("fabFinish(%q, %q) = %q, want %q", tt.expansion, tt.name, got, tt.want)
		}
	}
}

// TestFabPrintRun pins the split matchProduct looks a set up through: the
// datastore has one Monarch, Cardmarket has one expansion per print run of
// it, and the suffix that says which run is not part of any set's name.
// Only the run suffixes come off - Cardmarket spells plenty of other things
// after a dash, and those expansions name sets of their own.
func TestFabPrintRun(t *testing.T) {
	for _, tt := range []struct{ expansion, run, set string }{
		{"Monarch - First", "1st Edition", "Monarch"},
		{"Monarch - Unlimited", "Unlimited Edition", "Monarch"},
		{"Welcome to Rathe - Alpha", "1st Edition", "Welcome to Rathe"},
		{"Everfest - First", "1st Edition", "Everfest"},
		// The set sold in a single run says nothing, and neither does a
		// suffix naming something other than a run.
		{"Uprising", "", "Uprising"},
		{"Monarch - Boltyn Blitz Deck", "", "Monarch - Boltyn Blitz Deck"},
		{"History Pack 1 - Black Label", "", "History Pack 1 - Black Label"},
		{"Silver Age Deck - Bravo", "", "Silver Age Deck - Bravo"},
		// The suffix is the whole tail or it is not a run.
		{"First", "", "First"},
		{"Arcane Rising - Firstborn", "", "Arcane Rising - Firstborn"},
	} {
		run, set := fabPrintRun(tt.expansion)
		if run != tt.run || set != tt.set {
			t.Errorf("fabPrintRun(%q) = (%q, %q), want (%q, %q)",
				tt.expansion, run, set, tt.run, tt.set)
		}
	}
}

// TestFabTreatment pins which parenthetical comes off a product name as the
// printing's treatment and which stays as part of the card's own name.
func TestFabTreatment(t *testing.T) {
	for _, tt := range []struct{ in, treatment, card string }{
		{"Cracker Jax (Cold Foil)", "Cold Foil", "Cracker Jax"},
		{"Deep Blue (Regular)", "Normal", "Deep Blue"},
		{"Thump (Blue) (Rainbow Foil)", "Rainbow Foil", "Thump (Blue)"},
		{"Sigil of Suffering (Yellow)", "", "Sigil of Suffering (Yellow)"},
		{"Taylor", "", "Taylor"},
		// A set selling one card in several arts spells the art ahead of the
		// treatment. Only the treatment comes off: the art belongs to the
		// printing, which the datastore keeps a row of its own for.
		{"Twinning Blade (Extended Art Rainbow Foil)", "Rainbow Foil", "Twinning Blade (Extended Art)"},
		{"Twelve Petal Kasaya (Extended Art Cold Foil)", "Cold Foil", "Twelve Petal Kasaya (Extended Art)"},
		{"Fast and Furious (Extended Art Regular)", "Normal", "Fast and Furious (Extended Art)"},
		{"Channel Lake Frigid (Alternate Art Rainbow Foil)", "Rainbow Foil", "Channel Lake Frigid (Alternate Art)"},
		{"Man Overboard (Red) (Extended Art Rainbow Foil)", "Rainbow Foil", "Man Overboard (Red) (Extended Art)"},
		// The treatment is the tail the parenthetical ends on or it is not
		// one: "Golden" is an art of its own and names no finish.
		{"Sonata Arcanix (Cold Foil Golden)", "", "Sonata Arcanix (Cold Foil Golden)"},
	} {
		treatment, card := fabTreatment(tt.in)
		if treatment != tt.treatment || card != tt.card {
			t.Errorf("fabTreatment(%q) = (%q, %q), want (%q, %q)",
				tt.in, treatment, card, tt.treatment, tt.card)
		}
	}
}

// TestVersionTail pins the parenthetical Cardmarket tells same-name products
// apart with, which names its own version index and the rarity beside it and
// says nothing a matcher can use. A parenthetical that is part of the card's
// name stays.
func TestVersionTail(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"Charubin the Fire Knight (V.3 - Rare)", "Charubin the Fire Knight"},
		{"Dissolverock (V.1 - Common)", "Dissolverock"},
		{"Eevee (V.2)", "Eevee"},
		{"Sigil of Suffering (Yellow)", "Sigil of Suffering (Yellow)"},
		{"Go Bananas (Rainbow Foil)", "Go Bananas (Rainbow Foil)"},
	} {
		if got := versionTail.ReplaceAllString(tt.in, ""); got != tt.want {
			t.Errorf("strip(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestNumberTail pins the digits two catalogs numbering the same card agree
// about: Cardmarket numbers the oldest Yu-Gi-Oh sets by their original Asian
// print, the datastore by set.
func TestNumberTail(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"A015", "015"},
		{"LOB-015", "015"},
		{"001", "001"},
		{"155b", "155b"},
		{"", ""},
	} {
		if got := numberTail.FindString(tt.in); got != tt.want {
			t.Errorf("numberTail(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
