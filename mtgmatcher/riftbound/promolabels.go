package riftbound

// promoTypeLabels spells a qualifier the way the catalog writes it. The
// builder folds them to lower case before they reach the datastore, and no
// rule gets the spelling back: title-casing turns "GG EZ" into "Gg Ez".
//
// A qualifier missing from this list still reads as something - the backend
// falls back on title-casing the token - so a promo the catalog adds tomorrow
// shows up plainly spelled rather than not at all.
var promoTypeLabels = map[string]string{
	"alternateart":    "Alternate Art",
	"bestof":          "Best Of",
	"champion":        "Champion",
	"fistbumppromo":   "Fist Bump Promo",
	"ggez":            "GG EZ",
	"launchexclusive": "Launch Exclusive",
	"metal":           "Metal",
	"overnumbered":    "Overnumbered",
	"oversized":       "Oversized",
	"prizewall":       "Prize Wall",
	"rumble":          "Rumble",
	"setof3":          "Set of 3",
	"signature":       "Signature",
	"starter":         "Starter",
	"top8":            "Top 8",
	"ultimate":        "Ultimate",
	"vendetta":        "Vendetta",
}
