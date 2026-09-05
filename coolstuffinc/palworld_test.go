package coolstuffinc

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/palworld"
)

// TestPalworldNotes pins which tails come off a collector number and which
// stay on it. The six the catalog files a printing under have to survive:
// pulling one off leaves the note naming the base card, which is a wrong
// answer where leaving it on is at worst a miss.
func TestPalworldNotes(t *testing.T) {
	for _, tt := range []struct {
		desc, in, want string
	}{
		{"the rarity glued to a number is written back beside it",
			"EBP01-025RR", "EBP01-025 RR"},
		{"and a single-letter one too",
			"EBP01-027R", "EBP01-027 R"},
		{"a number the storefront already wrote apart is left alone",
			"EBP01-046 RR", "EBP01-046 RR"},
		{"a tail the catalog numbers a printing under stays on the number",
			"EBP01-025SSP", "EBP01-025SSP"},
		{"including the trial decks'",
			"ETD01-001TSR", "ETD01-001TSR"},
		{"a tail nobody has heard of stays on it as well",
			"EBP01-025ZZZ", "EBP01-025ZZZ"},
		{"the rest of the wording is kept",
			"ESOUL-002SSS (Gold)", "ESOUL-002 SSS (Gold)"},
		{"a number carrying no tail passes through", "EPR-004", "EPR-004"},
		{"so does a note saying nothing", "", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := palworldNotes(tt.in); got != tt.want {
				t.Errorf("palworldNotes(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestPalworldListing replays the sell listings this storefront refuses to
// be read as written, each checked as typed too: a row that stopped needing
// the correction is a row that should leave the table rather than sit there
// rewriting a listing that now means something.
func TestPalworldListing(t *testing.T) {
	path := os.Getenv("PALWORLD_PATH")
	if path == "" {
		t.Skip("Need PALWORLD_PATH variable set to run this test")
	}
	f, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := mtgmatcher.Open("palworld", f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name, edition, notes, uuid string
	}{
		{"Chillet - Dragon Whisperer (RR)", "Dawn of Palpagos", "EBP01-025RR", "ebp01-025_713917_foil"},
		{"Jormuntide - Surging Sea Serpent (R)", "Dawn of Palpagos", "EBP01-027R", "ebp01-027_713923_foil"},
		{"Fuak - Manic Wave Ripper (PR Card Pack Vol. 1)", "Promo", "EPR-004 PR ", "epr-004_714396_foil"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			asTyped := mtgmatcher.InputCard{Name: tt.name, Edition: tt.edition, Variation: tt.notes}
			id, err := b.Match(&asTyped)
			if err == nil {
				t.Errorf("Match(%q, %q) = %q, want the listing to reach nothing before it is read",
					tt.name, tt.notes, id)
			}
			card := mtgmatcher.InputCard{Name: palworldName(tt.name), Edition: tt.edition, Variation: palworldNotes(tt.notes)}
			id, err = b.Match(&card)
			if err != nil {
				t.Fatalf("Match(%q, %q) = %v", card.Name, card.Variation, err)
			}
			if id != tt.uuid {
				t.Errorf("Match(%q, %q) = %q, want %q", card.Name, card.Variation, id, tt.uuid)
			}
		})
	}
}
