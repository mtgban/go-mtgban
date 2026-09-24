package strikezone

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestProTourAndMediaShelf pins which set the Pro Tour and Media promo
// shelves land a name on when it sits in more than one. Both loops used to
// keep whichever set matched last, so a Regional PTQ retail listing always
// priced Snapcaster Mage at its unrelated Regional Championship Qualifiers
// 2023 printing, and any Media listing with a media-insert reprint
// overrode the wording's own San Diego Comic-Con set.
func TestProTourAndMediaShelf(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		desc, name, edition string
		wantSet, wantNumber string
	}{
		{
			desc: "a plain Pro Tour listing keeps PPRO",
			name: "Snapcaster Mage (2016 Regional PTQ)", edition: "Promos: Pro Tour",
			wantSet: "PPRO", wantNumber: "2016",
		},
		{
			desc: "wording naming the RCQ series takes PR23 instead",
			name: "Snapcaster Mage (Regional Championship Qualifiers 2023)", edition: "Promos: Pro Tour",
			wantSet: "PR23", wantNumber: "2",
		},
		{
			desc: "SDCC wording is left for the core SDCC rule, not PMEI",
			name: "Jace, Memory Adept (SDCC 2013 Exclusive)", edition: "Promos: Media",
			wantSet: "PSDC", wantNumber: "60★",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := preprocess(b, tt.name, tt.edition, "Foil")
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", tt.name, err)
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v", tt.name, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
				t.Errorf("%q got %s #%s, want %s #%s", tt.name, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}

// TestAutographedRefusal pins that a signed Secret Lair copy is refused
// rather than priced as the plain printing it shares a name with. The check
// happens before preprocess ever reads the backend, so a nil one does.
func TestAutographedRefusal(t *testing.T) {
	name := "Ajani Goldmane (745) (Autographed)"
	_, err := preprocess(nil, name, "Secret Lair", "Foil")
	if !errors.Is(err, mtgmatcher.ErrUnsupported) {
		t.Errorf("preprocess(%q) = %v, want ErrUnsupported", name, err)
	}
}
