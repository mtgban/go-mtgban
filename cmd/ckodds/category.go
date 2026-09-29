package main

import (
	"slices"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The set types of reprint products, and the frames of Booster Fun.
var (
	reprintSetTypes  = []string{"masters", "masterpiece", "from_the_vault", "spellbook", "premium_deck", "duel_deck"}
	boosterFunFrames = []string{"showcase", "extendedart", "inverted", "etched", "shatteredglass"}
)

// baseCategory is the category a card is measured in, the first that
// applies: the Reserved List, vintage (a set released through 1995), Secret
// Lair, Booster Fun (borderless, or a showcase, extended-art, inverted,
// etched or shattered-glass frame, since Throne of Eldraine), promo,
// Commander, Masters; anything else is "regular", which categoryOn splits by
// the set's age.
func baseCategory(b *mtgmatcher.Backend, co *mtgmatcher.CardObject) (category, released string) {
	set := b.Sets[co.SetCode]
	if set == nil {
		set = &mtgmatcher.Set{}
	}
	boosterFun := co.BorderColor == "borderless"
	for _, frame := range co.FrameEffects {
		boosterFun = boosterFun || slices.Contains(boosterFunFrames, frame)
	}
	switch {
	case co.IsReserved:
		return "reserved", set.ReleaseDate
	case set.ReleaseDate != "" && set.ReleaseDate <= "1995-12-31":
		return "vintage", set.ReleaseDate
	case co.SetCode == "SLD" || strings.Contains(set.Name, "Secret Lair"):
		return "secret lair", set.ReleaseDate
	case boosterFun && set.ReleaseDate >= "2019-10-04":
		return "booster fun", set.ReleaseDate
	case co.IsPromo || set.Type == "promo":
		return "promo", set.ReleaseDate
	case set.Type == "commander":
		return "commander", set.ReleaseDate
	case slices.Contains(reprintSetTypes, set.Type):
		return "masters", set.ReleaseDate
	}
	return "regular", set.ReleaseDate
}

// categoryOn is a card's category on a day: a regular printing is recent
// while its set is two years old or less, older after.
func categoryOn(category, released string, day time.Time) string {
	if category != "regular" {
		return category
	}
	if released >= day.AddDate(-2, 0, 0).Format(time.DateOnly) {
		return "recent"
	}
	return "older"
}
