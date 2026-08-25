package fleshandblood

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPaddedNumberMatches pins the padding rule. The catalog writes three
// numbers a digit wider than the printing wears them - "JDG0077" for JDG077 -
// and a storefront writes whichever it was handed, so both have to reach the
// same printing and neither may reach its neighbours.
func TestPaddedNumberMatches(t *testing.T) {
	for _, tt := range []struct {
		input, full string
		want        bool
	}{
		{"JDG0077", "JDG0077", true},
		{"JDG077", "JDG0077", true},
		{"HER0160", "HER160", true},
		{"HER156", "HER0156", true},
		// Padding is all that may be folded away: a different number is
		// still a different printing.
		{"JDG0077", "JDG024", false},
		{"HER0160", "HER035", false},
		{"HER0156", "HER0160", false},
		// A run of zeros inside the number is not padding.
		{"WTR040", "WTR40", true},
		{"WTR104", "WTR14", false},
	} {
		if got := numberMatches(tt.input, tt.full); got != tt.want {
			t.Errorf("numberMatches(%q, %q) = %v, want %v", tt.input, tt.full, got, tt.want)
		}
	}
}

// TestPaddedNumberExtracted pins that a padded number is read as a number at
// all: left unread, the match falls back on the name and answers with some
// other printing of it.
func TestPaddedNumberExtracted(t *testing.T) {
	for _, tt := range []struct{ variation, want string }{
		{"JDG0077", "JDG0077"},
		{"HER0160", "HER0160"},
		{"HER156", "HER156"},
		{"WTR215", "WTR215"},
		{"WTR040 // WTR039", "WTR040 // WTR039"},
	} {
		if got := extractNumber(tt.variation); got != tt.want {
			t.Errorf("extractNumber(%q) = %q, want %q", tt.variation, got, tt.want)
		}
	}
}

// TestEventTokenEditionUnsupported pins the refusal of the storefront
// expansion holding the organized-play tokens, which the shared token gate
// used to refuse for the whole matcher. The catalog carries six of its
// printings, every one reached by its TCGplayer id; the rest name printings
// it does not carry, and the expansion names no set, so nothing would keep
// them off whichever other set prints the same name.
func TestEventTokenEditionUnsupported(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
		err  string
	}{
		{
			desc: "a name the catalog carries only in another set",
			in:   mtgmatcher.InputCard{Name: "Chane // Hatchet of Body", Edition: "OP Event Tokens"},
			err:  mtgmatcher.ErrUnsupported.Error(),
		},
		{
			desc: "and one it carries in several",
			in:   mtgmatcher.InputCard{Name: "Rampart of the Ram's Head", Edition: "OP Event Tokens"},
			err:  mtgmatcher.ErrUnsupported.Error(),
		},
		{
			desc: "an ordinary set's token still resolves",
			in:   mtgmatcher.InputCard{Name: "Cracked Bauble", Edition: "Welcome to Rathe", Variation: "WTR224"},
			want: "wtr224_225302_unl",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			gotErr := ""
			if err != nil {
				gotErr = err.Error()
			}
			if id != tt.want || gotErr != tt.err {
				t.Errorf("Match(%q, %q) = (%q, %q), want (%q, %q)",
					tt.in.Name, tt.in.Edition, id, gotErr, tt.want, tt.err)
			}
		})
	}
}
