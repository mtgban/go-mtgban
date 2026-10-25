package cardgamecorner

import (
	"path"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The storefront's own names for the games it sells.
const (
	GameMagic         = "magic"
	GamePokemon       = "pokemon"
	GameYuGiOh        = "yugioh"
	GameOnePiece      = "onepiece"
	GameLorcana       = "lorcana"
	GameRiftbound     = "riftbound"
	GameFleshAndBlood = "fleshandblood"
)

// cgcGames is what NewScraper is built through: it names the storefront's
// game for each of ours, and a game named nowhere here is not one Card Game
// Corner is read for.
var cgcGames = map[mtgmatcher.Game]string{
	mtgmatcher.GameMagic:         GameMagic,
	mtgmatcher.GamePokemon:       GamePokemon,
	mtgmatcher.GameYuGiOh:        GameYuGiOh,
	mtgmatcher.GameOnePiece:      GameOnePiece,
	mtgmatcher.GameLorcana:       GameLorcana,
	mtgmatcher.GameRiftbound:     GameRiftbound,
	mtgmatcher.GameFleshAndBlood: GameFleshAndBlood,
}

// shelfSets names the set a game files the printings of a store shelf
// under, where the shelf's own name points at no single set. One Piece's
// promo shelves sell event printings, which share their number with the
// booster card they reprint; Pokemon's "Base" is the unlimited Base Set, not
// its Shadowless run; Yu-Gi-Oh's 25th anniversary and date reprints share
// theirs with the original's prints.
var shelfSets = map[mtgmatcher.Game]map[string]string{
	mtgmatcher.GameOnePiece: {
		"Promos":                     "OP-PR",
		"Unnumbered Promos":          "OP-PR",
		"Premium Bandai Promo":       "OP-PR",
		"Special Tournaments Promos": "OP-PR",
	},
	mtgmatcher.GamePokemon: {
		"Base": "BS",
	},
	mtgmatcher.GameYuGiOh: {
		"Legend of Blue Eyes White Dragon REPRINT 25th Anniversary Edition": "LOB-EN-23050",
		"Metal Raiders REPRINT 25th Anniversary Edition":                    "MRD-EN-23052",
		"Spell Ruler REPRINT 25th Anniversary Edition":                      "SRL-EN-23053",
		"Pharaoh’s Servant REPRINT 25th Anniversary Edition":                "PSV-EN-23054",
		"Dark Crisis REPRINT 25th Anniversary Edition":                      "DCR-EN-23055",
		"Invasion of Chaos REPRINT 25th Anniversary Edition":                "IOC-EN-23056",
		"Retro Pack 2 Reprint 2025":                                         "RP02-24352",
		"Light of Destruction - 2024 Reprint Unlimited":                     "LODT-23579",
	},
}

// promoNumberPrefixes is the code each Pokemon Black Star promo shelf leaves
// off its numbers, which the catalog numbers "SWSH011" where the storefront
// writes "011".
var promoNumberPrefixes = map[string]string{
	"BW Black Star Promos":              "BW",
	"XY Black Star Promos":              "XY",
	"Sun & Moon: Black Star Promos":     "SM",
	"Sword & Shield: Black Star Promos": "SWSH",
}

// gameNames holds the card names the store misspells, by game, as the
// datastore spells them, and one Yu-Gi-Oh name it gives in Italian whose
// image carries no English one.
var gameNames = map[mtgmatcher.Game]map[string]string{
	mtgmatcher.GameYuGiOh: {
		"Abyss Actors' Dress Rehersal":          "Abyss Actors' Dress Rehearsal",
		"Aphophis the Swamp Deity":              "Apophis the Swamp Deity",
		"Battlin‚Äô Boxer Lead Yolk":            "Battlin' Boxer Lead Yoke",
		"Carnot the Eternal Maching":            "Carnot the Eternal Machine",
		"Damage Vaccine O MAX":                  "Damage Vaccine Omega MAX",
		"Exstellarknight Constellar Ptolemy O7": "Exstellarknight Constellar Ptolemy Omega 7",
		"Falchionß":                             "Falchion Beta",
		"Goblin Biker Big Cabonga":              "Goblin Biker Big Gabonga",
		"Gold Pride - Chariot Carrier":          "Gold Pride - Chariot Carrie",
		"K9-ØØ Lupis":                           "K9-00 Lupis",
		"Majespecter Orthus - Nue":              "Majespecter Orthrus - Nue",
		"Maliss Q HEARTS OF CRYPTER":            "Maliss Q Hearts Crypter",
		"Outer Entity Azathoth":                 "Outer Entity Azathot",
		"Raffica di Vento":                      "Gust",
		"Reincarnation Unvealing Mail":          "Reincarnation Unveiling Mail",
		"Sky Striker Ace - Azelea Temperance":   "Sky Striker Ace - Azalea Temperance",
		"Supernatural Regeneration":             "Metaphysical Regeneration",
		"Terrors of the Overrroot":              "Terrors of the Overroot",
		"The Virtuos Vestals":                   "The Virtuous Vestals",
		"Urubonus, the Avatar of the Malice":    "Urubonus, the Avatar of Malice",
		"Xyz Posiedon Splash":                   "Xyz Poseidon Splash",
	},
	mtgmatcher.GameLorcana: {
		"Balu - Lively Bear": "Baloo - Fun-Loving Bear",
		"Miss Bianca - International Rescue Aid Society Age": "Miss Bianca - Rescue Aid Society Agent",
	},
	mtgmatcher.GameRiftbound: {
		"Corina Veraza":        "Corinna Veraza",
		"lascylla":             "Iascylla",
		"Vi, Piltover ZHorcer": "Vi, Piltover Enforcer",
	},
}

// vendorName returns the name a misspelled one stands for, keeping the
// version the store may have written after it.
func vendorName(game mtgmatcher.Game, name string) string {
	head, tail, hasTail := strings.Cut(name, " (")
	fixed, found := gameNames[game][head]
	if !found {
		return name
	}
	if hasTail {
		return fixed + " (" + tail
	}
	return fixed
}

// riftboundVersionRe matches the "(V.2 - Showcase)" a Riftbound name ends in,
// which the number already says: a showcase is "193a" beside the rare's
// "193".
var riftboundVersionRe = regexp.MustCompile(`\s*\(V\.[0-9]+ - [^)]*\)$`)

// oldShelfRe matches the name of a Yu-Gi-Oh shelf holding the first prints
// of an old set, which ends in the set's code.
var oldShelfRe = regexp.MustCompile(`ENG \(([A-Z0-9]+)\)$`)

// oldShelfSet returns the set an old Yu-Gi-Oh shelf holds. The North
// American and the European print of a card are both filed in it, the
// European one as "European <rarity>".
func oldShelfSet(edition string) (string, bool) {
	match := oldShelfRe.FindStringSubmatch(edition)
	if match == nil {
		return "", false
	}
	return match[1], true
}

// knownOldShelf returns the set oldShelfSet reads off a shelf, where the
// datastore knows it.
func knownOldShelf(b *mtgmatcher.Backend, edition string) (string, bool) {
	code, found := oldShelfSet(edition)
	if !found {
		return "", false
	}
	_, err := b.GetSet(code)
	return code, err == nil
}

// slugName reads the card an image is named after, where the store gave the
// name in Italian and the image directory holds the English one.
func slugName(b *mtgmatcher.Backend, extra string) string {
	slug := strings.ReplaceAll(path.Base(path.Dir(extra)), "-", " ")
	if !namesACard(b, slug) {
		return ""
	}
	return slug
}

// losesFirstEdition reports whether a first edition listing lands on a
// printing that is not one: the matcher falls back to the unlimited print
// where the datastore holds no first edition of the card.
func losesFirstEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	probe := *inCard
	cardID, err := b.Match(&probe)
	if err != nil {
		return false
	}
	co, err := b.GetUUID(cardID)
	if err != nil {
		return false
	}
	return co.Finish != mtgmatcher.FinishSlug(mtgmatcher.Run1stEdition)
}

