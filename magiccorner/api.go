package magiccorner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

// MCEdition is a set as Magic Corner files it for selling.
type MCEdition struct {
	ID   int    `json:"Id"`
	Name string `json:"Espansione"`
	Code string `json:"ImageUrl"`
}

// MCCard is one card on sale, in a condition and a language.
type MCCard struct {
	ID       int    `json:"IdProduct"`
	Name     string `json:"NomeEn"`
	Edition  string `json:"Category"`
	Code     string `json:"Icon"`
	Rarity   string `json:"Rarita"`
	Extra    string `json:"Image"`
	OrigName string `json:"NomeIt"`
	URL      string `json:"Url"`
	Variants []struct {
		ID        int     `json:"IdProduct"`
		Language  string  `json:"Lingua"`
		Foil      string  `json:"Foil"`
		Condition string  `json:"CondizioniShort"`
		Quantity  int     `json:"DispoWeb"`
		Price     float64 `json:"Price"`
	} `json:"Varianti"`
}

type mcResponse struct {
	Error string   `json:"Message"`
	Data  []MCCard `json:"d"`
}

type mcParam struct {
	SearchField   string `json:"f"`
	IDCategory    string `json:"IdCategory"`
	UIc           string `json:"UIc"`
	OnlyAvailable bool   `json:"SoloDispo"`
	ProductType   int    `json:"TipoProdotto"`
	IsBuy         bool   `json:"IsVendita"`
}

type mcBlob struct {
	Data string `json:"d"`
}

type mcEditionParam struct {
	UIc string `json:"UIc"`
}

const (
	mcBaseURL       = "https://www.cardgamecorner.com/12/modules/store/mcpub.asmx/"
	mcEditionsEndpt = "espansioni"
	mcCardsEndpt    = "carte"

	mcHotBuylistURL      = "https://www.cardgamecorner.com/webapi/mcbuylist/magic/-/0"
	mcEditionBuylistURL  = "https://www.cardgamecorner.com/webapi/mclistboxes/magic/it"
	mcAdvancedBuylistURL = "https://www.cardgamecorner.com/webapi/mcadvsearch"

	mcPromoEditionID      = 1113
	mcMerfolksVsGoblinsID = 1116
)

// MCClient reads the Magic Corner storefront, which serves its catalog and
// its buylist through different endpoints.
type MCClient struct {
	client *http.Client
}

// NewMCClient returns a client.
func NewMCClient() *MCClient {
	mc := MCClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	mc.client = client.StandardClient()
	return &mc
}

// GetEditionList returns the editions Magic Corner sells, by id and name.
func (mc *MCClient) GetEditionList(ctx context.Context, addPromoEd bool) ([]MCEdition, error) {
	param := mcEditionParam{
		UIc: "it",
	}
	reqBody, err := json.Marshal(&param)
	if err != nil {
		return nil, err
	}

	link := mcBaseURL + mcEditionsEndpt
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, link, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var blob mcBlob
	err = json.NewDecoder(resp.Body).Decode(&blob)
	if err != nil {
		return nil, err
	}

	var editionList []MCEdition
	// There is json in this json!
	err = json.NewDecoder(strings.NewReader(blob.Data)).Decode(&editionList)
	if err != nil {
		return nil, err
	}

	if addPromoEd {
		// This edition is not present in the normal callback
		editionList = append(editionList, MCEdition{
			ID:   mcPromoEditionID,
			Name: "Promo",
		})
	}

	return editionList, nil
}

