package mtgjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type Set struct {
	BaseSetSize  int    `json:"baseSetSize"`
	Code         string `json:"code"`
	Cards        []Card `json:"cards"`
	IsOnlineOnly bool   `json:"isOnlineOnly"`
	KeyruneCode  string `json:"keyruneCode"`
	Name         string `json:"name"`
	ParentCode   string `json:"parentCode"`
	ReleaseDate  string `json:"releaseDate"`
	Type         string `json:"type"`
}

type Card struct {
	Artist      string `json:"artist"`
	BorderColor string `json:"borderColor"`
	FlavorName  string `json:"flavorName"`
	FlavorText  string `json:"flavorText"`
	ForeignData []struct {
		Language string `json:"language"`
	} `json:"foreignData"`
	FrameEffects        []string          `json:"frameEffects"`
	HasFoil             bool              `json:"hasFoil"`
	HasNonFoil          bool              `json:"hasNonFoil"`
	Identifiers         map[string]string `json:"identifiers"`
	IsAlternative       bool              `json:"isAlternative"`
	IsFullArt           bool              `json:"isFullArt"`
	IsReserved          bool              `json:"isReserved"`
	Layout              string            `json:"layout"`
	Name                string            `json:"name"`
	Number              string            `json:"number"`
	OriginalReleaseDate string            `json:"originalReleaseDate"`
	Printings           []string          `json:"printings"`
	PromoTypes          []string          `json:"promoTypes"`
	Rarity              string            `json:"rarity"`
	SetCode             string            `json:"setCode"`
	Side                string            `json:"side"`
	UUID                string            `json:"uuid"`
	Variations          []string          `json:"variations"`
	Watermark           string            `json:"watermark"`
}

// Card implements the Stringer interface
func (c Card) String() string {
	return fmt.Sprintf("%s|%s|%s", c.Name, c.SetCode, c.Number)
}

type AllPrintings struct {
	Data map[string]*Set `json:"data"`
	Meta struct {
		Date    string `json:"date"`
		Version string `json:"version"`
	} `json:"meta"`
}

const (
	LayoutAftermath = "aftermath"
	LayoutFlip      = "flip"
	LayoutMeld      = "meld"
	LayoutNormal    = "normal"
	LayoutSplit     = "split"
	LayoutTransform = "transform"

	FrameEffectExtendedArt = "extendedart"
	FrameEffectFoilEtched  = "etched"
	FrameEffectInverted    = "inverted"
	FrameEffectShowcase    = "showcase"

	PromoTypeBundle     = "bundle"
	PromoTypeBuyABox    = "buyabox"
	PromoTypeGameDay    = "gameday"
	PromoTypePrerelease = "prerelease"
	PromoTypePromoPack  = "promopack"
	PromoTypeRelease    = "release"
	PromoTypeBoosterfun = "boosterfun"
	PromoTypeGodzilla   = "godzillaseries"

	BorderColorBorderless = "borderless"

	LanguageJapanese = "Japanese"

	SuffixLightMana = "†"
	SuffixSpecial   = "★"
	SuffixVariant   = "†"
)

func LoadAllPrintings(r io.Reader) (payload AllPrintings, err error) {
	err = json.NewDecoder(r).Decode(&payload)
	if err == nil && len(payload.Data) == 0 {
		err = errors.New("empty AllPrintings file")
	}
	return
}

func (c *Card) HasFrameEffect(fe string) bool {
	for _, effect := range c.FrameEffects {
		if effect == fe {
			return true
		}
	}
	return false
}

func (c *Card) HasPromoType(pt string) bool {
	for _, promoType := range c.PromoTypes {
		if promoType == pt {
			return true
		}
	}
	return false
}

func (c *Card) HasUniqueLanguage(lang string) bool {
	if len(c.ForeignData) != 1 {
		return false
	}
	return c.ForeignData[0].Language == lang
}

type TCGSku struct {
	Condition string `json:"condition"`
	Language  string `json:"language"`
	Printing  string `json:"printing"`
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
