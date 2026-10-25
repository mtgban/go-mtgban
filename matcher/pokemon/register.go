package pokemon

import "github.com/mtgban/go-mtgban/matcher"

func init() {
	matcher.RegisterGame(matcher.GamePokemon, Load)
}
