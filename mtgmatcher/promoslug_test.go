package mtgmatcher

import "testing"

func TestPromoTypeSlug(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		// Magic's types are already single words, so they pass through and
		// every "is:" query that worked keeps working.
		{"boosterfun", "boosterfun"},
		{"promopack", "promopack"},
		{"best of", "bestof"},
		{"Alternate Art", "alternateart"},
		{"Disney Parks & Stores", "disneyparksstores"},
		{"Illumineer's Quest", "illumineersquest"},
		{"Premium Card Collection -Best Selection Vol. 6-", "premiumcardcollectionbestselectionvol6"},
		// One promo the catalog spells two ways answers to one token.
		{"English Version 2nd  Anniversary Set", "englishversion2ndanniversaryset"},
		{"English Version 2nd Anniversary Set", "englishversion2ndanniversaryset"},
		{"", ""},
	} {
		if got := PromoTypeSlug(tt.in); got != tt.want {
			t.Errorf("PromoTypeSlug(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSlugDescribes(t *testing.T) {
	for _, tt := range []struct {
		wording, slug string
		want          bool
	}{
		// The words are joined back up a run at a time, so a slug made of
		// several words is found in wording that spells them out.
		{"Metal Best Of", "bestof", true},
		{"Metal Best Of", "metal", true},
		{"metal prize wall", "prizewall", true},
		{"OP01-001 Alternate Art", "alternateart", true},
		{"Champion Stamp", "champion", true},
		// A run has to start and end on word boundaries, which is what
		// keeps a slug from being found inside a longer word: a substring
		// test would call this a match.
		{"Metallic Foil", "metal", false},
		{"Bestowed Upon", "bestof", false},
		// Order matters, and unrelated wording says nothing.
		{"Of Best", "bestof", false},
		{"Prerelease Stamped", "prizewall", false},
		{"anything", "", false},
	} {
		if got := SlugDescribes(tt.wording, tt.slug); got != tt.want {
			t.Errorf("SlugDescribes(%q, %q) = %v, want %v", tt.wording, tt.slug, got, tt.want)
		}
	}
}
