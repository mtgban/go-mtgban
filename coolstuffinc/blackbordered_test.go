package coolstuffinc

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPreprocessBlackBorderedForeign pins the one shelf CSI files every
// foreign black-bordered print run under, regardless of era: the language
// word (in the note, the bracket, or both at once - "Italian language
// Italian" when a row carries both) is the only thing telling an FBB row
// from a 4BB one apart, and only those two have a row in the datastore at
// all. A card genuinely absent from both - Hell's Caretaker only exists as
// the Chronicles Japanese reprint, whatever CSI's own "(Italian)" bracket
// claims - correctly stays unresolved rather than being forced onto a set
// that never carried it.
func TestPreprocessBlackBorderedForeign(t *testing.T) {
	withGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	for _, tt := range []struct {
		desc         string
		name         string
		variant      string
		wantPreErr   bool // preprocess itself refuses the row
		wantEd       string
		wantLang     string
		wantResolved bool // whether Match should find a card
	}{
		{
			desc:         "a bare Italian bracket resolves against FBB",
			name:         "Animate Wall (Italian)",
			wantEd:       "FBB",
			wantLang:     "Italian",
			wantResolved: true,
		},
		{
			desc:         "a bare Japanese bracket resolves against 4BB",
			name:         "Alabaster Potion (Japanese)",
			wantEd:       "4BB",
			wantLang:     "Japanese",
			wantResolved: true,
		},
		{
			desc:         "a verbose note alongside the bracket still reads Italian",
			name:         "Bayou (Italian)",
			variant:      "Italian language.",
			wantEd:       "FBB",
			wantLang:     "Italian",
			wantResolved: true,
		},
		{
			desc:       "German has no row in the datastore and is refused",
			name:       "Animate Dead (German)",
			wantPreErr: true,
		},
		{
			desc:       "French has no row in the datastore and is refused",
			name:       "Badlands (French)",
			wantPreErr: true,
		},
		{
			desc:     "a card the shelf mislabels as Italian stays unresolved",
			name:     "Hell's Caretaker (Italian)",
			wantEd:   "FBB",
			wantLang: "Italian",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := preprocess(tt.name, "Black Bordered (foreign)", tt.variant, "")
			if tt.wantPreErr {
				if err != mtgmatcher.ErrUnsupported {
					t.Fatalf("preprocess(%q) = %v, want ErrUnsupported", tt.name, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", tt.name, err)
			}
			if card.Edition != tt.wantEd || card.Language != tt.wantLang {
				t.Errorf("preprocess(%q) = edition %q language %q, want %q/%q",
					tt.name, card.Edition, card.Language, tt.wantEd, tt.wantLang)
			}

			id, err := mtgmatcher.Match(card)
			if tt.wantResolved && (err != nil || id == "") {
				t.Errorf("Match(%+v) = %q, %v, want a resolved id", card, id, err)
			} else if !tt.wantResolved && err == nil {
				t.Errorf("Match(%+v) = %q, nil, want it to stay unresolved", card, id)
			}
		})
	}
}
