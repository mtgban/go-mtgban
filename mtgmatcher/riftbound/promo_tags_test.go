package riftbound

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPromoTagsAreSlugs pins the form the tags are stored in. A search query
// is split on whitespace before any filter sees it, so a tag is only askable
// as one word - the same shape Magic's own promo types have always had.
func TestPromoTagsAreSlugs(t *testing.T) {
	b := loadBackend(t)

	for _, tag := range b.AllPromoTypes {
		if slug := mtgmatcher.PromoTypeSlug(tag); slug != tag {
			t.Errorf("declared tag %q is not its own slug (%q), so is:%s cannot reach it", tag, slug, slug)
		}
	}
	for _, tag := range []string{"metal", "bestof", "prizewall", "alternateart"} {
		if !slices.Contains(b.AllPromoTypes, tag) {
			t.Errorf("tag %q is not declared", tag)
		}
	}
	// A qualifier that only repeats the collector number describes nothing.
	for _, tag := range b.AllPromoTypes {
		if len(tag) > 1 && tag[0] == 'r' && tag[1] >= '0' && tag[1] <= '9' {
			t.Errorf("declared tag %q is a collector number, not a description", tag)
		}
	}

	// The three printings that share name and number are told apart by tag.
	var bestOf int
	for _, uuid := range b.AllUUIDs {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		if co.Name == "Teemo, Swift Scout" && co.HasPromoType("bestof") {
			bestOf++
		}
	}
	if bestOf == 0 {
		t.Error("no Teemo, Swift Scout printing carries bestof")
	}
}
