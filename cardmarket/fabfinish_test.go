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
		// No run named: the treatment alone still names a printing, unless
		// it is the plain one the id already answers with.
		{"LSS Promos", "Go Bananas (Rainbow Foil)", "Rainbow Foil"},
		{"LSS Promos", "Taylor", ""},
		{"LSS Promos", "Ruu'di, Gem Keeper (Regular)", ""},
		// A parenthetical that is part of the card's name, not a treatment.
		{"LSS Promos", "Sigil of Suffering (Yellow)", ""},
	} {
		if got := fabFinish(tt.expansion, tt.name); got != tt.want {
			t.Errorf("fabFinish(%q, %q) = %q, want %q", tt.expansion, tt.name, got, tt.want)
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
