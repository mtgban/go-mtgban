package yugioh

import "github.com/mtgban/go-mtgban/matcher"

func init() {
	matcher.RegisterGame(matcher.GameYuGiOh, Load)
}
