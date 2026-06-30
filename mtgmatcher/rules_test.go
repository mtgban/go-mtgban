package mtgmatcher

import "testing"

// A Backend built by hand — no loader, so SetRules was never called — must
// fail cleanly rather than panic on every public entry point that consults
// the game rules.
func TestBackendWithoutRules(t *testing.T) {
	var b Backend
	b.CanonicalNames = map[string]string{}

	if _, err := b.GetSetByName("Some Set"); err != ErrDatastoreEmpty {
		t.Errorf("GetSetByName = %v, want %v", err, ErrDatastoreEmpty)
	}

	// Populate Sets so Match passes its emptiness check and actually reaches
	// the rules-dependent paths: the bracketed name routes through
	// GetSetByName (whose adjust step must be skipped without rules) and then
	// the nil-rules guard must fire before the pipeline proper.
	b.Sets = map[string]*Set{"XXX": {Name: "Some Other Set", Code: "XXX"}}
	if _, err := b.Match(&InputCard{Name: "Anything [Some Set]"}); err != ErrDatastoreEmpty {
		t.Errorf("Match = %v, want %v", err, ErrDatastoreEmpty)
	}
	if _, err := b.GetSetByName("Some Set"); err != ErrCardNotInEdition {
		t.Errorf("GetSetByName with sets = %v, want %v", err, ErrCardNotInEdition)
	}
	if b.hasPrinting("Anything", "promo_type", "boosterfun") {
		t.Errorf("hasPrinting = true, want false")
	}
}
