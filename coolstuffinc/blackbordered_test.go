package coolstuffinc

import (
	"testing"

	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/games"
)

// TestPreprocessBlackBorderedForeign pins the one shelf CSI files every
// foreign black-bordered print run under, regardless of era: the language
// word (in the note, the bracket, or both at once - "Italian language
// Italian" when a row carries both) is the only thing telling an FBB row
// from a 4BB one apart, and the language set within each. A card genuinely
// absent from both - Hell's Caretaker only exists as
// the Chronicles Japanese reprint, whatever CSI's own "(Italian)" bracket
// claims - correctly stays unresolved rather than being forced onto a set
// that never carried it.
func TestPreprocessBlackBorderedForeign(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	for _, tt := range []struct {
		desc         string
		name         string
		variant      string
		wantPreErr   bool // preprocess itself refuses the row
		wantEd       string
		wantLang     string
		wantResolved bool // whether Match should find a card
		wantSet      string
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
			desc:         "German resolves against FBB's German set",
			name:         "Animate Dead (German)",
			wantEd:       "FBB",
			wantLang:     "German",
			wantResolved: true,
			wantSet:      "FBBDE",
		},
		{
			desc:         "French resolves against FBB's French set",
			name:         "Badlands (French)",
			wantEd:       "FBB",
			wantLang:     "French",
			wantResolved: true,
			wantSet:      "FBBFR",
		},
		{
			desc:         "Spanish resolves against 4BB's Spanish set",
			name:         "Alabaster Potion (Spanish)",
			wantEd:       "4BB",
			wantLang:     "Spanish",
			wantResolved: true,
			wantSet:      "4BBES",
		},
		{
			desc:       "a language neither set was printed in is refused",
			name:       "Badlands (Russian)",
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
			card, err := preprocess(b, tt.name, "Black Bordered (foreign)", tt.variant, "", false)
			if tt.wantPreErr {
				if err != matcher.ErrUnsupported {
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

			id, err := b.Match(card)
			if tt.wantResolved && (err != nil || id == "") {
				t.Errorf("Match(%+v) = %q, %v, want a resolved id", card, id, err)
			} else if !tt.wantResolved && err == nil {
				t.Errorf("Match(%+v) = %q, nil, want it to stay unresolved", card, id)
			}
			co, _ := b.GetUUID(id)
			if tt.wantSet != "" && (co == nil || co.SetCode != tt.wantSet) {
				t.Errorf("Match(%+v) = %q, want a printing in %s", card, id, tt.wantSet)
			}
		})
	}
}
