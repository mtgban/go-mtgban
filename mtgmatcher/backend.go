package mtgmatcher

import (
	"fmt"
	"io"
	"log"
	"slices"
	"time"
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

	// For variable_config
	Chance int `json:"chance"`
	Weight int `json:"weight"`

	// This recursive definition is used for "variable" mode in which one
	// possible configuration is chosen at random
	Configs []map[string][]SealedContent `json:"configs"`
}

type DeckCard struct {
	Count    int    `json:"count"`
	IsEtched bool   `json:"isEtched"`
	IsFoil   bool   `json:"isFoil"`
	UUID     string `json:"uuid"`
}

type SealedProduct struct {
	Category    string                     `json:"category"`
	Contents    map[string][]SealedContent `json:"contents"`
	Identifiers map[string]string          `json:"identifiers"`
	Name        string                     `json:"name"`
	SetCode     string                     `json:"setCode"`
	CardCount   int                        `json:"cardCount"`
	ReleaseDate string                     `json:"releaseDate"`
	Subtype     string                     `json:"subtype"`
	UUID        string                     `json:"uuid"`
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
	TokenSetCode  string `json:"tokenSetCode"`
	Tokens        []Card `json:"tokens"`
	Type          string `json:"type"`

	// List of rarities present in the set
	Rarities []string
	// List of card colors present in the set
	Colors []string
	// Precomputed ReleaseDate value
	ReleaseDateTime time.Time

	Booster       map[string]Booster `json:"booster"`
	SealedProduct []SealedProduct    `json:"sealedProduct"`
	Decks         []struct {
		Code               string     `json:"code"`
		Commander          []DeckCard `json:"commander"`
		MainBoard          []DeckCard `json:"mainBoard"`
		DisplayCommander   []DeckCard `json:"displayCommander"`
		Planes             []DeckCard `json:"planes"`
		Schemes            []DeckCard `json:"schemes"`
		SideBoard          []DeckCard `json:"sideBoard"`
		Tokens             []DeckCard `json:"tokens"`
		Name               string     `json:"name"`
		SealedProductUUIDs []string   `json:"sealedProductUuids"`
	} `json:"decks"`
}

type Card struct {
	Artist              string              `json:"artist"`
	AttractionLights    []int               `json:"attractionLights"`
	BorderColor         string              `json:"borderColor"`
	Colors              []string            `json:"colors"`
	ColorIdentity       []string            `json:"colorIdentity"`
	FaceName            string              `json:"faceName"`
	FaceFlavorName      string              `json:"faceFlavorName"`
	FacePrintedName     string              `json:"facePrintedName"`
	Finishes            []string            `json:"finishes"`
	FlavorName          string              `json:"flavorName"`
	FlavorText          string              `json:"flavorText"`
	FrameEffects        []string            `json:"frameEffects"`
	FrameVersion        string              `json:"frameVersion"`
	HasContentWarning   bool                `json:"hasContentWarning"`
	Identifiers         map[string]string   `json:"identifiers"`
	IsAlternative       bool                `json:"isAlternative"`
	IsGameChanger       bool                `json:"isGameChanger"`
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
	PrintedName         string              `json:"printedName"`
	PrintedType         string              `json:"printedType"`
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
	Legalities          map[string]string   `json:"legalities"`
	Variations          []string            `json:"variations"`
	Watermark           string              `json:"watermark"`

	ForeignData []struct {
		Name        string            `json:"name"`
		Language    string            `json:"language"`
		Identifiers map[string]string `json:"identifiers"`
		Type        string            `json:"type"`
	} `json:"foreignData"`

	// A list of URLs containing the image of the card
	// At a minimum "full" and "thumbnail" versions should be provided
	Images map[string]string
}

