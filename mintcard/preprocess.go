package mintcard

import (
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

func setCodeExists(b *mtgmatcher.Backend, code string) bool {
	_, err := b.GetSet(code)
	return err == nil
}

func nameExists(b *mtgmatcher.Backend, name string) bool {
	uuids, err := b.SearchEquals(name)
	return err == nil && len(uuids) > 0
}

var nameTable = map[string]string{
	"Godzilla, King of Monsters":             "Godzilla, King of the Monsters",
	"Tavern Ruffian // Tavern Champion":      "Tavern Ruffian // Tavern Smasher",
	"Faithbound Judge // Sinner's Judgement": "Faithbound Judge // Sinner's Judgment",
	"Mothra's Giant Cocoon":                  "Mothra's Great Cocoon",
	"rathi Berserker":                        "Aerathi Berserker",
}

var name2edition = map[string]string{
	"Serra Angel":  "PWOS",
	"Fiendish Duo": "PKHM",
}

// codeTable maps the set codes the storefront invents onto the datastore's
var codeTable = map[string]string{
	"PVC": "DDE",
}

func preprocess(b *mtgmatcher.Backend, cardName, number, finish, langauge, edition, setCode string) (*mtgmatcher.InputCard, error) {
	if setCode == "FWB" {
		return nil, mtgmatcher.ErrUnsupported
	}
	// The double-faced helper cards a booster carries are the datastore's
	// substitute cards, filed in a set of their own beside the one they
	// came in and numbered from one: "Helper Card (9/9)" of Kaldheim is
	// card 9 of SKHM. A helper card of a set the datastore files none for
	// stays an insert below.
	if index, found := strings.CutPrefix(cardName, "Helper Card ("); found && setCodeExists(b, "S"+setCode) {
		number, _, _ = strings.Cut(strings.TrimSuffix(index, ")"), "/")
		cardName = "Double-Faced Substitute Card"
		setCode = "S" + setCode
	}
	// The inserts a booster carries beside its cards, and the emblems the
	// datastore files with the tokens, have no printing of their own here.
	// A name the datastore carries whole is a card whatever it says:
	// Signature Slam and Emblem of the Warmind are cards.
	if !nameExists(b, cardName) && (strings.Contains(cardName, "Theme Card") ||
		strings.Contains(cardName, "Helper Card") ||
		strings.HasPrefix(cardName, "Emblem ") ||
		strings.Contains(cardName, "Signature")) {
		return nil, mtgmatcher.ErrUnsupported
	}
	if fixup, found := codeTable[setCode]; found {
		setCode = fixup
	}
	if strings.Count(cardName, "Token") > 1 {
		return preprocessTokenPair(b, cardName, finish, setCode)
	}
	if strings.Contains(cardName, "Complete") && strings.Contains(cardName, "Set") {
		return nil, mtgmatcher.ErrUnsupported
	}
	if strings.Contains(cardName, "Signed") {
		return nil, mtgmatcher.ErrUnsupported
	}
	if strings.Contains(cardName, "Graded") {
		return nil, mtgmatcher.ErrUnsupported
	}

	cardName = strings.Replace(cardName, ")(", ") (", -1)
	s := mtgmatcher.SplitVariants(cardName)
	cardName = s[0]
	variant := ""
	if len(s) > 1 {
		variant = strings.Join(s[1:], " ")
	}
	variant = strings.Replace(variant, "HP", "", -1)
	variant = strings.Replace(variant, "MP", "", -1)
	variant = strings.Replace(variant, "DMG", "", -1)
	variant = strings.Replace(variant, "Damaged", "", -1)
	variant = strings.TrimSpace(variant)

	fixup, found := nameTable[cardName]
	if found {
		cardName = fixup
	}
	// A promo printed under a flavor name is listed by that name with the
	// card's own in the first parenthetical: "Fatalism (Arcane Denial)"
	if len(s) > 1 && !nameExists(b, cardName) && nameExists(b, s[1]) {
		cardName = s[1]
		variant = strings.TrimSpace(strings.Join(s[2:], " "))
	}

	switch setCode {
	case "PMSC":
		fixup, found := name2edition[cardName]
		if found {
			edition = fixup
		}
	case "PMF":
		if mtgmatcher.IsBasicLand(cardName) {
			edition = "PF19"
		}
	case "STA":
		if strings.Contains(variant, "Collector Booster") {
			variant += " Etched"
		}
	case "MYS":
		if variant == "Commander" {
			variant = "Commander 2011"
		}
	case "SLD":
		// The shelf holds the drops, the convention promos and the
		// commander decks alike, and only the card says which
		edition = setCode
		if len(b.MatchInSet(cardName, "SLD")) == 0 && len(b.MatchInSet(cardName, "SLP")) > 0 {
			edition = "SLP"
		}
		if len(b.MatchInSet(cardName, "SLC")) == 1 {
			edition = "SLC"
			if len(b.MatchInSet(cardName, "SLD")) > 0 && mtgmatcher.ExtractYear(variant) == "" {
				edition = "SLD"
			}
		}
	default:
		if setCodeExists(b, setCode) {
			edition = setCode
		}
	}

	foil := strings.Contains(finish, "Foil") || strings.Contains(variant, "Foil")

	if strings.Contains(finish, "Prerelease") {
		variant += " Prerelease"
	}

	number = strings.TrimLeft(number, "0")
	if number != "" && len(b.MatchInSetNumber(cardName, setCode, number)) == 1 {
		variant += " " + number
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      foil,
		Language:  langauge,
	}, nil
}

// tokenPairNumber matches the per-face number a two-sided token's own name
// carries beside it. Most sets spell it "Bird Token (2/21) // Saproling
// Token (16/21)" - Bird is C16's own token #2, Saproling its #16, the
// denominator its sheet's own token count and no part of either real
// number - but some ("Elephant Token (006) // Insect Token (007)", DFT)
// give the number bare, zero-padded, with no denominator at all. Both
// forms confirmed against real listings to agree with mtgjson's own token
// numbering exactly.
var tokenPairNumber = regexp.MustCompile(`\((\d+)(?:/\d+)?\)`)

// missingSpaceBeforeParen matches some catalog rows (measured: Commander
// 2014/2015) writing the number's own parenthetical straight against
// "Token" with no space - "Spirit Token(22/24)" - which
// magic.StripFaceWrapping (shared with every other vendor's own token-pair
// anchor) only trims starting at " (", so it would otherwise leave the
// parenthetical attached and never reduce the face down to "Spirit".
var missingSpaceBeforeParen = regexp.MustCompile(`(\S)(\(\d+(?:/\d+)?\))`)

// preprocessTokenPair resolves a two-sided token listing - one whose own
// name carries "Token" twice, sku2uuid's own TCGplayer sku lookup already
// having found nothing for it - to the combined entity mtgmatcher/magic
// derives for it, anchored by the catalog's own set code (setCode is
// already the datastore's own code here, unlike the wording other vendors
// give an edition) and each face's own number.
func preprocessTokenPair(b *mtgmatcher.Backend, cardName, finish, setCode string) (*mtgmatcher.InputCard, error) {
	foil := strings.Contains(finish, "Foil")
	cardName = missingSpaceBeforeParen.ReplaceAllString(cardName, "$1 $2")

	tokenSet := setCode
	if resolved := magic.SetTokenSetCode(b, setCode); resolved != "" {
		tokenSet = resolved
	}

	for _, m := range tokenPairNumber.FindAllStringSubmatch(cardName, -1) {
		number := strings.TrimLeft(m[1], "0")
		if number == "" {
			continue
		}
		if uuid := magic.MatchNativeTokenPair(b, tokenSet, number, cardName); uuid != "" {
			if id, err := b.MatchID(uuid, foil); err == nil {
				return &mtgmatcher.InputCard{ID: id}, nil
			}
		}
		if tcgID := magic.MatchTokenPairingBySetNumber(b, tokenSet, number, cardName, foil); tcgID != "" {
			if id, err := b.MatchID(tcgID, foil); err == nil {
				return &mtgmatcher.InputCard{ID: id}, nil
			}
		}
	}

	return nil, mtgmatcher.ErrUnsupported
}
