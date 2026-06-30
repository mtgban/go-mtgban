package mtgmatcher

// The Magic vocabulary lives in mtgmatcher/magic; these aliases keep the
// constants mtgban-website still resolves from core compiling until it
// migrates. Core cannot import the magic package (it would cycle), so the
// values are re-declared rather than re-exported.
//
// Deprecated: use the mtgmatcher/magic constants instead.
const (
	PromoTypeBoosterfun   = "boosterfun"
	PromoTypeBuyABox      = "buyabox"
	PromoTypePrerelease   = "prerelease"
	PromoTypePromoPack    = "promopack"
	PromoTypeThickDisplay = "thick"
)
