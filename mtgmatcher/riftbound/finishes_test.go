package riftbound

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestCardFinishes covers the fallback above all. The gallery says nothing
// about finish, so a datastore built before the builder recorded it has to
// keep loading as it always did - every card sold in both - rather than
// losing every printing to an empty finish list.
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
			// A finish the matcher has no uuid scheme for would otherwise
			// reach FoilUUIDs and answer for a printing that cannot be sold.
			name: "a finish this game does not have is dropped",
			card: GalleryCard{Finishes: []string{"etched", mtgmatcher.FinishFoil}},
			want: []string{mtgmatcher.FinishFoil},
		},
		{
			name: "dropping every finish falls back rather than leaving none",
			card: GalleryCard{Finishes: []string{"etched"}},
			want: both,
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
