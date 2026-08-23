package vegassingles

import (
	"encoding/json"
	"testing"
)

func TestPreprocessRiftbound(t *testing.T) {
	for _, tt := range []struct {
		display   string
		finish    string
		name      string
		variation string
		foil      bool
	}{
		{"Noxus Hopeful (012/298) - Origins Foil", "Foil", "Noxus Hopeful", "012", true},
		{"Jinx - Loose Cannon (Signature) (301*/298) - Origins Foil", "Foil", "Jinx - Loose Cannon", "301*", true},
		{"Pyke - Bloodharbor Ripper (Signature) (228*/219) - Unleashed Foil", "Foil", "Pyke - Bloodharbor Ripper", "228*", true},
	} {
		card, err := preprocessRiftbound(VSProduct{
			DisplayName:    tt.display,
			SelectedFinish: tt.finish,
			ProductData:    VSProductData{SetName: "Origins"},
		})
		if err != nil {
			t.Fatalf("%s: %v", tt.display, err)
		}
		if card.Name != tt.name || card.Variation != tt.variation || card.Foil != tt.foil {
			t.Errorf("%s:\n got  %q %q foil=%v\n want %q %q foil=%v",
				tt.display, card.Name, card.Variation, card.Foil, tt.name, tt.variation, tt.foil)
		}
	}
}

func TestPreprocessOnePiece(t *testing.T) {
	for _, tt := range []struct {
		display   string
		name      string
		variation string
	}{
		// The wording before the code stays with the name: the matcher picks
		// the printing the storefront is describing from it.
		{"Hody Jones (020) (Alternate Art) (OP06-020) - Wings of the Captain Foil", "Hody Jones (020) (Alternate Art)", "OP06-020"},
		{"Monkey.D.Luffy (Gen Con 2023) (P-037) - One Piece Promotion Cards Foil", "Monkey.D.Luffy (Gen Con 2023)", "P-037"},
		{"Roronoa Zoro (OP11-016) - A Fist of Divine Speed Release Event Cards", "Roronoa Zoro", "OP11-016"},
	} {
		card, err := preprocessOnePiece(VSProduct{DisplayName: tt.display})
		if err != nil {
			t.Fatalf("%s: %v", tt.display, err)
		}
		if card.Name != tt.name || card.Variation != tt.variation {
			t.Errorf("%s:\n got  %q %q\n want %q %q", tt.display, card.Name, card.Variation, tt.name, tt.variation)
		}
	}

	// A product with no card code names nothing the matcher could find.
	_, err := preprocessOnePiece(VSProduct{DisplayName: "The Time of Battle - Release Event Pack"})
	if err == nil {
		t.Error("expected an error for a codeless display name")
	}
}

func TestPreprocessPokemon(t *testing.T) {
	for _, tt := range []struct {
		display   string
		finish    string
		name      string
		variation string
		want      string
	}{
		{"Mewtwo GX 039/73  - Holofoil Shining Legends - Ultra Rare", "Holofoil", "Mewtwo GX", "039/73", "Holofoil"},
		{"Vaporeon 006/017  POP Series 3 - Rare", "Normal", "Vaporeon", "006/017", ""},
		{"Zapdos 042  - Holofoil XY  Evolutions - Holo Rare", "Holofoil", "Zapdos", "042", "Holofoil"},
		{"Glaceon VSTAR SWSH197  - Holofoil Prize Pack Series Cards - Promo", "Holofoil", "Glaceon VSTAR", "SWSH197", "Holofoil"},
		{"Shellos West Sea (SDCC 2007) 107  Miscellaneous Cards & Products - Common", "Normal", "Shellos West Sea (SDCC 2007)", "107", ""},
	} {
		card, err := preprocessPokemon(VSProduct{
			DisplayName:    tt.display,
			SelectedFinish: tt.finish,
		})
		if err != nil {
			t.Fatalf("%s: %v", tt.display, err)
		}
		if card.Name != tt.name || card.Variation != tt.variation || card.Finish != tt.want {
			t.Errorf("%s:\n got  %q %q finish=%q\n want %q %q finish=%q",
				tt.display, card.Name, card.Variation, card.Finish, tt.name, tt.variation, tt.want)
		}
	}
}

// The set field arrives as a plain code for most lines and as a whole object
// for some Pokemon products; either way only a string survives decoding.
func TestFlexibleSetField(t *testing.T) {
	for _, tt := range []struct {
		raw  string
		want string
	}{
		{`{"set": "sm35", "setName": "Shining Legends"}`, "sm35"},
		{`{"set": {"id": "pop3", "name": "POP Series 3"}, "setName": "POP Series 3"}`, "pop3"},
		{`{"setName": "Miscellaneous"}`, ""},
	} {
		var pd VSProductData
		if err := json.Unmarshal([]byte(tt.raw), &pd); err != nil {
			t.Fatalf("%s: %v", tt.raw, err)
		}
		if string(pd.Set) != tt.want {
			t.Errorf("%s: got %q, want %q", tt.raw, pd.Set, tt.want)
		}
	}
}
