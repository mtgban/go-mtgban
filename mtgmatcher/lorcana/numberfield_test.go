package lorcana

import (
	"testing"
)

// TestNumberField pins which field of a variation the number is read out of.
// A Lorcana number is written over what it is one of, so the field written
// that way is the number even where a storefront's prose put a digit in
// front of it; the first digit-leading field only answers where no field is
// written as a number at all.
func TestNumberField(t *testing.T) {
	for _, tt := range []struct{ desc, in, want string }{
		{"a number written as one", "177/204", "177/204"},
		{"a bare number is all there is to read", "87", "87"},
		{"prose in front of the number does not become it",
			"Chapter 1 version - 177/204", "177/204"},
		{"nor does prose behind a bare number displace it",
			"Enchanted 205 of 204", "205"},
		{"a wording with no number at all", "Alternate Art", ""},
		{"the trailing comma travels with the field", "36/P2, Puzzle Promo", "36/P2,"},
		{"a promo series marker stays on the number", "000B", "000B"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := numberField(tt.in); got != tt.want {
				t.Errorf("numberField(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestExtractNumberReadsTheWrittenNumber pins the number itself over the same
// wordings, since that is what FilterCards compares against a card's.
func TestExtractNumberReadsTheWrittenNumber(t *testing.T) {
	for _, tt := range []struct{ desc, in, want string }{
		{"the card is the one the number names, not the chapter",
			"Chapter 1 version - 177/204", "177"},
		{"and the same where the chapter is the larger number",
			"Chapter 1 version - 4/204", "4"},
		{"a bare number is unchanged", "87", "87"},
		{"leading zeros still go", "007/204", "7"},
		{"a number the zeros are the whole of comes back", "000B", "0B"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := extractNumber(tt.in); got != tt.want {
				t.Errorf("extractNumber(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
