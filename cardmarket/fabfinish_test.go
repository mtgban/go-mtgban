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
