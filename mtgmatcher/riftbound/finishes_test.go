package riftbound

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestCardFinishes pins how the finishes a card is sold in are read off its
// printings: TCGplayer's own names are placed, a name TCGplayer adds later
// arrives as its own finish, and one finish named twice is stored once.
func TestCardFinishes(t *testing.T) {
	type printing = struct {
		Finish string `json:"finish"`
		ID     string `json:"id"`
	}
	both := []string{mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil}

	tests := []struct {
		name      string
		printings []printing
		want      []string
	}{
		{
			name: "a card with no printing has no finish",
			want: nil,
		},
		{
			name:      "foil only is kept as foil only",
			printings: []printing{{"Foil", "x_foil"}},
			want:      []string{mtgmatcher.FinishFoil},
		},
		{
			name:      "nonfoil only is kept as nonfoil only",
			printings: []printing{{"Normal", "x"}},
			want:      []string{mtgmatcher.FinishNonfoil},
		},
		{
			name:      "TCGplayer's own names are placed, in the order given",
			printings: []printing{{"Normal", "x"}, {"Foil", "x_foil"}},
			want:      both,
		},
		{
			// The reason the vocabulary is open rather than a list kept
			// here: the printings under a TCGplayer category are the
			// vendor's to add, and a third one has to arrive as data.
			name:      "a printing TCGplayer adds later arrives as its own finish",
			printings: []printing{{"Normal", "x"}, {"Holofoil", "x_holofoil"}},
			want:      []string{mtgmatcher.FinishNonfoil, "holofoil"},
		},
		{
			name:      "a name is placed however it is spelled",
			printings: []printing{{"Cold Foil", "x_coldfoil"}},
			want:      []string{"coldfoil"},
		},
		{
			// Both spellings of one finish name one printing, and a card
			// listing each would otherwise be stored twice.
			name:      "one finish named twice is stored once",
			printings: []printing{{"Normal", "x"}, {mtgmatcher.FinishNonfoil, "y"}},
			want:      []string{mtgmatcher.FinishNonfoil},
		},
		{
			name:      "a printing published without a uuid is left out",
			printings: []printing{{"Normal", ""}, {"Foil", "x_foil"}},
			want:      []string{mtgmatcher.FinishFoil},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := cardFinishes(GalleryCard{Printings: test.printings})
			if !slices.Equal(got, test.want) {
				t.Errorf("cardFinishes(%v) = %v, want %v", test.printings, got, test.want)
			}
		})
	}
}
