package cardtrader

import "testing"

// TestFabNumber pins the collector numbers Card Trader misspells: a set code
// with a letter wrong, and the blueprints whose number names another card of
// the set.
func TestFabNumber(t *testing.T) {
	for _, tt := range []struct {
		id     int
		number string
		want   string
	}{
		{273455, "HBY242", "HVY242"},
		{273371, "HVY243", "HVY243"},
		{215358, "WTR087", "WTR187"},
		{215189, "WTR130", "WTR030"},
		{158167, "CHN010", "CHN009"},
		{1, "CHN010", "CHN010"},
		{1, "", ""},
	} {
		bp := Blueprint{ID: tt.id}
		if got := fabNumber(&bp, tt.number); got != tt.want {
			t.Errorf("fabNumber(%d, %q) = %q, want %q", tt.id, tt.number, got, tt.want)
		}
	}
	bp := Blueprint{ID: 273455}
	if got := gameVariation(GameFleshAndBlood, &bp, "HBY242"); got != "HVY242" {
		t.Errorf("gameVariation(HBY242) = %q, want HVY242", got)
	}
}

// TestFabPuzzle pins how a puzzle piece blueprint is read: the quoted face,
// spelled the datastore's way where the two differ, and the piece the
// version names, spelled so it describes either label the datastore uses
// for the middle piece.
func TestFabPuzzle(t *testing.T) {
	for _, tt := range []struct{ name, want string }{
		{`"Uzuri, Switchblade" Macro Puzzle Card`, "Uzuri, Switchblade"},
		{`"Chart the High Seas" Macro Puzzle Card`, "High Seas Map"},
		{"Uzuri, Switchblade", ""},
		{`"Uzuri, Switchblade" Art Card`, ""},
	} {
		if got := fabPuzzleFace(tt.name); got != tt.want {
			t.Errorf("fabPuzzleFace(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
	for _, tt := range []struct{ version, want string }{
		{"Center", "Middle Center"},
		{"Top Left", "Top Left"},
		{"Bottom Center", "Bottom Center"},
	} {
		if got := fabPuzzlePiece(tt.version); got != tt.want {
			t.Errorf("fabPuzzlePiece(%q) = %q, want %q", tt.version, got, tt.want)
		}
	}
	bp := Blueprint{Name: `"Treasure Island" Macro Puzzle Card`, Version: "Center"}
	if got := gameVariation(GameFleshAndBlood, &bp, "SEA-PLZ014"); got != "SEA-PLZ014 Middle Center" {
		t.Errorf("gameVariation(puzzle) = %q, want %q", got, "SEA-PLZ014 Middle Center")
	}
}

func TestFabNames(t *testing.T) {
	bp := Blueprint{Name: "Kassai of the Golden Sands"}
	if got := gameName(GameFleshAndBlood, &bp); got != "Kassai of the Golden Sand" {
		t.Errorf("gameName = %q, want the datastore's spelling", got)
	}
}
