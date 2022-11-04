package tcgplayer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	tcgApiListProductsURL = "https://api.tcgplayer.com/catalog/products"
	tcgApiListGroupsURL   = "https://api.tcgplayer.com/catalog/groups"

	tcgApiCategoriesURL = "https://api.tcgplayer.com/catalog/categories/"

	tcgApiVersion    = "v1.39.0"
	tcgApiProductURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/product/"
	tcgApiPricingURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/sku/"
	tcgApiBuylistURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/buy/sku/"

	tcgApiSKUsURL   = "https://api.tcgplayer.com/" + tcgApiVersion + "/catalog/products/%s/skus"
	tcgApiSearchURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/catalog/categories/1/search"

	tcgLatestSalesURL = "https://mpapi.tcgplayer.com/v2/product/%s/latestsales"
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

const (
	CategoryMagic   = 1
	CategoryYuGiOh  = 2
	CategoryPokemon = 3

	ProductTypeCards               = "Cards"
	ProductTypeBoosterBox          = "Booster Box"
	ProductTypeBoosterPack         = "Booster Pack"
	ProductTypeSealedProducts      = "Sealed Products"
	ProductTypeIntroPack           = "Intro Pack"
	ProductTypeFatPack             = "Fat Pack"
	ProductTypeBoxSets             = "Box Sets"
	ProductTypePreconEventDecks    = "Precon/Event Decks"
	ProductTypeMagicDeckPack       = "Magic Deck Pack"
	ProductTypeMagicBoosterBoxCase = "Magic Booster Box Case"
	ProductTypeAll5IntroPacks      = "All 5 Intro Packs"
	ProductTypeIntroPackDisplay    = "Intro Pack Display"
	ProductType3xMagicBoosterPacks = "3x Magic Booster Packs"
	ProductTypeBoosterBattlePack   = "Booster Battle Pack"

	MaxLimit = 100
)

type TCGResponse struct {
	TotalItems int             `json:"totalItems"`
	Success    bool            `json:"success"`
	Errors     []string        `json:"errors"`
	Results    json.RawMessage `json:"results"`
}

// Perform an authenticated GET request on any URL
func (tcg *TCGClient) Get(url string) (*TCGResponse, error) {
	resp, err := tcg.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response TCGResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), string(data))
	}

	return &response, nil
}

type TCGProduct struct {
	ProductId    int      `json:"productId"`
	Name         string   `json:"name"`
	CleanName    string   `json:"cleanName"`
	ImageUrl     string   `json:"imageUrl"`
	GroupId      int      `json:"groupId"`
	URL          string   `json:"url"`
	ModifiedOn   string   `json:"modifiedOn"`
	Skus         []TCGSKU `json:"skus,omitempty"`
	ExtendedData []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Value       string `json:"value"`
	} `json:"extendedData,omitempty"`
}

func (tcg *TCGClient) TotalProducts(category int, productTypes []string) (int, error) {
	return tcg.queryTotal(tcgApiListProductsURL, category, productTypes)
}

func (tcg *TCGClient) queryTotal(link string, category int, productTypes []string) (int, error) {
	u, err := url.Parse(link)
	if err != nil {
		return 0, err
	}
	v := url.Values{}
	v.Set("categoryId", fmt.Sprint(category))
	if productTypes != nil {
		v.Set("productTypes", strings.Join(productTypes, ","))
	}
	v.Set("limit", fmt.Sprint(1))
	u.RawQuery = v.Encode()

	response, err := tcg.Get(u.String())
	if err != nil {
		return 0, err
	}
	return response.TotalItems, nil
}

