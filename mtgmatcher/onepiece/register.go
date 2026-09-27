package onepiece

import "github.com/mtgban/go-mtgban/mtgmatcher"

func init() {
	mtgmatcher.RegisterGame(mtgmatcher.GameOnePiece, Load)
}
