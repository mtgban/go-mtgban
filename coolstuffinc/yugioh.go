package coolstuffinc

import (
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// yugiohCodes spells the set codes the buylist writes unpadded the way the
// catalog does: the Dinosaurs Rage deck is SD9 here and SD09 there, and the
// third Hobby League is HL3 where the catalog says HL03. Both the Number and
// the note carry them.
var yugiohCodes = strings.NewReplacer(
	"SD9-SS1", "SD09-ENSS1",
	"SD9-EN", "SD09-EN",
	"HL3EN", "HL03-EN",
	"HL3-EN", "HL03-EN",
)

// yugiohImageCode matches the set code and number a Yu-Gi-Oh image sku
// carries: "BLMMEN053" is BLMM-EN053.
var yugiohImageCode = regexp.MustCompile(`^([A-Za-z0-9]{2,5}?)EN(\d{3})`)

// yugiohImageCard retries a refused buylist row with the code its own image
// sku carries in place of its Number, where the two disagree, and answers the
// id it lands on. The landing has to be the same card, so a sku naming
// another card's code reaches nothing.
func yugiohImageCard(b *mtgmatcher.Backend, card *mtgmatcher.InputCard, product CSIPriceEntry) string {
	match := yugiohImageCode.FindStringSubmatch(product.Image)
	if match == nil || product.Number == "" {
		return ""
	}
	code := match[1] + "-EN" + match[2]
	if strings.EqualFold(code, product.Number) {
		return ""
	}
	retried := *card
	retried.Variation = strings.Replace(card.Variation, product.Number, code, 1)
	id, err := b.Match(&retried)
	if err != nil {
		return ""
	}
	co, err := b.GetUUID(id)
	head, _, _ := strings.Cut(card.Name, " (")
	if err != nil || !mtgmatcher.Equals(co.Name, head) {
		return ""
	}
	return id
}
