package coolstuffinc

import (
	"testing"
)

// TestOnePieceRenamedTreatment pins what the rename may and may not reach.
// The guard that carries it is the number: a card whose own number holds a
// real Full Art is answered by it, and one holding a single alternate
// printing under another label is the one the storefront means.
func TestOnePieceRenamedTreatment(t *testing.T) {
	b := readGameDatastore(t, "onepiece", "ONEPIECE_PATH")

	for _, tt := range []struct {
		desc, id, name, want string
	}{
		{"a word the set does not use reaches the one printing it can mean",
			"st21-003_615568_foil", "Sanji - 003 (Full Art)", "st21-003_615569_foil"},
		{"a number holding the word's own printing is left alone",
			"st29-002_671548", "Nami - 002 (Full Art)", ""},
		{"a set wearing several labels still answers a number holding one",
			"st30-012_693396_foil", "Monkey.D.Luffy - 012 (Full Art)", "st30-012_693397_foil"},
		{"a second qualifier is not answered by the treatment alone",
			"op05-006_527663_foil", "Koala (006) (Dash Pack) (Full Art)", ""},
		{"a listing naming nothing is left alone",
			"st21-003_615568_foil", "Sanji - 003", ""},
		{"and so is one already on the printing it names",
			"st21-003_615569_foil", "Sanji - 003 (Full Art)", ""},
		{"a word the catalog never uses reaches nothing",
			"st21-003_615568_foil", "Sanji - 003 (Shiny Chrome)", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := onePieceRenamedTreatment(b, tt.id, tt.name); got != tt.want {
				t.Errorf("onePieceRenamedTreatment(%q, %q) = %q, want %q", tt.id, tt.name, got, tt.want)
			}
		})
	}
}
