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
		setName   string
		number    int
		edition   string
		variation string
	}{
		// The code names the set the prose does, so nothing is second-guessed.
		{"Hallowed Fountain (RVR-280) - Ravnica Remastered", "rvr", "Ravnica Remastered", 280, "rvr", "280"},
		// The number field is null for every Secret Lair drop, so the drop's
		// number comes off the display name instead of the whole set aliasing
		// onto one card.
		{"Anguished Unmaking (1800) (Rainbow Foil) (SLD-1800) - Secret Lair Drop Series Foil", "sld", "Secret Lair Drop Series", 0, "Secret Lair Drop Series", "1800"},
		{"Brain Freeze (Halo Foil) (SLC-029) - Secret Lair Countdown Kit Foil", "slc", "Secret Lair Countdown Kit", 0, "Secret Lair Countdown Kit", "029"},
		{"Ancient Tomb (0136) (Borderless) (Galaxy Foil) (EOS-136) - Edge of Eternities: Stellar Sights Foil", "eos", "Edge of Eternities: Stellar Sights", 0, "eos", "136"},
		// Some codes are the storefront's own filing rather than a set's, and
		// the number counts within it. No set answers to it, so it is
		// withdrawn rather than sent.
		{"Sword of Forge and Frontier (Borderless) (UMP-002) - Unique and Miscellaneous Promos", "ump", "Unique and Miscellaneous Promos", 0, "Unique and Miscellaneous Promos", ""},
		// The code names the parent set of a promo printing, the prose names
		// the promo set itself, and the number needs the promo way of saying
		// it.
		{"Archangel of Tithes (PPOTJ-002) - Outlaws of Thunder Junction Promos", "ppotj", "Outlaws of Thunder Junction Promos", 2, "Outlaws of Thunder Junction Promos", "2p"},
		{"Adeline, Resplendent Cathar (PPMID-001) - Innistrad: Midnight Hunt Promos Foil", "ppmid", "Innistrad: Midnight Hunt Promos", 1, "Innistrad: Midnight Hunt Promos", "1p"},
		{"Archmage's Charm (MH1-007) - Modern Horizons 1 Timeshifts Foil", "mh1", "Modern Horizons 1 Timeshifts", 7, "Modern Horizons 1 Timeshifts", "7"},
		{"Tannuk, Steadfast Second (PRE-162) - Prerelease Cards Foil", "pre", "Prerelease Cards", 162, "Prerelease Cards", "162"},
		// A heading no set answers to keeps the code's reading rather than
		// losing the row.
		{"Vexing Arcanix (8th Edition) (OVER-319) - Oversize Cards", "over", "Oversize Cards", 319, "over", "319"},
		// The prose is a heading that files a promo under a set it does not
		// belong to, and the printing it reaches is not the one the listing
		// numbers. The display name spells the right set out behind its last
		// dash, qualifiers and all.
		{"Snapcaster Mage (PTP-002) - Regional Championship Qualifiers 2023 (Borderless) Foil", "ptp", "Pro Tour Promos", 2, "Regional Championship Qualifiers 2023", "2"},
		{"Slith Firewalker (JSS-001) - Junior Super Series", "jss", "Junior Series Promos", 10, "Junior Super Series", "10"},
		{"The One Ring (UMP-451) - The Lord of the Rings: Tales of Middle-earth (Borderless) Foil", "ump", "Unique and Miscellaneous Promos", 451, "The Lord of the Rings: Tales of Middle-earth", "451"},
		// The heading names no set at all, so the code's reading is all master
		// had, and it aliases across the two sets numbering this card alike.
		{"Interplanar Tunnel (OVER-002) - Planechase 2012 Planes", "over", "Oversize Cards", 2, "Planechase 2012 Planes", "2"},
		{"Vampiric Tutor (JDG-002) - Judge Gift Cards 2018 Foil", "jdg", "Judge Promos", 2, "Judge Gift Cards 2018", "2"},
		// The display name names no set of its own either, and the heading
		// reaches a prerelease printing filed at a number the listing never
		// says. The code's reading is the one that answers to the 2.
		{"Spectacular Spider-Man (Borderless) (MEDIA-002) - Media Promos Foil", "media", "Media Promos", 0, "media", "002"},
	} {
		card, err := preprocessMagic(VSProduct{
			DisplayName: tt.display,
			ProductData: VSProductData{
				Set:                       flexString(tt.set),
				SetName:                   tt.setName,
				CollectorNumberNormalized: tt.number,
			},
		})
		if err != nil {
			t.Fatalf("%s: %v", tt.display, err)
		}
		if card.Edition != tt.edition || card.Variation != tt.variation {
			t.Errorf("%s:\n got  %q %q\n want %q %q",
				tt.display, card.Edition, card.Variation, tt.edition, tt.variation)
		}
	}
}

