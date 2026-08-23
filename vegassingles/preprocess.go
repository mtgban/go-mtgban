package vegassingles

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// preprocess turns a storefront product into the matcher's input, in the
// grammar its game's display names follow.
func preprocess(product VSProduct, game string) (*mtgmatcher.InputCard, error) {
	switch game {
	case GameRiftbound:
		return preprocessRiftbound(product)
	case GameOnePiece:
		return preprocessOnePiece(product)
	case GamePokemon:
		return preprocessPokemon(product)
	}
	return preprocessMagic(product)
}

func preprocessMagic(product VSProduct) (*mtgmatcher.InputCard, error) {
	// Display name format: "Hallowed Fountain (RVR-280) - Ravnica Remastered"
	// Extract card name by finding the first " ("
	cardName := product.DisplayName
	if idx := strings.Index(cardName, " ("); idx != -1 {
		cardName = cardName[:idx]
	}

	edition := string(product.ProductData.Set)
	if edition == "" {
		edition = product.ProductData.SetName
	}

	variant := ""
	if product.ProductData.CollectorNumberNormalized > 0 {
		variant = strconv.Itoa(product.ProductData.CollectorNumberNormalized)
	}

	foil := product.SelectedFinish == "foil"

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      foil,
	}, nil
}

// riftboundNumber is the collector group riftbound display names carry, like
// "(301*/298)": the printing's own number, starred for the showcase variants,
// over the set size the matcher does not need.
var riftboundNumber = regexp.MustCompile(`\((\d+[a-z]?\*?)/\d+\)`)

// A riftbound display name reads
//
//	Jinx - Loose Cannon (Signature) (301*/298) - Origins Foil
//
// The name stops at the first parenthesis: a tag like (Signature) only
// restates what the starred number already says, so it is dropped rather
// than left for the matcher to see through.
func preprocessRiftbound(product VSProduct) (*mtgmatcher.InputCard, error) {
	match := riftboundNumber.FindStringSubmatch(product.DisplayName)
	if match == nil {
		return nil, errors.New("no collector number in display name")
	}

	cardName := product.DisplayName
	idx := strings.Index(cardName, " (")
	if idx != -1 {
		cardName = cardName[:idx]
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   product.ProductData.SetName,
		Variation: match[1],
		Foil:      strings.EqualFold(product.SelectedFinish, "foil"),
	}, nil
}

// onePieceCode is the card code One Piece display names carry, like
// "(OP06-020)" or "(P-037)", which names the printing on its own.
var onePieceCode = regexp.MustCompile(`\(([A-Z]+\d*-\d+[a-z]?)\)`)

// A One Piece display name reads
//
//	Hody Jones (020) (Alternate Art) (OP06-020) - Wings of the Captain Foil
//
// Everything before the code stays in the name: the matcher reads variant
// wording like (Alternate Art) or (Parallel) from it to pick the printing
// the storefront is describing.
func preprocessOnePiece(product VSProduct) (*mtgmatcher.InputCard, error) {
	loc := onePieceCode.FindStringSubmatchIndex(product.DisplayName)
	if loc == nil {
		return nil, errors.New("no card code in display name")
	}

	return &mtgmatcher.InputCard{
		Name:      strings.TrimSpace(product.DisplayName[:loc[0]]),
		Edition:   product.ProductData.SetName,
		Variation: product.DisplayName[loc[2]:loc[3]],
		Foil:      strings.EqualFold(product.SelectedFinish, "foil"),
	}, nil
}

// pokemonNumber is the collector number Pokemon display names carry inline,
// like "039/73", "042" or "SWSH197", set apart by spaces rather than
// parentheses.
var pokemonNumber = regexp.MustCompile(` ([A-Z]{0,5}\d+[a-zA-Z]?(?:/\d+)?) `)

// A Pokemon display name reads
//
//	Mewtwo GX 039/73  - Holofoil Shining Legends - Ultra Rare
//
// The name is what stands before the number, promo parentheticals like
// (SDCC 2007) included; the finish travels in its own field for this game,
// where the matcher tells Holofoil from Reverse Holofoil by wording.
func preprocessPokemon(product VSProduct) (*mtgmatcher.InputCard, error) {
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
