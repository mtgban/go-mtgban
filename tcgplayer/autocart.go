package tcgplayer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
)

const tcgAdd2CartURL = "https://mpgateway.tcgplayer.com/v1/cart/%s/item/add"

type TCGAutoClient struct {
	client *http.Client
	cartId string
}

func NewTCGAutoClient(cartId string) *TCGAutoClient {
	tcg := TCGAutoClient{}
	tcg.client = cleanhttp.DefaultClient()
	tcg.cartId = cartId
	return &tcg
}

type TCGAutocartRequest struct {
	SKU               int    `json:"sku"`
	SellerKey         string `json:"sellerKey"`
	ChannelID         int    `json:"channelId"`
	RequestedQuantity int    `json:"requestedQuantity"`
	Price             int    `json:"price"`
	IsDirect          bool   `json:"isDirect"`
}

type TCGAutocartResponse struct {
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Results []struct {
		IsDirect                bool    `json:"isDirect"`
		SellerQuantityAvailable int     `json:"sellerQuantityAvailable"`
		ItemQuantityInCart      int     `json:"itemQuantityInCart"`
		CurrentPrice            float64 `json:"currentPrice"`
		Status                  int     `json:"status"`
	}
}

func (tcg *TCGAutoClient) AddProductToCart(ctx context.Context, sellerKey string, skuId, qty int, isDirect bool) (*TCGAutocartResponse, error) {
	var params TCGAutocartRequest
	params.SKU = skuId
	params.SellerKey = sellerKey
	params.RequestedQuantity = qty
	params.IsDirect = isDirect

	payload, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	link := fmt.Sprintf(tcgAdd2CartURL, tcg.cartId)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, link, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tcg.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response TCGAutocartResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("%s: %s", response.Errors[0].Code, response.Errors[0].Message)
	}

	return &response, nil
}
