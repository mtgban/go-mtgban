package lorcana

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestExtractPool pins what a storefront's number says about the run. A
// Lorcana number is written over what it is one of, so the tail is either a
// count - the set's size - or the name of a promo run. It is returned as
// written now: the card's SetTotal holds the same denominator in the
// datastore's own spelling, where it used to be slugged to match a promo
// type.
func TestExtractPool(t *testing.T) {
	for _, tt := range []struct{ desc, in, want string }{
		{"a promo run names itself", "5/P3", "P3"},
		{"the initialisms keep their capitals", "1/PD1", "PD1"},
		{"a count is the set's size, not a run", "87/204", ""},
		{"a bare number says nothing about a run", "87", ""},
		{"nor does a number with an empty tail", "87/", ""},
		{"the run survives a trailing comma", "5/P3, Foil", "P3"},
		{"only the field the number is written in counts", "Enchanted 5/P3", "P3"},
		{"which is not the digit a storefront's prose leads with",
			"Chapter 1 version - 5/P3", "P3"},
		{"a wording with no number at all", "Alternate Art", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := extractPool(tt.in); got != tt.want {
				t.Errorf("extractPool(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestPoolTiebreak pins that the run decides between two printings a number
// cannot separate. Each promo run is numbered from one, so a card promoted
// twice wears the same number in both and only the denominator tells them
// apart - which is what a storefront writes there, "1/D23" against "1/P1",
// exactly where a card of the set writes "87/204".
//
// It is read off SetTotal. It used to be read off PromoTypes, where the run
// travelled because nothing else carried it; a numbering pool is not a
// promotion, and the datastore says so by publishing it as the total.
//
// The cards are built here rather than loaded because no pair in today's
// datastore needs this to be separated - every one of them is told apart by
// its finish first. The tiebreak is the guard for when that stops being
// true, so the guard is what gets tested.
func TestPoolTiebreak(t *testing.T) {
	p1 := mtgmatcher.Card{UUID: "659", Name: "Mickey Mouse - Brave Little Tailor", SetTotal: "P1"}
	d23 := mtgmatcher.Card{UUID: "1191", Name: "Mickey Mouse - Brave Little Tailor", SetTotal: "D23"}
	set := mtgmatcher.Card{UUID: "1", Name: "Ariel - On Human Legs", SetTotal: "204"}
	// Sealed and the few printings that publish no denominator at all.
	untotalled := mtgmatcher.Card{UUID: "x", Name: "Mickey Mouse - Brave Little Tailor"}

	for _, tt := range []struct {
		desc  string
		pool  string
		cards []mtgmatcher.Card
		want  []string
	}{
		{"the named run wins", "P1", []mtgmatcher.Card{p1, d23}, []string{"659"}},
		{"and so does the other one", "D23", []mtgmatcher.Card{p1, d23}, []string{"1191"}},
		{"the storefront's own casing still reaches it", "p1",
			[]mtgmatcher.Card{p1, d23}, []string{"659"}},
		{"a run no candidate carries keeps the whole tier", "P4",
			[]mtgmatcher.Card{p1, d23}, []string{"659", "1191"}},
		{"nothing written decides nothing", "",
			[]mtgmatcher.Card{p1, d23}, []string{"659", "1191"}},
		{"one candidate is already the answer", "P1",
			[]mtgmatcher.Card{d23}, []string{"1191"}},
		{"a set's size separates a set card from a run", "204",
			[]mtgmatcher.Card{set, p1}, []string{"1"}},
		{"a card carrying no total at all cannot be chosen", "P1",
			[]mtgmatcher.Card{untotalled, d23}, []string{"x", "1191"}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			var got []string
			for _, card := range poolTiebreak(tt.pool, tt.cards) {
				got = append(got, card.UUID)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("poolTiebreak(%q) = %v, want %v", tt.pool, got, tt.want)
			}
		})
	}
}
