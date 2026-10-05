package strikezone

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
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
		wantPromoType            string
	}{
		{
			desc: "a borderless printing the listing does not call borderless",
			name: "Betor, Kin to All (Showcase)", shelf: "Tarkir: Dragonstorm",
			notes: "Near Mint Normal English", wantPromoType: "showcase",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			co := tieBreak(t, b, tt.name, tt.shelf, tt.notes)
			if co == nil {
				t.Fatalf("%q was left aliased", tt.name)
			}
			if !co.HasPromoType(tt.wantPromoType) {
				t.Errorf("%q landed on %s #%s %v, want %s", tt.name, co.SetCode, co.Number, co.PromoTypes, tt.wantPromoType)
			}
		})
	}
}
