package mtgmatcher

import "testing"

func TestNormalizeFinish(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Cold Foil", "coldfoil"},
		{"cold-foil", "coldfoil"},
		{"RainbowPillars", "rainbowpillars"},
		{"FreeForm1", "freeform1"},
		{"", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NormalizeFinish(test.name); got != test.want {
				t.Errorf("NormalizeFinish(%q) = %q, want %q", test.name, got, test.want)
			}
		})
	}
}

// TestCanonicalFinish pins the shared vocabulary: the names every game
// answers for, and the silence that lets a game name the rest itself.
func TestCanonicalFinish(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Normal", FinishNonfoil},
		{"non-foil", FinishNonfoil},
		{"Foil", FinishFoil},
		{"Foil Etched", FinishEtched},
		{"etched", FinishEtched},
		{"Holofoil", ""},
		{"Cold Foil", ""},
		{"", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanonicalFinish(test.name); got != test.want {
				t.Errorf("CanonicalFinish(%q) = %q, want %q", test.name, got, test.want)
			}
		})
	}
}
