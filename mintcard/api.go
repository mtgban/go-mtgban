package mintcard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
)

type Card struct {
	ID       string `json:"Id"`
	Name     string `json:"Name"`
	Number   string `json:"Number,omitempty"`
	Quantity int    `json:"Quantity,omitempty,string"`
	Price    string `json:"Price,omitempty"`
	BuyPrice string `json:"Buy Price,omitempty"`
}

// Map with Edition as keys
type MintData map[string]struct {
	Abbreviation string `json:"Abbreviation"`
	EditionId    string `json:"Edition Id"`
	// Maps of Language - Finish - Condition - Rarity as keys
	Cards map[string]map[string]map[string]map[string][]Card `json:"Cards"`
}

type MintProductList struct {
	Ack       string   `json:"Ack"`
	Products  MintData `json:"Products"`
	Timestamp string   `json:"Timestamp"`
}

const (
	mintPricelistURL = "https://mtgban.mtgmintcard.com"
	mintUserAgent    = "MTGBAN"
)

type MintClient struct {
	client *http.Client
	token  string
}

func NewMintClient() (*MintClient, error) {
	mint := MintClient{}
	mint.client = cleanhttp.DefaultClient()

	req, err := http.NewRequest("POST", mintPricelistURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", mintUserAgent)
	req.Header.Add("API-CALL-NAME", "FetchToken")
	req.Header.Add("Content-Type", "application/json")

	resp, err := mint.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var authData map[string]string
	err = json.Unmarshal(data, &authData)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %s", string(data))
	}
	if authData["Ack"] != "Success" {
		return nil, fmt.Errorf("invalid request: %s", string(data))
	}
	mint.token = authData["Token"]

	return &mint, nil
}

func (mint *MintClient) GetProductList() (MintData, error) {
	req, err := http.NewRequest("POST", mintPricelistURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", mintUserAgent)
	req.Header.Add("API-CALL-NAME", "GetProducts")
	req.Header.Add("API-TOKEN", mint.token)

	resp, err := mint.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var productlist MintProductList
	err = json.Unmarshal(data, &productlist)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	if productlist.Ack != "Success" {
		return nil, fmt.Errorf("invalid request")
	}

	return productlist.Products, nil
}
