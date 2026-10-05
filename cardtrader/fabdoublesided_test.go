package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// fabFixtureBackend builds a minimal in-memory backend from a handful of
// hand-written card numbers, standing in for the shapes measured on the real
// Flesh and Blood datastore without loading it: two faces shared by more
// than one double-sided pairing (MON002, mirroring Prism), a face carried by
// only one pairing (WTR078, not enough on its own), and a number that also
// has a printing of its own beside its pairings (MST158, mirroring Spectral
// Shield's real MST158-A/-B), beside an unrelated number that merely
// extends MON002 with another digit, and a number whose only standalone
// printing is a foil (UPR103, mirroring Iyslander's Marvel cold foil).
func fabFixtureBackend() *mtgmatcher.Backend {
	rows := []struct {
		number string
		foil   bool
	}{
		{"MON002//MON001", false},
		{"MON088//MON002", false},
		{"MON002//MON221", false},
		{"MON0021", false},
		{"WTR040//WTR078", false},
		{"MST158-A", false},
		{"MST003//MST158", false},
		{"MST002//MST158", false},
		{"UPR103", true},
		{"UPR103//UPR046", false},
		{"UPR102//UPR103", false},
	}
	b := &mtgmatcher.Backend{UUIDs: map[string]*mtgmatcher.CardObject{}}
	for i, row := range rows {
		uuid := "fixture" + string(rune('0'+i))
		b.UUIDs[uuid] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{Number: row.number}, Foil: row.foil}
		b.AllUUIDs = append(b.AllUUIDs, uuid)
	}
	return b
}

func TestFabDoubleSidedFace(t *testing.T) {
	b := fabFixtureBackend()

	tests := []struct {
		desc   string
		number string
		foil   bool
		want   bool
	}{
		{"a face shared by three pairings and never standalone", "MON002", false, true},
		{"a face carried by only one pairing", "WTR078", false, false},
		{"a face with a printing of its own beside its pairings", "MST158", false, false},
		{"a regular listing beside a foil-only printing of its own", "UPR103", false, true},
		{"a foil listing beside that foil printing", "UPR103", true, false},
		{"a number that names no pairing at all", "SEA999", false, false},
		{"a composite pair number is never a lone face", "MON002//MON001", false, false},
		{"empty", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := fabDoubleSidedFace(b, tt.number, tt.foil)
			if got != tt.want {
				t.Errorf("fabDoubleSidedFace(%q, %v) = %v, want %v", tt.number, tt.foil, got, tt.want)
			}
		})
	}
}
