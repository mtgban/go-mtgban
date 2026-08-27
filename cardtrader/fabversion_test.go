package cardtrader

import "testing"

// TestFabWording pins which half of a Flesh and Blood blueprint's version
// reaches the matcher. The treatment half is already spoken for - the listing
// states it as its own finish - and repeating it narrows nothing, while the
// wording beside it is the only thing telling two same-numbered printings
// apart.
func TestFabWording(t *testing.T) {
	for _, tt := range []struct {
		desc, version, number, want string
	}{
		{"the treatment is dropped and the wording kept",
			"Extended Art | Rainbow Foil", "ROS167ea", "Extended Art"},
		{"a version naming only a treatment says nothing",
			"Rainbow Foil", "CRU003", ""},
		{"nor does the other treatment",
			"Cold Foil", "MST068", ""},
		{"a version restating the number would only stutter",
			"DYN115", "DYN115", ""},
		{"and the wording beside that restatement survives",
			"DYN116 | Reverse", "DYN116", "Reverse"},
		{"the print run rides through",
			"Lost Treasure | Cold Foil", "SEA150-TP", "Lost Treasure"},
		{"a treatment the finish has no word for is kept",
			"Golden Cold Foil", "EVR155", "Golden Cold Foil"},
		{"a bare wording is kept whole",
			"Marvel", "HER116", "Marvel"},
		{"an empty version stays empty", "", "WTR001", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := fabWording(tt.version, tt.number); got != tt.want {
				t.Errorf("fabWording(%q, %q) = %q, want %q", tt.version, tt.number, got, tt.want)
			}
		})
	}
}

// TestGameVariationFab pins what the card is finally asked for.
func TestGameVariationFab(t *testing.T) {
	for _, tt := range []struct {
		version, number, want string
	}{
		{"Extended Art | Rainbow Foil", "ROS167ea", "ROS167ea Extended Art"},
		{"Rainbow Foil", "CRU003", "CRU003"},
		{"DYN115", "DYN115", "DYN115"},
		{"", "WTR001", "WTR001"},
		// The Classic Constructed marker comes off the number and is said
		// the way the catalog says it, which is where the twin printings
		// filed at that same number differ.
		{"CC Label", "APS011cc", "APS011 CC Tag"},
		{"CC Label", "AGB008cc", "AGB008 CC Tag"},
		{"", "AGB008", "AGB008"},
		// A number that merely ends in those letters keeps them: only a
		// marker sits behind a digit.
		{"Rainbow Foil", "MSTcc", "MSTcc"},
	} {
		bp := Blueprint{Version: tt.version}
		if got := gameVariation(GameFleshAndBlood, &bp, tt.number); got != tt.want {
			t.Errorf("gameVariation(FaB, %q, %q) = %q, want %q", tt.version, tt.number, got, tt.want)
		}
	}
}

// TestFabPlainNumber pins what the "cc" a CardTrader collector number can end
// with means. The catalog files the Classic Constructed printing at the plain
// number and separates it from its twin by a tag, so the marker has to leave
// the number to reach anything at all.
func TestFabPlainNumber(t *testing.T) {
	for _, tt := range []struct {
		desc, in, want string
		wantCC         bool
	}{
		{"the marker comes off a numbered card", "APS011cc", "APS011", true},
		{"and off the deck that has a tagged twin", "AGB008cc", "AGB008", true},
		{"a number without it is untouched", "APS011", "APS011", false},
		{"letters that are not a marker stay", "MSTcc", "MSTcc", false},
		{"so does a number that is only the marker", "cc", "cc", false},
		{"an empty number stays empty", "", "", false},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, gotCC := fabPlainNumber(tt.in)
			if got != tt.want || gotCC != tt.wantCC {
				t.Errorf("fabPlainNumber(%q) = %q, %v, want %q, %v", tt.in, got, gotCC, tt.want, tt.wantCC)
			}
		})
	}
}
