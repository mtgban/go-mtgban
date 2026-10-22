package cardgamecorner

import (
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

// preprocessGame reads a card of any game but Magic, whose listings name the
// card, its set and its collector number plainly.
func preprocessGame(b *mtgmatcher.Backend, card *Card, index int) (*mtgmatcher.InputCard, error) {
	return gameInput(b.Game, card, index), nil
}

// gameInput reads a listing's wording the way the game's rules expect it.
func gameInput(game mtgmatcher.Game, card *Card, index int) *mtgmatcher.InputCard {
	name := card.Name
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
	}
	return &mtgmatcher.InputCard{
		Name:      name,
		Edition:   card.Edition,
		Variation: card.Number,
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
