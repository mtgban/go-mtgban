package gundam

import (
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoTypeLabels are the words behind a token. Decoration is this side's
// job - the datastore publishes a slug and nothing else, and a slug cannot
// give back the boundaries it dropped, so "worldchampionship" reads as
// "Worldchampionship" to a title-caser and has to be written down instead.
//
// Only the tokens a title-caser gets wrong are here. A single word it gets
// right on its own - "champion", "finalist", "winner", "head" - so listing
// those would only be a second place to keep them in step with the first.
//
// The whole of a promotion's name is the variant beside it, which the builder
// publishes untouched: these are the words of the token, not of the printing.
var promoTypeLabels = map[string]string{
	"1stanniversary":         "1st Anniversary",
	"1stplace":               "1st Place",
	"2ndplace":               "2nd Place",
	"3rdplace":               "3rd Place",
	"animeexpo":              "Anime Expo",
	"bandaicardgamesfest":    "Bandai Card Games Fest",
	"boostkit":               "Boost Kit",
	"collaborationpack":      "Collaboration Pack",
	"deckbuildbox":           "Deck Build Box",
	"earlytrialevent":        "Early Trial Event",
	"eventpack":              "Event Pack",
	"eventpromo":             "Event Promo",
	"firstcombat":            "First Combat",
	"gamaexpo":               "GAMA Expo",
	"gencon":                 "Gen Con",
	"ggenerationeternal":     "G Generation Eternal",
	"gundambase":             "Gundam Base",
	"judgepack":              "Judge Pack",
	"launchevent":            "Launch Event",
	"launchkit":              "Launch Kit",
	"lefthand":               "Left Hand",
	"linkrare":               "Link Rare",
	"movierelease":           "Movie Release",
	"newtypechallenge":       "Newtype Challenge",
	"officialcardcaseset":    "Official Card Case Set",
	"participationpack":      "Participation Pack",
	"premiumaccessory":       "Premium Accessory",
	"premiumcardcollection":  "Premium Card Collection",
	"regionalchampionship":   "Regional Championship",
	"releaseevent":           "Release Event",
	"resourcepack":           "Resource Pack",
	"resourceset":            "Resource Set",
	"righthand":              "Right Hand",
	"sandiegocomiccon":       "San Diego Comic-Con",
	"serialnumbered":         "Serial Numbered",
	"sp":                     "SP",
	"starterdeckbattleevent": "Starter Deck Battle Event",
	"storetournament":        "Store Tournament",
	"storetrialevent":        "Store Trial Event",
	"winnerpack":             "Winner Pack",
	"worldchampionship":      "World Championship",
}

// promoTypeLabel spells a token the way a reader should see it: the words
// above where a title-caser cannot work them out, and the title-cased token
// otherwise.
func promoTypeLabel(promoType string) string {
	if label, found := promoTypeLabels[promoType]; found {
		return label
	}
	return mtgmatcher.Title(promoType)
}