// Card implements the Stringer interface
func (c Card) String() string {
	if c.Number == "" {
		return fmt.Sprintf("[%s] %s", c.SetCode, c.Name)
	}
	return fmt.Sprintf("%s|%s|%s", c.Name, c.SetCode, c.Number)
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

const (
	FinishNonfoil = "nonfoil"
	FinishFoil    = "foil"
	FinishEtched  = "etched"

	FrameEffectExtendedArt = "extendedart"
	FrameEffectInverted    = "inverted"
	FrameEffectShowcase    = "showcase"
	FrameEffectShattered   = "shatteredglass"

	PromoTypeArenaLeague       = "arenaleague"
	PromoTypeBoosterfun        = "boosterfun"
	PromoTypeBundle            = "bundle"
	PromoTypeBuyABox           = "buyabox"
	PromoTypeConcept           = "concept"
	PromoTypeConfettiFoil      = "confettifoil"
	PromoTypeDoubleExposure    = "doubleexposure"
	PromoTypeDoubleRainbow     = "doublerainbow"
	PromoTypeDracula           = "draculaseries"
	PromoTypeDraftWeekend      = "draftweekend"
	PromoTypeEmbossed          = "embossed"
	PromoTypeFNM               = "fnm"
	PromoTypeFractureFoil      = "fracturefoil"
	PromoTypeGalaxyFoil        = "galaxyfoil"
	PromoTypeGameDay           = "gameday"
	PromoTypeGilded            = "gilded"
	PromoTypeGlossy            = "glossy"
	PromoTypeGodzilla          = "godzillaseries"
	PromoTypeHaloFoil          = "halofoil"
	PromoTypeIntroPack         = "intropack"
	PromoTypeInvisibleInk      = "invisibleink"
	PromoTypeJudgeGift         = "judgegift"
	PromoTypeManaFoil          = "manafoil"
	PromoTypeNeonInk           = "neonink"
	PromoTypeOilSlick          = "oilslick"
	PromoTypePlayPromo         = "playpromo"
	PromoTypePlayerRewards     = "playerrewards"
	PromoTypePoster            = "poster"
	PromoTypePrerelease        = "prerelease"
	PromoTypePromoPack         = "promopack"
	PromoTypeRainbowFoil       = "rainbowfoil"
	PromoTypeRaisedFoil        = "raisedfoil"
	PromoTypeRelease           = "release"
	PromoTypeRippleFoil        = "ripplefoil"
	PromoTypeSChineseAltArt    = "schinesealtart"
	PromoTypeScroll            = "scroll"
	PromoTypeSerialized        = "serialized"
	PromoTypeSilverFoil        = "silverfoil"
	PromoTypeStarterDeck       = "starterdeck"
	PromoTypeStepAndCompleat   = "stepandcompleat"
	PromoTypeStoreChampionship = "storechampionship"
	PromoTypeSurgeFoil         = "surgefoil"
	PromoTypeTextured          = "textured"
	PromoTypeThickDisplay      = "thick"
	PromoTypeWPN               = "wizardsplaynetwork"

	BorderColorBorderless = "borderless"

	LanguageJapanese  = "Japanese"
	LanguagePhyrexian = "Phyrexian"

	SuffixSpecial = "★"
	SuffixVariant = "†"
	SuffixPhiLow  = "φ"
)

// CardObject is an extension of Card, containing fields that cannot
// be easily represented in the original object.
type CardObject struct {
	Card
	Edition string
	Foil    bool
	Etched  bool
	Sealed  bool
}

// Card implements the Stringer interface
func (co CardObject) String() string {
	if co.Sealed {
		return co.Card.String()
	}
	finish := "nonfoil"
	if co.Etched {
		finish = "etched"
	} else if co.Foil {
		finish = "foil"
	}
	return fmt.Sprintf("%s|%s", co.Card, finish)
}

type AlternateProps struct {
	OriginalName   string
	OriginalNumber string
	IsFlavor       bool
}

var defaultBackend Backend

type Backend struct {
	// Slice of all set codes loaded
	AllSets []string

	// Map of set code : Set
	Sets map[string]*Set

	// Map of normalized name : canonical name
	// This is slightly different for tokens, as they are tagged as such
	CanonicalNames map[string]string

	// Map of uuid : CardObject
	UUIDs map[string]CardObject

	// Slice with token names (not normalized and without any "Token" tags)
	Tokens []string

	// Slice with every uniquely normalized name
	AllNames []string
	// Slice with every unique name, as it would appear on a card
	AllCanonicalNames []string
	// Slice with every unique name, lower case
	AllLowerNames []string

	// Slice with every uniquely normalized product name
	AllSealed []string
	// Slice with every unique product name, as defined by mtgjson
	AllCanonicalSealed []string
	// Slice with every unique product name, lower case
	AllLowerSealed []string

	// Map of all normalized names to slice of uuids
	Hashes map[string][]string

	// Map of face/flavor names to set of canonical properties, such as original
	// name, and number, as well as a way to determine FlavorNames
	// Neither key nor values are normalized
	AlternateProps map[string]AlternateProps

	// Slice with every possible non-sealed uuid
	AllUUIDs []string
	// Slice with every possible sealed uuid
	AllSealedUUIDs []string

	// Non-sealed uuids bucketed by set code, each bucket sorted
	SetUUIDs map[string][]string
	// Sealed uuids bucketed by set code, each bucket sorted
	SetSealedUUIDs map[string][]string

	// Non-MTGBAN UUID to a card (or product) UUID
	ExternalIdentifiers map[string]string

	// A list of keywords mapped to the full Commander set name
	CommanderKeywordMap map[string]string

	// A list of promo types as exported by mtgjson
	AllPromoTypes []string

	// A list of deck names of Secret Lair Commander cards
	SLDDeckNames []string

	// Game-specific identification hooks used by Match, attached by the
	// game's datastore loader via SetRules.
	rules GameRules
}

var logger = log.New(io.Discard, "", log.LstdFlags)

const (
	suffixFoil   = "_f"
	suffixEtched = "_e"
)

func SetGlobalDatastore(b *Backend) {
	defaultBackend = *b
}

func SetGlobalLogger(userLogger *log.Logger) {
	logger = userLogger
}
