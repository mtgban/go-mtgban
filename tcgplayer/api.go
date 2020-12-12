package tcgplayer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

const (
	defaultConcurrency = 8
	defaultAPIRetry    = 5
	maxIdsInRequest    = 250

	pagesPerRequest = 50
	tcgBaseURL      = "https://shop.tcgplayer.com/productcatalog/product/getpricetable?productId=0&gameName=magic&useV2Listings=true&page=0&pageSize=0&sortValue=price"

	tcgApiTokenURL = "https://api.tcgplayer.com/token"

	tcgApiVersion    = "v1.39.0"
	tcgApiProductURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/product/"
	tcgApiPricingURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/sku/"
	tcgApiBuylistURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/buy/sku/"

	tcgApiSKUsURL   = "https://api.tcgplayer.com/" + tcgApiVersion + "/catalog/products/%s/skus"
	tcgApiSearchURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/catalog/categories/1/search"
)

type TCGClient struct {
	client *retryablehttp.Client
}

func NewTCGClient(publicId, privateId string) *TCGClient {
	tcg := TCGClient{}
	tcg.client = retryablehttp.NewClient()
	tcg.client.Logger = nil
	tcg.client.HTTPClient.Transport = &authTransport{
		Parent:    tcg.client.HTTPClient.Transport,
		PublicId:  publicId,
		PrivateId: privateId,

		// Set a relatively high rate to prevent unexpected limits later
		Limiter: rate.NewLimiter(80, 20),

		mtx: sync.RWMutex{},
	}
	return &tcg
}

type authTransport struct {
	Parent    http.RoundTripper
	PublicId  string
	PrivateId string
	token     string
	expires   time.Time
	Limiter   *rate.Limiter
	mtx       sync.RWMutex
}

func (t *authTransport) authToken() (string, time.Time, error) {
	params := url.Values{}
	params.Set("grant_type", "client_credentials")
	params.Set("client_id", t.PublicId)
	params.Set("client_secret", t.PrivateId)
	body := strings.NewReader(params.Encode())

	resp, err := cleanhttp.DefaultClient().Post(tcgApiTokenURL, "application/json", body)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, err
	}

	var response struct {
		AccessToken string        `json:"access_token"`
		ExpiresIn   time.Duration `json:"expires_in"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return "", time.Time{}, err
	}

	expires := time.Now().Add(response.ExpiresIn * time.Second)
	return response.AccessToken, expires, nil
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	err := t.Limiter.Wait(context.Background())
	if err != nil {
		return nil, err
	}
	if t.PublicId == "" || t.PrivateId == "" {
		return nil, fmt.Errorf("missing public or private id")
	}

	// Retrieve the static values
	t.mtx.RLock()
	token := t.token
	expires := t.expires
	t.mtx.RUnlock()

	// If there is a token, make sure it's still valid
	if token != "" || time.Now().After(expires.Add(-1*time.Hour)) {
		// If not valid, ask for generating a new one
		t.mtx.Lock()
		token = ""
		t.mtx.Unlock()
	}

	// Generate a new token
	if token == "" {
		t.mtx.Lock()
		// Only perform this action once, for the routine that got the mutex first
		// The others will just use the updated token immediately after
		if token == t.token {
			t.token, t.expires, err = t.authToken()
		}
		token = t.token
		t.mtx.Unlock()
		// If anything fails
		if err != nil {
			return nil, err
		}
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	return t.Parent.RoundTrip(req)
}

type TCGPrice struct {
	ProductId      int     `json:"productId"`
	LowPrice       float64 `json:"lowPrice"`
	MarketPrice    float64 `json:"marketPrice"`
	MidPrice       float64 `json:"midPrice"`
	DirectLowPrice float64 `json:"directLowPrice"`
	SubTypeName    string  `json:"subTypeName"`
}

func (tcg *TCGClient) TCGPricesForIds(productIds []string) ([]TCGPrice, error) {
	resp, err := tcg.client.Get(tcgApiProductURL + strings.Join(productIds, ","))
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
		return nil, err
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
	resp, err := tcg.client.Get(fmt.Sprintf(tcgApiSKUsURL, productId))
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

type TCGSKUPrice struct {
	SkuId int `json:"skuId"`

	// Only availabe from TCGPricesForSKUs()
	ProductId          int     `json:"productId"`
	LowPrice           float64 `json:"lowPrice"`
	LowestShipping     float64 `json:"lowestShipping"`
	LowestListingPrice float64 `json:"lowestListingPrice"`
	MarketPrice        float64 `json:"marketPrice"`
	DirectLowPrice     float64 `json:"directLowPrice"`

	// Only available from TCGBuylistPricesForSKUs()
	BuylistPrices struct {
		High   float64 `json:"high"`
		Market float64 `json:"market"`
	} `json:"prices"`
}

func (tcg *TCGClient) TCGPricesForSKUs(ids []string) ([]TCGSKUPrice, error) {
	resp, err := tcg.client.Get(tcgApiPricingURL + strings.Join(ids, ","))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool          `json:"success"`
		Errors  []string      `json:"errors"`
		Results []TCGSKUPrice `json:"results"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	return response.Results, nil
}

func (tcg *TCGClient) TCGBuylistPricesForSKUs(ids []string) ([]TCGSKUPrice, error) {
	resp, err := tcg.client.Get(tcgApiBuylistURL + strings.Join(ids, ","))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool          `json:"success"`
		Errors  []string      `json:"errors"`
		Results []TCGSKUPrice `json:"results"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	return response.Results, nil
}

func (tcg *TCGClient) IdSearch(name, edition string) ([]int, error) {
	type tcgFilter struct {
		Name   string   `json:"name"`
		Values []string `json:"values"`
	}

	var searchReq struct {
		Sort    string      `json:"sort"`
		Limit   int         `json:"limit"`
		Offset  int         `json:"offset"`
		Filters []tcgFilter `json:"filters"`
	}

	// Default values
	searchReq.Sort = "name"
	searchReq.Limit = 100

	if name != "" {
		searchReq.Filters = append(searchReq.Filters, tcgFilter{
			Name:   "ProductName",
			Values: []string{name},
		})
	}
	if edition != "" {
		searchReq.Filters = append(searchReq.Filters, tcgFilter{
			Name:   "SetName",
			Values: []string{edition},
		})
	}

	reqBody, err := json.Marshal(&searchReq)
	if err != nil {
		return nil, err
	}

	resp, err := tcg.client.Post(tcgApiSearchURL, "application/json", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		TotalItems int      `json:"totalItems"`
		Success    bool     `json:"success"`
		Errors     []string `json:"errors"`
		Results    []int    `json:"results"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, fmt.Errorf(strings.Join(response.Errors, "|"))
	}
	if len(response.Results) < 1 {
		return nil, fmt.Errorf("empty search response")
	}

	return response.Results, nil
}
