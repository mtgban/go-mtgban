package mtgmatcher

import (
	"fmt"
	"io"
	"log"
	"slices"
	"strings"
	"time"
)

// Sheet is one pool a booster draws from, with the weight of each card in it.
type Sheet struct {
	AllowDuplicates bool           `json:"allowDuplicates"`
	BalanceColors   bool           `json:"balanceColors"`
	Cards           map[string]int `json:"cards"`
	Fixed           bool           `json:"fixed"`
	Foil            bool           `json:"foil"`
	TotalWeight     int            `json:"totalWeight"`
}

// Booster describes how a product's boosters are built: the sheets available
// and the weighted configurations that draw from them.
type Booster struct {
	Boosters []struct {
		Contents map[string]int `json:"contents"`
		Weight   int            `json:"weight"`
	} `json:"boosters"`
	BoostersTotalWeight int              `json:"boostersTotalWeight"`
	Sheets              map[string]Sheet `json:"sheets"`
	Name                string           `json:"name"`
}

// SealedContent is one component of a sealed product: a card, a pack, a deck,
// or a set of alternative configurations chosen at random.
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

// DeckCard is one card in a preconstructed deck, with its finish.
type DeckCard struct {
	Count    int    `json:"count"`
	IsEtched bool   `json:"isEtched"`
	IsFoil   bool   `json:"isFoil"`
	UUID     string `json:"uuid"`
}

// SealedProduct is a sealed item and, where it is known, what opening it can
// produce.
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

// Set is an edition and everything printed in it, cards and sealed alike.
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

// Card is one printing, with the properties that tell it apart from every
// other printing of the same name. Fields follow the MTGJSON project.
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

	// OriginalNumber is Number with the game's own decorations stripped -
	// Magic's ★ and †, Riftbound's star, Lorcana's variant letter - and so
	// it is the plain number a search matches. It is never longer than
	// Number: a loader that widens it here instead of narrowing it turns
	// the ordinary number search into the strict one.
	OriginalNumber string

	// SetTotal is the set size the card's own face prints beside its
	// number, the "167" of "082/167", which is what tells a reprint from
	// its original - Cascoon is 44/130 in Diamond & Pearl and 44/127 in
	// Platinum. It is the card's own rather than the set's, because the
	// pooled sets - World Championship Decks, the promo pools - hold cards
	// that keep the total of wherever they were first printed, and the set
	// they sit in has no one size. Empty means the face prints no total at
	// all, which is most promos, and that emptiness is meaningful: a
	// bare-numbered promo is a different printing from its totalled
	// reprint, so nothing may fill it in from Set.BaseSetSize.
	SetTotal string

	// FoilUUIDs holds one entry per finish the printing is sold in, mapping
	// it to the uuid that carries it: every finish has a uuid of its own and
	// no uuid answers for two. The three shared finishes are keyed by their
	// constant, since output() resolves the caller's flags to one of them
	// and pulls the uuid from here (the standard foil stays under
	// FinishFoil whatever the printing calls it); a finish past them -
	// Lorcana's "rainbowpillars" - is keyed by the game's canonical name
	// for it, the same name CardObject.Finish carries. Loaders populate it; a Card
	// without it falls back to the suffix rules.
	FoilUUIDs map[string]string

	// FinishAliases maps a finish name onto the FoilUUIDs key answering for
	// it on this printing, for the names whose meaning is the printing's own
	// business rather than the game's: Lorcana keys a printing's standard
	// foil under FinishFoil whatever its foil type is called, so the loader
	// registers that type's own name here, along with the name TCGplayer
	// prices the printing's special treatment under. Aliases are spellings,
	// never finishes - they add no uuid and hide none.
	FinishAliases map[string]string

	// Finish is the canonical name of the finish this specific entry
	// carries, as the game's rules spell it (GameRules.CanonicalFinish):
	// FinishNonfoil, FinishFoil and FinishEtched where the game has no name
	// of its own, and the game's own name where it has one - Lorcana's
	// "silver" or "rainbowpillars". It is set per stored uuid, not on the
	// set-level card, which represents every finish, and it is what makes
	// two entries the Foil flag cannot tell apart distinguishable. Sealed
	// entries carry no finish.
	Finish string

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

// HasFinish reports whether the printing was sold in this finish.
func (c *Card) HasFinish(fi string) bool {
	return slices.Contains(c.Finishes, fi)
}

// HasFrameEffect reports whether the printing carries this frame effect.
func (c *Card) HasFrameEffect(fe string) bool {
	return slices.Contains(c.FrameEffects, fe)
}

// HasPromoType reports whether the printing carries this promo type.
func (c *Card) HasPromoType(pt string) bool {
	return slices.Contains(c.PromoTypes, pt)
}

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

// AlternateProps carries the name and number a printing is also known by, for
// the reskins and flavor names storefronts list instead of the real one.
type AlternateProps struct {
	OriginalName   string
	OriginalNumber string
	IsFlavor       bool
}

var defaultBackend Backend

