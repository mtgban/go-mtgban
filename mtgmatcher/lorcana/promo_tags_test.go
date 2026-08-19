package lorcana

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPromoTagsAreSlugs pins the stored form. Lorcana writes none of this
// into the name - not one card name carries a parenthesis - so the tags come
// from the datastore's own promo fields, slugged like every other game's.
func TestPromoTagsAreSlugs(t *testing.T) {
	b := loadDatastore(t)

	for _, tag := range b.AllPromoTypes {
		if slug := mtgmatcher.PromoTypeSlug(tag); slug != tag {
			t.Errorf("declared tag %q is not its own slug (%q)", tag, slug)
		}
	}
	for _, tag := range []string{"d23", "organizedplay", "highgloss"} {
		if !slices.Contains(b.AllPromoTypes, tag) {
			t.Errorf("tag %q is not declared", tag)
		}
	}
	var tagged int
	for _, uuid := range b.AllUUIDs {
		co, err := b.GetUUID(uuid)
		if err == nil && co.HasPromoType("d23") {
			tagged++
		}
	}
	if tagged == 0 {
		t.Error("no printing carries d23, so is:d23 answers with nothing")
	}
}
