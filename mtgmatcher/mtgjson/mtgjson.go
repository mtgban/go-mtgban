package mtgjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/exp/slices"
)

type Sheet struct {
	AllowDuplicates bool           `json:"allowDuplicates"`
	BalanceColors   bool           `json:"balanceColors"`
	Cards           map[string]int `json:"cards"`
	Fixed           bool           `json:"fixed"`
	Foil            bool           `json:"foil"`
	TotalWeight     int            `json:"totalWeight"`
}

type Booster struct {
	Boosters []struct {
		Contents map[string]int `json:"contents"`
		Weight   int            `json:"weight"`
	} `json:"boosters"`
	BoostersTotalWeight int              `json:"boostersTotalWeight"`
	Sheets              map[string]Sheet `json:"sheets"`
	Name                string           `json:"name"`
}

type SealedContent struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
	Foil  bool   `json:"foil"`
	Name  string `json:"name"`
	Set   string `json:"set"`
	UUID  string `json:"uuid"`

	// This recursive definition is used for "variable" mode in which one
	// possible configuration is chosen at random
	Configs []map[string][]SealedContent `json:"configs"`
}

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

	Booster       map[string]Booster `json:"booster"`
	SealedProduct []struct {
		Category    string                     `json:"category"`
		Contents    map[string][]SealedContent `json:"contents"`
		Identifiers map[string]string          `json:"identifiers"`
		Name        string                     `json:"name"`
		ProductSize int                        `json:"productSize"`
		Subtype     string                     `json:"subtype"`
		UUID        string                     `json:"uuid"`
	} `json:"sealedProduct"`
	Decks []struct {
		Cards []struct {
			Count  int    `json:"count"`
			Finish string `json:"finish"`
			UUID   string `json:"uuid"`
		} `json:"cards"`
		Name               string   `json:"name"`
		SealedProductUUIDs []string `json:"sealedProductUuids"`
	} `json:"decks"`
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
	FrameEffects        []string            `json:"frameEffects"`
	FrameVersion        string              `json:"frameVersion"`
	Identifiers         map[string]string   `json:"identifiers"`
	IsAlternative       bool                `json:"isAlternative"`
	IsFullArt           bool                `json:"isFullArt"`
	IsFunny             bool                `json:"isFunny"`
	IsOnlineOnly        bool                `json:"isOnlineOnly"`
	IsOversized         bool                `json:"isOversized"`
	IsPromo             bool                `json:"isPromo"`
	IsReserved          bool                `json:"isReserved"`
	Language            string              `json:"language"`
	Layout              string              `json:"layout"`
	Name                string              `json:"name"`
	Number              string              `json:"number"`
	OriginalReleaseDate string              `json:"originalReleaseDate"`
	Printings           []string            `json:"printings"`
	PromoTypes          []string            `json:"promoTypes"`
	Rarity              string              `json:"rarity"`
	SetCode             string              `json:"setCode"`
	SourceProducts      map[string][]string `json:"sourceProducts"`
	Side                string              `json:"side"`
	Subsets             []string            `json:"subsets"`
	Types               []string            `json:"types"`
	Subtypes            []string            `json:"subtypes"`
	Supertypes          []string            `json:"supertypes"`
	UUID                string              `json:"uuid"`
	Variations          []string            `json:"variations"`
	Watermark           string              `json:"watermark"`
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
	PromoTypeGilded     = "gilded"
	PromoTypeTextured   = "textured"
	PromoTypeNeonInk    = "neonink"
	PromoTypeGalaxyFoil = "galaxyfoil"
	PromoTypeSurgeFoil  = "surgefoil"
	PromoTypeGlossy     = "glossy"
	PromoTypeAmpersand  = "ampersand"
	PromoTypeSerialized = "serialized"
	PromoTypeHaloFoil   = "halofoil"

	PromoTypeStepAndCompleat = "stepandcompleat"
	PromoTypeOilSlick        = "oilslick"
	PromoTypeConcept         = "concept"
	PromoTypeConfettiFoil    = "confettifoil"

	PromoTypeThickDisplay  = "thick"
	PromoTypeJudgeGift     = "judgegift"
	PromoTypeArenaLeague   = "arenaleague"
	PromoTypePlayerRewards = "playerrewards"
	PromoTypeStarterDeck   = "starterdeck"
	PromoTypeDraftWeekend  = "draftweekend"

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

var AllPromoTypes = []string{
	PromoTypeAmpersand,
	PromoTypeArenaLeague,
	PromoTypeBoosterfun,
	PromoTypeBundle,
	PromoTypeBuyABox,
	PromoTypeConcept,
	PromoTypeConfettiFoil,
	PromoTypeDracula,
	PromoTypeDraftWeekend,
	PromoTypeGalaxyFoil,
	PromoTypeGameDay,
	PromoTypeGilded,
	PromoTypeGlossy,
	PromoTypeGodzilla,
	PromoTypeHaloFoil,
	PromoTypeIntroPack,
	PromoTypeJudgeGift,
	PromoTypeNeonInk,
	PromoTypeOilSlick,
	PromoTypePlayPromo,
	PromoTypePlayerRewards,
	PromoTypePrerelease,
	PromoTypePromoPack,
	PromoTypeRelease,
	PromoTypeSChineseAltArt,
	PromoTypeSerialized,
	PromoTypeStarterDeck,
	PromoTypeStepAndCompleat,
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
	return slices.Contains(c.Finishes, fi)
}

func (c *Card) HasFrameEffect(fe string) bool {
	return slices.Contains(c.FrameEffects, fe)
}

func (c *Card) HasPromoType(pt string) bool {
	return slices.Contains(c.PromoTypes, pt)
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
