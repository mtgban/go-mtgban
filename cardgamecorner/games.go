package cardgamecorner

import (
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
	return &mtgmatcher.InputCard{
		Name:      card.Name,
		Edition:   card.Edition,
		Variation: card.Number,
		Foil:      card.Variants[index].Foil == "Foil",
	}
}