func (tcg *TCGClient) ListAllProducts(category int, productTypes []string, includeSkus bool, offset int, limit int) ([]TCGProduct, error) {
	u, err := url.Parse(tcgApiListProductsURL)
	if err != nil {
		return nil, err
	}

	if limit > MaxLimit {
		return nil, errors.New("invalid limit parameter")
	}

	v := url.Values{}
	v.Set("getExtendedFields", "true")
	v.Set("categoryId", fmt.Sprint(category))
	if productTypes != nil {
		v.Set("productTypes", strings.Join(productTypes, ","))
	}
	if includeSkus {
		v.Set("includeSkus", "true")
	}
	v.Set("offset", fmt.Sprint(offset))

	v.Set("limit", fmt.Sprint(limit))
	u.RawQuery = v.Encode()

	resp, err := tcg.Get(u.String())
	if err != nil {
		return nil, err
	}

	var out []TCGProduct
	err = json.Unmarshal(resp.Results, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type TCGGroup struct {
	GroupID      int    `json:"groupId"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Supplemental bool   `json:"supplemental"`
	PublishedOn  string `json:"publishedOn"`
	ModifiedOn   string `json:"modifiedOn"`
	CategoryID   int    `json:"categoryId"`
}

func (tcg *TCGClient) TotalGroups(category int) (int, error) {
	return tcg.queryTotal(tcgApiListGroupsURL, category, nil)
}

func (tcg *TCGClient) ListAllGroups(category int, offset int, limit int) ([]TCGGroup, error) {
	u, err := url.Parse(tcgApiListGroupsURL)
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	v.Set("categoryId", fmt.Sprint(category))
	v.Set("offset", fmt.Sprint(offset))
	v.Set("limit", fmt.Sprint(limit))
	u.RawQuery = v.Encode()

	resp, err := tcg.Get(u.String())
	if err != nil {
		return nil, err
	}

	var out []TCGGroup
	err = json.Unmarshal(resp.Results, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (tcg *TCGClient) EditionMap(category int) (map[int]TCGGroup, error) {
	totals, err := tcg.TotalGroups(category)
	if err != nil {
		return nil, err
	}

	results := map[int]TCGGroup{}
	for i := 0; i < totals; i += MaxLimit {
		groups, err := tcg.ListAllGroups(category, i, MaxLimit)
		if err != nil {
			return nil, err
		}

		for _, group := range groups {
			results[group.GroupID] = group
		}
	}

	return results, nil
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
	resp, err := tcg.Get(tcgApiProductURL + strings.Join(productIds, ","))
	if err != nil {
		return nil, err
	}

	var out []TCGPrice
	err = json.Unmarshal(resp.Results, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type TCGSKU struct {
	SkuId       int `json:"skuId"`
	ProductId   int `json:"productId"`
	LanguageId  int `json:"languageId"`
	PrintingId  int `json:"printingId"`
	ConditionId int `json:"conditionId"`
}

func (tcg *TCGClient) SKUsForId(productId string) ([]TCGSKU, error) {
	resp, err := tcg.Get(fmt.Sprintf(tcgApiSKUsURL, productId))
	if err != nil {
		return nil, err
	}

	var out []TCGSKU
	err = json.Unmarshal(resp.Results, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
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
	resp, err := tcg.Get(tcgApiPricingURL + strings.Join(ids, ","))
	if err != nil {
		return nil, err
	}

	var out []TCGSKUPrice
	err = json.Unmarshal(resp.Results, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (tcg *TCGClient) TCGBuylistPricesForSKUs(ids []string) ([]TCGSKUPrice, error) {
	resp, err := tcg.Get(tcgApiBuylistURL + strings.Join(ids, ","))
	if err != nil {
		return nil, err
	}

	var out []TCGSKUPrice
	err = json.Unmarshal(resp.Results, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type TCGCategory struct {
	CategoryID        int    `json:"categoryId"`
	Name              string `json:"name"`
	ModifiedOn        string `json:"modifiedOn"`
	DisplayName       string `json:"displayName"`
	SeoCategoryName   string `json:"seoCategoryName"`
	SealedLabel       string `json:"sealedLabel"`
	NonSealedLabel    string `json:"nonSealedLabel"`
	ConditionGuideURL string `json:"conditionGuideUrl"`
	IsScannable       bool   `json:"isScannable"`
	Popularity        int    `json:"popularity"`
}

func (tcg *TCGClient) TCGCategoriesDetails(ids []int) ([]TCGCategory, error) {
	link := tcgApiCategoriesURL
	for i, id := range ids {
		if i != 0 {
			link += ","
		}
		link += fmt.Sprint(id)
	}

	resp, err := tcg.Get(link)
	if err != nil {
		return nil, err
	}

	var out []TCGCategory
	err = json.Unmarshal(resp.Results, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type latestSalesRequest struct {
	Variants    []string `json:"variants"`
	Conditions  []string `json:"conditions"`
	Languages   []string `json:"languages"`
	ListingType string   `json:"listingType"`
	Limit       int      `json:"limit"`
}

type latestSalesResponse struct {
	PreviousPage string            `json:"previousPage"`
	NextPage     string            `json:"nextPage"`
	ResultCount  int               `json:"resultCount"`
	TotalResults int               `json:"totalResults"`
	Data         []LatestSalesData `json:"data"`
}

type LatestSalesData struct {
	Condition       string    `json:"condition"`
	Variant         string    `json:"variant"`
	Language        string    `json:"language"`
	Quantity        int       `json:"quantity"`
	Title           string    `json:"title"`
	ListingType     string    `json:"listingType"`
	CustomListingID string    `json:"customListingId"`
	PurchasePrice   float64   `json:"purchasePrice"`
	ShippingPrice   float64   `json:"shippingPrice"`
	OrderDate       time.Time `json:"orderDate"`
}

const (
	defaultListingTypeLatestSales = "All"
	defaultLimitLastestSales      = 25
)

func TCGLatestSales(tcgProductId string, foil ...bool) (*latestSalesResponse, error) {
	link := fmt.Sprintf(tcgLatestSalesURL, tcgProductId)

	var params latestSalesRequest
	params.ListingType = defaultListingTypeLatestSales
	params.Limit = defaultLimitLastestSales

	if len(foil) > 0 {
		if foil[0] {
			params.Variants = []string{"2"}
		} else {
			params.Variants = []string{"1"}
		}
	}

	payload, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	resp, err := cleanhttp.DefaultClient().Post(link, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response latestSalesResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), string(data))
	}

	return &response, nil
}
