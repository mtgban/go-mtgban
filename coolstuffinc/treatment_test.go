package coolstuffinc

import "testing"

// TestCatalogTreatment pins the patterned holos the storefront and the catalog
// disagree on. The disagreement is not cosmetic: "foil" is a finish the
// matcher already reads, so a Master Ball asked for by the storefront's name
// answers with the set's reverse holo rather than missing outright.
func TestCatalogTreatment(t *testing.T) {
	for _, tt := range []struct {
		desc, in, want string
	}{
		{"the storefront calls the pattern a foil",
			"024/086 Master Ball Foil", "024/086 Master Ball Pattern"},
		{"and the other pattern too",
			"059/131 Poke Ball Foil", "059/131 Poke Ball Pattern"},
		{"an ordinary foil is left alone",
			"099 Reverse Foil", "099 Reverse Foil"},
		{"so is a plain number",
			"024/086", "024/086"},
		{"and a wording neither spells",
			"Holo Promo", "Holo Promo"},
		{"an empty variation stays empty", "", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := catalogTreatment(tt.in); got != tt.want {
				t.Errorf("catalogTreatment(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
