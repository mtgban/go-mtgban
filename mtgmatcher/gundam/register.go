package gundam

import "github.com/mtgban/go-mtgban/mtgmatcher"

func init() {
	mtgmatcher.RegisterGame(mtgmatcher.GameGundam, Load)
}
