package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestRiftboundSKUCardReadsTheBuylistSKU pins that the sku the buylist
// feed hands over bare resolves the same way the one inside a sale
// listing's image url does. The feed's Image field is the same name, and
// the buylist carries the champion subtitles that reach no card just as
// the sale listings do.
func TestRiftboundSKUCardReadsTheBuylistSKU(t *testing.T) {
	b := readGameDatastore(t, "riftbound", "RIFTBOUND_PATH")

	for _, tt := range []struct {
		desc, stem string
		foil       bool
		wantID     string
	}{
		{"a plain sku", "VEN021", true, "ven-021-166_foil"},
		{"an alternate art", "UNL147aOVR", true, "unl-147a-219_foil"},
		// The nine signature printings the storefront buys are all
		// filed at a starred number, and the number without the star is
		// the overnumbered sibling of the same card.
		{"a signature printing", "SFD224SIG", true, "sfd-224-star-221_foil"},
		{"a signature printing, lowercased", "unl233sig", true, "unl-233-star-219_foil"},
		{"a bare TCGplayer id", "m653158", false, "ogs-023-024"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card := riftboundSKUCard(b, tt.stem, tt.foil)
			if card == nil {
				t.Fatalf("riftboundSKUCard(%q) named no card", tt.stem)
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%v) = %v", card, err)
			}
			if id != tt.wantID {
				co, _ := b.GetUUID(id)
				t.Errorf("Match(%v) = %q (%s), want %q", card, id, co, tt.wantID)
			}
		})
	}
}

// TestRiftboundSKUCardSeparatesTheSignature pins the printing the
// signature suffix must not be confused with. Dropping the suffix rather
// than reading it answers the overnumbered printing the set files at the
// same number, which is another card entirely at another price - the
// storefront buys the signature Aphelios at $450.
func TestRiftboundSKUCardSeparatesTheSignature(t *testing.T) {
	b := readGameDatastore(t, "riftbound", "RIFTBOUND_PATH")

	signature := riftboundSKUCard(b, "SFD224SIG", true)
	overnumbered := riftboundSKUCard(b, "SFD224", true)
	if signature == nil || overnumbered == nil {
		t.Fatal("one of the two printings named no card")
	}

	signatureID, err := b.Match(signature)
	if err != nil {
		t.Fatalf("Match(%v) = %v", signature, err)
	}
	overnumberedID, err := b.Match(overnumbered)
	if err != nil {
		t.Fatalf("Match(%v) = %v", overnumbered, err)
	}
	if signatureID == overnumberedID {
		t.Errorf("both skus answered %q; the suffix told two printings apart", signatureID)
	}
}

// TestRiftboundSKUCardRefusesWhatItCannotName pins that a feed row with
// no sku of its own answers nothing rather than guessing.
func TestRiftboundSKUCardRefusesWhatItCannotName(t *testing.T) {
	b := readGameDatastore(t, "riftbound", "RIFTBOUND_PATH")

	for _, tt := range []struct{ desc, stem string }{
		{"no sku at all", ""},
		{"the storefront's own logo", "RiftboundLogo"},
		{"a token the catalog does not carry", "UNLT02"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if card := riftboundSKUCard(b, tt.stem, false); card != nil {
				t.Errorf("riftboundSKUCard(%q) = %v, want none", tt.stem, card)
			}
		})
	}
}
