package riftbound

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestQualifiedBaseNameFollowsCurrentGallery(t *testing.T) {
	const (
		baseUUID = "ogn-066-298"
		altUUID  = "ogn-066a-298"
	)
	b := &mtgmatcher.Backend{
		CanonicalNames: map[string]string{
			mtgmatcher.Normalize("Ahri"): "Ahri",
		},
		Hashes: map[string][]string{
			mtgmatcher.Normalize("Ahri"): {baseUUID, altUUID},
		},
		UUIDs: map[string]*mtgmatcher.CardObject{
			baseUUID: {Card: mtgmatcher.Card{Name: "Ahri", Number: "66"}},
			altUUID:  {Card: mtgmatcher.Card{Name: "Ahri", Number: "66a"}},
		},
	}

	in := mtgmatcher.InputCard{Name: "Ahri, Alluring", Variation: "066a"}
	(Rules{}).AdjustName(b, &in)
	if in.Name != "Ahri" {
		t.Fatalf("AdjustName changed %q to %q, want Ahri", "Ahri, Alluring", in.Name)
	}

	in = mtgmatcher.InputCard{Name: "Ahri, Alluring", Variation: "227*"}
	(Rules{}).AdjustName(b, &in)
	if in.Name != "Ahri, Alluring" {
		t.Fatalf("AdjustName accepted a number absent from the base name: %q", in.Name)
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
			b := &mtgmatcher.Backend{
				CanonicalNames: map[string]string{
					mtgmatcher.Normalize(tt.name): tt.name,
					mtgmatcher.Normalize(tt.base): tt.base,
				},
				Hashes: map[string][]string{
					mtgmatcher.Normalize(tt.name): {promoUUID},
					mtgmatcher.Normalize(tt.base): {"base"},
				},
				UUIDs: map[string]*mtgmatcher.CardObject{
					promoUUID: {Card: mtgmatcher.Card{SetCode: "PR", Number: "66"}},
					"base":    {Card: mtgmatcher.Card{SetCode: "OGN", Number: "66"}},
				},
				Sets: map[string]*mtgmatcher.Set{
					"PR":  {Code: "PR", Type: "promo"},
					"OGN": {Code: "OGN"},
				},
			}
			in := mtgmatcher.InputCard{Name: tt.name, Variation: "66"}
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
	b := &mtgmatcher.Backend{
		CanonicalNames: map[string]string{
			mtgmatcher.Normalize(name):     name,
			mtgmatcher.Normalize(baseName): baseName,
		},
		Hashes: map[string][]string{
			mtgmatcher.Normalize(name):     {promoID},
			mtgmatcher.Normalize(baseName): {"base"},
		},
		UUIDs: map[string]*mtgmatcher.CardObject{
			promoID: {Card: mtgmatcher.Card{SetCode: "PR", Number: "66"}},
			"base":  {Card: mtgmatcher.Card{SetCode: "OGN", Number: "66"}},
		},
		Sets: map[string]*mtgmatcher.Set{
			"PR":  {Code: "PR", Type: "promo"},
			"OGN": {Code: "OGN"},
		},
	}
	in := mtgmatcher.InputCard{Name: name}
	(Rules{}).Prefilter(b, &in)
	if in.Name != baseName || in.Variation != "66" {
		t.Fatalf("Prefilter produced name=%q variation=%q, want %q %q", in.Name, in.Variation, baseName, "66")
	}

	in = mtgmatcher.InputCard{Name: name, Variation: "Alternate Art"}
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
	b := &mtgmatcher.Backend{
		CanonicalNames: map[string]string{
			mtgmatcher.Normalize(name):     name,
			mtgmatcher.Normalize(baseName): baseName,
		},
		Hashes: map[string][]string{
			mtgmatcher.Normalize(name):     {"promo"},
			mtgmatcher.Normalize(baseName): {"base"},
		},
		UUIDs: map[string]*mtgmatcher.CardObject{
			"promo": {Card: mtgmatcher.Card{SetCode: "PR", Number: "66"}},
			"base":  {Card: mtgmatcher.Card{SetCode: "OGN", Number: "66"}},
		},
		Sets: map[string]*mtgmatcher.Set{
			"PR":  {Code: "PR", Type: "promo"},
			"OGN": {Code: "OGN"},
		},
	}
	in := mtgmatcher.InputCard{Name: name, Variation: "66", Edition: "Promos"}
	(Rules{}).Prefilter(b, &in)
	if in.Name != name || in.Variation != "66" {
		t.Fatalf("Prefilter changed promo input to name=%q variation=%q", in.Name, in.Variation)
	}
}

func TestPrefilterLeavesUnknownDashedNameAlone(t *testing.T) {
	const baseName = "Dark Child"
	b := &mtgmatcher.Backend{
		CanonicalNames: map[string]string{mtgmatcher.Normalize(baseName): baseName},
		Hashes:         map[string][]string{mtgmatcher.Normalize(baseName): {"base"}},
		UUIDs: map[string]*mtgmatcher.CardObject{
			"base": {Card: mtgmatcher.Card{SetCode: "OGN", Number: "66"}},
		},
		Sets: map[string]*mtgmatcher.Set{"OGN": {Code: "OGN"}},
	}
	in := mtgmatcher.InputCard{Name: "Dark Child - Starter", Variation: "66"}
	(Rules{}).Prefilter(b, &in)
	if in.Name != "Dark Child - Starter" || in.Variation != "66" {
		t.Fatalf("Prefilter changed dashed input to name=%q variation=%q", in.Name, in.Variation)
	}
}

func TestCanonicalGalleryNameRestoresVendettaRecruitQualifier(t *testing.T) {
	card := GalleryCard{}
	card.Set.Value.ID = "VEN"
	card.CollectorNumber = 4
	card.Name = "Recruit"
	if got := canonicalGalleryName(card); got != "Recruit (NX)" {
		t.Fatalf("canonicalGalleryName() = %q, want Recruit (NX)", got)
	}

	card.Set.Value.ID = "OGN"
	if got := canonicalGalleryName(card); got != "Recruit" {
		t.Fatalf("canonicalGalleryName() changed non-Vendetta card to %q", got)
	}
}
