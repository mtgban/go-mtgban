package gamenerdz

import (
	"errors"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// preprocess turns a storefront product into the matcher's input, in the
// grammar its game's display names follow. The finish always comes from the
// selectedFinish field, which names the printing in each game's own words;
// the display name only restates it as a tail on the set.
func preprocess(product GNProduct, game string) (*mtgmatcher.InputCard, error) {
	switch game {
	case GameLorcana:
		return preprocessLorcana(product)
	case GamePokemon:
		return preprocessPokemon(product)
	case GameOnePiece:
		return preprocessOnePiece(product)
	case GameFleshAndBlood:
		return preprocessFleshAndBlood(product)
	}
	return preprocessMagic(product)
}

// magicCode is the set-and-number tag Magic display names carry, like
// "(TLE-265)" in "Aang, Air Nomad (0265) (TLE-265) - ...": always the last
// such tag, after any bare display number the name also shows.
var magicCode = regexp.MustCompile(`\(([0-9A-Z]{2,6})-([0-9A-Za-z]+)\)`)

// A Magic display name reads
//
//	Aang's Shelter - Teferi's Protection (Borderless) (TLE-007) - Avatar...
//
// The name stops at the first parenthesis: the collector number pins the
// printing, so variant wording like (Borderless) only restates it.
func preprocessMagic(product GNProduct) (*mtgmatcher.InputCard, error) {
	matches := magicCode.FindAllStringSubmatch(product.DisplayName, -1)
	if matches == nil {
		return nil, errors.New("no set code in display name")
	}
	number := matches[len(matches)-1][2]

	cardName := product.DisplayName
	if idx := strings.Index(cardName, " ("); idx != -1 {
		cardName = cardName[:idx]
	}

	edition := string(product.ProductData.Set)
	if edition == "" {
		edition = product.ProductData.SetName
	}

	card := &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: number,
		Foil:      strings.EqualFold(product.SelectedFinish, "foil") || nameSaysFoil(product.DisplayName),
	}

	// The storefront files every prerelease stamp under one pseudo-set,
	// whatever set the card was printed for. That code names no set the
	// catalog has, so the edition selects nothing and the number answers
	// alone - with the ordinary printing, the one the stamp is not. The
	// number is the card's own, so naming the stamp is enough to reach the
	// prerelease printing without ever knowing which set it belongs to.
	// The catalog answering is what says the shelf was right: a handful of
	// ordinary cards sit on it by mistake, and no stamp to reach is what
	// tells them apart, so they keep the reading they already had.
	if strings.EqualFold(edition, preShelf) {
		stamped := strings.TrimSpace(number + " " + preStamp)
		probe := mtgmatcher.InputCard{
			Name:      cardName,
			Edition:   edition,
			Variation: stamped,
			Foil:      card.Foil,
		}
		_, err := mtgmatcher.Match(&probe)
		if err == nil {
			card.Variation = stamped
		}
	}

	return card, nil
}

// The set code the storefront shelves every prerelease stamp under, and the
// word the catalog tells those printings apart by.
const (
	preShelf = "pre"
	preStamp = "Prerelease"
)

// bracketed is anything this storefront brackets into a display name: the
// set-and-number tag, the variant wording, the grade it tails a played copy
// with. What is left is the shelf the product is sold from, which ends in
// the finish.
var bracketed = regexp.MustCompile(`\s*\([^)]*\)`)

// nameSaysFoil reads the finish off the name a product is sold under, which
// the finish field beside it sometimes contradicts. Only this direction is
// worth reading: a name spelling the finish is a statement, while a name
// that does not is silence - Aether Revolt's Ajani, Valiant Protector is
// printed in foil only and its name never says so - so the name adds a
// finish the field left off and never takes one away.
//
// Reading the name whole would not do: a set can be called a foil edition
// without the product being one, so only the finish the shelf ends in
// counts.
func nameSaysFoil(displayName string) bool {
	shelf := strings.TrimSpace(bracketed.ReplaceAllString(displayName, ""))
	return strings.HasSuffix(shelf, "Foil")
}

// lorcanaNumber is the collector group Lorcana display names carry, like
// "(17/204)": the printing's own number over the set size the matcher does
// not need.
var lorcanaNumber = regexp.MustCompile(`\((\d+[a-z]?)/\d+\)`)

// A Lorcana display name reads
//
//	4*Town - Hottest Band of the Year (17/204) - Attack of the Vine
//
// The dash is part of the card's own name-and-subtitle, so the name only
// stops at the number's parenthesis.
func preprocessLorcana(product GNProduct) (*mtgmatcher.InputCard, error) {
	loc := lorcanaNumber.FindStringSubmatchIndex(product.DisplayName)
	if loc == nil {
		return nil, errors.New("no collector number in display name")
	}

	finish := product.SelectedFinish
	if strings.EqualFold(finish, "Normal") {
		finish = ""
	}

	return &mtgmatcher.InputCard{
		Name:      strings.TrimSpace(product.DisplayName[:loc[0]]),
		Edition:   product.ProductData.SetName,
		Variation: product.DisplayName[loc[2]:loc[3]],
		Finish:    finish,
		Foil:      finish != "",
	}, nil
}

