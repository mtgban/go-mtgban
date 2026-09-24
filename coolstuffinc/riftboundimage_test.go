package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestRiftboundImageCardNamesThePrinting pins the printing a product image
// names, for the listings whose wording cannot say which one they are. The
// storefront writes the champion's subtitle into the product name where the
// catalog files every one of a champion's cards under the champion alone, so
// Vendetta's Akali 021 and 038 read alike and neither reached a card.
func TestRiftboundImageCardNamesThePrinting(t *testing.T) {
	b := readGameDatastore(t, "riftbound", "RIFTBOUND_PATH")

	const base = "https://res.cloudinary.com/csicdn/image/upload/" +
		"c_pad,fl_lossy,h_186,q_auto,w_186/v1/Images/Products/Misc%20Art/"

	for _, tt := range []struct {
		desc, img string
		foil      bool
		wantID    string
	}{
		// The two Akali cards a shelf sells side by side, told apart by
		// nothing the catalog writes.
		{"a champion's first card", "Vendetta/full/VEN021.jpg", true, "ven-021-166_foil"},
		{"a champion's second card", "Vendetta/full/VEN038.jpg", true, "ven-038-166_foil"},
		// The alternate art is a number of its own, behind a suffix the
		// storefront adds to the file name and the catalog does not.
		{"an alternate art", "Vendetta/full/VEN021aOVR.jpg", true, "ven-021a-166_foil"},
		// The number is padded to three digits here and not in the catalog.
		{"a padded number", "Spiritforged/full/SFD109.jpg", true, "sfd-109-221_foil"},
		// Origins names an image after the revision it is on, and files
		// some behind a letter or two of the storefront's own.
		{"an image revision", "Origins/full/OGN119v2.jpg", true, "ogn-119-298_foil"},
		{"a storefront prefix", "Origins/full/sOGN156_.jpg", true, "ogn-156-298_foil"},
		{"a two-letter prefix", "Origins/full/kdOGN247_.jpg", true, "ogn-247-298_foil"},
		// The oldest products are pictured by TCGplayer product id rather
		// than by sku, in two spellings.
		{"a TCGplayer id", "Origins/full/652842_in_700x.jpg", true, "ogn-066-298_foil"},
		{"a bare TCGplayer id", "Origins/full/m653158.jpg", false, "ogs-023-024"},
		// The extension is the storefront's to change.
		{"a shouted extension", "Vendetta/full/VEN021.JPG", true, "ven-021-166_foil"},
		// The signature suffix is a number, not noise: the same set
		// files an overnumbered printing of the same card at 224
		// without the star, at another price entirely.
		{"a signature printing", "Spiritforged/full/SFD224SIG.jpg", true, "sfd-224-star-221_foil"},
		// A print index repeats the card's own number with a tally tacked
		// on - the sku names one card sold under two images, not two cards.
		{"a print index", "Origins/full/OGN148_2.jpg", true, "ogn-148-298_foil"},
		// An underscore between the code and the number is not a print
		// index: the whole "_148" has to survive rather than being read
		// as a tally tacked onto an empty number.
		{"a code separated by an underscore", "Origins/full/OGN_148.jpg", true, "ogn-148-298_foil"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card := riftboundImageCard(b, base+tt.img, tt.foil)
			if card == nil {
				t.Fatalf("riftboundImageCard(%q) named no card", tt.img)
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

// TestRiftboundImageCardRefusesWhatItCannotName pins the refusals that keep
// the image a fallback rather than a guess: a number the set does not carry,
// a number that names more than one card, and a file name that is not a sku
// at all each answer nothing, so the listing stays refused rather than
// landing on a neighbour.
func TestRiftboundImageCardRefusesWhatItCannotName(t *testing.T) {
	b := readGameDatastore(t, "riftbound", "RIFTBOUND_PATH")

	const base = "https://res.cloudinary.com/csicdn/image/upload/v1/Images/Products/Misc%20Art/"

	for _, tt := range []struct{ desc, img string }{
		// The tokens are sold and the catalog carries no row for any of
		// them, so T01 reaches nothing in Unleashed.
		{"a token the catalog does not carry", "Unleashed/full/UNLT01.jpg"},
		// A suffix of the storefront's own that is not one of the two
		// this reads: the listing stays refused rather than landing on
		// the number without it.
		{"an unread suffix", "Origins/full/tOGN197as.jpg"},
		{"a number no set holds", "Vendetta/full/VEN999.jpg"},
		// The organized play promos file several cards at one number -
		// OPP-125 is both Lunar Boon and Hungry Wolf - and a number that
		// does not tell printings apart is no answer to a listing being
		// read for which printing it is.
		{"a number naming more than one card", "Promo/full/OPP125.jpg"},
		{"a name that is not a sku", "Promo/full/viktororistamp.jpg"},
		{"the placeholder image", "Vendetta/full/noimage_1.jpg"},
		{"no image at all", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			img := tt.img
			if img != "" {
				img = base + img
			}
			if card := riftboundImageCard(b, img, false); card != nil {
				t.Errorf("riftboundImageCard(%q) = %v, want none", tt.img, card)
			}
		})
	}
}
