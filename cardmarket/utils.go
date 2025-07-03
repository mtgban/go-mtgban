package cardmarket

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/go-cleanhttp"
)

var filteredExpansionsTags = []string{
	"Filler Cards",
	"Gatherers' Tavern",
	"GnD Cards",
	"Heroes of the Realm",
	"MKM Series",
	"Oversized",
	"Player Cards",
	"Revista Serra Promos",
	"Rk post Products",
	"SAWATARIX",
	"Starcity Games: Creature Collection",
	"Starcity",
	"Street Clans",
	"Three for One",
	"Token",
	"TokyoMTG Products",
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Version     int          `json:"version"`
		CreatedAt   string       `json:"createdAt"`
		PriceGuides []PriceGuide `json:"priceGuides"`
	}
	err = json.Unmarshal(data, &response)
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Version   int           `json:"version"`
		CreatedAt string        `json:"createdAt"`
		Products  []ProductList `json:"products"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	return response.Products, nil
}