// GetInventoryForEdition returns everything on sale in one edition.
func (mc *MCClient) GetInventoryForEdition(ctx context.Context, edition MCEdition) ([]MCCard, error) {
	// This breaks on the main website too, just skip it
	if edition.ID == mcMerfolksVsGoblinsID {
		return nil, nil
	}

	// The last field before || is the language
	// 0 - any language, 72 - english only
	langCode := 0
	if edition.ID == mcPromoEditionID {
		langCode = 72
	}
	param := mcParam{
		// Search string for Id and Language
		SearchField: fmt.Sprintf("%d|0|0|0|0|%d||true|0|", edition.ID, langCode),

		// The edition/category id
		IDCategory: fmt.Sprintf("%d", edition.ID),

		// Returns entries with available quantity
		OnlyAvailable: true,

		// Only mtg
		ProductType: 1,

		// No idea what these fields are for
		UIc:   "it",
		IsBuy: false,
	}
	reqBody, err := json.Marshal(&param)
	if err != nil {
		return nil, err
	}

	link := mcBaseURL + mcCardsEndpt
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, link, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s - %v", edition.Name, err)
	}
	defer resp.Body.Close()

	var response mcResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", edition.Name, err)
	}
	if response.Error != "" {
		return nil, fmt.Errorf("%s - %d: %v", edition.Name, resp.StatusCode, response.Error)
	}

	return response.Data, nil
}

// MCExpansion is a set as Magic Corner files it for buying, which does not
// always agree with MCEdition.
type MCExpansion struct {
	ID      int    `json:"Id"`
	Name    string `json:"Espansione"`
	Enabled bool   `json:"Enabled"`
}

// MCBuylistEditionResponse is what the buylist edition endpoint answers with.
type MCBuylistEditionResponse struct {
	Expansions []MCExpansion `json:"Expansions"`
}

// GetBuylistEditions returns the editions Magic Corner is buying from.
func (mc *MCClient) GetBuylistEditions(ctx context.Context) ([]MCExpansion, error) {
	link := mcEditionBuylistURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response MCBuylistEditionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}

	return response.Expansions, nil
}

// MCBuylistRequest is one page of a buylist query.
type MCBuylistRequest struct {
	Q              string  `json:"q"`
	Game           string  `json:"game"`
	Edition        int     `json:"edition"`
	Rarity         string  `json:"rarity"`
	Color          string  `json:"color"`
	Firstedition   string  `json:"firstedition"`
	Foil           string  `json:"foil"`
	Language       *string `json:"language"`
	Page           int     `json:"page"`
	Sort           int     `json:"sort"`
	IsBuyList      bool    `json:"isBuyList"`
	OnlyHotBuyList bool    `json:"onlyHotBuyList"`
	OnlyAvailable  bool    `json:"onlyAvailable"`
}

// MCBuylistResponse is what the buylist endpoint answers with.
type MCBuylistResponse struct {
	Result MCBuylistResult `json:"Result"`

	ID              int  `json:"Id"`
	Status          int  `json:"Status"`
	IsCanceled      bool `json:"IsCanceled"`
	IsCompleted     bool `json:"IsCompleted"`
	CreationOptions int  `json:"CreationOptions"`
	IsFaulted       bool `json:"IsFaulted"`
}

// MCBuylistResult is one page of buylist entries.
type MCBuylistResult struct {
	Products []MCProduct `json:"Products"`
	Total    int         `json:"Total"`
}

// MCProduct is one card Magic Corner is buying, with the price they pay.
type MCProduct struct {
	ID           string  `json:"Id"`
	Game         string  `json:"Game"`
	ModelEn      string  `json:"ModelEn"`
	Rarity       string  `json:"Rarity"`
	Category     string  `json:"Category"`
	Quantity     int     `json:"Quantity"`
	MinAcquisto  float64 `json:"MinAcquisto"`
	MaxAcquisto  float64 `json:"MaxAcquisto"`
	Language     int     `json:"Language"`
	SerialNumber int     `json:"SerialNumber"`
}

// GetBuylistForEdition returns one page of what Magic Corner buys in an
// edition.
func (mc *MCClient) GetBuylistForEdition(ctx context.Context, edition, page int) (*MCBuylistResult, error) {
	payload, err := json.Marshal(&MCBuylistRequest{
		IsBuyList: true,
		Game:      "magic",
		Page:      page,
		Edition:   edition,
		Sort:      5,
	})
	if err != nil {
		return nil, err
	}

	link := mcAdvancedBuylistURL
	if page > 1 {
		link = fmt.Sprintf("%s?p=%d", mcAdvancedBuylistURL, page)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, link, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response MCBuylistResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}

	return &response.Result, nil
}
