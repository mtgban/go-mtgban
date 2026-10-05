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
	"premium card collection - film red edition":     "Premium Card Collection ONE PIECE FILM RED Edition",
}

// onePiecePIDs names the TCGplayer product a storefront product is, for the
// listings whose wording cannot reach it: the storefront describes a DON!!
// card and a reprinted promo in prose the catalog never uses, and the catalog
// tells them apart by promo types and watermarks the storefront does not
// write. The product id is the storefront's own and both feeds share it.
var onePiecePIDs = map[string]string{
	"353757": "456059", // OP01 DON "The King of the Pirates" = OP01 Manga Alternate Art
	"357696": "482273", // OP02 DON "To Put An End To This War!!" = OP02 Manga
	"368617": "483145", // DON (Tournament Pack Vol. 2) participant, no stamp
	"368620": "482237", // DON Red (DON!! Card Pack Vol. 1)
	"368621": "483170", // DON Teal (Vol. 2)
	"368674": "482239", // DON Blue (Vol. 1)
	"368675": "483171", // DON Bronze (Vol. 2)
	"368676": "483167", // DON Green (Vol. 2)
	"371229": "483168", // DON Orange (Vol. 2)
	"371230": "483169", // DON Pink (Vol. 2)
	"371231": "482240", // DON Purple (Vol. 1)
	"371232": "482241", // DON Silver (Vol. 1)
	"371233": "482238", // DON Yellow (Vol. 1)
	"371821": "517476", // OP04 DON "Will you call me your Shipmate?!!" = OP04 Alternate Art
	"378041": "529792", // OP05 DON "Luffy Punching Kaido" = OP05 Alternate Art
	"384872": "541671", // OP06 DON "Zoro & Sanji" = OP06 Alternate Art
	"391902": "555894", // OP07 DON "Five Remaining Warlords" = OP07 Alternate Art
	"404440": "604246", // OP09 DON "Are you that afraid of the new era?!" = OP09 Alternate Art
	"417147": "636742", // OP11 DON "... They Believe In Me" = OP11 Alternate Art
	"422589": "646574", // OP12 DON "It's My Student's Farewell." = OP12 Alternate Art
	"433483": "672738", // OP14 DON (Sand Tornado) DP Vol. 9 = Crocodile
	"433484": "672736", // OP14 DON (Smile and Strings) DP Vol. 9 = Donquixote Doflamingo
	"440300": "698313", // OP16 DON "We'll Have To Break Out" = OP16 Alternate Art
	"440301": "698314", // OP16 DON same (GOLD) = Alternate Art Gold
	"447228": "712747", // OP17 DON Luffy = Alternate Art
	"447229": "710859", // OP17 DON Luffy (GOLD) = Alternate Art Gold
	"447230": "712748", // OP17 DON Luffy & Loki
	"447231": "710860", // OP17 DON Luffy & Loki (GOLD)
	"447232": "712750", // OP17 DON Four Emperors
	"447233": "711421", // OP17 DON Four Emperors (GOLD)
	"447234": "712749", // OP17 DON Xebec = Alternate Art Rocks
	"447235": "711420", // OP17 DON Xebec (BLUE) = Rocks Special Foil
	"447236": "715682", // OP17 DON Rocks.D.Xebec (Double Pack Vol. 12)
	"447636": "681871", // DON (Red Bull Don!!)
	"354630": "457032", // Shanks P-016 nonfoil = Film Red
	"354622": "457039", // Monkey.D.Luffy P-022 (Film Red) nonfoil
	"379043": "518702", // Monkey.D.Luffy P-055 nonfoil = Sealed Battle Kit Vol. 1
	"379048": "518696", // Usopp P-049 nonfoil = Sealed Battle Kit Vol. 1
	"387345": "537438", // Roronoa Zoro P-045 (OP06 Pre-Release Tournament) nonfoil = Participant
	"424023": "656166", // Sabo P-044 White Border = PRB-02 Reprint
	"424027": "656192", // Koala P-069 White Border = PRB-02 Reprint
	"424028": "656199", // Carrot P-070 White Border = PRB-02 Reprint
	"424029": "656209", // Sabo P-073 = PRB-02 Reprint
	"424030": "656212", // Portgas.D.Ace P-074 = PRB-02 Reprint
	"424031": "656216", // Monkey.D.Luffy P-075 = PRB-02 Reprint
	"424032": "656223", // Adio P-078 = PRB-02 plain
	"424033": "656229", // Lim P-079 = PRB-02 plain
	"424036": "656237", // Shanks P-083 = PRB-02 Reprint
	"424037": "656230", // Jewelry Bonney P-085 = PRB-02 Reprint
	"424038": "656220", // Trafalgar Law P-088 = PRB-02 Reprint
	"425300": "656177", // Jinbe P-063 = PRB-02 Reprint
	"384878": "541670", // Rebecca OP05-091 (SP) In Sunflower Field = OP06 SP
	"423973": "654571", // Rebecca OP05-091 (SP) Stitched Together Border = PRB-02 SP
}

