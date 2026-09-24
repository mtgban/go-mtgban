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
// extends MON002 with another digit.
func fabFixtureBackend() *mtgmatcher.Backend {
	numbers := []string{
		"MON002//MON001",
		"MON088//MON002",
		"MON002//MON221",
		"MON0021",
		"WTR040//WTR078",
		"MST158-A",
		"MST003//MST158",
		"MST002//MST158",
	}
	b := &mtgmatcher.Backend{UUIDs: map[string]*mtgmatcher.CardObject{}}
	for i, number := range numbers {
		uuid := "fixture" + string(rune('0'+i))
		b.UUIDs[uuid] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{Number: number}}
		b.AllUUIDs = append(b.AllUUIDs, uuid)
	}
	return b
}

func TestFabDoubleSidedFace(t *testing.T) {
	b := fabFixtureBackend()

	tests := []struct {
		desc   string
		number string
		want   bool
	}{
		{"a face shared by three pairings and never standalone", "MON002", true},
		{"a face carried by only one pairing", "WTR078", false},
		{"a face with a printing of its own beside its pairings", "MST158", false},
		{"a number that names no pairing at all", "SEA999", false},
		{"a composite pair number is never a lone face", "MON002//MON001", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := fabDoubleSidedFace(b, tt.number); got != tt.want {
				t.Errorf("fabDoubleSidedFace(%q) = %v, want %v", tt.number, got, tt.want)
			}
		})
	}
}
