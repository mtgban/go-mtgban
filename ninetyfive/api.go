package ninetyfive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	baseURL  = "https://shop.95gamecenter.com/jsons"
	staticUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:135.0) Gecko/20100101 Firefox/135.0"
)

type NFCard map[string]struct {
	CardName     string `json:"card_name"`
	SetName      string `json:"set_name"`
	CardNum      string `json:"card_num"`
	SetCode      string `json:"set_code"`
	SetSupertype string `json:"set_supertype"`
	DedFoil      string `json:"ded_foil"`
}

// example for sell prices
//
//	"pG07_6": {
//	  "pG07_6_MT_EN_true": {},
//	  "pG07_6_NM_EN_true": {},
//	  "pG07_6_LP_EN_true": {}
//	}
//
// example for buy prices
//
//	"FDN_227": {
//	  "EN": {}
//	}
type NFPrice map[string]map[string]struct {
	// Only for sell prices
	Quan  int    `json:"quan,omitempty,string"`
	Price string `json:"price,omitempty"`

	// Only for buy prices
	BuyPrice    string `json:"buy_price,omitempty"`
	CardLang    string `json:"card_lang,omitempty"`
	QuantityBuy int    `json:"quantity_buy,omitempty,string"`
}

type NFClient struct {
	client *http.Client
}

func NewNFClient() *NFClient {
	nf := NFClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	nf.client = client.StandardClient()
	return &nf
}

func (nf *NFClient) getIndexList(ctx context.Context) ([]string, error) {
	data, err := nf.getFile(ctx, "card_index", "[")
	if err != nil {
		return nil, err
	}

	var list []string
	err = json.Unmarshal(data, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (nf *NFClient) getPrices(ctx context.Context) (NFPrice, error) {
	data, err := nf.getFile(ctx, "sku_index", "[")
	if err != nil {
		return nil, err
	}

	var list []string
	err = json.Unmarshal(data, &list)
	if err != nil {
		return nil, err
	}

	data, err = nf.getFile(ctx, list[0], "{")
	if err != nil {
		return nil, err
	}

	var prices NFPrice
	err = json.Unmarshal(data, &prices)
	if err != nil {
		return nil, err
	}

	return prices, nil
}

func (nf *NFClient) getBuyPrices(ctx context.Context) (NFPrice, error) {
	data, err := nf.getFile(ctx, "price_index", "[")
	if err != nil {
		return nil, err
	}

	var list []string
	err = json.Unmarshal(data, &list)
	if err != nil {
		return nil, err
	}

	data, err = nf.getFile(ctx, list[0], "{")
	if err != nil {
		return nil, err
	}

	var prices NFPrice
	err = json.Unmarshal(data, &prices)
	if err != nil {
		return nil, err
	}

	return prices, nil
}

func (nf *NFClient) getCards(ctx context.Context, name string) (NFCard, error) {
	data, err := nf.getFile(ctx, name, "{")
	if err != nil {
		return nil, err
	}

	var card NFCard
	err = json.Unmarshal(data, &card)
	if err != nil {
		return nil, err
	}

	return card, nil
}

func (nf *NFClient) getFile(ctx context.Context, name, separator string) ([]byte, error) {
	u, err := url.Parse(baseURL + "/" + name + ".js")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", staticUA)

	resp, err := nf.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	base := string(data)
	base = strings.Replace(base, "TL;DR", "TLDR", 1)
	base = strings.Split(base, ";")[0]
	idx := strings.Index(base, separator)
	if idx == -1 {
		return nil, fmt.Errorf("malformed file %s", name)
	}

	return []byte(base[idx:]), nil
}
