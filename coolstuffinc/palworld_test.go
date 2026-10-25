package coolstuffinc

import (
	"testing"

	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/palworld"
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
	b := readGameDatastore(t, "palworld", "PALWORLD_PATH")

	for _, tt := range []struct {
		name, edition, notes, uuid string
	}{
		{"Chillet - Dragon Whisperer (RR)", "Dawn of Palpagos", "EBP01-025RR", "ebp01-025_713917_foil"},
		{"Jormuntide - Surging Sea Serpent (R)", "Dawn of Palpagos", "EBP01-027R", "ebp01-027_713923_foil"},
		{"Fuak - Manic Wave Ripper (PR Card Pack Vol. 1)", "Promo", "EPR-004 PR ", "epr-004_714396_foil"},
		{"Cattiva - Prototype", "Souls & Misc.", "Prototype", "716343"},
		{"Soul - Prototype", "Souls & Misc.", "Prototype", "716360"},
		{"Soul - Chillet (Demo Caravan Promo)", "Promo", "ESOUL-000 PR Long Blue Creature in Pool Ring", "esoul-003_714403_foil"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			asTyped := matcher.InputCard{Name: tt.name, Edition: tt.edition, Variation: tt.notes}
			id, err := b.Match(&asTyped)
			if err == nil {
				t.Errorf("Match(%q, %q) = %q, want the listing to reach nothing before it is read",
					tt.name, tt.notes, id)
			}
			card := matcher.InputCard{Name: palworldName(tt.name), Edition: tt.edition, Variation: palworldNotes(tt.notes)}
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

// TestPalworldBuylistNote pins that a buy row leaving the number empty and
// writing it in the note reaches the printing the note names.
func TestPalworldBuylistNote(t *testing.T) {
	b := readGameDatastore(t, "palworld", "PALWORLD_PATH")

	for _, tt := range []struct {
		notes, uuid string
	}{
		{"ESS01-002 R", "ess01-002_723233"},
		{"ESS01-002OSR Full Art", "ess01-002osr_723234_foil"},
	} {
		product := CSIPriceEntry{Name: "Chillet - Final Frostneedle (Sleeve & Card Set Vol. 1)", ItemSet: "Souls & Misc.", Notes: tt.notes, IsFoil: 1}
		card := matcher.InputCard{Name: palworldName(product.Name), Edition: product.ItemSet, Variation: palworldNotes(buylistVariation(product)), Foil: true}
		id, err := b.Match(&card)
		if err != nil || id != tt.uuid {
			t.Errorf("Match(%q) = %q, %v, want %q", tt.notes, id, err, tt.uuid)
		}
	}
}
