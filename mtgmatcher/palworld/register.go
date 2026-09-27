package palworld

import "github.com/mtgban/go-mtgban/mtgmatcher"

func init() {
	mtgmatcher.RegisterGame(mtgmatcher.GamePalworld, Load)
}
