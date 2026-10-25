package onepiece

import "testing"

// TestErrataRun pins the two spellings a wording carries a corrected run in.
// Cardtrader letters the collector number with the Greek letter and writes
// the word out in the version text, and a listing quoting only one of them
// still names the run.
func TestErrataRun(t *testing.T) {
	for _, tt := range []struct {
		desc string
		text string
		want string
	}{
		{"the letter on the number", "OP01-002α", "alpha"},
		{"the other letter", "OP01-002β", "beta"},
		{"the word in the version text", "ST03-009 Alpha Errata Card", "alpha"},
		{"the word beside the letter", "OP01-047β Beta Pre-Errata Card", "beta"},
		{"a letter behind the art suffix", "OP01-002aβ", "beta"},
		{"a letter ahead of it", "OP01-016βa", "beta"},
		{"the catalog's own label", "Parallel Alpha Pre-Errata", "alpha"},
		{"a correction naming no run", "OP01-003 Pre-Errata Card", ""},
		{"the plain label", "Pre-Errata", ""},
		{"a wording about something else", "OP16 Release Event", ""},
		{"nothing at all", "", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := errataRun(tt.text); got != tt.want {
				t.Errorf("errataRun(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

// TestErrataTreatment pins what a corrected printing's label says beyond the
// correction and the run. The catalog spells the treatment several ways, so
// the words are read off the label rather than looked for: a rule assuming
// the alternate art is always called "Parallel" prices OP01-064, whose
// alternate art this catalog files as a Box Topper.
func TestErrataTreatment(t *testing.T) {
	for _, tt := range []struct {
		label string
		want  string
	}{
		{"Pre-Errata", ""},
		{"Alpha Pre-Errata", ""},
		{"Beta Pre-Errata", ""},
		{"Parallel Pre-Errata", "Parallel"},
		{"Parallel Alpha Pre-Errata", "Parallel"},
		{"Parallel Beta Pre-Errata", "Parallel"},
		{"Box Topper Pre-Errata", "Box Topper"},
		{"Pre-Errata Demo Deck", "Demo Deck"},
		{"", ""},
	} {
		t.Run(tt.label, func(t *testing.T) {
			if got := errataTreatment(tt.label); got != tt.want {
				t.Errorf("errataTreatment(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}
