package gundam

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// rejoinBackend is a datastore of the shapes rejoinName exists for: a card
// whose name carries a parenthetical beside a plain card of the same head, a
// promo printing of one, a name with the parenthetical in the middle, and a
// suit sold plain and as a parallel, which must go on reading as a rarity.
func rejoinBackend() *mtgmatcher.Backend {
	payload := &Datastore{Game: "gundam"}
	payload.Sets = map[string]struct {
		Name        string `json:"name"`
		ReleaseDate string `json:"releaseDate"`
		Type        string `json:"type,omitempty"`
	}{
		"GD01":   {Name: "Newtype Rising", ReleaseDate: "2025-04-25"},
		"GD02":   {Name: "Dual Impact", ReleaseDate: "2025-08-01"},
		"GD03":   {Name: "Steel Requiem", ReleaseDate: "2025-11-14"},
		"GD04":   {Name: "Phantom Aria", ReleaseDate: "2026-02-20"},
		"GCG-PR": {Name: "Gundam Promotional Cards", ReleaseDate: "2025-04-25", Type: "promo"},
	}
	// The backend groups printings by product, so every card here is its
	// own product, numbered from the tail of its id.
	var product int
	card := func(id, name, number, set, rarity, variant string, promo ...string) DatastoreCard {
		product++
		c := DatastoreCard{ID: id, Name: name, Number: number, SetCode: set, Rarity: rarity, Variant: variant, Finish: "Normal", PromoTypes: promo}
		c.ExternalLinks.TcgPlayerID = product
		return c
	}
	payload.Cards = []DatastoreCard{
		card("gd01-001_1", "Gundam", "GD01-001", "GD01", "Legend Rare", ""),
		card("gd01-001_2", "Gundam", "GD01-001", "GD01", "LR+", ""),
		card("gd01-006_3", "Delta Plus", "GD01-006", "GD01", "Rare", ""),
		card("gd02-017_4", "Delta Plus (Waverider Mode)", "GD02-017", "GD02", "Common", ""),
		card("gd03-076_5", "Freedom Gundam (METEOR)", "GD03-076", "GD03", "Rare", ""),
		card("gd03-076_6", "Freedom Gundam (METEOR)", "GD03-076", "GCG-PR", "Rare", "Boost Kit 01", "boostkit"),
		card("gd04-009_7", "Guncannon (108) & Guncannon (109)", "GD04-009", "GD04", "Rare", ""),
		card("gd03-032_8", "Zaku (Four Snake Eyes') [YETI] (GQ)", "GD03-032", "GD03", "Common", ""),
		card("gd01-098_9", "Elan Ceres (Enhanced Person Number 4)", "GD01-098", "GD01", "Common", ""),
		card("gd04-009_10", "Guncannon (108) & Guncannon (109)", "GD04-009", "GD04", "R+", ""),
		card("gd01-082_11", "Gundam Aerial", "GD01-082", "GD01", "Common", ""),
	}
	return payload.newBackend()
}

// TestRejoinNameGivesBackTheParenthetical pins that a storefront writing the
// head of a card's name plain, with the card's own parenthetical beside it,
// reaches the card and not a plain card of the same head - and that the
// words which were never the name's, a promo run or a rarity, stay behind
// as the variation the filters read.
func TestRejoinNameGivesBackTheParenthetical(t *testing.T) {
	b := rejoinBackend()
	for _, test := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"the head is another card", mtgmatcher.InputCard{
			Name: "Delta Plus", Variation: "Waverider Mode", Edition: "Dual Impact"}, "gd02-017_4"},
		{"the head is the plain card when nothing is beside it", mtgmatcher.InputCard{
			Name: "Delta Plus", Edition: "Newtype Rising"}, "gd01-006_3"},
		{"the whole name written as the card spells it", mtgmatcher.InputCard{
			Name: "Delta Plus (Waverider Mode)", Edition: "Dual Impact"}, "gd02-017_4"},
		{"the promo run stays behind as the variation", mtgmatcher.InputCard{
			Name: "Freedom Gundam", Variation: "Meteor Boost Kit 01", Edition: "Gundam Promos"}, "gd03-076_6"},
		{"the parenthetical spelled in another case", mtgmatcher.InputCard{
			Name: "Freedom Gundam (Meteor)", Edition: "Steel Requiem"}, "gd03-076_5"},
		{"a parenthetical in the middle of the name", mtgmatcher.InputCard{
			Name: "Guncannon & Guncannon", Variation: "108 109", Edition: "Phantom Aria"}, "gd04-009_7"},
		{"two of them, around a bracket", mtgmatcher.InputCard{
			Name: "Zaku [YETI]", Variation: "Four Snake Eyes' GQ", Edition: "Steel Requiem"}, "gd03-032_8"},
		{"a head that is no card on its own", mtgmatcher.InputCard{
			Name: "Elan Ceres", Variation: "Enhanced Person Number 4", Edition: "Newtype Rising"}, "gd01-098_9"},
		{"a rarity beside a plain name is still the rarity", mtgmatcher.InputCard{
			Name: "Gundam", Variation: "LR+", Edition: "Newtype Rising"}, "gd01-001_2"},
		{"the number written first", mtgmatcher.InputCard{
			Name: "Delta Plus", Variation: "GD02-017 Waverider Mode", Edition: "Dual Impact"}, "gd02-017_4"},
		{"the number first and the promo run last", mtgmatcher.InputCard{
			Name: "Freedom Gundam", Variation: "GD03-076 Meteor Boost Kit 01", Edition: "Gundam Promos"}, "gd03-076_6"},
		{"the parallel of a name with a parenthetical in the middle", mtgmatcher.InputCard{
			Name: "Guncannon & Guncannon", Variation: "GD04-009 108 109 R+", Edition: "Phantom Aria"}, "gd04-009_10"},
		{"a head the canonical name does not hold stays the head", mtgmatcher.InputCard{
			Name: "Gundam Aerial", Variation: "Waverider Mode", Edition: "Newtype Rising"}, "gd01-082_11"},
	} {
		in := test.in
		got, err := b.Match(&in)
		if test.want == "" {
			if err == nil {
				t.Errorf("%s: Match(%v) = %s, want an error", test.desc, test.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: Match(%v) = %v", test.desc, test.in, err)
			continue
		}
		if got != test.want {
			t.Errorf("%s: Match(%v) = %s, want %s", test.desc, test.in, got, test.want)
		}
	}
}
