package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestCanonicalFinish pins the one finish Magic names itself. The shared
// vocabulary answers "" for it, so a game that does not sell an etched foil
// cannot be handed one by a vendor spelling it.
func TestCanonicalFinish(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Foil Etched", mtgmatcher.FinishEtched},
		{"etched foil", mtgmatcher.FinishEtched},
		{"ETCHED", mtgmatcher.FinishEtched},
		{"Foil", mtgmatcher.FinishFoil},
		{"Normal", mtgmatcher.FinishNonfoil},
		{"Holofoil", ""},
		{"", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := (Rules{}).CanonicalFinish(test.name)
			if got != test.want {
				t.Errorf("CanonicalFinish(%q) = %q, want %q", test.name, got, test.want)
			}
			if test.want == mtgmatcher.FinishEtched && mtgmatcher.CanonicalFinish(test.name) != "" {
				t.Errorf("the shared vocabulary still places %q", test.name)
			}
		})
	}
}
