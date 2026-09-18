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
