package coolstuffinc

import (
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