// Backend is a loaded datastore: every set and printing of one game, with the
// indexes Match needs and the game's own rules attached. Build one through a
// game's Load, not by hand.
type Backend struct {
	// Slice of all set codes loaded
	AllSets []string

	// Map of set code : Set
	Sets map[string]*Set

	// Map of Normalize(set name) : Set, for name-based lookups
	NormalizedSets map[string]*Set

	// Map of normalized name : canonical name
	// This is slightly different for tokens, as they are tagged as such
	CanonicalNames map[string]string

	// Map of uuid : CardObject. Values are shared with every caller
	// of GetUUID and must never be modified after the load completes.
	UUIDs map[string]*CardObject

	// Slice with token names (not normalized and without any "Token" tags)
	Tokens []string

	// Slice with every uniquely normalized name
	AllNames []string
	// Slice with every unique name, as it would appear on a card
	AllCanonicalNames []string
	// Slice with every unique name, lower case
	AllLowerNames []string

	// What AddName and AddSealed have filed already, one set per list: two
	// spellings can normalize or lowercase to one string, so each list is
	// deduped on what it holds rather than on one of the others.
	seenNames           map[string]bool
	seenLowerNames      map[string]bool
	seenCanonicalNames  map[string]bool
	seenSealed          map[string]bool
	seenLowerSealed     map[string]bool
	seenCanonicalSealed map[string]bool

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

	// sealedIdx is the sealed namespace as ResolveSealed reads it, built by
	// SortSealed once every product is filed.
	sealedIdx *sealedIndex

	// Non-sealed uuids bucketed by set code, each bucket sorted
	SetUUIDs map[string][]string
	// Sealed uuids bucketed by set code, each bucket sorted
	SetSealedUUIDs map[string][]string

	// Non-MTGBAN identifiers to a card (or product) UUID, one map per id
	// space so the integer spaces never shadow each other. The uuid filed
	// for a shared identifier is the printing's base sibling: the nonfoil,
	// or the foil where no nonfoil was sold, or the etched where nothing
	// else was.
	ExternalIdentifiers map[string]map[string]string

	// A list of keywords mapped to the full Commander set name
	CommanderKeywordMap map[string]string

	// A list of promo types as exported by mtgjson
	AllPromoTypes []string

	// Map of a promo type to the words it was made from, for the games that
	// slug a qualifier the storefront wrote in full ("premiumcardcollection
	// bestselectionvol6" was "Premium Card Collection -Best Selection Vol.
	// 6-"). A promo type is a token so that it can be typed into a search;
	// this is what puts the words back for a reader. Empty for Magic, whose
	// types are tokens at the source and have no fuller spelling to keep.
	PromoTypeLabels map[string]string

	// A list of deck names of Secret Lair Commander cards
	SLDDeckNames []string

	// Game-specific identification hooks used by Match, attached by the
	// game's datastore loader via SetRules.
	rules         GameRules
	knownFinishes map[string]bool
}

// Logger receives the matcher's diagnostics. It discards them until
// SetGlobalLogger says otherwise.
var Logger = log.New(io.Discard, "", log.LstdFlags)

const (
	suffixFoil   = "_f"
	suffixEtched = "_e"
)

// IndexSets indexes Sets by their normalized name, so name lookups need one
// map access instead of rescanning and renormalizing the whole set list.
// Codes are visited in sorted order so that two sets normalizing to the same
// name resolve deterministically (lowest code wins - the linear scan this
// replaces followed random map order). Every loader has to call this once
// its Sets are populated, including the ones living in their own package,
// which is why it is exported.
func (b *Backend) IndexSets() {
	codes := make([]string, 0, len(b.Sets))
	for code := range b.Sets {
		codes = append(codes, code)
	}
	slices.Sort(codes)

	b.NormalizedSets = map[string]*Set{}
	for _, code := range codes {
		name := Normalize(b.Sets[code].Name)
		_, found := b.NormalizedSets[name]
		if !found {
			b.NormalizedSets[name] = b.Sets[code]
		}
	}
}

// SetGlobalDatastore installs the datastore the package-level Match, MatchID
// and the rest resolve against. It copies the value, so later changes to b do
// not reach the installed one.
func SetGlobalDatastore(b *Backend) {
	// The sealed index is read-only and built from the datastore alone, so a
	// datastore whose loader never filed a sealed product through SortSealed
	// gets one here rather than paying for one on every lookup.
	if b.sealedIdx == nil {
		b.sealedIdx = b.buildSealedIndex()
	}
	defaultBackend = *b
}

// SetGlobalLogger points the matcher's diagnostics at a logger of your own.
func SetGlobalLogger(userLogger *log.Logger) {
	Logger = userLogger
}

// AddName files a card name in each search index that does not already hold
// it: normalized in AllNames, lower case in AllLowerNames, and as written in
// AllCanonicalNames.
//
// Each list is deduped on what it actually holds rather than on one of the
// others. Two spellings can normalize or lowercase to one string, and a key
// stored twice returns its whole hash bucket twice — so a name already
// present in one list may still be missing from another, and asking the
// wrong list is how a duplicate gets in.
//
// The lists say what they hold, and so does this: AllLowerNames is
// documented as every unique name in lower case, and a loader that files the
// name as written leaves it holding something else.
func (b *Backend) AddName(name string) {
	if b.seenNames == nil {
		b.seenNames = map[string]bool{}
		b.seenLowerNames = map[string]bool{}
		b.seenCanonicalNames = map[string]bool{}
	}
	if n := Normalize(name); !b.seenNames[n] {
		b.seenNames[n] = true
		b.AllNames = append(b.AllNames, n)
	}
	if lower := strings.ToLower(name); !b.seenLowerNames[lower] {
		b.seenLowerNames[lower] = true
		b.AllLowerNames = append(b.AllLowerNames, lower)
	}
	if !b.seenCanonicalNames[name] {
		b.seenCanonicalNames[name] = true
		b.AllCanonicalNames = append(b.AllCanonicalNames, name)
	}
}