// preprocessGame reads a card of any game but Magic, whose listings name the
// card, its set and its collector number plainly.
func preprocessGame(b *mtgmatcher.Backend, card *Card, index int) (*mtgmatcher.InputCard, error) {
	// Graded slabs are sold on shelves of their own
	if strings.Contains(card.Edition, "Gradat") {
		return nil, mtgmatcher.ErrUnsupported
	}

	inCard := gameInput(b.Game, card, index)
	code, found := shelfSets[b.Game][card.Edition]
	oldShelf := b.Game == mtgmatcher.GameYuGiOh && oldShelfRe.MatchString(card.Edition)
	if oldShelf {
		if !namesACard(b, inCard.Name) {
			name := slugName(b, card.Extra)
			if name != "" {
				inCard.Name = name
			}
		}
		if !found {
			code, found = knownOldShelf(b, card.Edition)
		}
	}
	if found {
		set, err := b.GetSet(code)
		if err != nil {
			return nil, err
		}
		inCard.Edition = set.Name
	}

	// A copy of a first edition the datastore lacks, the European ones
	// still, would be priced as the unlimited it lands on
	if oldShelf && inCard.Finish == mtgmatcher.Run1stEdition && losesFirstEdition(b, inCard) {
		return nil, mtgmatcher.ErrUnsupported
	}
	return inCard, nil
}

