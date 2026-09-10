package riftbound

// promoTypeLabels spells a qualifier the way the catalog writes it. The
// builder folds them to lower case before they reach the datastore, and no
// rule gets the spelling back: title-casing turns "GG EZ" into "Gg Ez".
//
// A qualifier missing from this list still reads as something - the backend
// falls back on title-casing the token - so a promo the catalog adds tomorrow
// shows up plainly spelled rather than not at all.
var promoTypeLabels = map[string]string{
	"alternateart": "Alternate Art",
	"bestof":       "Best Of",
	"champion":     "Champion",
	// The catalog writes "Fist Bump Promo" and the builder trims the
	// " Promo" every qualifier here is published without, so the token is
	// the two words that are left.
	"fistbump":        "Fist Bump",
	"fullart":         "Full Art",
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
	// The five T1 Worlds Champion cards, filed by the bundle each copy came
	// in: the commemoration is one token and the bundle another, so a
	// query for either reaches the printing.
	"playerbundle":           "Player Bundle",
	"serialnumbered":         "Serial Numbered",
	"signatureeditionbundle": "Signature Edition Bundle",
	"t1worldschampion":       "T1 Worlds Champion",
}
