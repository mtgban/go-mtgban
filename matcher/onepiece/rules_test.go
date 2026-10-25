package onepiece

import "testing"

// TestTruncates pins the reading that lets a wording unseat the set code
// the storefront wrote: only a name with its tail cut off, spelled in the
// order the set spells it. Which words the event sets append to their base
// name says nothing on its own - fifteen of them draw the marker from one
// vocabulary, and "Cards" closes all fifteen while never opening one - so
// only where a word sits tells "Pillars of Strength Cards" from "Pillars
// of Strength Pre-Release". The four Super Pre-Release decks spell their
// marker in front instead, leaving no leading run of the base name to cut.
func TestTruncates(t *testing.T) {
	tests := []struct {
		edition, name string
		want          bool
	}{
		{"The Azure Sea's Seven Release", "The Azure Sea's Seven Release Event Cards", true},
		{"The Azure Sea's Seven", "The Azure Sea's Seven Release Event Cards", true},
		{"The Azure Sea's Seven Cards", "The Azure Sea's Seven Release Event Cards", false},
		{"Pillars of Strength Pre-Release", "Pillars of Strength Pre-Release Cards", true},
		{"Pillars of Strength Cards", "Pillars of Strength Pre-Release Cards", false},
		{"Starter Deck 4: Animal Kingdom Pirates", "Super Pre-Release Starter Deck 4: Animal Kingdom Pirates", false},
		{"Premium Booster", "Premium Booster -The Best- Vol. 2", true},
		{"Premium Booster -The Best- Promo", "Premium Booster -The Best- Vol. 2", false},
		{"Extra Booster: Memorial Collection", "Extra Booster: Memorial Collection", true},
		{"Extra Booster: Memorial Collection Cards", "Extra Booster: Memorial Collection", false},
		{"", "Romance Dawn", false},
	}

	for _, test := range tests {
		if got := truncates(test.edition, test.name); got != test.want {
			t.Errorf("truncates(%q, %q) = %v, want %v", test.edition, test.name, got, test.want)
		}
	}
}

// TestFullNumberShapes pins which collector numbers count as one, the
// shape extractNumber reads before it falls back to the first
// digit-leading word. cardtrader's collectorNumberRe holds a copy of this
// regexp to decide whether a blueprint's Version may ride behind the
// number, and cardtrader/utils_test.go asserts the same table: a shape
// gained here has to be gained there too, or the scraper starts appending
// wording to a number the matcher cannot parse.
func TestFullNumberShapes(t *testing.T) {
	tests := map[string]bool{
		"OP01-001":   true,
		"P-043":      true,
		"OP01-001a":  true,
		"OP07-047P2": false,
		"P-L":        false,
		"2024":       false,
		"":           false,
	}

	for number, want := range tests {
		if got := fullNumberRe.MatchString(number); got != want {
			t.Errorf("fullNumberRe.MatchString(%q) = %v, want %v", number, got, want)
		}
	}
}

// TestTreatmentSaid pins how the gold-bordered half of a DON!! pair is
// asked for, against the art sentences a storefront publishes beside the
// name. Both say the word; only one of them is a claim about the printing.
func TestTreatmentSaid(t *testing.T) {
	tests := map[string]bool{
		"PRB-01 (Pink Glove) 02 Vinsmoke Reiju Gold Foil": true,
		"Gold text/Border Boa Hancock GOLD":               true,
		"(Gold Border)(We'll Have To Break Out Of This)":  true,
		"Alternate Art - Luffy (Gold Ver.) Luffy Gold":    true,
		"PRB-01 (Gold Hook) 10 Crocodile":                 false,
		"PRB-02 (Gold Bell) 43 Kalgara":                   false,
		"Alternate Art - I'm gonna ring that golden bell": false,
		"Alternate Art - I'd Like To Meet This Fellow!!!": false,
		"": false,
	}

	for wording, want := range tests {
		if got := treatmentSaid(wording); got != want {
			t.Errorf("treatmentSaid(%q) = %v, want %v", wording, got, want)
		}
	}
}
