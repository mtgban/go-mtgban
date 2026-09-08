package riftbound

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestCardFinishes pins the two spellings a datastore reaches here in and
// the fallback under both. The gallery says nothing about finish, so a
// datastore built before the builder recorded it has to keep loading as it
// always did - every card sold in both - rather than losing every printing
// to an empty finish list.
func TestCardFinishes(t *testing.T) {
	both := []string{mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil}

	tests := []struct {
		name string
		card GalleryCard
		want []string
	}{
		{
			name: "a datastore without the field falls back to both",
			card: GalleryCard{},
			want: both,
		},
		{
			name: "foil only is kept as foil only",
			card: GalleryCard{Finishes: []string{mtgmatcher.FinishFoil}},
			want: []string{mtgmatcher.FinishFoil},
		},
		{
			name: "nonfoil only is kept as nonfoil only",
			card: GalleryCard{Finishes: []string{mtgmatcher.FinishNonfoil}},
			want: []string{mtgmatcher.FinishNonfoil},
		},
		{
			name: "both are kept in the order given",
			card: GalleryCard{Finishes: both},
			want: both,
		},
		{
			// What the builder publishes now: the printing name TCGplayer
			// prices the sku under, not the matcher's own spelling.
			name: "TCGplayer's own names are placed",
			card: GalleryCard{Finishes: []string{"Normal", "Foil"}},
			want: both,
		},
		{
			// The reason the vocabulary is open rather than a list kept
			// here: the printings under a TCGplayer category are the
			// vendor's to add, and a third one has to arrive as data.
			name: "a printing TCGplayer adds later arrives as its own finish",
			card: GalleryCard{Finishes: []string{"Normal", "Holofoil"}},
			want: []string{mtgmatcher.FinishNonfoil, "holofoil"},
		},
		{
			name: "a name is placed however it is spelled",
			card: GalleryCard{Finishes: []string{"Cold Foil"}},
			want: []string{"coldfoil"},
		},
		{
			// Both spellings of one finish name one printing, and a card
			// listing each would otherwise be stored twice.
			name: "one finish named twice is stored once",
			card: GalleryCard{Finishes: []string{"Normal", mtgmatcher.FinishNonfoil}},
			want: []string{mtgmatcher.FinishNonfoil},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := cardFinishes(test.card)
			if !slices.Equal(got, test.want) {
				t.Errorf("cardFinishes(%v) = %v, want %v", test.card.Finishes, got, test.want)
			}
		})
	}
}

// TestCanonicalFinish pins that Riftbound places TCGplayer's printing names
// and the matcher's own spelling on the same finish, and hands back anything
// else rather than refusing it - a printing the vendor adds to the category
// must reach a uuid without a release of this package.
func TestCanonicalFinish(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		// TCGplayer's whole vocabulary for the category today
		{"Normal", mtgmatcher.FinishNonfoil},
		{"Foil", mtgmatcher.FinishFoil},
		// The spelling the datastores built before that carry
		{"nonfoil", mtgmatcher.FinishNonfoil},
		{"foil", mtgmatcher.FinishFoil},
		// However a storefront writes it
		{"NON-FOIL", mtgmatcher.FinishNonfoil},
		{"normal", mtgmatcher.FinishNonfoil},
		// A printing name the category does not have yet
		{"Holofoil", "holofoil"},
		{"Cold Foil", "coldfoil"},
		{"", ""},
	}

	for _, test := range tests {
		if got := (Rules{}).CanonicalFinish(test.in); got != test.want {
			t.Errorf("CanonicalFinish(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}
