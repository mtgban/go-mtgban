package cardmarket

import (
	"encoding/json"
	"io"
	"sort"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"golang.org/x/exp/slices"
)

var filteredExpansions = []string{
	"GnD Cards",
	"MKM Series",
	"Rk post Products",
	"Starcity Games: Creature Collection",
	"Three for One",
	"Filler Cards",
	"Revista Serra Promos",
	"SAWATARIX",
	"TokyoMTG Products",
}

type MKMExpansionIdPair struct {
	IdExpansion int
	Name        string
}

func (mkm *MKMClient) ListExpansionIds() ([]MKMExpansionIdPair, error) {
	expansions, err := mkm.MKMExpansions()
	if err != nil {
		return nil, err
	}

	var out []MKMExpansionIdPair
	for _, exp := range expansions {
		if slices.Contains(filteredExpansions, exp.Name) {
			continue
		}
		if strings.Contains(exp.Name, "Token") ||
			strings.Contains(exp.Name, "Starcity") ||
			strings.Contains(exp.Name, "Oversized") ||
			strings.Contains(exp.Name, "Player Cards") {
			continue
		}
		out = append(out, MKMExpansionIdPair{
			IdExpansion: exp.IdExpansion,
			Name:        exp.Name,
		})
	}

	// Keep list sorted for reproducible results
	sort.Slice(out, func(i, j int) bool {
		return out[i].IdExpansion < out[j].IdExpansion
	})

	return out, nil
}

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

const (
	PriceGuideURL         = "https://downloads.s3.cardmarket.com/productCatalog/priceGuide/price_guide_1.json"
	ProductListSinglesURL = "https://downloads.s3.cardmarket.com/productCatalog/productList/products_singles_1.json"
	ProductListSealedURL  = "https://downloads.s3.cardmarket.com/productCatalog/productList/products_nonsingles_1.json"
)

func GetPriceGuide() ([]PriceGuide, error) {
	resp, err := cleanhttp.DefaultClient().Get(PriceGuideURL)
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

func GetProductListSingles() ([]ProductList, error) {
	return getProductList(ProductListSinglesURL)
}

func GetProductListSealed() ([]ProductList, error) {
	return getProductList(ProductListSealedURL)
}

func getProductList(link string) ([]ProductList, error) {
	resp, err := cleanhttp.DefaultClient().Get(link)
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
