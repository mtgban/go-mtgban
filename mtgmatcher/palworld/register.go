package palworld

import "github.com/mtgban/go-mtgban/mtgmatcher"

func init() {
	mtgmatcher.RegisterGame("palworld", Load)
}
