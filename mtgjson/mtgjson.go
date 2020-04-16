package mtgjson

import (
	"encoding/json"
	"io"
	"os"
)

type Set struct {
	Name         string `json:"name"`
	IsOnlineOnly bool   `json:"isOnlineOnly"`
	Type         string `json:"type"`
	BaseSetSize  int    `json:"baseSetSize"`
	Cards        []Card `json:"cards"`
	ReleaseDate  string `json:"releaseDate"`
	ParentCode   string `json:"parentCode"`
}

type Card struct {
	Name       string `json:"name"`
	HasNonFoil bool   `json:"hasNonFoil"`
	UUID       string `json:"uuid"`

	HasFoil       bool     `json:"hasFoil"`
	Artist        string   `json:"artist"`
	BorderColor   string   `json:"borderColor"`
	FrameEffects  []string `json:"frameEffects"`
	Layout        string   `json:"layout"`
	Names         []string `json:"names"`
	Number        string   `json:"number"`
	IsAlternative bool     `json:"isAlternative"`
	IsFullArt     bool     `json:"isFullArt"`
	IsStarter     bool     `json:"isStarter"`
	ScryfallId    string   `json:"scryfallId"`
	Variations    []string `json:"variations"`
	FlavorText    string   `json:"flavorText"`
	Watermark     string   `json:"watermark"`
	ForeignData   []struct {
		Language string `json:"language"`
	} `json:"foreignData"`
}

type SimpleCard struct {
	Layout    string   `json:"layout"`
	Name      string   `json:"name"`
	Names     []string `json:"names"`
	Printings []string `json:"printings"`
	Side      string   `json:"side"`
}

const (
	LayoutAftermath = "aftermath"
	LayoutFlip      = "flip"
	LayoutMeld      = "meld"
	LayoutNormal    = "normal"
	LayoutSplit     = "split"
	LayoutTransform = "transform"

	FrameEffectExtendedArt = "extendedart"
	FrameEffectInverted    = "inverted"
	FrameEffectShowcase    = "showcase"

	BorderColorBorderless = "borderless"

	LanguageJapanese = "Japanese"

	SuffixLightMana = "†"
	SuffixSpecial   = "★"
	SuffixVariant   = "†"
)

type SetDatabase map[string]*Set
type CardDatabase map[string]*SimpleCard

// Load a MTGJSON AllPrinting.json file and return a SetDatabase map.
func LoadAllPrintings(allPrintingsPath string) (SetDatabase, error) {
	allPrintingsReader, err := os.Open(allPrintingsPath)
	if err != nil {
		return nil, err
	}
	defer allPrintingsReader.Close()

	return LoadAllPrintingsFromReader(allPrintingsReader)
}

func LoadAllPrintingsFromReader(r io.Reader) (SetDatabase, error) {
	dec := json.NewDecoder(r)
	_, err := dec.Token()
	if err != nil {
		return nil, err
	}

	allPrintingsDb := SetDatabase{}
	for dec.More() {
		val, err := dec.Token()
		if err != nil {
			return nil, err
		}

		code, ok := val.(string)
		if !ok {
			continue
		}

		var set Set
		err = dec.Decode(&set)
		if err != nil {
			return nil, err
		}

		// Skip online-only sets
		if set.IsOnlineOnly {
			continue
		}

		allPrintingsDb[code] = &set
	}

	return allPrintingsDb, nil
}

// Load a MTGJSON AllCards.json file and return a CardDatabase map.
func LoadAllCards(allCardPath string) (CardDatabase, error) {
	allCardsReader, err := os.Open(allCardPath)
	if err != nil {
		return nil, err
	}
	defer allCardsReader.Close()

	return LoadAllCardsFromReader(allCardsReader)
}

func LoadAllCardsFromReader(r io.Reader) (CardDatabase, error) {
	dec := json.NewDecoder(r)
	_, err := dec.Token()
	if err != nil {
		return nil, err
	}

	allCardsDb := CardDatabase{}
	for dec.More() {
		val, err := dec.Token()
		if err != nil {
			return nil, err
		}

		name, ok := val.(string)
		if !ok {
			continue
		}

		var card SimpleCard
		err = dec.Decode(&card)
		if err != nil {
			return nil, err
		}

		// Normalize card name for easier retrieval later
		name = Normalize(name)
		allCardsDb[name] = &card
	}

	return allCardsDb, nil
}

func (c *Card) HasFrameEffect(fe string) bool {
	for _, effect := range c.FrameEffects {
		if effect == fe {
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
