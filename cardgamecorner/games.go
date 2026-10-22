package cardgamecorner

import (
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
// its Shadowless run; Yu-Gi-Oh's 25th anniversary reprints share theirs with
// both of the original's regional prints.
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

// riftboundVersionRe matches the "(V.2 - Showcase)" a Riftbound name ends in,
// which the number already says: a showcase is "193a" beside the rare's
// "193".
var riftboundVersionRe = regexp.MustCompile(`\s*\(V\.[0-9]+ - [^)]*\)$`)

// preprocessGame reads a card of any game but Magic, whose listings name the
// card, its set and its collector number plainly.
func preprocessGame(b *mtgmatcher.Backend, card *Card, index int) (*mtgmatcher.InputCard, error) {
	// Graded slabs are sold on shelves of their own
	if strings.Contains(card.Edition, "Gradat") {
		return nil, mtgmatcher.ErrUnsupported
	}

	inCard := gameInput(b.Game, card, index)
	code, found := shelfSets[b.Game][card.Edition]
	if found {
		set, err := b.GetSet(code)
		if err != nil {
			return nil, err
		}
		inCard.Edition = set.Name
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
	case mtgmatcher.GameYuGiOh:
		// A number holds the card at several rarities, which the name
		// spells only from its second version on
		number = strings.TrimSpace(number + " " + card.Rarity)
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
		Name:      name,
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
