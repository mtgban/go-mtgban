package mintcard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
)

// Card is one entry of the price list, carrying both sides of the book.
type Card struct {
	ID       string `json:"Id"`
	Name     string `json:"Name"`
	Number   string `json:"Number,omitempty"`
	Quantity int    `json:"Quantity,omitempty,string"`
	Price    string `json:"Price,omitempty"`
	BuyPrice string `json:"Buy Price,omitempty"`

	// The matching SKU id of the card
	TCGplayerID int `json:"TCGPlayerId,omitempty,string"`
}

// MintData is the price list keyed by edition, then by the card entries
// inside it.
type MintData map[string]struct {
	Abbreviation string `json:"Abbreviation"`
	EditionID    string `json:"Edition Id"`
	// Maps of Language - Finish - Condition - Rarity as keys
	Cards map[string]map[string]map[string]map[string][]Card `json:"Cards"`
}

// MintProductList is what the price-list endpoint answers with.
type MintProductList struct {
	Ack       string   `json:"Ack"`
	Products  MintData `json:"Products"`
	Timestamp string   `json:"Timestamp"`
}

const (
	mintPricelistURL = "https://mtgban.mtgmintcard.com"
	mintUserAgent    = "MTGBAN"
)

// MintClient reads MTG Mint Card's price list.
type MintClient struct {
	client *http.Client
	token  string
}

// NewMintClient returns a client, failing if the session cannot be opened.
func NewMintClient(ctx context.Context) (*MintClient, error) {
	mint := MintClient{}
	mint.client = cleanhttp.DefaultClient()

	req, err := http.NewRequestWithContext(ctx, "POST", mintPricelistURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", mintUserAgent)
	req.Header.Set("API-CALL-NAME", "FetchToken")
	req.Header.Set("Content-Type", "application/json")

	resp, err := mint.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var authData map[string]string
	err = json.NewDecoder(resp.Body).Decode(&authData)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	if authData["Ack"] != "Success" {
		return nil, fmt.Errorf("invalid request: %v", authData)
	}
	mint.token = authData["Token"]

	return &mint, nil
}

// GetProductList downloads the whole price list in one call.
func (mint *MintClient) GetProductList(ctx context.Context) (MintData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mintPricelistURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", mintUserAgent)
	req.Header.Set("API-CALL-NAME", "GetProducts")
	req.Header.Set("API-TOKEN", mint.token)

	resp, err := mint.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var productlist MintProductList
	err = json.NewDecoder(resp.Body).Decode(&productlist)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	if productlist.Ack != "Success" {
		return nil, fmt.Errorf("invalid request: %v", productlist)
	}

	return productlist.Products, nil
}
