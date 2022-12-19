package mtgjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Set struct {
	BaseSetSize   int    `json:"baseSetSize"`
	Code          string `json:"code"`
	Cards         []Card `json:"cards"`
	IsFoilOnly    bool   `json:"isFoilOnly"`
	IsNonFoilOnly bool   `json:"isNonFoilOnly"`
	IsOnlineOnly  bool   `json:"isOnlineOnly"`
	KeyruneCode   string `json:"keyruneCode"`
	Name          string `json:"name"`
	ParentCode    string `json:"parentCode"`
	ReleaseDate   string `json:"releaseDate"`
	Tokens        []Card `json:"tokens"`
	Type          string `json:"type"`

	TCGPlayerGroupId int `json:"tcgplayerGroupId"`

	SealedProduct []struct {
		Identifiers map[string]string `json:"identifiers"`
		Name        string            `json:"name"`
		UUID        string            `json:"uuid"`
	} `json:"sealedProduct"`
}

type Card struct {
	Artist           string   `json:"artist"`
	AttractionLights []int    `json:"attractionLights,omitempty"`
	BorderColor      string   `json:"borderColor"`
	Colors           []string `json:"colors"`
	ColorIdentity    []string `json:"colorIdentity"`
	FaceName         string   `json:"faceName"`
	FaceFlavorName   string   `json:"faceFlavorName"`
	Finishes         []string `json:"finishes"`
	FlavorName       string   `json:"flavorName"`
	FlavorText       string   `json:"flavorText"`
	ForeignData      []struct {
		Language string `json:"language"`
	} `json:"foreignData"`
	FrameEffects        []string          `json:"frameEffects"`
	FrameVersion        string            `json:"frameVersion"`
	Identifiers         map[string]string `json:"identifiers"`
	IsAlternative       bool              `json:"isAlternative"`
	IsFullArt           bool              `json:"isFullArt"`
	IsFunny             bool              `json:"isFunny"`
	IsOnlineOnly        bool              `json:"isOnlineOnly"`
	IsOversized         bool              `json:"isOversized"`
	IsPromo             bool              `json:"isPromo"`
	IsReserved          bool              `json:"isReserved"`
	Language            string            `json:"language"`
	Layout              string            `json:"layout"`
	Name                string            `json:"name"`
	Number              string            `json:"number"`
	OriginalReleaseDate string            `json:"originalReleaseDate"`
	Printings           []string          `json:"printings"`
	PromoTypes          []string          `json:"promoTypes"`
	Rarity              string            `json:"rarity"`
	SetCode             string            `json:"setCode"`
	Side                string            `json:"side"`
	Types               []string          `json:"types"`
	Subtypes            []string          `json:"subtypes"`
	Supertypes          []string          `json:"supertypes"`
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

	FinishNonfoil = "nonfoil"
	FinishFoil    = "foil"
	FinishEtched  = "etched"

	FrameEffectExtendedArt = "extendedart"
	FrameEffectEtched      = "etched"
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
	PromoTypeGateway    = "gateway"
	PromoTypeGilded     = "gilded"
	PromoTypeTextured   = "textured"
	PromoTypeNeonInk    = "neonink"
	PromoTypeGalaxyFoil = "galaxyfoil"
	PromoTypeSurgeFoil  = "surgefoil"
	PromoTypeGlossy     = "glossy"

	PromoTypeThickDisplay  = "thick"
	PromoTypeJudgeGift     = "judgegift"
	PromoTypeArenaLeague   = "arenaleague"
	PromoTypePlayerRewards = "playerrewards"

	PromoTypeSChineseAltArt = "schinesealtart"

	BorderColorBorderless = "borderless"
	BorderColorGold       = "gold"

	LanguageJapanese  = "Japanese"
	LanguagePhyrexian = "Phyrexian"

	SuffixLightMana = "†"
	SuffixSpecial   = "★"
	SuffixVariant   = "†"
)

var AllPromoTypes = []string{
	PromoTypeArenaLeague,
	PromoTypeBoosterfun,
	PromoTypeBundle,
	PromoTypeBuyABox,
	PromoTypeDracula,
	PromoTypeGalaxyFoil,
	PromoTypeGameDay,
	PromoTypeGateway,
	PromoTypeGilded,
	PromoTypeGlossy,
	PromoTypeGodzilla,
	PromoTypeIntroPack,
	PromoTypeJudgeGift,
	PromoTypeNeonInk,
	PromoTypePlayPromo,
	PromoTypePlayerRewards,
	PromoTypePrerelease,
	PromoTypePromoPack,
	PromoTypeRelease,
	PromoTypeSChineseAltArt,
	PromoTypeSurgeFoil,
	PromoTypeTextured,
	PromoTypeThickDisplay,
	PromoTypeWPN,
}

func LoadAllPrintings(r io.Reader) (payload AllPrintings, err error) {
	err = json.NewDecoder(r).Decode(&payload)
	if err == nil && len(payload.Data) == 0 {
		err = errors.New("empty AllPrintings file")
	}
	return
}

func (c *Card) HasFinish(fi string) bool {
	for i := range c.Finishes {
		if c.Finishes[i] == fi {
			return true
		}
	}
	return false
}

func (c *Card) HasFrameEffect(fe string) bool {
	for i := range c.FrameEffects {
		if c.FrameEffects[i] == fe {
			return true
		}
	}
	return false
}

func (c *Card) HasPromoType(pt string) bool {
	for i := range c.PromoTypes {
		if c.PromoTypes[i] == pt {
			return true
		}
	}
	return false
}

func (c *Card) IsPlaneswalker() bool {
	for _, typeLine := range c.Types {
		if typeLine == "Planeswalker" {
			return true
		}
	}
	return false
}

// Check if a dual-faced card has the same for both faces
func (c *Card) IsDFCSameName() bool {
	idx := strings.Index(c.Name, " // ")
	if idx < 0 {
		return false
	}
	left := c.Name[:idx]
	right := c.Name[idx+4:]
	return left == right
}

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
