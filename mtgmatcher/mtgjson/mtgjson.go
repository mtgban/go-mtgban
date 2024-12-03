package mtgjson

import (
	"encoding/json"
	"errors"
	"io"
)

const (
	LayoutAftermath = "aftermath"
	LayoutFlip      = "flip"
	LayoutMeld      = "meld"
	LayoutNormal    = "normal"
	LayoutSplit     = "split"
	LayoutTransform = "transform"

	FinishNonfoil = "nonfoil"
	FinishFoil    = "foil"
	FinishEtched  = "etched"

	FrameEffectExtendedArt = "extendedart"
	FrameEffectInverted    = "inverted"
	FrameEffectShowcase    = "showcase"
	FrameEffectShattered   = "shatteredglass"

	PromoTypeBundle     = "bundle"
	PromoTypeBuyABox    = "buyabox"
	PromoTypeGameDay    = "gameday"
	PromoTypeIntroPack  = "intropack"
	PromoTypePrerelease = "prerelease"
	PromoTypePromoPack  = "promopack"
	PromoTypeRelease    = "release"
	PromoTypeBoosterfun = "boosterfun"
	PromoTypeGodzilla   = "godzillaseries"
	PromoTypeDracula    = "draculaseries"
	PromoTypePlayPromo  = "playpromo"
	PromoTypeWPN        = "wizardsplaynetwork"
	PromoTypeGilded     = "gilded"
	PromoTypeTextured   = "textured"
	PromoTypeNeonInk    = "neonink"
	PromoTypeGalaxyFoil = "galaxyfoil"
	PromoTypeSurgeFoil  = "surgefoil"
	PromoTypeGlossy     = "glossy"
	PromoTypeEmbossed   = "embossed"
	PromoTypeSerialized = "serialized"
	PromoTypeHaloFoil   = "halofoil"
	PromoTypeScroll     = "scroll"
	PromoTypePoster     = "poster"
	PromoTypeSilverFoil = "silverfoil"
	PromoTypeRippleFoil = "ripplefoil"
	PromoTypeRaisedFoil = "raisedfoil"
	PromoTypeFNM        = "fnm"

	PromoTypeStoreChampionship = "storechampionship"

	PromoTypeStepAndCompleat = "stepandcompleat"
	PromoTypeOilSlick        = "oilslick"
	PromoTypeConcept         = "concept"
	PromoTypeConfettiFoil    = "confettifoil"
	PromoTypeDoubleRainbow   = "doublerainbow"
	PromoTypeRainbowFoil     = "rainbowfoil"
	PromoTypeFractureFoil    = "fracturefoil"

	PromoTypeThickDisplay  = "thick"
	PromoTypeJudgeGift     = "judgegift"
	PromoTypeArenaLeague   = "arenaleague"
	PromoTypePlayerRewards = "playerrewards"
	PromoTypeStarterDeck   = "starterdeck"
	PromoTypeDraftWeekend  = "draftweekend"
	PromoTypeInvisibleInk  = "invisibleink"

	PromoTypeSChineseAltArt = "schinesealtart"

	BorderColorBorderless = "borderless"
	BorderColorGold       = "gold"

	LanguageJapanese  = "Japanese"
	LanguagePhyrexian = "Phyrexian"

	SuffixSpecial = "★"
	SuffixVariant = "†"
	SuffixPhiUp   = "Φ"
	SuffixPhiLow  = "φ"
)

type TCGSku struct {
	Condition string `json:"condition"`
	Language  string `json:"language"`
	Printing  string `json:"printing"`
	Finish    string `json:"finish"`
	ProductId int    `json:"productId"`
	SkuId     int    `json:"skuId"`
}

type AllTCGSkus struct {
	Data map[string][]TCGSku `json:"data"`
	Meta struct {
		Date    string `json:"date"`
		Version string `json:"version"`
	} `json:"meta"`
}

func LoadAllTCGSkus(r io.Reader) (payload AllTCGSkus, err error) {
	err = json.NewDecoder(r).Decode(&payload)
	if err == nil && len(payload.Data) == 0 {
		err = errors.New("empty AllTCGSkus file")
	}
	return
}
