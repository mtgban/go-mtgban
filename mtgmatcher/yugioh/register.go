package yugioh

import "github.com/mtgban/go-mtgban/mtgmatcher"

func init() {
	mtgmatcher.RegisterGame("yugioh", Load)
}