// The storefront states a magic finish twice and its selectedFinish field is
// stale in both directions, so the display name's own last word is what the
// finish is read from. The storefront lists the two printings of Aether
// Channeler as two products that differ only by that word, both of them
// saying selectedFinish=foil.
func TestPreprocessMagicFinish(t *testing.T) {
	for _, tt := range []struct {
		display string
		finish  string
		foil    bool
	}{
		{"Acererak the Archlich (SLD-1784) - Secret Lair Drop (Borderless) Foil", "nonfoil", true},
		{"Acererak the Archlich (SLD-1784) - Secret Lair Drop (Borderless)", "nonfoil", false},
		{"Aether Channeler (GAME-011) - Store Championships Foil", "foil", true},
		{"Aether Channeler (GAME-011) - Store Championships", "foil", false},
	} {
		card, err := preprocessMagic(VSProduct{
			DisplayName:    tt.display,
			SelectedFinish: tt.finish,
		})
		if err != nil {
			t.Fatalf("%s: %v", tt.display, err)
		}
		if card.Foil != tt.foil {
			t.Errorf("%s: got foil=%v, want %v", tt.display, card.Foil, tt.foil)
		}
	}
}

// Etched is said only in the display name, in two spellings, and it rides in
// the variation because that is the only place the matcher reads it from.
func TestPreprocessMagicEtched(t *testing.T) {
	for _, tt := range []struct {
		display   string
		set       string
		setName   string
		number    int
		variation string
	}{
		{"Arid Mesa (MH2-436) - Modern Horizons 2 Etched Foil", "mh2", "Modern Horizons 2", 436, "436 Etched"},
		{"Azorius Signet (Foil Etched) (SLD-286) - Secret Lair Drop Series Foil", "sld", "Secret Lair Drop Series", 0, "286 Etched"},
		{"Talisman of Hierarchy (MH1-036) - Modern Horizons 1 Timeshifts Etched Foil", "mh1", "Modern Horizons 1 Timeshifts", 36, "36 Etched"},
		// Three Timeshifts listings name the parent set rather than the
		// Timeshifts one, and land where no etched printing stands. Asking
		// for one there answers with the nonfoil printing, further from the
		// listing than the foil one it has today, so the word is dropped.
		{"Force of Negation (MH1-009) - Modern Horizons 1 Timeshifts Etched Foil", "mh1", "Modern Horizons", 9, "9"},
	} {
		card, err := preprocessMagic(VSProduct{
			DisplayName: tt.display,
			ProductData: VSProductData{
				Set:                       flexString(tt.set),
				SetName:                   tt.setName,
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
		// The runes state a number and no set size at all, and the signature
		// promos lead theirs with letters. Both are printings the datastore
		// holds, and reading neither is what kept them off the shelf.
		{"Chaos Rune (R05) - Vendetta Foil", "Foil", "Chaos Rune", "R05", true},
		{"Body Rune (R04a) (R04a) - Vendetta Foil", "Foil", "Body Rune", "R04a", true},
		{"Calm Rune (Alternate Art) (R02a) - Spiritforged Foil", "Foil", "Calm Rune", "R02a Alternate Art", true},
		{"Order Rune (R06c) (R06c) - Riftbound Organized Play Promotional Cards", "Normal", "Order Rune", "R06c", false},
		{"Ahri - Inquisitive (SP3/006) - Vendetta Foil", "Foil", "Ahri - Inquisitive", "SP3", true},
		// A trailing parenthetical that only carries digits is not the
		// number: the first group in the name is, and it is read first.
		{"Teemo - Swift Scout (Alternate Art) (263a/298) - Riftbound Promotional Cards Foil (Unique) (699149)", "Foil", "Teemo - Swift Scout", "263a Alternate Art", true},
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

	// A name whose only digits are the unique copy's own id states no
	// collector number, and answering one anyway would price a card the
	// listing never named.
	for _, display := range []string{
		"Teemo - Swift Scout (Alternate Art) - Riftbound Promotional Cards Foil (Unique) (699149)",
		"Ahri - Inquisitive - Vendetta Foil",
	} {
		card, err := preprocessRiftbound(VSProduct{
			DisplayName:    display,
			SelectedFinish: "Foil",
			ProductData:    VSProductData{SetName: "Origins"},
		})
		if err == nil {
			t.Errorf("%s: read a number and answered %q, want no number", display, card.Variation)
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
		// A subset a set prints alongside itself letters its denominator too.
		// The number used to be unreadable there, and the search walked past
		// it into the edition half and came back with the era code.
		{"Altaria TG11/TG30  - Holofoil SWSH12 Silver Tempest Trainer Gallery - Ultra Rare", "Holofoil", "Altaria", "TG11/TG30", "Holofoil"},
		{"Aether Foundation Employee SV81/SV94  - Holofoil Hidden Fates Shiny Vault - Shiny Holo Rare", "Holofoil", "Aether Foundation Employee", "SV81/SV94", "Holofoil"},
		{"Absol GG16/GG70  - Holofoil Crown Zenith Galarian Gallery - Ultra Rare", "Holofoil", "Absol", "GG16/GG70", "Holofoil"},
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

// TestPreprocessMagicSpelledNumber pins the numbers the storefront's int
// field cannot hold. It is an int: the star on a War of the Spark alt-art
// and the letter behind an Unstable variant both fall off it, and the two
// printings then answer to one number.
func TestPreprocessMagicSpelledNumber(t *testing.T) {
	for _, tt := range []struct {
		display   string
		number    int
		setName   string
		variation string
	}{
		{"Ajani, the Greathearted (WAR-184★) - War of the Spark Foil", 184, "War of the Spark", "184★"},
		{"Ugin, the Ineffable (WAR-02★) - War of the Spark Foil", 2, "War of the Spark", "02★"},
		{"Brothers Yamazaki (CHK-160B) - Champions of Kamigawa Foil", 160, "Champions of Kamigawa", "160B"},
		{"Everythingamajig (UST-147C) - Unstable Foil", 147, "Unstable", "147C"},
		{"Ghalta, Primal Hunger (PPM20-130P) - Rivals of Ixalan Promos Foil", 130, "Rivals of Ixalan Promos", "130P"},
		// A plain number is left exactly as the field states it.
		{"Hallowed Fountain (RVR-280) - Ravnica Remastered", 280, "Ravnica Remastered", "280"},
		// Padding is not more: the tidier spelling stands.
		{"Archmage's Charm (MH1-007) - Modern Horizons 1 Timeshifts Foil", 7, "Modern Horizons 1 Timeshifts", "7"},
	} {
		product := VSProduct{DisplayName: tt.display, SelectedFinish: "Foil"}
		product.ProductData.SetName = tt.setName
		product.ProductData.CollectorNumberNormalized = tt.number
		card, err := preprocessMagic(product)
		if err != nil {
			t.Errorf("%s: %v", tt.display, err)
			continue
		}
		if card.Variation != tt.variation {
			t.Errorf("%s:\n got  %q\n want %q", tt.display, card.Variation, tt.variation)
		}
	}
}

// TestPreprocessMagicWording pins the name a display name really states and
// the wording standing beside it. Cutting at the first bracket loses both.
func TestPreprocessMagicWording(t *testing.T) {
	for _, tt := range []struct {
		display   string
		setName   string
		number    int
		name      string
		variation string
	}{
		// A name carrying a parenthesis of its own survives, which is what
		// SplitVariants is for - cutting at the first bracket asked for
		// "Dwight, Assistant". The reskin behind the dash is read further
		// down, where it reaches Baral.
		{"Dwight, Assistant (to the) King - Baral, Chief of Compliance (SLD-2168) - Secret Lair Drop Series",
			"Secret Lair Drop Series", 2168, "Dwight, Assistant (to the) King", "2168"},
		// The wording rescues a listing nothing else answered: the bin's
		// own code names no set and its number no printing, and the year
		// is the whole of what says which promo this is.
		{"Forest (Year of the Snake 2025) (SSP-006) - Standard Showdown Promos Foil",
			"Standard Showdown Promos", 0, "Forest", "Year of the Snake 2025"},
		// A wording naming a foil is left behind: the finish is the tail's
		// to state, and this listing's tail does not say it.
		{"Endless Sands (0060) (Borderless) (Galaxy Foil) (EOS-060) - Edge of Eternities: Stellar Sights",
			"Edge of Eternities: Stellar Sights", 60, "Endless Sands", "60"},
		// An ordinary listing is untouched.
		{"Hallowed Fountain (RVR-280) - Ravnica Remastered",
			"Ravnica Remastered", 280, "Hallowed Fountain", "280"},
	} {
		product := VSProduct{DisplayName: tt.display}
		product.ProductData.SetName = tt.setName
		product.ProductData.CollectorNumberNormalized = tt.number
		card, err := preprocessMagic(product)
		if err != nil {
			t.Errorf("%s: %v", tt.display, err)
			continue
		}
		if card.Name != tt.name || card.Variation != tt.variation {
			t.Errorf("%s:\n got  %q %q\n want %q %q", tt.display, card.Name, card.Variation, tt.name, tt.variation)
		}
	}
}
