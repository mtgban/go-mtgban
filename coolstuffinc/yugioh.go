package coolstuffinc

import (
	"errors"
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

// yugiohWording corrects the words a listing's name or note types the catalog
// cannot read. The matcher takes a bare number in the wording for the
// collector number, so "Version 1 - " and "4 Corners" are rewritten away from
// it; "No Stamp" says the opposite of the word the stamp promo type is read
// from; and GB1-001 is the number GBI-001 mistyped.
var yugiohWording = strings.NewReplacer(
	"Version 1 - ", "",
	"Version 2 - ", "",
	"Flowers in 4 Corners", "Flowers in Four Corners",
	"GB1-001", "GBI-001",
	"(No Stamp ", "(",
	" - No Stamp", "",
	"(Normal Art) Dragon Art", "A Alternate Art",
	"(Alt Art) Hero Art", "B HERO Art",
)

// yugiohNumbers holds the numbers the buylist mistypes, keyed by name and the
// number as typed. A blind replacement cannot hold them: LCGX-EN040 is the
// real number of another card.
var yugiohNumbers = map[string]string{
	"Ten Thousand Dragon|BLAR-EN093":     "BLAR-EN10K",
	"Gladiator Beast Secutor|LCGX-EN040": "LCGX-EN240",
}

// yugiohAlternateArts lists the Gold Letter alternate arts of Battles of
// Legend - Chapter 1 whose listing leaves "Alternate Art" out, keyed by name
// and edition. Both printings of these numbers are alternate arts, and only
// that word tells the Gold Letter one from the Silver.
var yugiohAlternateArts = map[string]bool{
	"Borrelsword Dragon|Battles of Legend - Chapter 1":                  true,
	"Number 39: Utopia (Astral Language)|Battles of Legend - Chapter 1": true,
}

// yugiohSkillCards names the Speed Duel skill cards a listing sells under the
// bare name, keyed by name and edition. The catalog files them as
// "<name> (Skill Card)", and the bare name finds another card of that name.
var yugiohSkillCards = map[string]string{
	"Call of the Haunted|Speed Duel: Arena of Lost Souls": "Call of the Haunted (Skill Card)",
	"Zombie Master|Speed Duel: Trials of the Kingdom":     "Zombie Master (Skill Card)",
}

// yugiohListing answers the card a Yu-Gi-Oh listing describes. The variation
// is the note on the sell listing and the number and note on the buylist, and
// notes is the note alone, which tells the print runs apart.
func yugiohListing(name, edition, notes, variation, rarity string, foil bool) *mtgmatcher.InputCard {
	listed := name + "|" + edition
	if skill, found := yugiohSkillCards[listed]; found {
		name = skill
	}
	variation = jpArtWording(yugiohWording.Replace(variation))
	if yugiohAlternateArts[listed] {
		variation += " Alternate Art"
	}
	return &mtgmatcher.InputCard{
		Name:      catalogColor(catalogSpelling(jpArtWording(yugiohWording.Replace(name)))),
		Edition:   printRunEdition(edition, notes),
		Variation: strings.TrimSpace(variation + " " + catalogRarity(rarity)),
		Foil:      foil,
	}
}

// yugiohBuylistVariation is buylistVariation with the number a vendor typo
// replaced and the codes spelled the catalog's way.
func yugiohBuylistVariation(product CSIPriceEntry) string {
	product.Number = yugiohBuylistNumber(product)
	return yugiohCodes.Replace(buylistVariation(product))
}

// yugiohBuylistNumber is a buylist row's Number as yugiohBuylistVariation
// writes it.
func yugiohBuylistNumber(product CSIPriceEntry) string {
	fixed, found := yugiohNumbers[product.Name+"|"+product.Number]
	if found {
		return fixed
	}
	return yugiohCodes.Replace(product.Number)
}

// yugiohArtNotes names the artwork a note describes as the catalog's
// qualifier for it. The Limited Pack World Championship 2026 prints an
// alternate art of five Secret Rares, and the note describes its border and
// lettering where the catalog names it Emblazoned.
var yugiohArtNotes = map[string]string{
	"Red Inner Border, Japanese Letters in Art": "Emblazoned Alternate Art",
}

// yugiohArtCard answers a listing whose note is one of yugiohArtNotes, or nil.
// The qualifier names exactly one printing of the number, so the bracket the
// name carries and the rarity beside it are left behind: either one makes the
// matcher read the Emblazoned Secret Rare of the same number instead.
func yugiohArtCard(name, edition, number, notes string, foil bool) *mtgmatcher.InputCard {
	qualifier, found := yugiohArtNotes[strings.TrimSpace(notes)]
	if !found {
		return nil
	}
	head, _, _ := strings.Cut(name, " (")
	return &mtgmatcher.InputCard{
		Name:      catalogColor(catalogSpelling(head)),
		Edition:   printRunEdition(edition, notes),
		Variation: strings.TrimSpace(number + " " + qualifier),
		Foil:      foil,
	}
}

// yugiohRetries reports whether a refused listing is asked again by its
// image, on either feed: not where the matcher skips it, and not where the
// note says it is one of several printings.
func yugiohRetries(err error, notes string) bool {
	return err != nil && !errors.Is(err, mtgmatcher.ErrUnsupported) && !csiAmbiguousNote.MatchString(notes)
}

// yugiohImageCode matches the set code and number a Yu-Gi-Oh image sku
// carries: "BLMMEN053" is BLMM-EN053.
var yugiohImageCode = regexp.MustCompile(`^([A-Za-z0-9]{2,5}?)EN(\d{3})`)

// yugiohImageCard retries a refused buylist row with the code its own image
// sku carries in place of its Number, as yugiohBuylistNumber spells it, where
// the two disagree, and answers the id it lands on. The landing has to be the
// same card, so a sku naming another card's code reaches nothing.
func yugiohImageCard(b *mtgmatcher.Backend, card *mtgmatcher.InputCard, product CSIPriceEntry) string {
	match := yugiohImageCode.FindStringSubmatch(product.Image)
	number := yugiohBuylistNumber(product)
	if match == nil || number == "" {
		return ""
	}
	code := match[1] + "-EN" + match[2]
	if strings.EqualFold(code, number) {
		return ""
	}
	return yugiohRetry(b, card, strings.Replace(card.Variation, number, code, 1), code)
}

// yugiohSKUCard retries a refused sell listing with the number its product
// image names, and answers the id it lands on. The image file is the code with
// its dash left out, so the number is the datastore's own that spells it:
// LEHDENA23 is LEHD-ENA23 and MRL103 is MRL-103, which no pattern could split.
// numbers is skuNumbers of the backend.
func yugiohSKUCard(b *mtgmatcher.Backend, numbers map[string][]string, card *mtgmatcher.InputCard, imgURL string) string {
	match := imageStem.FindStringSubmatch(imgURL)
	if match == nil {
		return ""
	}
	for _, number := range numbers[strings.ToUpper(match[1])] {
		id := yugiohRetry(b, card, number+" "+card.Variation, number)
		if id != "" {
			return id
		}
	}
	return ""
}

// yugiohRetry matches the card again with another variation and answers the id
// only if it is the printing at number and the same card, the catalog's deck
// letter "Monster Reborn (A)" aside.
func yugiohRetry(b *mtgmatcher.Backend, card *mtgmatcher.InputCard, variation, number string) string {
	retried := *card
	retried.Variation = variation
	id, err := b.Match(&retried)
	if err != nil {
		return ""
	}
	co, err := b.GetUUID(id)
	if err != nil || !strings.EqualFold(co.Number, number) {
		return ""
	}
	head, _, _ := strings.Cut(card.Name, " (")
	landed, _, _ := strings.Cut(co.Name, " (")
	if !mtgmatcher.Equals(landed, head) {
		return ""
	}
	return id
}
