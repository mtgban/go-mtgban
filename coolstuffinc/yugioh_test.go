package coolstuffinc

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestYugiohImageCardNeedsTheSameName pins the retry that reads the set code
// off the image sku where the Number field disagrees with it: the vendor
// mistypes a number now and then, and a sku naming another card's code must
// not answer for this one.
func TestYugiohImageCardNeedsTheSameName(t *testing.T) {
	b := readGameDatastore(t, "yugioh", "YUGIOH_PATH")

	product := CSIPriceEntry{
		Name: "Enlilgirsu, the Orcust Mekk-Knight (Ultra Rare)", ItemSet: "Battles of Legend - Monster Mayhem",
		Number: "BLMM-EN052", RarityName: "Ultra Rare", Image: "BLMMEN053",
	}
	card := &mtgmatcher.InputCard{Name: product.Name, Edition: product.ItemSet, Variation: "BLMM-EN052 Ultra Rare"}
	_, err := b.Match(card)
	if err == nil {
		t.Fatalf("Match(%v) landed as typed, want the typo to refuse", card)
	}
	id := yugiohImageCard(b, card, product)
	if id == "" {
		t.Fatalf("yugiohImageCard(%v) found nothing", card)
	}
	co, _ := b.GetUUID(id)
	if co.Number != "BLMM-EN053" {
		t.Errorf("landed %q (%s), want the card at BLMM-EN053", id, co.Number)
	}

	product.Image = "BLMMEN052SLR"
	id = yugiohImageCard(b, card, product)
	if id != "" {
		co, _ := b.GetUUID(id)
		t.Errorf("a sku naming another card's code landed %q (%s)", id, co.Name)
	}
}

// TestYugiohSKUCardReadsTheNumberOffTheImage pins the retry of a refused sell
// listing with the number its product image names, where the wording gives
// several printings of the card at once and the image file leaves the dash
// out of the number.
func TestYugiohSKUCardReadsTheNumberOffTheImage(t *testing.T) {
	b := readGameDatastore(t, "yugioh", "YUGIOH_PATH")

	const dir = "https://res.cloudinary.com/csicdn/image/upload/v1/Images/Products/YuGiOh%20Art/"
	for _, test := range []struct {
		name, edition, rarity, img, want string
	}{
		{"Serpent Night Dragon", "Magic Ruler", "Secret Rare", "Magic%20Ruler/full/MRL103.jpg", "mrl-103_22981_unlimited"},
		{"Monster Reborn", "Legendary Hero Decks", "Common", "Legendary%20Hero%20Decks/full/LEHDENA23.jpg", "lehd-ena23_177611_1stedition"},
		{"Monster Reborn", "Legendary Hero Decks", "Common", "Legendary%20Hero%20Decks/full/LEHDENC16.jpg", "lehd-enc16_177613_1stedition"},
		// An image naming another card's number answers nothing.
		{"Monster Reborn", "Legendary Hero Decks", "Common", "Magic%20Ruler/full/MRL103.jpg", ""},
	} {
		card := yugiohListing(test.name, test.edition, "", "", test.rarity, false)
		got := yugiohSKUCard(b, card, dir+test.img)
		if got != test.want {
			t.Errorf("%s (%s) landed %q, want %q", test.name, test.img, got, test.want)
		}
	}
}

// TestYugiohArtCardReachesTheEmblazonedPrinting pins the alternate art the
// storefront describes by its border and lettering.
func TestYugiohArtCardReachesTheEmblazonedPrinting(t *testing.T) {
	b := readGameDatastore(t, "yugioh", "YUGIOH_PATH")

	card := yugiohArtCard("Dracotail Faimena (Alternate Art Secret Rare)", "Limited Pack World Championship 2026", "26LP-EN002", "Red Inner Border, Japanese Letters in Art", false)
	if card == nil {
		t.Fatal("the note named no artwork")
	}
	id, err := b.Match(card)
	if err != nil {
		t.Fatalf("Match(%v) = %v", card, err)
	}
	co, _ := b.GetUUID(id)
	if !slices.Contains(co.PromoTypes, "emblazonedalternateart") {
		t.Errorf("landed %q (%v), want the emblazonedalternateart printing", id, co.PromoTypes)
	}
	if yugiohArtCard("Dracotail Faimena", "Limited Pack World Championship 2026", "26LP-EN002", "Reverse Foil", false) != nil {
		t.Error("a note naming no artwork was read as one")
	}
}

