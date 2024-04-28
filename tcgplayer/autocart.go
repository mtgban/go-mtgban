package tcgplayer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
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
		Code    string `json:"Code"`
		Message string `json:"Message"`
	} `json:"errors"`
	Results []struct {
		IsDirect                bool    `json:"IsDirect"`
		SellerQuantityAvailable int     `json:"SellerQuantityAvailable"`
		ItemQuantityInCart      int     `json:"ItemQuantityInCart"`
		CurrentPrice            float64 `json:"CurrentPrice"`
		Status                  int     `json:"Status"`
	}
}

func (tcg *TCGAutoClient) AddProductToCart(sellerKey string, skuId, qty int, isDirect bool) (*TCGAutocartResponse, error) {
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

	resp, err := tcg.client.Post(link, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response TCGAutocartResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), string(data))
	}
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("%s: %s", response.Errors[0].Code, response.Errors[0].Message)
	}

	return &response, nil
}
