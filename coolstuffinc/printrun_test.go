package coolstuffinc

import "testing"

// TestPrintRunEdition pins the copyright date onto the set that printed it.
// The two runs of these editions are reissued under one another's numbers
// and one storefront name, so the date in the note is the only thing that
// says which of the two a listing is, and a listing that says nothing must
// keep the edition it arrived with.
func TestPrintRunEdition(t *testing.T) {
	for _, tt := range []struct {
		desc, edition, notes, want string
	}{
		{"the later date names the reprint",
			"Metal Raiders", "2020 Copyright Date", "Metal Raiders (25th Anniversary Edition)"},
		{"the earlier date names the original",
			"Metal Raiders", "Copyright Date 1996", "Metal Raiders"},
		{"the date is read wherever the note puts it",
			"Invasion of Chaos", "1996 Copyright Date", "Invasion of Chaos"},
		{"a tier written in front of the date does not hide it",
			"Light of Destruction", "Ultra Rare - 2020 Copyright Date", "Light of Destruction (2020 Date Reprint)"},
		{"the catalog's The is restored on the way to the original",
			"Legend of Blue Eyes White Dragon", "Copyright Date 1996", "The Legend of Blue Eyes White Dragon"},
		{"and dropped again on the way to the reprint",
			"Legend of Blue Eyes White Dragon", "2020 Copyright Date", "Legend of Blue Eyes White Dragon (25th Anniversary Edition)"},
		{"the pack the storefront numbers is the one the catalog does not",
			"Retro Pack 1", "1996 Copyright Date", "Retro Pack"},
		{"a note that names no date leaves the edition alone",
			"Metal Raiders", "", "Metal Raiders"},
		{"so does a note about something else entirely",
			"Metal Raiders", "Reprints LOB-001", "Metal Raiders"},
		{"an edition sold in one run is never rewritten",
			"Rarity Collection 5", "2020 Copyright Date", "Rarity Collection 5"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := printRunEdition(tt.edition, tt.notes); got != tt.want {
				t.Errorf("printRunEdition(%q, %q) = %q, want %q", tt.edition, tt.notes, got, tt.want)
			}
		})
	}
}
