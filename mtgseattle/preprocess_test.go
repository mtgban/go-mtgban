package mtgseattle

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/magic"
)

var (
	datastoreOnce    sync.Once
	datastoreErr     error
	datastoreBackend *matcher.Backend
)

// realDatastore reads the Magic datastore the first time a test asks for it,
// and skips where the run carries none.
func realDatastore(t *testing.T) *matcher.Backend {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		backend, err := datastore.Read("magic", path)
		if err != nil {
			datastoreErr = err
			return
		}
		datastoreBackend = backend
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if datastoreBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return datastoreBackend
}

// TestMysteryBoosterCardsResolvesToPLSTNumber pins a Mystery Booster Cards
// listing to the PLST printing the MB1 retail booster bundles for that name
// in the listing's finish.
func TestMysteryBoosterCardsResolvesToPLSTNumber(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		cardName string
		variant  string
		number   string
	}{
		{"Gonti, Lord of Luxury", "", "KLD-84"},
		{"Krosan Verge", "", "C18-263"},
		{"Storm Crow", "", "POR-69"},
		{"Storm Crow", "Foil", "9ED-100"},
	}
	for _, test := range tests {
		theCard, err := preprocess(b, test.cardName, "Mystery Booster Cards", test.variant)
		if err != nil {
			t.Fatalf("%s: preprocess: %v", test.cardName, err)
		}

		cardID, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("%s: Match: %v", test.cardName, err)
		}
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatalf("%s: GetUUID: %v", test.cardName, err)
		}
		if co.SetCode != "PLST" || co.Number != test.number {
			t.Errorf("%s %s: matched %s %s, want PLST %s", test.cardName, test.variant, co.SetCode, co.Number, test.number)
		}
	}

	// The booster names only a nonfoil printing, so a foil listing keeps
	// its own wording.
	theCard, err := preprocess(b, "Skullclamp", "Mystery Booster Cards", "Foil")
	if err != nil {
		t.Fatalf("Skullclamp: preprocess: %v", err)
	}
	if theCard.Edition == "PLST" {
		t.Errorf("foil Skullclamp sent to PLST %s", theCard.Variation)
	}
}

// TestShelfWordingResolvesToItsPrinting pins listings whose wording is read
// as an art index, a release event, an emblem's planeswalker or a token's
// letter, not a collector number, and the edition names MTG Seattle gives
// each language of a foreign-only set.
func TestShelfWordingResolvesToItsPrinting(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		cardName string
		edition  string
		variant  string
		set      string
		number   string
	}{
		{"Abbey Matron", "Homelands", "1", "HML", "2a"},
		{"Abbey Matron", "Homelands", "2", "HML", "2b"},
		{"Aliban's Tower", "Homelands", "2", "HML", "61b"},
		{"Azorius Guildmage", "Pre-Release Promos", "Foil Launch Promo", "PDIS", "141★"},
		{"Blight Herder", "Pre-Release Promos", "Foil Launch Promo", "PBFZ", "2"},
		{"Emblem", "Modern Masters", "Elspeth, Knight-Errant", "TMMA", "16"},
		{"Emblem", "Commander Anthology Vol. II", "Daretti", "TCM2", "18"},
		{"Wurm Token (B)", "Scars of Mirrodin", "", "TSOM", "8"},
		{"Beast Token A", "Duel Decks: Anthology", "", "TGVL", "8"},
		{"Beast Token B", "Duel Decks: Garruk vs Liliana", "", "TDDD", "T2"},
		{"Badlands", "Italian Revised FBB", "", "FBB", "282"},
		{"Badlands", "French Revised FBB", "", "FBBFR", "282"},
		{"Badlands", "German Revised FBB", "", "FBBDE", "282"},
		{"Angry Mob", "German Renaissance", "", "RENDE", "4"},
		{"Plateau", "German Revised FWB", "", "FWBDE", "284"},
	}
	for _, test := range tests {
		theCard, err := preprocess(b, test.cardName, test.edition, test.variant)
		if err != nil {
			t.Fatalf("%s: preprocess: %v", test.cardName, err)
		}

		cardID, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("%s: Match: %v", test.cardName, err)
		}
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatalf("%s: GetUUID: %v", test.cardName, err)
		}
		if co.SetCode != test.set || co.Number != test.number {
			t.Errorf("%s %s: matched %s %s, want %s %s", test.cardName, test.variant, co.SetCode, co.Number, test.set, test.number)
		}
	}

	// An index past the card's arts is left as the shelf wrote it.
	theCard, err := preprocess(b, "Abbey Matron", "Homelands", "3")
	if err != nil {
		t.Fatalf("Abbey Matron (3): preprocess: %v", err)
	}
	if theCard.Variation != "3" {
		t.Errorf("Abbey Matron (3): variant rewritten to %q", theCard.Variation)
	}
}

// TestTokenPair pins a two-sided token to the pairing the datastore derives
// for it, never to one of its faces, and refuses one no face anchors.
func TestTokenPair(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		cardName string
		edition  string
		variant  string
		set      string
		number   string
	}{
		{"Dragon Token (005) // Goblin Token (006)", "Ravnica Allegiance Guild Kits", "", "TGK2", "5//6"},
		{"Bird Token // Cat Token (020)", "Core Set 2021", "Foil", "TM21", "2//20"},
		{"Champion of Wits Token // Insect Token", "Hour of Devastation", "", "THOU", "2//12"},
	}
	for _, test := range tests {
		theCard, err := preprocess(b, test.cardName, test.edition, test.variant)
		if err != nil {
			t.Fatalf("%s: preprocess: %v", test.cardName, err)
		}

		cardID, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("%s: Match: %v", test.cardName, err)
		}
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatalf("%s: GetUUID: %v", test.cardName, err)
		}
		foil := test.variant == "Foil"
		if co.SetCode != test.set || co.Number != test.number || co.Foil != foil || co.Identifiers["derivedTokenPair"] != "true" {
			t.Errorf("%s: matched %s %s foil=%t, want the %s %s pairing", test.cardName, co.SetCode, co.Number, co.Foil, test.set, test.number)
		}
	}

	// A foil listing of a native card printed only nonfoil, and a graded
	// sheet, are refused like any other such listing.
	for _, test := range [][3]string{
		{"Copy Token (001) // Horror Token (002)", "Guilds of Ravnica Guild Kit", "Foil"},
		{"Dragon Token (005) // Goblin Token (006)", "Ravnica Allegiance Guild Kits", "BGS 9.5"},
	} {
		theCard, err := preprocess(b, test[0], test[1], test[2])
		if err == nil {
			t.Errorf("%s %s: preprocess = %v, want refused", test[0], test[2], theCard)
		}
	}

	// The emblem's planeswalker sits in the variant, so the title names no
	// second face to pair.
	_, err := preprocess(b, "Monk Token // Emblem", "Double Masters 2022", "Wrenn and Six Foil")
	if !errors.Is(err, matcher.ErrUnsupported) {
		t.Errorf("Monk Token // Emblem: preprocess error %v, want unsupported", err)
	}
}
