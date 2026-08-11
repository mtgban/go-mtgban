package fleshandblood

import "github.com/mtgban/go-mtgban/mtgmatcher"

func init() {
	mtgmatcher.RegisterGame("fleshandblood", Load)
}
