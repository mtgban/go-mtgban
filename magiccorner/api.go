package magiccorner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	http "github.com/hashicorp/go-retryablehttp"
)

type MCEdition struct {
	Id   int    `json:"Id"`
	Set  string `json:"Espansione"`
	Code string `json:"ImageUrl"`
}

type MCCard struct {
	Id       int    `json:"IdProduct"`
	Name     string `json:"NomeEn"`
	Set      string `json:"Category"`
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
	mcBaseURL       = "https://www.magiccorner.it/12/modules/store/mcpub.asmx/"
	mcEditionsEndpt = "espansioni"
	mcCardsEndpt    = "carte"

	mcReinassanceId       = 73
	mcRevisedEUFBBId      = 1041
	mcPromoEditionId      = 1113
	mcMerfolksVsGoblinsId = 1116
)

type MCClient struct {
	client *http.Client
}

func NewMCClient() *MCClient {
	mc := MCClient{}
	mc.client = http.NewClient()
	mc.client.Logger = nil
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

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var blob mcBlob
	err = json.Unmarshal(data, &blob)
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
			Id:  mcPromoEditionId,
			Set: "Promo",
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
		return nil, fmt.Errorf("%s - %v", edition.Set, err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s - %d: %v", edition.Set, resp.StatusCode, err)
	}

	var response mcResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("%s - %d: %v", edition.Set, resp.StatusCode, err)
	}
	if response.Error != "" {
		return nil, fmt.Errorf("%s - %d: %v", edition.Set, resp.StatusCode, response.Error)
	}

	return response.Data, nil
}
