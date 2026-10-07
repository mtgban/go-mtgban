package coolstuffinc

import (
	"testing"
)

// TestOnePieceParallelPrinting pins which "(Alternate Art)" listings are
// the catalog's Parallel: the shelf's own set must file the number's alternate
// art that way, and the name must say nothing else about the printing.
func TestOnePieceParallelPrinting(t *testing.T) {
	b := readGameDatastore(t, "onepiece", "ONEPIECE_PATH")

	const romanceDawn = "OP01 - Romance Dawn"
	for _, tt := range []struct {
		desc, id, edition, name, notes, want string
	}{
		{"a reprint set's alternate art is not the shelf's printing",
			"op01-024_586178_foil", romanceDawn, "Monkey.D.Luffy - 024 (Alternate Art)", "OP01-024", "op01-024_453509_foil"},
		{"a manga row is not the alternate art",
			"op02-013_485862_foil", "OP02 - Paramount War", "Portgas.D.Ace - 013 (Alternate Art)", "", "op02-013_486333_foil"},
		{"a set filing its alternate art under that name is left alone",
			"op02-041_482337_foil", "OP02 - Paramount War", "Monkey.D.Luffy - 041 (Alternate Art)", "", ""},
		{"a note naming the reprint keeps the reprint",
			"op01-120_586194_foil", romanceDawn, "Shanks (120) (Alternate Art)", "PRB01 Reprint - OP01-120", ""},
		{"a name saying more than the treatment is left alone",
			"op01-024_586178_foil", romanceDawn, "Monkey.D.Luffy - 024 (PRB01 Alternate Art)", "", ""},
		{"and so is one naming no treatment",
			"op01-024_453508_foil", romanceDawn, "Monkey.D.Luffy - 024", "", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := onePieceParallelPrinting(b, tt.id, tt.edition, tt.name, tt.notes)
			if got != tt.want {
				t.Errorf("onePieceParallelPrinting(%q, %q, %q, %q) = %q, want %q", tt.id, tt.edition, tt.name, tt.notes, got, tt.want)
			}
		})
	}
}
