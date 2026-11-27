package toamagic

import (
	"cmp"
	"errors"
	"regexp"
	"slices"
	"strings"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/matcher/magic"
	"github.com/mtgban/go-mtgban/matcher/sealed"
)

var cardTable = map[string]string{
	"Scornful Ã†ther-Lich": "Scornful Aether-Lich",

	"Invasion of Belenon // Belenon War Antherm": "Invasion of Belenon // Belenon War Anthem",
	"Invasion of Moag // Bloomweaver Dryads":     "Invasion of Moag // Bloomwielder Dryads",
	"Shrouded Shepherd // Cleave Shadws":         "Shrouded Shepherd // Cleave Shadows",

	"Arlinn, Embraces by the Moon": "Arlinn, Embraced by the Moon",
}

var promoTags = []string{
	"2016 Welcome Deck",
	"Foil",
	"Godzilla Lands",
}

// foilTreatments name a finish with the word Foil in it, which the matcher
// reads whole.
var foilTreatments = []string{
	"First Place Foil",
	"Mana Foil",
}

// doubleSided closes the planeswalker half of a two-sided emblem listing,
// "Oko, Shadowmoor Scion Double-Sided Token".
var doubleSided = regexp.MustCompile(`(?i)\s*doubled?-sided token`)

// faceNumber is the collector number a face of a two-sided emblem may be
// listed with, "Eldrazi (0001)".
var faceNumber = regexp.MustCompile(`\s*\(\d+\)`)

