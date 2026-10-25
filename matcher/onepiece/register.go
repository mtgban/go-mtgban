package onepiece

import "github.com/mtgban/go-mtgban/matcher"

func init() {
	matcher.RegisterGame(matcher.GameOnePiece, Load)
}
