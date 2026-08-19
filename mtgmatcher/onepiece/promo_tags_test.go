package onepiece

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPromoTagsAreSlugs pins the stored form for the game that needs it
// most: every One Piece card is named after a character, so the qualifier is
// the only thing that says which printing, and there are 464 of them.
func TestPromoTagsAreSlugs(t *testing.T) {
	b := loadBackend(t)

	for _, tag := range b.AllPromoTypes {
		if slug := mtgmatcher.PromoTypeSlug(tag); slug != tag {
			t.Errorf("declared tag %q is not its own slug (%q)", tag, slug)
		}
	}
	for _, tag := range []string{"alternateart", "parallel", "manga", "premiumcardcollectionbestselectionvol6"} {
		if !slices.Contains(b.AllPromoTypes, tag) {
			t.Errorf("tag %q is not declared", tag)
		}
	}

	// The printing that started this: reachable by the event that issued it.
	var hits int
	for _, uuid := range b.AllUUIDs {
		co, err := b.GetUUID(uuid)
		if err == nil && co.HasPromoType("premiumcardcollectionbestselectionvol6") {
			hits++
		}
	}
	if hits == 0 {
		t.Error("no printing carries the Best Selection Vol. 6 tag")
	}
}