// gameInput reads a listing's wording the way the game's rules expect it.
func gameInput(game mtgmatcher.Game, card *Card, index int) *mtgmatcher.InputCard {
	name := card.Name
	number := card.Number
	variant := card.Variants[index]
	foil := variant.Foil == "Foil"
	var rainbow bool
	switch game {
	case mtgmatcher.GameFleshAndBlood:
		// Every name ends in its finish, which hides the pitch before it
		// from the rules, and is the only place a rainbow foil says so
		name, rainbow = strings.CutSuffix(name, " (Rainbow Foil)")
		name = strings.TrimSuffix(name, " (Normal)")
		foil = foil || rainbow
		// A double-sided card is numbered "001/002" with its faces in
		// ascending order, which the catalog does not keep, and both
		// faces are in the name
		if strings.Contains(name, " // ") {
			number = ""
		}
	case mtgmatcher.GameYuGiOh:
		// A number holds the card at several rarities, which the name
		// spells only from its second version on
		number = strings.TrimSpace(number + " " + card.Rarity)
		// The European print, an old shelf's other one, numbers its
		// cards on its own
		if oldShelfRe.MatchString(card.Edition) && variant.Language == "EN" {
			number = "European " + card.Rarity
		}
	case mtgmatcher.GameRiftbound:
		name = riftboundVersionRe.ReplaceAllString(name, "")
	case mtgmatcher.GamePokemon:
		// The storefront spells the δ of a Delta Species card as a "d"
		// before the words it stands for, which the catalog files as the
		// variant of the plain name instead
		name = strings.TrimSuffix(strings.TrimSpace(name), " d Delta Species")
		prefix, found := promoNumberPrefixes[card.Edition]
		if found && !strings.HasPrefix(number, prefix) {
			number = prefix + number
		}
	}
	return &mtgmatcher.InputCard{
		Name:      vendorName(game, name),
		Edition:   card.Edition,
		Variation: number,
		Foil:      foil,
		Finish:    gameFinish(game, card, variant, rainbow),
	}
}

// gameFinish names the finish a listing prices, for the games whose
// printings the foil flag cannot tell apart: a print run, and Pokemon's two
// holo treatments.
func gameFinish(game mtgmatcher.Game, card *Card, variant Variant, rainbow bool) string {
	firstEdition := variant.Print == "First Edition"
	switch game {
	case mtgmatcher.GameYuGiOh:
		if firstEdition {
			return "1st Edition"
		}
	case mtgmatcher.GamePokemon:
		if variant.Foil == "Reverse Holo" {
			return "Reverse Holofoil"
		}
		var treatment string
		if card.Rarity == "Rare Holo" {
			treatment = "Holofoil"
		}
		if firstEdition {
			return mtgmatcher.PrintingFinish("1st Edition", treatment)
		}
		return treatment
	case mtgmatcher.GameFleshAndBlood:
		treatment := "Normal"
		if rainbow {
			treatment = "Rainbow Foil"
		}
		switch {
		case firstEdition:
			return mtgmatcher.PrintingFinish("1st Edition", treatment)
		case variant.Print == "Unl.":
			return mtgmatcher.PrintingFinish("Unlimited Edition", treatment)
		case rainbow:
			return treatment
		}
	}
	return ""
}
