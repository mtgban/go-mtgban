package tcgplayer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	http "github.com/hashicorp/go-retryablehttp"
)

type TCGClient struct {
	client *http.Client
}

func NewTCGClient(publicId, privateId string) *TCGClient {
	tcg := TCGClient{}
	tcg.client = http.NewClient()
	tcg.client.Logger = nil
	tcg.client.HTTPClient.Transport = &authTransport{
		Parent:    tcg.client.HTTPClient.Transport,
		PublicId:  publicId,
		PrivateId: privateId,
	}
	return &tcg
}

type TCGPrice struct {
	LowPrice       float64 `json:"lowPrice"`
	MarketPrice    float64 `json:"marketPrice"`
	MidPrice       float64 `json:"midPrice"`
	DirectLowPrice float64 `json:"directLowPrice"`
	SubTypeName    string  `json:"subTypeName"`
}

func (tcg *TCGClient) PricesForId(productId string) ([]TCGPrice, error) {
	resp, err := tcg.client.Get(tcgApiProductURL + productId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool       `json:"success"`
		Errors  []string   `json:"errors"`
		Results []TCGPrice `json:"results"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		if strings.Contains(string(data), "<head><title>403 Forbidden</title></head>") {
			err = fmt.Errorf("403 Forbidden")
		}
		return nil, err
	}
	if !response.Success {
		return nil, fmt.Errorf(strings.Join(response.Errors, "|"))
	}

	return response.Results, nil
}

type TCGSKU struct {
	SkuId       int `json:"skuId"`
	ProductId   int `json:"productId"`
	LanguageId  int `json:"languageId"`
	PrintingId  int `json:"printingId"`
	ConditionId int `json:"conditionId"`
}

func (tcg *TCGClient) SKUsForId(productId string) ([]TCGSKU, error) {
	resp, err := tcg.client.Get(fmt.Sprintf(tcgApiSKUURL, productId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool     `json:"success"`
		Errors  []string `json:"errors"`
		Results []TCGSKU `json:"results"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, fmt.Errorf(strings.Join(response.Errors, "|"))
	}

	return response.Results, nil
}

type TCGBuylistPrice struct {
	ProductId int `json:"productId"`
	Prices    struct {
		High   float64 `json:"high"`
		Market float64 `json:"market"`
	} `json:"prices"`
	SKUs []struct {
		SkuId  int `json:"skuId"`
		Prices struct {
			High   float64 `json:"high"`
			Market float64 `json:"market"`
		} `json:"prices"`
	} `json:"skus"`
}

func (tcg *TCGClient) BuylistPricesForId(productId string) (*TCGBuylistPrice, error) {
	resp, err := tcg.client.Get(tcgApiBuylistURL + productId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool              `json:"success"`
		Errors  []string          `json:"errors"`
		Results []TCGBuylistPrice `json:"results"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, fmt.Errorf(strings.Join(response.Errors, "|"))
	}
	if len(response.Results) < 1 {
		return nil, fmt.Errorf("empty buylist response")
	}

	return &response.Results[0], nil
}
