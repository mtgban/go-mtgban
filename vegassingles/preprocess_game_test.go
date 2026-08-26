package vegassingles

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// The magic preprocessor asks the datastore what a display name's own reading
// of a product resolves to before preferring it, so its tests need the real
// one. The other games' preprocessors read the display name alone.
func TestMain(m *testing.M) {
	allprintingsPath := os.Getenv("ALLPRINTINGS5_PATH")
	if allprintingsPath == "" {
		log.Fatalln("Need ALLPRINTINGS5_PATH variable set to run tests")
	}

	reader, err := os.Open(allprintingsPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer reader.Close()

	ds, err := magic.Load(reader)
	if err != nil {
		log.Fatalln(err)
	}
	mtgmatcher.SetGlobalDatastore(ds)

	os.Exit(m.Run())
}

func TestPreprocessMagic(t *testing.T) {
	for _, tt := range []struct {
		display   string
		set       string
		number    int
		variation string
	}{
		// The storefront's own field is authoritative wherever it is set.
		{"Hallowed Fountain (RVR-280) - Ravnica Remastered", "rvr", 280, "280"},
		// It is null for every Secret Lair drop, so the drop's number comes
		// off the display name instead of the whole set aliasing onto one
		// card.
		{"Anguished Unmaking (1800) (Rainbow Foil) (SLD-1800) - Secret Lair Drop Series Foil", "sld", 0, "1800"},
		{"Brain Freeze (Halo Foil) (SLC-029) - Secret Lair Countdown Kit Foil", "slc", 0, "029"},
		{"Ancient Tomb (0136) (Borderless) (Galaxy Foil) (EOS-136) - Edge of Eternities: Stellar Sights Foil", "eos", 0, "136"},
		// Some codes are the storefront's own filing rather than a set's, and
		// the number counts within it. No set answers to it, so it is
		// withdrawn rather than sent.
		{"Sword of Forge and Frontier (Borderless) (UMP-002) - Unique and Miscellaneous Promos", "ump", 0, ""},
		{"Get Lost (UMP-001) - Unique and Miscellaneous Promos", "ump", 0, ""},
	} {
		card, err := preprocessMagic(VSProduct{
			DisplayName: tt.display,
			ProductData: VSProductData{
				Set:                       flexString(tt.set),
				CollectorNumberNormalized: tt.number,
			},
		})
		if err != nil {
			t.Fatalf("%s: %v", tt.display, err)
		}
		if card.Variation != tt.variation {
			t.Errorf("%s:\n got  %q\n want %q", tt.display, card.Variation, tt.variation)
		}
	}
}

func TestPreprocessRiftbound(t *testing.T) {
	for _, tt := range []struct {
		display   string
		finish    string
		name      string
		variation string
		foil      bool
	}{
		{"Noxus Hopeful (012/298) - Origins Foil", "Foil", "Noxus Hopeful", "012", true},
		// A tag rides behind the number rather than being dropped. For the
		// starred printings it only restates the star, and the same card
		// answers either way; for the promotional ones sharing a plain
		// number it is the only thing between two cards.
		{"Jinx - Loose Cannon (Signature) (301*/298) - Origins Foil", "Foil", "Jinx - Loose Cannon", "301* Signature", true},
		{"Pyke - Bloodharbor Ripper (Signature) (228*/219) - Unleashed Foil", "Foil", "Pyke - Bloodharbor Ripper", "228* Signature", true},
		{"Rengar - Trophy Hunter (Champion) (120/219) - Riftbound Organized Play Promotional Cards Foil", "Foil", "Rengar - Trophy Hunter", "120 Champion", true},
		{"Guardian Angel (Top 8) (051/221) - Riftbound Organized Play Promotional Cards Foil", "Foil", "Guardian Angel", "051 Top 8", true},
		{"Teemo - Swift Scout (Alternate Art) (263a/298) - Riftbound Promotional Cards Foil", "Foil", "Teemo - Swift Scout", "263a Alternate Art", true},
		// A name with no tag keeps the bare number.
		{"Ivern - Green Father (Overnumbered) (233/219) - Unleashed Foil", "Foil", "Ivern - Green Father", "233 Overnumbered", true},
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
