package lorcana

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestExtractTotal pins what a storefront's number says about the printing. A
// Lorcana number is written over what it is one of, so the tail is either a
// count - the set's size - or the name of a promo run, and either is the
// denominator the face prints. It is returned as written: the card's SetTotal
// holds the same denominator in the datastore's own spelling, where it used
// to be slugged to match a promo type.
func TestExtractTotal(t *testing.T) {
	for _, tt := range []struct{ desc, in, want string }{
		{"a promo run names itself", "5/P3", "P3"},
		{"the initialisms keep their capitals", "1/PD1", "PD1"},
		{"a count is the set's size, and says so", "87/204", "204"},
		{"a bare number says nothing about a total", "87", ""},
		{"nor does a number with an empty tail", "87/", ""},
		{"the total survives a trailing comma", "5/P3, Foil", "P3"},
		{"only the field the number is written in counts", "Enchanted 5/P3", "P3"},
		{"which is not the digit a storefront's prose leads with",
			"Chapter 1 version - 5/P3", "P3"},
		{"a wording with no number at all", "Alternate Art", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := extractTotal(tt.in); got != tt.want {
				t.Errorf("extractTotal(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestTotalTiebreak pins that the denominator decides between two printings a
// number cannot separate. Each promo run is numbered from one, so a card
// promoted twice wears the same number in both and only the denominator tells
// them apart - which is what a storefront writes there, "1/D23" against
// "1/P1", exactly where a card of the set writes "87/204".
//
// A count separates the same way: a set card and a promo of it stand at one
// number and print different totals under it, which is how Fabled's "Stitch -
// Rock Star" at "3/204" is told from the Disney Parks printing at "3/DIS".
//
// It is read off SetTotal. It used to be read off PromoTypes, where the run
// travelled because nothing else carried it; a numbering pool is not a
// promotion, and the datastore says so by publishing it as the total.
func TestTotalTiebreak(t *testing.T) {
	p1 := mtgmatcher.Card{UUID: "659", Name: "Mickey Mouse - Brave Little Tailor", SetTotal: "P1"}
	d23 := mtgmatcher.Card{UUID: "1191", Name: "Mickey Mouse - Brave Little Tailor", SetTotal: "D23"}
	set := mtgmatcher.Card{UUID: "1", Name: "Ariel - On Human Legs", SetTotal: "204"}
	// The pair Cool Stuff Inc's buylist answered with both of at once.
	stitch := mtgmatcher.Card{UUID: "1939", Name: "Stitch - Rock Star", SetTotal: "204"}
	parks := mtgmatcher.Card{UUID: "3245", Name: "Stitch - Rock Star", SetTotal: "DIS"}
	// Sealed and the few printings that publish no denominator at all.
	untotalled := mtgmatcher.Card{UUID: "x", Name: "Mickey Mouse - Brave Little Tailor"}

	for _, tt := range []struct {
		desc  string
		total string
		cards []mtgmatcher.Card
		want  []string
	}{
		{"the named run wins", "P1", []mtgmatcher.Card{p1, d23}, []string{"659"}},
		{"and so does the other one", "D23", []mtgmatcher.Card{p1, d23}, []string{"1191"}},
		{"the storefront's own casing still reaches it", "p1",
			[]mtgmatcher.Card{p1, d23}, []string{"659"}},
		{"a total no candidate carries keeps the whole tier", "P4",
			[]mtgmatcher.Card{p1, d23}, []string{"659", "1191"}},
		{"nothing written decides nothing", "",
			[]mtgmatcher.Card{p1, d23}, []string{"659", "1191"}},
		{"one candidate is already the answer", "P1",
			[]mtgmatcher.Card{d23}, []string{"1191"}},
		{"a set's size separates a set card from a run", "204",
			[]mtgmatcher.Card{set, p1}, []string{"1"}},
		{"and separates a set card from a promo of itself", "204",
			[]mtgmatcher.Card{stitch, parks}, []string{"1939"}},
		{"the promo's own total reaches the promo", "DIS",
			[]mtgmatcher.Card{stitch, parks}, []string{"3245"}},
		{"a card carrying no total at all cannot be chosen", "P1",
			[]mtgmatcher.Card{untotalled, d23}, []string{"x", "1191"}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			var got []string
			for _, card := range totalTiebreak(tt.total, tt.cards) {
				got = append(got, card.UUID)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("totalTiebreak(%q) = %v, want %v", tt.total, got, tt.want)
			}
		})
	}
}
