package magiccorner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

type MCEdition struct {
	Id   int    `json:"Id"`
	Name string `json:"Espansione"`
	Code string `json:"ImageUrl"`
}

type MCCard struct {
	Id       int    `json:"IdProduct"`
	Name     string `json:"NomeEn"`
	Edition  string `json:"Category"`
	Code     string `json:"Icon"`
	Rarity   string `json:"Rarita"`
	Extra    string `json:"Image"`
	OrigName string `json:"NomeIt"`
	URL      string `json:"Url"`
	Variants []struct {
		Id        int     `json:"IdProduct"`
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
	IdCategory    string `json:"IdCategory"`
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

	mcPromoEditionId      = 1113
	mcMerfolksVsGoblinsId = 1116
)

type MCClient struct {
	client *http.Client
}

func NewMCClient() *MCClient {
	mc := MCClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	mc.client = client.StandardClient()
	return &mc
}

// Retrieve the available edition ids and names
func (mc *MCClient) GetEditionList(addPromoEd bool) ([]MCEdition, error) {
	param := mcEditionParam{
		UIc: "it",
	}
	reqBody, err := json.Marshal(&param)
	if err != nil {
		return nil, err
	}

	resp, err := mc.client.Post(mcBaseURL+mcEditionsEndpt, "application/json", bytes.NewReader(reqBody))
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
	err = json.Unmarshal([]byte(blob.Data), &editionList)
	if err != nil {
		return nil, err
	}

	if addPromoEd {
		// This edition is not present in the normal callback
		editionList = append(editionList, MCEdition{
			Id:   mcPromoEditionId,
			Name: "Promo",
		})
	}

	return editionList, nil
}

func (mc *MCClient) GetInventoryForEdition(edition MCEdition) ([]MCCard, error) {
	// This breaks on the main website too, just skip it
	if edition.Id == mcMerfolksVsGoblinsId {
		return nil, nil
	}

	// The last field before || is the language
	// 0 - any language, 72 - english only
	langCode := 0
	if edition.Id == mcPromoEditionId {
		langCode = 72
	}
	param := mcParam{
		// Search string for Id and Language
		SearchField: fmt.Sprintf("%d|0|0|0|0|%d||true|0|", edition.Id, langCode),

		// The edition/category id
		IdCategory: fmt.Sprintf("%d", edition.Id),

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

	resp, err := mc.client.Post(mcBaseURL+mcCardsEndpt, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("%s - %v", edition.Name, err)
	}
	defer resp.Body.Close()

	var response mcResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("%s - %d: %v", edition.Name, resp.StatusCode, err)
	}
	if response.Error != "" {
		return nil, fmt.Errorf("%s - %d: %v", edition.Name, resp.StatusCode, response.Error)
	}

	return response.Data, nil
}

type MCExpansion struct {
	Id      int    `json:"Id"`
	Name    string `json:"Espansione"`
	Enabled bool   `json:"Enabled"`
}

type MCBuylistEditionResponse struct {
	Expansions []MCExpansion `json:"Expansions"`
}

func (mc *MCClient) GetBuylistEditions() ([]MCExpansion, error) {
	resp, err := mc.client.Get(mcEditionBuylistURL)
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}
	defer resp.Body.Close()

	var response MCBuylistEditionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}

	return response.Expansions, nil
}

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

type MCBuylistResponse struct {
	Result MCBuylistResult `json:"Result"`

	ID              int  `json:"Id"`
	Status          int  `json:"Status"`
	IsCanceled      bool `json:"IsCanceled"`
	IsCompleted     bool `json:"IsCompleted"`
	CreationOptions int  `json:"CreationOptions"`
	IsFaulted       bool `json:"IsFaulted"`
}

type MCBuylistResult struct {
	Products []MCProduct `json:"Products"`
	Total    int         `json:"Total"`
}

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

func (mc *MCClient) GetHotBuylistPage(page int) ([]MCProduct, error) {
	resp, err := mc.client.Get(mcHotBuylistURL + "?p=" + fmt.Sprint(page))
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}
	defer resp.Body.Close()

	var response MCBuylistResult
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}

	return response.Products, nil
}

func (mc *MCClient) GetBuylistForEdition(edition, page int) (*MCBuylistResult, error) {
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
	resp, err := mc.client.Post(link, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}
	defer resp.Body.Close()

	var response MCBuylistResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}

	return &response.Result, nil
}
