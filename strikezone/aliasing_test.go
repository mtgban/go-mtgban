package strikezone

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// tieBreak runs a listing through preprocess and the match the way processRow
// does. It fails the test unless the match is ambiguous, and returns the
// printing resolveAliasing settles on, nil when it settles on none.
func tieBreak(t *testing.T, b *mtgmatcher.Backend, name, shelf, notes string) *mtgmatcher.CardObject {
	t.Helper()

	card, err := preprocess(b, name, shelf, notes)
	if err != nil {
		t.Fatalf("preprocess(%q) = %v", name, err)
	}
	_, err = b.Match(card)
	var alias *mtgmatcher.AliasingError
	if !errors.As(err, &alias) {
		t.Fatalf("Match(%q) = %v, want an aliasing", name, err)
	}
	id := resolveAliasing(b, aliasedListing{name: name, shelf: shelf, card: card}, alias.Probe())
	if id == "" {
		return nil
	}
	co, err := b.GetUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	return co
}

// TestAliasingContradictions pins one listing per way the wording rules a
// printing out, each reaching a printing the other candidates lack.
func TestAliasingContradictions(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		desc, name, shelf, notes string
		ok                       func(co *mtgmatcher.CardObject) bool
		want                     string
	}{
		{
			desc: "a borderless printing the listing does not call borderless",
			name: "Betor, Kin to All (Showcase)", shelf: "Tarkir: Dragonstorm",
			notes: "Near Mint Normal English",
			ok:    func(co *mtgmatcher.CardObject) bool { return !co.HasPromoType(magic.PromoTypeBorderless) },
			want:  "a printing that is not borderless",
		},
		{
			desc: "a flavor name the listing does not write",
			name: "Beast Within (Borderless)", shelf: "Marvel Universe Eternal-Legal",
			notes: "Near Mint Normal English",
			ok:    func(co *mtgmatcher.CardObject) bool { return co.FlavorName == "" },
			want:  "the printing with no flavor name",
		},
		{
			desc: "a printing never made in the finish on sale",
			name: "Lightning Bolt", shelf: "Promos: Magicfest",
			notes: "Near Mint Normal English",
			ok:    func(co *mtgmatcher.CardObject) bool { return co.HasFinish(mtgmatcher.FinishNonfoil) },
			want:  "a printing made in nonfoil",
		},
		{
			desc: "several printings left, the one in the set the shelf is named for",
			name: "Delighted Halfling (Borderless)", shelf: "Universes Beyond: The Lord of the Rings: Tales of Middle-earth",
			notes: "Foil",
			ok:    func(co *mtgmatcher.CardObject) bool { return co.SetCode == "LTR" },
			want:  "the printing in LTR",
		},
		{
			desc: "a promo pack land a later set reprinted, on the shelf of the set it came in",
			name: "Deserted Beach", shelf: "Promo Pack: Innistrad: Midnight Hunt",
			notes: "Near Mint Normal English",
			ok:    func(co *mtgmatcher.CardObject) bool { return co.SetCode == "PMID" },
			want:  "the printing in PMID",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			co := tieBreak(t, b, tt.name, tt.shelf, tt.notes)
			if co == nil {
				t.Fatalf("%q was left aliased", tt.name)
			}
			if !tt.ok(co) {
				t.Errorf("%q landed on %s #%s %v, want %s", tt.name, co.SetCode, co.Number, co.PromoTypes, tt.want)
			}
		})
	}
}

// TestAliasingStands pins the listings left aliased: a lone survivor on a
// promo shelf that is not a promo, and several survivors with nothing to tell
// them apart.
func TestAliasingStands(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		desc, name, shelf, notes string
	}{
		{"a non-promo is not what a promo shelf holds", "Scute Swarm", "Promos: Pro Tour", "Near Mint Foil English"},
		{"two printings the wording fits equally", "Lightning Bolt", "Promos: Magicfest", "Near Mint Foil English"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			co := tieBreak(t, b, tt.name, tt.shelf, tt.notes)
			if co != nil {
				t.Errorf("%q landed on %s #%s, want it left aliased", tt.name, co.SetCode, co.Number)
			}
		})
	}
}
