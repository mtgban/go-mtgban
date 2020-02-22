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
	TotalSetSize int    `json:"totalSetSize"`
	Cards        []Card `json:"cards"`
	ReleaseDate  string `json:"releaseDate"`
}

type Card struct {
	Name       string `json:"name"`
	HasNonFoil bool   `json:"hasNonFoil"`
	UUID       string `json:"uuid"`

	Artist                 string   `json:"artist"`
	BorderColor            string   `json:"borderColor"`
	FrameEffect            string   `json:"frameEffect"`
	FrameEffects           []string `json:"frameEffects"`
	Layout                 string   `json:"layout"`
	Names                  []string `json:"names"`
	Number                 string   `json:"number"`
	Type                   string   `json:"type"`
	IsAlternative          bool     `json:"isAlternative"`
	IsDateStamped          bool     `json:"isDateStamped"`
	IsPromo                bool     `json:"isPromo"`
	IsFullArt              bool     `json:"isFullArt"`
	IsStarter              bool     `json:"isStarter"`
	ScryfallId             string   `json:"scryfallId"`
	ScryfallIllustrationId string   `json:"scryfallIllustrationId"`
	ScryfallOracleId       string   `json:"scryfallOracleId"`
	Variations             []string `json:"variations"`
	FlavorText             string   `json:"flavorText"`
	ForeignData            []struct {
		Language string `json:"language"`
	} `json:"foreignData"`
}

const (
	LayoutAftermath = "aftermath"
	LayoutFlip      = "flip"
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
)

type MTGDB map[string]Set

// Load a MTGJSON AllPrinting.json file and return a MTGDB map.
func LoadAllPrintings(allPrintingsPath string) (MTGDB, error) {
	allPrintingsReader, err := os.Open(allPrintingsPath)
	if err != nil {
		return nil, err
	}
	defer allPrintingsReader.Close()

	return LoadAllPrintingsFromReader(allPrintingsReader)
}

func LoadAllPrintingsFromReader(r io.Reader) (MTGDB, error) {
	dec := json.NewDecoder(r)
	_, err := dec.Token()
	if err != nil {
		return nil, err
	}

	allPrintingsDb := MTGDB{}
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

		allPrintingsDb[code] = set
	}

	return allPrintingsDb, nil
}

func (c *Card) HasFrameEffect(fe string) bool {
	for _, effect := range c.FrameEffects {
		if effect == fe {
			return true
		}
	}
	return false
}
