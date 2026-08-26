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
	} {
		bp := Blueprint{Version: tt.version}
		if got := gameVariation(GameFleshAndBlood, &bp, tt.number); got != tt.want {
			t.Errorf("gameVariation(FaB, %q, %q) = %q, want %q", tt.version, tt.number, got, tt.want)
		}
	}
}
