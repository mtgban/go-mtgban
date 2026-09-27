package fleshandblood

import "github.com/mtgban/go-mtgban/mtgmatcher"

func init() {
	mtgmatcher.RegisterGame(mtgmatcher.GameFleshAndBlood, Load)
}
