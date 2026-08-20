package yugioh

import (
	"slices"
	"testing"
)

// TestPromoTagsAndQualifiedNames pins what tells sibling products apart.
// Four Duelist League printings of Dark Magician share the number
// DL11-EN001, the rarity Rare and the print run, and only the color
// separates them; the site prints a qualifier only when the backend
// declares it, so both halves have to hold.
func TestPromoTagsAndQualifiedNames(t *testing.T) {
	b := loadBackend(t)

	for _, tag := range []string{"purple", "alternateart", "duelterminal", "otsstamp"} {
		if !slices.Contains(b.AllPromoTypes, tag) {
			t.Errorf("promo type %q is not declared, so nothing will print it", tag)
		}
	}

	// The rarity is the axis most sibling products differ on, but it is a
	// field of its own that the card already carries: as a tag it would
	// declare a third of the catalog promotional.
	for _, rarity := range []string{"common", "ultrarare", "quartercenturysecretrare"} {
		if slices.Contains(b.AllPromoTypes, rarity) {
			t.Errorf("rarity %q is declared as a promo type", rarity)
		}
	}

	bare, err := b.SearchEquals("Dark Magician")
	if err != nil {
		t.Fatal(err)
	}
	if len(bare) < 2 {
		t.Errorf("bare name reached %d printings, expected every printing of the name", len(bare))
	}

	for _, tt := range []struct{ query, promoType string }{
		{"Dark Magician (Purple)", "purple"},
		{"Dark Magician (Green)", "green"},
	} {
		got, err := b.SearchEquals(tt.query)
		if err != nil {
			t.Errorf("%q: %v", tt.query, err)
			continue
		}
		if len(got) != 1 {
			t.Errorf("%q reached %d printings, want the one it names", tt.query, len(got))
			continue
		}
		co, err := b.GetUUID(got[0])
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(co.PromoTypes, tt.promoType) {
			t.Errorf("%q reached a printing tagged %v, want one carrying %q", tt.query, co.PromoTypes, tt.promoType)
		}
	}

	// A spelling the datastore already carries as a name of its own keeps
	// only its own printings: "Dark Magician Girl (A)" is the YGLD card's
	// name, and the RA03 printing that files the same "A" as a label beside
	// the plain name must not pour into its bucket.
	named, err := b.SearchEquals("Dark Magician Girl (A)")
	if err != nil {
		t.Fatal(err)
	}
	for _, uuid := range named {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.Name != "Dark Magician Girl (A)" {
			t.Errorf("the name's bucket reached %q from %s, want only the card of that name", co.Name, co.SetCode)
		}
	}
}
