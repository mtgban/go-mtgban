package mtgmatcher

import "testing"

func TestPromoTypeSlug(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		// Magic's own types are already one lower-case word, so the slug
		// leaves them alone and "is:" keeps working exactly as it did.
		{"boosterfun", "boosterfun"},
		{"promopack", "promopack"},
		// The other games spell theirs as the storefront wrote them.
		{"best of", "bestof"},
		{"Alternate Art", "alternateart"},
		{"Disney Parks & Stores", "disneyparksstores"},
		{"Illumineer's Quest", "illumineersquest"},
		{"Premium Card Collection -Best Selection Vol. 6-", "premiumcardcollectionbestselectionvol6"},
		// Two spellings of one promo answer to one token, which is what
		// makes the tag askable at all: the catalog writes this event both
		// with and without the stray second space.
		{"English Version 2nd  Anniversary Set", "englishversion2ndanniversaryset"},
		{"English Version 2nd Anniversary Set", "englishversion2ndanniversaryset"},
		{"", ""},
	} {
		if got := PromoTypeSlug(tt.in); got != tt.want {
			t.Errorf("PromoTypeSlug(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
