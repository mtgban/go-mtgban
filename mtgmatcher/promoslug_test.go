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

func TestSlugDescribesAny(t *testing.T) {
	for _, tt := range []struct {
		wording string
		slugs   []string
		want    bool
	}{
		// The tag the wording names is the second one, which is the whole
		// point: asking only the first would answer no.
		{"OMN133 Extended Art", []string{"blue", "extendedart"}, true},
		{"OMN133 Blue", []string{"blue", "extendedart"}, true},
		{"OMN133 Cold Foil", []string{"blue", "extendedart"}, false},
		{"anything", nil, false},
	} {
		if got := SlugDescribesAny(tt.wording, tt.slugs); got != tt.want {
			t.Errorf("SlugDescribesAny(%q, %v) = %v, want %v", tt.wording, tt.slugs, got, tt.want)
		}
	}
}

// TestDescribedVariants pins the two halves of the ranking on the shape that
// forced it: one Yu-Gi-Oh number printed as a colour, an alternate art, and
// the two together.
func TestDescribedVariants(t *testing.T) {
	blue := Card{UUID: "blue", PromoTypes: []string{"blue"}}
	green := Card{UUID: "green", PromoTypes: []string{"green"}}
	art := Card{UUID: "art", PromoTypes: []string{"alternateart"}}
	artBlue := Card{UUID: "artblue", PromoTypes: []string{"alternateart", "blue"}}
	artGreen := Card{UUID: "artgreen", PromoTypes: []string{"alternateart", "green"}}
	cards := []Card{blue, green, art, artBlue, artGreen}

	for _, tt := range []struct {
		wording string
		want    []string
	}{
		// Naming both beats naming either, so the siblings sharing one tag
		// stand aside rather than aliasing the answer away.
		{"Alternate Art Blue", []string{"artblue"}},
		{"Alternate Art Green", []string{"artgreen"}},
		// Naming one keeps the printing wearing only that: the wording said
		// nothing about an alternate art and must not be read as asking for
		// one.
		{"Blue", []string{"blue"}},
		{"Alternate Art", []string{"art"}},
		// A tag named by nothing leaves the caller to its other tiers.
		{"Cold Foil", nil},
	} {
		var got []string
		for _, card := range DescribedVariants(tt.wording, cards) {
			got = append(got, card.UUID)
		}
		if len(got) != len(tt.want) {
			t.Errorf("DescribedVariants(%q) = %v, want %v", tt.wording, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("DescribedVariants(%q) = %v, want %v", tt.wording, got, tt.want)
				break
			}
		}
	}
}

// TestDescribedVariantsContainedLabel pins the third rule on the shape that
// forced it: a catalog writing the whole distinction as one label, so the
// printings wear one tag apiece and one tag contains the other whole. One
// Piece sells "Monkey.D.Luffy (Super Alternate Art)" beside the same card
// "(Red Super Alternate Art)"; both are named by a wording spelling the
// longer out, and counting tags cannot tell them apart.
func TestDescribedVariantsContainedLabel(t *testing.T) {
	superArt := Card{UUID: "superart", PromoTypes: []string{"superalternateart"}}
	redSuperArt := Card{UUID: "redsuperart", PromoTypes: []string{"redsuperalternateart"}}
	parallel := Card{UUID: "parallel", PromoTypes: []string{"parallel"}}
	pack := Card{UUID: "pack", PromoTypes: []string{"cs2023eventpack"}}
	finalist := Card{UUID: "finalist", PromoTypes: []string{"cs2023eventpackfinalistver"}}
	cards := []Card{superArt, redSuperArt, parallel, pack, finalist}

	for _, tt := range []struct {
		wording string
		want    []string
	}{
		// The longer label spells more of the wording out, so it answers a
		// wording that spelled it out rather than aliasing with the label
		// it contains.
		{"Red Super Alternate Art", []string{"redsuperart"}},
		{"CS 2023 Event Pack Finalist Ver.", []string{"finalist"}},
		// The shorter label still answers its own wording: the longer one
		// is not named by a wording that never spelled its extra words.
		{"Super Alternate Art", []string{"superart"}},
		{"CS 2023 Event Pack", []string{"pack"}},
		// A label sharing nothing with the wording is untouched by any of it.
		{"Parallel", []string{"parallel"}},
	} {
		var got []string
		for _, card := range DescribedVariants(tt.wording, cards) {
			got = append(got, card.UUID)
		}
		if len(got) != len(tt.want) {
			t.Errorf("DescribedVariants(%q) = %v, want %v", tt.wording, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("DescribedVariants(%q) = %v, want %v", tt.wording, got, tt.want)
				break
			}
		}
	}
}

// TestDescribedVariantsUnrelatedLabels pins the boundary of that rule. Two
// labels a wording names at once are only settled where one spells the other
// out; labels that merely differ stay a tie, for the caller to break with
// what it knows about its own catalog.
//
// Flesh and Blood writes the finish into the same wording as the variant, so
// "UPR043 Cold Foil" on a card whose name says "(Marvel)" names a cold foil
// label and a marvel one. Settling that by length hands it to the cold foil,
// and the Flesh and Blood matcher never gets to ask again without the words
// its finish already consumed - which is how it reaches the marvel.
func TestDescribedVariantsUnrelatedLabels(t *testing.T) {
	coldFoil := Card{UUID: "coldfoil", PromoTypes: []string{"coldfoil"}}
	marvel := Card{UUID: "marvel", PromoTypes: []string{"marvel"}}
	cards := []Card{coldFoil, marvel}

	got := DescribedVariants("UPR043 Cold Foil Marvel", cards)
	if len(got) != 2 {
		var names []string
		for _, card := range got {
			names = append(names, card.UUID)
		}
		t.Errorf("DescribedVariants named %v, want both labels left to the caller", names)
	}

	// Each still answers a wording that names it alone.
	for _, tt := range []struct {
		wording string
		want    string
	}{
		{"UPR043 Cold Foil", "coldfoil"},
		{"UPR043 Marvel", "marvel"},
	} {
		got := DescribedVariants(tt.wording, cards)
		if len(got) != 1 || got[0].UUID != tt.want {
			t.Errorf("DescribedVariants(%q) = %v, want %s", tt.wording, got, tt.want)
		}
	}
}

// TestDescribedVariantsSameLabel pins the other half of that boundary: two
// printings wearing the same label spell each other out, so containment says
// nothing about which the wording meant. Answering with either would settle
// by position what the labels do not settle at all - One Piece tells those
// apart by the edition they sit in, further down.
func TestDescribedVariantsSameLabel(t *testing.T) {
	first := Card{UUID: "first", PromoTypes: []string{"parallel"}}
	second := Card{UUID: "second", PromoTypes: []string{"parallel"}}

	got := DescribedVariants("Parallel", []Card{first, second})
	if len(got) != 2 {
		t.Errorf("DescribedVariants named %d printings, want both left to the caller", len(got))
	}
}