func preprocess(b *matcher.Backend, cardName, edition, variant string) (*matcher.InputCard, error) {
	if cardName == "Emblem" && strings.Contains(variant, " // ") ||
		strings.HasSuffix(cardName, " // Emblem") {
		return preprocessEmblemPair(b, cardName, edition, variant)
	}

	s := matcher.SplitVariants(cardName)
	cardName = s[0]
	if len(s) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(s[1:], " ")
	}

	for _, tag := range promoTags {
		if strings.HasSuffix(cardName, tag) {
			if cardName == "Foil" {
				continue
			}
			cardName = strings.TrimSuffix(cardName, " "+tag)
			if variant != "" {
				variant += " "
			}
			variant += tag
		}
	}

	variant = strings.ReplaceAll(variant, "First-Place", "First Place")
	isFoil := strings.Contains(variant, "Foil")
	namesFoil := slices.ContainsFunc(foilTreatments, func(t string) bool {
		return strings.Contains(variant, t)
	})
	if isFoil && !namesFoil {
		variant = strings.Replace(variant, "Foil", "", 1)
		variant = strings.TrimSpace(variant)
	}

	// The store files an emblem as "Emblem - <planeswalker>", which the
	// datastore names "<planeswalker> Emblem".
	if cardName == "Emblem" && variant != "" {
		cardName, variant = emblemName(b, edition, variant), ""
	}

	switch edition {
	case "Portal":
		if variant == "1" {
			variant = "Flavor Text"
		} else if variant == "2" {
			variant = "Reminder Text"
		}
	case "The Brothers' War":
		if strings.Contains(variant, "Retro") {
			edition = "BRR"
		}
	case "Duskmourn: House of Horror":
		// A legend's showcase is its double exposure; the textured showcase
		// says so.
		if strings.Contains(variant, "Showcase") && !strings.Contains(variant, "Textured") &&
			hasDoubleExposure(b, cardName) {
			variant = strings.Replace(variant, "Showcase", "Borderless", 1)
		}
	case "Mystery Booster Cards":
		// A listing naming its original set, "Mortify (C18)", says which,
		// unless it names it by a code the datastore does not use ("DVD").
		number := plstNumber(b, cardName, variant)
		if number == "" {
			number = mb1PLSTNumber(b, cardName, isFoil)
		}
		if number != "" {
			edition = "PLST"
			variant = number
		}
	case "Archenemy Nicol Bolas":
		if cardName == "Highland Lake" || cardName == "Submerged Boneyard" {
			return nil, errors.New("does not exist")
		}
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	return &matcher.InputCard{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}

// preprocessEmblemPair lands a two-sided emblem on the pairing the datastore
// derives for it. The store lists one as "Emblem - <planeswalker> // <token>
// Double-Sided Token" or "<token> // Emblem - <planeswalker> Double-Sided
// Token".
func preprocessEmblemPair(b *matcher.Backend, cardName, edition, variant string) (*matcher.InputCard, error) {
	isFoil := strings.HasSuffix(variant, " Foil")
	variant = strings.TrimSuffix(variant, " Foil")
	variant = doubleSided.ReplaceAllString(variant, "")
	cardName = faceNumber.ReplaceAllString(cardName, "")
	variant = faceNumber.ReplaceAllString(variant, "")

	var pair string
	if cardName == "Emblem" {
		walker, token, _ := strings.Cut(variant, " // ")
		// "Zombie Token Blue" tells apart two tokens of one name.
		token, _, _ = strings.Cut(token, " Token")
		pair = emblemName(b, edition, walker) + " // " + token
	} else {
		token := strings.TrimSuffix(cardName, " // Emblem")
		pair = token + " // " + emblemName(b, edition, variant)
	}

	pairID := magic.MatchTokenPairingByNamesAndEdition(b, pair, edition, isFoil)
	if pairID != "" {
		id, err := b.MatchID(pairID, isFoil)
		if err == nil {
			return &matcher.InputCard{ID: id, Foil: isFoil}, nil
		}
	}
	return nil, matcher.ErrUnsupported
}

// emblemName names the emblem of the planeswalker the store calls walker,
// often by a single name ("Gideon", "Nixilis"), the way the edition's token
// set does, or walker's own emblem when the set holds no single match.
func emblemName(b *matcher.Backend, edition, walker string) string {
	walker = cmp.Or(cardTable[walker], walker)
	name := walker + " Emblem"
	tokenSet := magic.EditionTokenSetCode(b, edition)
	var found []string
	for _, id := range b.GetUUIDsInSet(tokenSet) {
		co, err := b.GetUUID(id)
		if err != nil {
			continue
		}
		if co.Name == name {
			return name
		}
		isEmblem := strings.HasSuffix(co.Name, " Emblem") && !strings.Contains(co.Name, " // ")
		if isEmblem && strings.Contains(co.Name, walker) && !slices.Contains(found, co.Name) {
			found = append(found, co.Name)
		}
	}
	if len(found) == 1 {
		return found[0]
	}
	return name
}

// hasDoubleExposure reports whether Duskmourn prints cardName with a double
// exposure, which only its foil names.
func hasDoubleExposure(b *matcher.Backend, cardName string) bool {
	ids, _ := b.SearchEquals(cardName)
	for _, id := range ids {
		co, err := b.GetUUID(id)
		if err == nil && co.SetCode == "DSK" && slices.Contains(co.PromoTypes, magic.PromoTypeDoubleExposure) {
			return true
		}
	}
	return false
}

// plstNumber answers the PLST printing of cardName reprinted from the set
// coded code, or "".
func plstNumber(b *matcher.Backend, cardName, code string) string {
	if code == "" {
		return ""
	}
	for _, card := range b.MatchInSet(cardName, "PLST") {
		if strings.HasPrefix(card.Number, code+"-") {
			return card.Number
		}
	}
	return ""
}

// mb1PLSTBooster is the Mystery Booster sealed product mb1PLSTNumber reads
// booster contents from.
const mb1PLSTBooster = "Mystery Booster Booster Pack (Retail Edition)"

// mb1PLSTNumber answers which PLST printing Mystery Booster's own booster
// bundles for cardName in the listing's finish, or "" when the booster names
// none sold in that finish.
func mb1PLSTNumber(b *matcher.Backend, cardName string, isFoil bool) string {
	finish := matcher.FinishNonfoil
	if isFoil {
		finish = matcher.FinishFoil
	}
	for _, uuid := range b.GetSealedUUIDsInSet("MB1") {
		co, err := b.GetUUID(uuid)
		if err != nil || co.Name != mb1PLSTBooster {
			continue
		}

		probs, err := sealed.ProductCounts(b, "MB1", uuid)
		if err != nil {
			return ""
		}
		for _, p := range probs {
			card, err := b.GetUUID(p.UUID)
			if err == nil && card.SetCode == "PLST" && card.Name == cardName && card.HasFinish(finish) {
				return card.Number
			}
		}
		return ""
	}
	return ""
}
