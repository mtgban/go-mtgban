package pokemon

import "github.com/mtgban/go-mtgban/mtgmatcher"

func init() {
	mtgmatcher.RegisterGame("pokemon", Load)
}