// TestYugiohListingReachesItsPrinting pins the listings whose wording the
// matcher reads as something else: a bare digit in a note as the collector
// number, the word Stamp inside "No Stamp" as the stamp promo type, a number
// the vendor mistyped, and a skill card or alternate art sold under the name
// and wording of another printing.
func TestYugiohListingReachesItsPrinting(t *testing.T) {
	b := readGameDatastore(t, "yugioh", "YUGIOH_PATH")

	tests := []struct {
		name, edition, notes, number, rarity string
		want                                 string
	}{
		{"Black Rose Dragon (Super Rare)", "Quarter Century Stampede", "Version 1 - Super Rare", "", "Super Rare", "ra04-en057_626965_1stedition"},
		{"Black Rose Dragon (Quarter Century Secret Rare)", "Quarter Century Stampede", "Version 2 - Quarter Century Secret Rare", "", "Quarter Century Secret Rare", "ra04-en057_626963_1stedition"},
		{"Ghost Sister & Spooky Dogwood", "Maximum Gold", "(Normal) Flowers in 4 Corners", "", "Premium Gold Rare", "mago-en013_227431_1stedition"},
		{"Slifer the Sky Dragon", "Promo", "GB1-001", "", "Ultra Rare", "gbi-001_25371_limited"},
		{"Raigeki (No Stamp Ultra Rare)", "Rarity Collection 5", "Ultra Rare - No Stamp", "", "Ultra Rare", "ra05-en110_689625_1stedition"},
		{"Vanquish Soul Razen (No Stamp Starlight Rare)", "Rarity Collection 5", "Starlight Rare - No Stamp", "", "Starlight Rare", "ra05-en134_689639_1stedition"},
		{"Polymerization (Secret Rare)", "Quarter Century Bonanza", "(Normal Art) Dragon Art - Secret Rare", "", "Secret Rare", "ra03-en051_593867_1stedition"},
		{"Polymerization [Alt Art] (Secret Rare)", "Quarter Century Bonanza", "(Alt Art) Hero Art - Secret Rare", "", "Secret Rare", "ra03-en051_592525_1stedition"},
		{"Ten Thousand Dragon", "Battles of Legend - Armageddon", "", "BLAR-EN093", "Secret Rare", "blar-en10k_218039_1stedition"},
		{"Gladiator Beast Secutor", "Legendary Collection 2", "", "LCGX-EN040", "Secret Rare", "lcgx-en240_56880_unlimited"},
		{"Borrelsword Dragon", "Battles of Legend - Chapter 1", "Gold Letter Ultra Rare", "", "Ultra Rare", "blc1-en023_538495_1stedition"},
		{"Call of the Haunted", "Speed Duel: Arena of Lost Souls", "", "", "Ultra Rare", "sbls-ens03_186714_1stedition"},
	}
	for _, test := range tests {
		variation := test.notes
		if test.number != "" {
			variation = yugiohBuylistVariation(CSIPriceEntry{Name: test.name, Number: test.number, Notes: test.notes})
		}
		card := yugiohListing(test.name, test.edition, test.notes, variation, test.rarity, false)
		id, err := b.Match(card)
		if err != nil {
			t.Errorf("Match(%v) = %v", card, err)
			continue
		}
		if id != test.want {
			t.Errorf("%s (%s) landed %q, want %q", test.name, test.notes, id, test.want)
		}
	}
}
