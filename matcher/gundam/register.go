package gundam

import "github.com/mtgban/go-mtgban/matcher"

func init() {
	matcher.RegisterGame(matcher.GameGundam, Load)
}
