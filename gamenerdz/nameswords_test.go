package gamenerdz

import "testing"

// TestContainsWords pins that a name is found in a display name only as whole
// words, read the way Normalize reads both.
func TestContainsWords(t *testing.T) {
	for _, tt := range []struct {
		outer, inner string
		want         bool
	}{
		{"Torgal, Clive's Companion - Yoshimaru, Ever Faithful (MFP-001) - MagicFest Cards Foil", "Yoshimaru, Ever Faithful", true},
		{"Delver of Secrets // Insectile Aberration", "Delver of Secrets", true},
		{"Lim-Dul's Vault (ALL-130) - Alliances", "Lim-Dûl's Vault", true},
		{"Fire & Ice (APC-128) - Apocalypse", "Fire and Ice", true},
		{"Fog Bank (BBD-104) - Battlebond", "Fog", true},
		{"Shockmaw Dragon (M19-157) - Core Set 2019", "Shock", false},
		{"Waste Land", "Wasteland", false},
		{"Wasteland (TMP-330) - Tempest", "Waste Land", false},
		{"Delver of Secrets", "", false},
	} {
		got := containsWords(tt.outer, tt.inner)
		if got != tt.want {
			t.Errorf("containsWords(%q, %q) = %v, want %v", tt.outer, tt.inner, got, tt.want)
		}
	}
}
