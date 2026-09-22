package lorcana

import (
	"slices"
	"testing"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestExtractOrdinal pins what a "(V.N)" wording says about a listing's
// position among same-named, same-numbered siblings. Core Match splits the
// parenthetical off the name and appends it to Variation as "V.N" before
// this ever runs.
func TestExtractOrdinal(t *testing.T) {
	for _, tt := range []struct {
		desc, in string
		want     int
	}{
		{"the position is read off the wording", "5 V.2", 2},
		{"a bare number claims no position", "5", 0},
		{"a position of zero is not one a storefront writes", "5 V.0", 0},
		{"a non-numeric tail is not a position", "5 V.a", 0},
		{"a wording with no number at all", "Alternate Art", 0},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := extractOrdinal(tt.in); got != tt.want {
				t.Errorf("extractOrdinal(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

// TestPromoOrdinalTiebreak pins that a Cardmarket "(V.N)" wording decides
// between two Special printings a bare number cannot: "Maleficent -
// Monstrous Dragon" is card 5 of the P1 promo pool and card 5 of the P3
// pool, wave to wave, and Cardmarket sends a bare "5" for both - no
// denominator for totalTiebreak to read - so only the wording tells them
// apart, and Cardmarket writes it on every wave but the first. The pool
// itself is not read here; wave order stands in for it, which is what the
// wording is counting.
//
// An explicit "(V.N)" is trusted on its own; an unsuffixed wording is
// trusted as "V.1" only alongside an edition naming the promo wave shape,
// so a same-numbered Special tie some other storefront reaches - never
// having written "(V.N)" at all - stays refused rather than guessed at.
func TestPromoOrdinalTiebreak(t *testing.T) {
	b := &mtgmatcher.Backend{Sets: map[string]*mtgmatcher.Set{
		"1": {ReleaseDateTime: time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)},
		"9": {ReleaseDateTime: time.Date(2025, 9, 5, 0, 0, 0, 0, time.UTC)},
	}}
	wave1 := mtgmatcher.Card{UUID: "663_holofoil", Name: "Maleficent - Monstrous Dragon", SetCode: "1", Rarity: "special"}
	wave3 := mtgmatcher.Card{UUID: "2183_holofoil", Name: "Maleficent - Monstrous Dragon", SetCode: "9", Rarity: "special"}
	// A collision the datastore's own numbering produced, not a Cardmarket
	// promo wave - nothing about "(V.N)" wording claims anything here.
	setCard := mtgmatcher.Card{UUID: "163", Name: "Let It Go", SetCode: "1", Rarity: "rare"}
	otherSetCard := mtgmatcher.Card{UUID: "2626", Name: "Let It Go", SetCode: "11", Rarity: "rare"}

	for _, tt := range []struct {
		desc      string
		variation string
		edition   string
		cards     []mtgmatcher.Card
		want      []string
	}{
		{"an unsuffixed wording is Cardmarket's own V.1", "5", "Promos Year 3",
			[]mtgmatcher.Card{wave3, wave1}, []string{"663_holofoil"}},
		{"an explicit (V.2) reaches the later wave", "5 V.2", "Promos Year 3",
			[]mtgmatcher.Card{wave1, wave3}, []string{"2183_holofoil"}},
		{"an explicit (V.2) needs no edition confirmation at all", "5 V.2", "",
			[]mtgmatcher.Card{wave1, wave3}, []string{"2183_holofoil"}},
		{"a position past every candidate keeps the whole tier", "5 V.3", "Promos Year 3",
			[]mtgmatcher.Card{wave1, wave3}, []string{"663_holofoil", "2183_holofoil"}},
		{"one candidate is already the answer", "5 V.2", "Promos Year 3",
			[]mtgmatcher.Card{wave3}, []string{"2183_holofoil"}},
		{"negative: a tie no candidate is Special keeps the whole tier", "163 V.2", "Promos Year 3",
			[]mtgmatcher.Card{setCard, otherSetCard}, []string{"163", "2626"}},
		{"negative: one non-Special candidate keeps the whole tier", "5 V.2", "Promos Year 3",
			[]mtgmatcher.Card{wave1, setCard}, []string{"663_holofoil", "163"}},
		{"negative: an unrecognized pool is its own claim, not silence",
			"5/P9, Made Up Pool", "Promos Year 3", []mtgmatcher.Card{wave1, wave3},
			[]string{"663_holofoil", "2183_holofoil"}},
		{"negative: an unsuffixed wording elsewhere is not Cardmarket's V.1",
			"5", "", []mtgmatcher.Card{wave1, wave3},
			[]string{"663_holofoil", "2183_holofoil"}},
		{"negative: an unrelated edition does not confirm the wave either",
			"5", "The First Chapter", []mtgmatcher.Card{wave1, wave3},
			[]string{"663_holofoil", "2183_holofoil"}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			inCard := &mtgmatcher.InputCard{Variation: tt.variation, Edition: tt.edition}
			var got []string
			for _, card := range promoOrdinalTiebreak(b, inCard, tt.cards) {
				got = append(got, card.UUID)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("promoOrdinalTiebreak(%q, edition=%q) = %v, want %v", tt.variation, tt.edition, got, tt.want)
			}
		})
	}
}
