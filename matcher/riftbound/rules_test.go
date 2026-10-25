package riftbound

import (
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

func TestQualifiedBaseNameFollowsCurrentGallery(t *testing.T) {
	const (
		baseUUID = "ogn-066-298"
		altUUID  = "ogn-066a-298"
	)
	b := &matcher.Backend{
		CanonicalNames: map[string]string{
			matcher.Normalize("Ahri"): "Ahri",
		},
		Hashes: map[string][]string{
			matcher.Normalize("Ahri"): {baseUUID, altUUID},
		},
		UUIDs: map[string]*matcher.CardObject{
			baseUUID: {Card: matcher.Card{Name: "Ahri", Number: "66"}},
			altUUID:  {Card: matcher.Card{Name: "Ahri", Number: "66a"}},
		},
	}

	in := matcher.InputCard{Name: "Ahri, Alluring", Variation: "066a"}
	(Rules{}).AdjustName(b, &in)
	if in.Name != "Ahri" {
		t.Fatalf("AdjustName changed %q to %q, want Ahri", "Ahri, Alluring", in.Name)
	}

	in = matcher.InputCard{Name: "Ahri, Alluring", Variation: "227*"}
	(Rules{}).AdjustName(b, &in)
	if in.Name != "Ahri, Alluring" {
		t.Fatalf("AdjustName accepted a number absent from the base name: %q", in.Name)
	}
}

// TestQualifiedBaseNameRetriesUnhashedDashedName covers a dash-joined name
// that names no card at all - not even a wrong one. promoOnlyName answers
// false for an empty bucket, so the same "not promo-only" test that gates
// the promo case must not block this one, or the split is skipped whenever
// nothing is hashed under the compound name.
// A collector number is what earns the retry: it is the only thing tying
// an unrecognized dashed name back to a real champion's own printing.
func TestQualifiedBaseNameRetriesUnhashedDashedName(t *testing.T) {
	const baseUUID = "ogn-009-298"
	b := &matcher.Backend{
		CanonicalNames: map[string]string{
			matcher.Normalize("Champion"): "Champion",
		},
		Hashes: map[string][]string{
			matcher.Normalize("Champion"): {baseUUID},
		},
		UUIDs: map[string]*matcher.CardObject{
			baseUUID: {Card: matcher.Card{Name: "Champion", Number: "9"}},
		},
	}

	if got := qualifiedBaseName(b, "Champion - Title", "9"); got != "Champion" {
		t.Fatalf("qualifiedBaseName(%q) = %q, want %q", "Champion - Title", got, "Champion")
	}

	// No number to anchor the retry: the name stays unmatched.
	if got := qualifiedBaseName(b, "Champion - Title", ""); got != "" {
		t.Fatalf("qualifiedBaseName with no number = %q, want \"\"", got)
	}
}

func TestPrefilterReaimsPromoQualifiedCurrentName(t *testing.T) {
	for _, tt := range []struct {
		name string
		base string
		want string
	}{
		{name: "Ahri, Alluring", base: "Ahri", want: "Ahri"},
		{name: "Master Yi - Meditative", base: "Master Yi", want: "Master Yi"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const promoUUID = "promo"
			b := &matcher.Backend{
				CanonicalNames: map[string]string{
					matcher.Normalize(tt.name): tt.name,
					matcher.Normalize(tt.base): tt.base,
				},
				Hashes: map[string][]string{
					matcher.Normalize(tt.name): {promoUUID},
					matcher.Normalize(tt.base): {"base"},
				},
				UUIDs: map[string]*matcher.CardObject{
					promoUUID: {Card: matcher.Card{SetCode: "PR", Number: "66"}},
					"base":    {Card: matcher.Card{SetCode: "OGN", Number: "66"}},
				},
				Sets: map[string]*matcher.Set{
					"PR":  {Code: "PR", Type: "promo"},
					"OGN": {Code: "OGN"},
				},
			}
			in := matcher.InputCard{Name: tt.name, Variation: "66"}
			(Rules{}).Prefilter(b, &in)
			if in.Name != tt.want {
				t.Fatalf("Prefilter changed %q to %q, want %q", tt.name, in.Name, tt.want)
			}
		})
	}
}

func TestPrefilterKeepsNumberFromPromoQualifiedName(t *testing.T) {
	const (
		name     = "Ahri, Alluring"
		promoID  = "promo"
		baseName = "Ahri"
	)
	b := &matcher.Backend{
		CanonicalNames: map[string]string{
			matcher.Normalize(name):     name,
			matcher.Normalize(baseName): baseName,
		},
		Hashes: map[string][]string{
			matcher.Normalize(name):     {promoID},
			matcher.Normalize(baseName): {"base"},
		},
		UUIDs: map[string]*matcher.CardObject{
			promoID: {Card: matcher.Card{SetCode: "PR", Number: "66"}},
			"base":  {Card: matcher.Card{SetCode: "OGN", Number: "66"}},
		},
		Sets: map[string]*matcher.Set{
			"PR":  {Code: "PR", Type: "promo"},
			"OGN": {Code: "OGN"},
		},
	}
	in := matcher.InputCard{Name: name}
	(Rules{}).Prefilter(b, &in)
	if in.Name != baseName || in.Variation != "66" {
		t.Fatalf("Prefilter produced name=%q variation=%q, want %q %q", in.Name, in.Variation, baseName, "66")
	}

	in = matcher.InputCard{Name: name, Variation: "Alternate Art"}
	(Rules{}).Prefilter(b, &in)
	if in.Name != baseName || in.Variation != "66 Alternate Art" {
		t.Fatalf("Prefilter produced name=%q variation=%q, want %q %q", in.Name, in.Variation, baseName, "66 Alternate Art")
	}
}