// pokemonNumber is the collector number Pokemon display names carry inline,
// like "63", "65/130" or "SWSH197", set apart by spaces rather than
// parentheses.
var pokemonNumber = regexp.MustCompile(` ([A-Z]{0,5}\d+[a-zA-Z]?(?:/\d+)?) `)

// A Pokemon display name reads
//
//	Abra 65/130 - Base Set 2 Reverse Holofoil
//
// The name is what stands before the number; the finish travels in its own
// field for this game, where the matcher tells Holofoil from Reverse
// Holofoil by wording.
func preprocessPokemon(product GNProduct) (*mtgmatcher.InputCard, error) {
	loc := pokemonNumber.FindStringSubmatchIndex(product.DisplayName)
	if loc == nil {
		return nil, errors.New("no collector number in display name")
	}

	finish := product.SelectedFinish
	if strings.EqualFold(finish, "Normal") {
		finish = ""
	}

	return &mtgmatcher.InputCard{
		Name:      strings.TrimSpace(product.DisplayName[:loc[0]]),
		Edition:   product.ProductData.SetName,
		Variation: product.DisplayName[loc[2]:loc[3]],
		Finish:    finish,
	}, nil
}

// onePieceCode is the card code One Piece display names carry, like
// "(OP12-042)" or "(ST25-001)", which names the printing on its own.
var onePieceCode = regexp.MustCompile(`\(([A-Z]+\d*-\d+[a-z]?)\)`)

// A One Piece display name reads
//
//	Ama no Murakumo Sword (Jolly Roger Foil) (OP06-056) Premium Booster...
//
// Everything before the code stays in the name: the matcher reads variant
// wording like (Jolly Roger Foil) or (Reprint) from it to pick the printing
// the storefront is describing. The Premium Booster listings write the code
// twice, once more ahead of that wording - "Baby 5 (OP04-032) (Jolly Roger
// Foil) (OP04-032)" - so the name runs to the last code and sheds the code's
// earlier copy, keeping the wording between them.
// squareDecorations rewrites the brackets this storefront hangs a finishing
// place in - "(Online Regional 2024 Vol. 2) [Winner]" - as the parentheses it
// writes every other qualifier in. The matcher splits a parenthetical off the
// name and reads it as the printing being named; a bracket it leaves in the
// name, where the place is lost and the listing lands on the printing that
// was awarded none. That is the whole of the difference between a $558 card
// and a $4 one at the same number.
var squareDecorations = strings.NewReplacer("[", "(", "]", ")")

func preprocessOnePiece(product GNProduct) (*mtgmatcher.InputCard, error) {
	locs := onePieceCode.FindAllStringSubmatchIndex(product.DisplayName, -1)
	if locs == nil {
		return nil, errors.New("no card code in display name")
	}
	loc := locs[len(locs)-1]

	code := product.DisplayName[loc[2]:loc[3]]
	cardName := strings.ReplaceAll(product.DisplayName[:loc[0]], "("+code+")", "")
	cardName = squareDecorations.Replace(cardName)

	return &mtgmatcher.InputCard{
		Name:      strings.Join(strings.Fields(cardName), " "),
		Edition:   product.ProductData.SetName,
		Variation: code,
		Foil:      strings.EqualFold(product.SelectedFinish, "foil"),
	}, nil
}

// fabCode is the number tag Flesh and Blood display names carry, like
// "(ARC017)" or "(FAB113)", after the pitch-color parenthetical that is part
// of the card's name.
var fabCode = regexp.MustCompile(`\(([0-9A-Z]{2,5}\d{3}[a-z]?)\)`)

// A Flesh and Blood display name reads
//
//	Aether Sink (ARC017) Arcane Rising 1st Edition Rainbow Foil
//
// The selected finish is already the datastore's own vocabulary, the print
// run crossed with the treatment, so it passes through whole.
func preprocessFleshAndBlood(product GNProduct) (*mtgmatcher.InputCard, error) {
	loc := fabCode.FindStringSubmatchIndex(product.DisplayName)
	if loc == nil {
		return nil, errors.New("no card number in display name")
	}

	return &mtgmatcher.InputCard{
		Name:      strings.TrimSpace(product.DisplayName[:loc[0]]),
		Edition:   product.ProductData.SetName,
		Variation: product.DisplayName[loc[2]:loc[3]],
		Finish:    product.SelectedFinish,
		Foil:      strings.Contains(product.SelectedFinish, "Foil"),
	}, nil
}
