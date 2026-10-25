package fleshandblood

import "github.com/mtgban/go-mtgban/matcher"

func init() {
	matcher.RegisterGame(matcher.GameFleshAndBlood, Load)
}