func TestPrefilterKeepsPromoQualifiedNameForPromoEdition(t *testing.T) {
	const (
		name     = "Ahri, Alluring"
		baseName = "Ahri"
	)
	b := &matcher.Backend{
		CanonicalNames: map[string]string{
			matcher.Normalize(name):     name,
			matcher.Normalize(baseName): baseName,
		},
		Hashes: map[string][]string{
			matcher.Normalize(name):     {"promo"},
			matcher.Normalize(baseName): {"base"},
		},
		UUIDs: map[string]*matcher.CardObject{
			"promo": {Card: matcher.Card{SetCode: "PR", Number: "66"}},
			"base":  {Card: matcher.Card{SetCode: "OGN", Number: "66"}},
		},
		Sets: map[string]*matcher.Set{
			"PR":  {Code: "PR", Type: "promo"},
			"OGN": {Code: "OGN"},
		},
	}
	in := matcher.InputCard{Name: name, Variation: "66", Edition: "Promos"}
	(Rules{}).Prefilter(b, &in)
	if in.Name != name || in.Variation != "66" {
		t.Fatalf("Prefilter changed promo input to name=%q variation=%q", in.Name, in.Variation)
	}
}

func TestPrefilterLeavesUnknownDashedNameAlone(t *testing.T) {
	const baseName = "Dark Child"
	b := &matcher.Backend{
		CanonicalNames: map[string]string{matcher.Normalize(baseName): baseName},
		Hashes:         map[string][]string{matcher.Normalize(baseName): {"base"}},
		UUIDs: map[string]*matcher.CardObject{
			"base": {Card: matcher.Card{SetCode: "OGN", Number: "66"}},
		},
		Sets: map[string]*matcher.Set{"OGN": {Code: "OGN"}},
	}
	in := matcher.InputCard{Name: "Dark Child - Starter", Variation: "66"}
	(Rules{}).Prefilter(b, &in)
	if in.Name != "Dark Child - Starter" || in.Variation != "66" {
		t.Fatalf("Prefilter changed dashed input to name=%q variation=%q", in.Name, in.Variation)
	}
}

// TestPrefilterRetriesWhenDirectMatchConflictsWithEdition covers a name that
// is not promo-only - so the ordinary promo-qualified-name retry never
// triggers - but whose direct canonical match still names the wrong card: a
// storefront's "Champion - Title" spelling normalizes the same as an
// unrelated "Champion, Title" name that happens to be real (Ivern's "Green
// Father" against Secret Garden's "Ivern, Green Father" is the live case).
// The edition is the tiebreaker: a direct match with no printing in the
// stated edition is the signal to retry as the bare title instead.
func TestPrefilterRetriesWhenDirectMatchConflictsWithEdition(t *testing.T) {
	const (
		wrongUUID = "wrong"
		rightUUID = "right"
	)
	b := &matcher.Backend{
		CanonicalNames: map[string]string{
			matcher.Normalize("Champion, Title"): "Champion, Title",
			matcher.Normalize("Title"):           "Title",
		},
		Hashes: map[string][]string{
			matcher.Normalize("Champion, Title"): {wrongUUID},
			matcher.Normalize("Title"):           {rightUUID},
		},
		UUIDs: map[string]*matcher.CardObject{
			wrongUUID: {Card: matcher.Card{Name: "Champion, Title", SetCode: "OPP", Number: "9"}},
			rightUUID: {Card: matcher.Card{Name: "Title", SetCode: "UNL", Number: "9"}},
		},
		Sets: map[string]*matcher.Set{
			"OPP": {Code: "OPP"},
			"UNL": {Code: "UNL"},
		},
	}

	// The direct match ("Champion, Title") has no printing in the stated
	// edition (UNL); retry as the bare title, which does.
	in := matcher.InputCard{Name: "Champion - Title", Variation: "9", Edition: "UNL"}
	(Rules{}).Prefilter(b, &in)
	if in.Name != "Title" {
		t.Fatalf("Prefilter kept the edition-conflicting direct match, got %q, want %q", in.Name, "Title")
	}

	// The direct match's own edition is left alone: no retry, no rename.
	in = matcher.InputCard{Name: "Champion - Title", Variation: "9", Edition: "OPP"}
	(Rules{}).Prefilter(b, &in)
	if in.Name != "Champion - Title" {
		t.Fatalf("Prefilter changed a non-conflicting direct match to %q", in.Name)
	}

	// No edition at all proves no conflict; the direct match stands.
	in = matcher.InputCard{Name: "Champion - Title", Variation: "9"}
	(Rules{}).Prefilter(b, &in)
	if in.Name != "Champion - Title" {
		t.Fatalf("Prefilter changed an edition-less input to %q", in.Name)
	}
}

func TestCanonicalGalleryNameRestoresVendettaRecruitQualifier(t *testing.T) {
	card := GalleryCard{}
	card.SetCode = "VEN"
	card.Number = "T04"
	card.Name = "Recruit"
	if got := canonicalGalleryName(card); got != "Recruit (NX)" {
		t.Fatalf("canonicalGalleryName() = %q, want Recruit (NX)", got)
	}

	card.SetCode = "OGN"
	if got := canonicalGalleryName(card); got != "Recruit" {
		t.Fatalf("canonicalGalleryName() changed non-Vendetta card to %q", got)
	}
}