// onePieceStarterDeck matches the starter deck a name states in brackets.
var onePieceStarterDeck = regexp.MustCompile(`\(Starter Deck (\d+)\)`)

// onePieceNotePlace is the finishing place a buy row's note names. The name
// carries the event and the note the place, and the catalog labels the
// printing by both: without the place, the plain card wins.
var onePieceNotePlace = regexp.MustCompile(`(?i)\b(?:participant|winner|finalist)\b`)

// onePieceSpellings spells the One Piece wording this storefront writes its
// own way: the Heroines Edition event card lost a word, the Event Pack card
// lost the dash before its number, and the eighth Winner Pack is the catalog's
// October to December one. Only the part that differs is rewritten, so the
// number or bracket behind it is kept.
var onePieceSpellings = strings.NewReplacer(
	"But If We See Each Other Again...Will You Call Me Your Shipmate?!!", "But If We Ever See Each Other Again... Will You Call Me Your Shipmate?!!",
	"Kouzuki Momonosuke P-064", "Kouzuki Momonosuke - P-064",
	"Winner Pack Vol. 8", "Winner Pack 2024 Oct.-Dec.",
)

// onePieceSpelling spells One Piece wording the way the catalog does.
func onePieceSpelling(wording string) string {
	return onePieceSpellings.Replace(wording)
}

// onePieceShelf answers the set a One Piece listing belongs to, which is the
// shelf it arrived on except where that shelf says only "Promo".
//
// A starter deck card reprinted as a promo is filed here under the promo
// shelf with the deck named in brackets, and the promo shelf holds a printing
// of its own at the same number: the P-041 Luffy is both the plain promo and
// the Starter Deck 18 card. The two meet, and the $0.50 deck card would be
// priced as the $60.00 promo standing beside it.
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
// card, and "" wherever the listing is already answered.
//
// This storefront calls the Gear5 starter deck's premium printing "Full Art"
// where the catalog files every alternate printing of that set as "Parallel",
// so the word named no label there and the row settled on the plain card - a
// $15.00 Monkey.D.Luffy priced as the $0.50 one beside it.
//
// The guard is what keeps it from touching a real Full Art. The word must
// name a label the catalog uses somewhere, so a typo reaches nothing; the
// card's own number must hold no printing of it, which is false for every
// real Full Art printing; and the number must hold a single alternate
// printing, so the one printing the storefront can mean is the one the
// number carries. A listing naming a second qualifier is not answered by the
// treatment alone, so it is left where it landed.
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
		if card.Number != co.Number || len(card.PromoTypes) == 0 {
			continue
		}
		if alternate != "" && alternate != card.UUID {
			return ""
		}
		alternate = card.UUID
		for _, promoType := range card.PromoTypes {
			labels[promoType] = true
		}
	}
	if alternate == "" {
		return ""
	}

	qualifiers := nameQualifierList(name)
	if len(qualifiers) != 1 {
		return ""
	}
	slug := mtgmatcher.PromoTypeSlug(qualifiers[0])
	if slug == "" || labels[slug] || !slices.Contains(b.AllPromoTypes, slug) {
		return ""
	}
	return alternate
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
