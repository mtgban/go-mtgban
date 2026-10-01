package coolstuffinc

import (
	"regexp"
	"slices"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// onePieceEvents spells a One Piece event the way the catalog names it, for
// the names this storefront gives it instead. They are its own: the catalog
// sells the card in "BANDAI Card Games Fest 25-26" and it goes up here as
// the Afro Luffy promo, after the art rather than the pack. A nickname
// names one product and nothing else, which is why they are listed one at a
// time rather than read for.
var onePieceEvents = map[string]string{
	"afro luffy promo":   "BANDAI Card Games Fest 25-26",
	"l.a. dodgers promo": "Dodgers x One Piece",
	// The playmat and the participation pack name the product the card came
	// in, where the catalog names the event that handed it out.
	"bcgf playmat promo":                             "Official Playmat -Bandai Card Games Fest 24-25 Edition-",
	"offline regional participation pack 2024 vol.2": "Offline Regional 2024 Vol. 2 Participant",
}

// onePieceStarterDeck matches the starter deck a name states in brackets.
var onePieceStarterDeck = regexp.MustCompile(`\(Starter Deck (\d+)\)`)

// onePieceSpellings spells the One Piece names this storefront writes its
// own way: the Heroines Edition event card lost a word.
var onePieceSpellings = map[string]string{
	"But If We See Each Other Again...Will You Call Me Your Shipmate?!!": "But If We Ever See Each Other Again... Will You Call Me Your Shipmate?!!",
}

// onePieceSpelling spells a One Piece name the way the catalog does.
func onePieceSpelling(name string) string {
	if spelled, found := onePieceSpellings[name]; found {
		return spelled
	}
	return name
}

// onePieceShelf answers the set a One Piece listing belongs to, which is the
// shelf it arrived on except where that shelf says only "Promo".
//
// A starter deck card reprinted as a promo is filed here under the promo
// shelf with the deck named in brackets, and the promo shelf holds a printing
// of its own at the same number: the P-041 Luffy is both the plain promo and
// the Starter Deck 18 card. The two met, and a $0.50 deck card was priced as
// the $60.00 promo standing beside it.
//
// The bracket only decides where the shelf has nothing to say. Every other
// listing naming a deck arrives on a real set already - a Backlight on ST11,
// a Kuzan on OP12 - and there the shelf is what the catalog agrees with,
// while the bracket names the deck the card was reprinted from.
func onePieceShelf(shelf, name string) string {
	if shelf != "Promo" {
		return shelf
	}
	match := onePieceStarterDeck.FindStringSubmatch(name)
	if match == nil {
		return shelf
	}
	return "Starter Deck " + match[1]
}

// onePieceRenamedTreatment answers the printing a One Piece listing means
// when it names its treatment with a word the catalog does not use for that
// set, and "" wherever the listing is already answered.
//
// This storefront calls the Gear5 starter deck's premium printing "Full Art"
// where the catalog files every alternate printing of that set as "Parallel",
// so the word named no label there and the row settled on the plain card - a
// $15.00 Monkey.D.Luffy priced as the $0.50 one beside it.
//
// The guard is what keeps it from touching a real Full Art. The word must
// name a label the catalog uses somewhere, so a typo reaches nothing; the
// card's own set must hold no printing of it, which is false for all 75 real
// Full Art printings, since their sets are the ones that use the name; and
// the set must wear a single premium label throughout, so the one printing
// the storefront can mean is the one the number carries.
func onePieceRenamedTreatment(b *mtgmatcher.Backend, id, name string) string {
	co, err := b.GetUUID(id)
	if err != nil || len(co.PromoTypes) > 0 {
		return ""
	}
	set, err := b.GetSet(co.SetCode)
	if err != nil {
		return ""
	}

	labels := map[string]bool{}
	var alternate string
	for _, card := range set.Cards {
		for _, promoType := range card.PromoTypes {
			labels[promoType] = true
		}
		if card.Number == co.Number && len(card.PromoTypes) > 0 {
			if alternate != "" && alternate != card.UUID {
				return ""
			}
			alternate = card.UUID
		}
	}
	if len(labels) != 1 || alternate == "" {
		return ""
	}

	for _, match := range nameParenthetical.FindAllStringSubmatch(name, -1) {
		slug := mtgmatcher.PromoTypeSlug(strings.TrimSpace(match[1]))
		if slug == "" || labels[slug] || !slices.Contains(b.AllPromoTypes, slug) {
			continue
		}
		return alternate
	}
	return ""
}

// eventNamed adds the catalog's name for every event the wording gives its
// own name to. The storefront's words stay: they are what the listing says
// about the art, and the catalog's name is only what files it.
func eventNamed(wording string) string {
	lower := strings.ToLower(wording)
	for nickname, event := range onePieceEvents {
		if strings.Contains(lower, nickname) {
			wording += " " + event
		}
	}
	return wording
}

// onePieceDonHead matches the wrapping the storefront hangs in front of
// a DON!! card's description, numbered or not.
var onePieceDonHead = regexp.MustCompile(`(?i)^DON!!\s*(?:\([0-9]+\))?\s*-?\s*`)

// onePieceDonCharacters spells the four characters this storefront
// names in full where the catalog names them by the name the promo type
// carries. They are not misspellings and not this storefront's
// invention - both names are the character's - so a listing saying
// "Edward Newgate" is describing the card the catalog files under
// "whitebeard", and neither wording reaches the other on its own.
//
// Read as a second attempt, never as a correction. Some promo types
// carry the full name themselves - the first anniversary's DON!! is
// "monkeydluffy1st" - so rewriting every listing would lose the cards
// whose catalog wording the storefront already matched. Only the four
// the storefront actually differs on are here, and a character the
// catalog spells differently again stays refused: every DON!! in a set
// shares a name and a number, so a guess buys another card.
var onePieceDonCharacters = strings.NewReplacer(
	"Edward Newgate", "Whitebeard",
	"Charlotte Linlin", "Big Mom",
	"Monkey.D.Luffy", "Luffy",
	"Portgas.D.Ace", "Ace",
)

// onePieceDonName answers a DON!! listing the way the catalog files it,
// and reports whether the listing is one at all.
//
// The catalog names all 238 of the game's DON!! cards "DON!! Card" and
// tells them apart by promo type alone - the character, the artwork,
// the border - which the matcher already reads out of a listing's own
// wording. The storefront writes those same words into the product name
// and publishes no collector number for a DON!! at all, so the name
// reaches nothing and the wording never gets as far as the rules that
// would have read it. Hand over the name the catalog uses and leave the
// description where a description belongs.
func onePieceDonName(name string) (string, string, bool) {
	if !strings.HasPrefix(strings.ToUpper(name), "DON!!") {
		return "", "", false
	}
	return "DON!! Card", onePieceDonHead.ReplaceAllString(name, ""), true
}

// onePieceDonRenamed answers the same description with the characters
// the catalog names differently swapped in, and an empty string when it
// names none of them.
func onePieceDonRenamed(description string) string {
	renamed := onePieceDonCharacters.Replace(description)
	if renamed == description {
		return ""
	}
	return renamed
}
