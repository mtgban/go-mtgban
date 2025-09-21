package cardmarket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
)

var filteredExpansionsTags = []string{
	"Boomer Tokns",
	"Filler Cards",
	"For Science!",
	"Gatherers' Tavern",
	"GnD Cards",
	"Heroes of the Realm",
	"Mana ZenZero",
	"MKM Series",
	"Oversized",
	"Player Cards",
	"Revista Serra Promos",
	"Rk post Products",
	"SAWATARIX",
	"Starcity",
	"Street Clans",
	"Three for One",
	"Token",
	"TokyoMTG Products",
	"Vanlubow",
}

func FilterAndSortExpansions(expansions []MKMExpansion) []MKMExpansion {
	var out []MKMExpansion
	for _, exp := range expansions {
		var skip bool
		for _, tag := range filteredExpansionsTags {
			if strings.Contains(exp.Name, tag) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		out = append(out, exp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

const (
	GameIdMagic = iota + 1
	GameIdWorldOfWarcraft
	GameIdYugioh
	_
	GameIdTheSpoils
	GameIdPokemon
	GameIdForceOfWill
	GameIdCardfightVanguard
	GameIdFinalFantasy
	GameIdWeissSchwarz
	GameIdDragoborne
	GameIdMyLittlePony
	GameIdDragonBallSuper
	_
	GameIdStarWarsDestiny
	GameIdFleshAndBlood
	GameIdDigimon
	GameIdOnePiece
	GameIdLorcana
	GameIdBattleSpiritsSaga
	GameIdStarWarsUnlimited
)

const (
	priceGuideURL         = "https://downloads.s3.cardmarket.com/productCatalog/priceGuide/price_guide_%d.json"
	productListSinglesURL = "https://downloads.s3.cardmarket.com/productCatalog/productList/products_singles_%d.json"
	productListSealedURL  = "https://downloads.s3.cardmarket.com/productCatalog/productList/products_nonsingles_%d.json"
)

type PriceGuide struct {
	IdProduct        int     `json:"idProduct"`
	AvgSellPrice     float64 `json:"avg"`
	LowPrice         float64 `json:"low"`
	TrendPrice       float64 `json:"trend"`
	FoilAvgSellPrice float64 `json:"avg-foil"`
	FoilLowPrice     float64 `json:"low-foil"`
	FoilTrendPrice   float64 `json:"trend-foil"`
	AvgDay1          float64 `json:"avg1"`
	AvgDay7          float64 `json:"avg7"`
	AvgDay30         float64 `json:"avg30"`
	FoilAvgDay1      float64 `json:"avg1-foil"`
	FoilAvgDay7      float64 `json:"avg7-foil"`
	FoilAvgDay30     float64 `json:"avg30-foil"`
}

func GetPriceGuide(gameId int) ([]PriceGuide, error) {
	resp, err := cleanhttp.DefaultClient().Get(fmt.Sprintf(priceGuideURL, gameId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Version     int          `json:"version"`
		CreatedAt   string       `json:"createdAt"`
		PriceGuides []PriceGuide `json:"priceGuides"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.PriceGuides, nil
}

type ProductList struct {
	IdProduct    int    `json:"idProduct"`
	Name         string `json:"name"`
	CategoryID   int    `json:"idCategory"`
	CategoryName string `json:"categoryName"`
	ExpansionID  int    `json:"idExpansion"`
	MetacardID   int    `json:"idMetacard"`
	DateAdded    string `json:"dateAdded"`
}

func GetProductListSingles(gameId int) ([]ProductList, error) {
	return getProductList(gameId, productListSinglesURL)
}

func GetProductListSealed(gameId int) ([]ProductList, error) {
	return getProductList(gameId, productListSealedURL)
}

func getProductList(gameId int, link string) ([]ProductList, error) {
	resp, err := cleanhttp.DefaultClient().Get(fmt.Sprintf(link, gameId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Version   int           `json:"version"`
		CreatedAt string        `json:"createdAt"`
		Products  []ProductList `json:"products"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Products, nil
}

// Make sure there are no duplicate names within the same edition
func SanitizeProductList(productList []ProductList) {
	// Lower product id means lower version number
	for i := range productList {
		name := productList[i].Name
		// Skip already processed entries
		if strings.Contains(name, "(V.") {
			continue
		}

		version := 0
		first := 0
		for j := range productList {
			// Look through the current edition only
			if productList[i].ExpansionID != productList[j].ExpansionID {
				continue
			}

			if name == productList[j].Name {
				// Save the reference to the first element as it's not guaranteed that
				// a. we'll find duplicates in the same edition
				// b. duplicates are grouped together (they might have wide gaps
				// At least the rule of lower id -> lower version number still stands
				if version == 0 {
					first = j
				}
				version++

				// If multiple ids are found, we need to update the version of the first
				// element (and only the first time) and then update the version of the
				// current entry
				if version > 1 {
					if version == 2 {
						productList[first].Name = fmt.Sprintf("%s (V.%d)", name, 1)
					}
					productList[j].Name = fmt.Sprintf("%s (V.%d)", name, version)
				}
			}
		}
	}
}

func BuildURL(idProduct, idGame int, affiliate string, foil bool) string {
	game := ""
	switch idGame {
	case GameIdMagic:
		game = "Magic"
	case GameIdLorcana:
		game = "Lorcana"
	default:
		return ""
	}

	u, err := url.Parse(fmt.Sprintf("https://www.cardmarket.com/en/%s/Products", game))
	if err != nil {
		return ""
	}

	v := url.Values{}

	v.Set("idProduct", fmt.Sprint(idProduct))

	// Set English as preferred language, it switches to the default one
	// automatically in case the card has is non-English only
	v.Set("language", "1")

	if foil {
		v.Set("isFoil", "Y")
	}

	if affiliate != "" {
		v.Set("utm_source", affiliate)
		v.Set("utm_medium", "text")
		v.Set("utm_campaign", "card_prices")
	}

	u.RawQuery = v.Encode()
	return u.String()
}
