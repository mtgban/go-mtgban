package tcgplayer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"golang.org/x/time/rate"
)

const (
	defaultConcurrency = 8
	defaultAPIRetry    = 5
	maxIdsInRequest    = 250

	tcgApiTokenURL = "https://api.tcgplayer.com/token"

	tcgApiListProductsURL = "https://api.tcgplayer.com/catalog/products"
	tcgApiListGroupsURL   = "https://api.tcgplayer.com/catalog/groups"

	tcgApiCategoriesURL = "https://api.tcgplayer.com/catalog/categories/"
	tcgApiPrintingsURL  = "https://api.tcgplayer.com/catalog/categories/%d/printings"

	tcgApiVersion    = "v1.39.0"
	tcgApiProductURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/product/"
	tcgApiPricingURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/sku/"
	tcgApiBuylistURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/buy/sku/"
	tcgApiSKUsURL    = "https://api.tcgplayer.com/" + tcgApiVersion + "/catalog/products/%s/skus"

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

	data, err := io.ReadAll(resp.Body)
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
	CategoryLorcana = 71

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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response TCGResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), string(data))
	}
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf(strings.Join(response.Errors, " "))
	}

	return &response, nil
}

type TCGProduct struct {
	ProductId  int      `json:"productId"`
	Name       string   `json:"name"`
	CleanName  string   `json:"cleanName"`
	ImageUrl   string   `json:"imageUrl"`
	GroupId    int      `json:"groupId"`
	URL        string   `json:"url"`
	ModifiedOn string   `json:"modifiedOn"`
	Skus       []TCGSKU `json:"skus,omitempty"`

	// Only available for catalog/products API calls
	ExtendedData []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Value       string `json:"value"`
	} `json:"extendedData,omitempty"`
}

func (tcgp *TCGProduct) GetNumber() string {
	for _, extData := range tcgp.ExtendedData {
		if extData.Name == "Number" {
			num := strings.TrimLeft(extData.Value, "0")
			num = strings.Split(num, "/")[0]
			return num
		}
	}
	return ""
}

func (tcgp *TCGProduct) GetNameAndVariant() (string, string) {
	cardName := tcgp.Name
	variant := ""

	if strings.Contains(cardName, " - ") {
		fields := strings.Split(cardName, " - ")
		cardName = fields[0]
		if len(fields) > 1 {
			variant = strings.Join(fields[1:], " ")
		}
	}
	if strings.Contains(cardName, " [") {
		cardName = strings.Replace(cardName, "[", "(", -1)
		cardName = strings.Replace(cardName, "]", ")", -1)
	}
	if strings.Contains(cardName, " (") {
		fields := mtgmatcher.SplitVariants(cardName)
		cardName = fields[0]
		if len(fields) > 1 {
			if variant != "" {
				variant += " "
			}
			variant += strings.Join(fields[1:], " ")

			variant = strings.TrimSuffix(variant, " CE")
			variant = strings.TrimSuffix(variant, " IE")
		}
	}

	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)

	return cardName, variant
}

func (tcgp *TCGProduct) IsToken() bool {
	for _, extData := range tcgp.ExtendedData {
		if extData.Name == "SubType" && strings.Contains(extData.Value, "Token") {
			return true
		}
	}
	// There are some tokens not marked as such
	if strings.Contains(tcgp.CleanName, "Token") {
		return true
	}
	return false
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

type TCGPrinting struct {
	PrintingId int    `json:"printingId"`
	Name       string `json:"name"`
}

func (tcg *TCGClient) ListCategoryPrintings(category int) ([]TCGPrinting, error) {
	resp, err := tcg.Get(fmt.Sprintf(tcgApiPrintingsURL, category))
	if err != nil {
		return nil, err
	}

	var out []TCGPrinting
	err = json.Unmarshal(resp.Results, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (tcg *TCGClient) ListProducts(productIds []int, includeSkus bool) ([]TCGProduct, error) {
	link := tcgApiListProductsURL + "/"
	for _, pid := range productIds {
		link += fmt.Sprintf("%d,", pid)
	}
	link = strings.TrimLeft(link, ",")

	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	v := url.Values{}
	v.Set("getExtendedFields", "true")
	if includeSkus {
		v.Set("includeSkus", "true")
	}

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

var SKUConditionMap = map[int]string{
	1: "NM",
	2: "SP",
	3: "MP",
	4: "HP",
	5: "PO",
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

	data, err := io.ReadAll(resp.Body)
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

const (
	SellersPageURL     = "https://shop.tcgplayer.com/sellers"
	SellerInventoryURL = "https://mp-search-api.tcgplayer.com/v1/search/request?q=&isList=true&mpfev=1953"
	SellerListingURL   = "https://mp-search-api.tcgplayer.com/v1/product/%d/listings"

	DefaultSellerRequestSize = 50

	MaxGlobalScrapingValue = 8000
)

func SellerKeyExists(sellerKey string) bool {
	client := cleanhttp.DefaultClient()

	// Do not follow redirects
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Get("https://shop.tcgplayer.com/sellerfeedback/" + sellerKey)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func SellerName2ID(sellerName string) (string, error) {
	if sellerName == "" {
		return "", errors.New("missing seller name")
	}

	v := url.Values{}
	v.Set("name", "foo")
	v.Set("SellerName", sellerName)
	v.Set("isDirect", "false")
	v.Set("isCertified", "false")
	v.Set("isGoldStar", "false")
	v.Set("categoryId", "1") // 1 = mtg
	v.Set("returnUrl", "")

	req, err := http.NewRequest(http.MethodPost, SellersPageURL, strings.NewReader(v.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	link, ok := doc.Find(`div.scTitle a`).Attr("href")
	if !ok {
		return "", errors.New("not found")
	}

	return strings.TrimPrefix(link, "/sellerfeedback/"), nil
}

type sellerInventoryRequest struct {
	Algorithm string `json:"algorithm"`
	From      int    `json:"from"`
	Size      int    `json:"size"`
	Filters   struct {
		Term struct {
			ProductLineName []string `json:"productLineName,omitempty"`
			ProductTypeName []string `json:"productTypeName,omitempty"`
			SetName         []string `json:"setName,omitempty"`
		} `json:"term"`
		Range struct {
		} `json:"range"`
		Match struct {
		} `json:"match"`
	} `json:"filters"`
	ListingSearch struct {
		Filters struct {
			Term struct {
				SellerStatus  string   `json:"sellerStatus,omitempty"`
				ChannelID     int      `json:"channelId"`
				DirectSeller  bool     `json:"direct-seller,omitempty"`
				DirectProduct bool     `json:"directProduct,omitempty"`
				SellerKey     []string `json:"sellerKey,omitempty"`
				Printing      []string `json:"printing,omitempty"`
			} `json:"term"`
			Range struct {
				Quantity struct {
					GreaterThanOrEqual int `json:"gte"`
				} `json:"quantity"`
				DirectInventory struct {
					GreaterThanOrEqual int `json:"gte"`
				} `json:"directInventory"`
			} `json:"range"`
			Exclude struct {
				ChannelExclusion int `json:"channelExclusion"`
			} `json:"exclude"`
		} `json:"filters"`
		Context struct {
			Cart struct {
			} `json:"cart"`
		} `json:"context"`
	} `json:"listingSearch"`
	Context struct {
		Cart struct {
		} `json:"cart"`
		ShippingCountry string `json:"shippingCountry"`
	} `json:"context"`
	Settings struct {
		UseFuzzySearch bool `json:"useFuzzySearch"`
	} `json:"settings"`
	Sort struct {
		Field string `json:"field"`
		Order string `json:"order"`
	} `json:"sort"`
}

type sellerInventoryResponse struct {
	Title  string `json:"title"`
	Status int    `json:"status"`

	Results []struct {
		Aggregations struct {
			SetName []struct {
				URLValue string  `json:"urlValue"`
				IsActive bool    `json:"isActive"`
				Value    string  `json:"value"`
				Count    float64 `json:"count"`
			} `json:"setName"`
		} `json:"aggregations"`
		TotalResults int                     `json:"totalResults"`
		ResultID     string                  `json:"resultId"`
		Algorithm    string                  `json:"algorithm"`
		SearchType   string                  `json:"searchType"`
		Results      []SellerInventoryResult `json:"results"`
	} `json:"results"`
}

type SellerListing struct {
	ChannelID            float64 `json:"channelId"`
	Condition            string  `json:"condition"`
	ConditionID          float64 `json:"conditionId"`
	DirectInventory      float64 `json:"directInventory"`
	DirectProduct        bool    `json:"directProduct"`
	DirectSeller         bool    `json:"directSeller"`
	ForwardFreight       bool    `json:"forwardFreight"`
	GoldSeller           bool    `json:"goldSeller"`
	Language             string  `json:"language"`
	LanguageAbbreviation string  `json:"languageAbbreviation"`
	LanguageID           float64 `json:"languageId"`
	ListingID            float64 `json:"listingId"`
	ListingType          string  `json:"listingType"`
	Price                float64 `json:"price"`
	Printing             string  `json:"printing"`
	ProductConditionID   float64 `json:"productConditionId"`
	ProductID            float64 `json:"productId"`
	Quantity             float64 `json:"quantity"`
	RankedShippingPrice  float64 `json:"rankedShippingPrice"`
	Score                float64 `json:"score"`
	SellerID             string  `json:"sellerId"`
	SellerKey            string  `json:"sellerKey"`
	SellerName           string  `json:"sellerName"`
	SellerRating         float64 `json:"sellerRating"`
	SellerSales          string  `json:"sellerSales"`
	SellerShippingPrice  float64 `json:"sellerShippingPrice"`
	ShippingPrice        float64 `json:"shippingPrice"`
	VerifiedSeller       bool    `json:"verifiedSeller"`
}

type SellerInventoryResult struct {
	FoilOnly                bool    `json:"foilOnly"`
	ImageCount              float64 `json:"imageCount"`
	LowestPrice             float64 `json:"lowestPrice"`
	LowestPriceWithShipping float64 `json:"lowestPriceWithShipping"`
	MarketPrice             float64 `json:"marketPrice"`
	MaxFulfillableQuantity  float64 `json:"maxFulfillableQuantity"`
	NormalOnly              bool    `json:"normalOnly"`
	ProductID               float64 `json:"productId"`
	ProductLineID           float64 `json:"productLineId"`
	ProductLineName         string  `json:"productLineName"`
	ProductLineURLName      string  `json:"productLineUrlName"`
	ProductName             string  `json:"productName"`
	ProductStatusID         float64 `json:"productStatusId"`
	ProductTypeID           float64 `json:"productTypeId"`
	ProductTypeName         string  `json:"productTypeName"`
	ProductURLName          string  `json:"productUrlName"`
	RarityName              string  `json:"rarityName"`
	Score                   float64 `json:"score"`
	Sealed                  bool    `json:"sealed"`
	SellerListable          bool    `json:"sellerListable"`
	Sellers                 float64 `json:"sellers"`
	SetCode                 string  `json:"setCode"`
	SetID                   float64 `json:"setId"`
	SetName                 string  `json:"setName"`
	SetURLName              string  `json:"setUrlName"`
	ShippingCategoryID      float64 `json:"shippingCategoryId"`
	TotalListings           float64 `json:"totalListings"`
	CustomAttributes        struct {
		Number string `json:"number"`
	} `json:"customAttributes"`
	Listings []SellerListing `json:"listings"`
}

func NewTCGSellerClient() *TCGClient {
	tcg := TCGClient{}
	tcg.client = retryablehttp.NewClient()
	tcg.client.Logger = nil
	return &tcg
}

func (tcg *TCGClient) TCGInventoryForSeller(sellerKeys []string, size, page int, useDirect bool, finishes []string, sets ...string) (*sellerInventoryResponse, error) {
	var params sellerInventoryRequest
	params.Algorithm = "revenue_synonym_v2"
	params.From = size * page
	params.Size = size
	params.Filters.Term.ProductLineName = []string{"magic"}
	params.Filters.Term.ProductTypeName = []string{"Cards"}
	params.Filters.Term.SetName = sets
	params.ListingSearch.Filters.Term.SellerStatus = "Live"
	params.ListingSearch.Filters.Term.SellerKey = sellerKeys
	params.ListingSearch.Filters.Term.Printing = finishes
	if useDirect {
		params.ListingSearch.Filters.Term.DirectProduct = true
		params.ListingSearch.Filters.Term.DirectSeller = true
		params.ListingSearch.Filters.Range.DirectInventory.GreaterThanOrEqual = 1
	}
	params.ListingSearch.Filters.Range.Quantity.GreaterThanOrEqual = 1
	params.Context.ShippingCountry = "US"
	params.Settings.UseFuzzySearch = true
	params.Sort.Field = "product-sorting-name"
	params.Sort.Order = "asc"

	payload, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	resp, err := tcg.client.Post(SellerInventoryURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response sellerInventoryResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), string(data))
	}

	if response.Title != "" {
		return nil, fmt.Errorf("error: %s (%d)", response.Title, response.Status)
	}
	if len(response.Results) == 0 {
		return nil, fmt.Errorf("emtpy results in response at page %d", page)
	}

	return &response, nil
}

type sellerInventoryListingRequest struct {
	Filters struct {
		Term struct {
			SellerStatus  string   `json:"sellerStatus"`
			ChannelID     int      `json:"channelId"`
			DirectSeller  bool     `json:"direct-seller,omitempty"`
			DirectProduct bool     `json:"directProduct,omitempty"`
			Language      []string `json:"language,omitempty"`
		} `json:"term"`
		Range struct {
			Quantity struct {
				Gte int `json:"gte"`
			} `json:"quantity"`
			DirectInventory struct {
				Gte int `json:"gte,omitempty"`
			} `json:"directInventory,omitempty"`
		} `json:"range"`
		Exclude struct {
			ChannelExclusion int `json:"channelExclusion"`
		} `json:"exclude"`
	} `json:"filters"`
	Context struct {
		ShippingCountry string `json:"shippingCountry"`
		Cart            struct {
		} `json:"cart"`
	} `json:"context"`
	Aggregations []string `json:"aggregations"`
	From         int      `json:"from"`
	Size         int      `json:"size"`
	Sort         struct {
		Field string `json:"field"`
		Order string `json:"order"`
	} `json:"sort"`
}

type sellerInventoryListingResponse struct {
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Results []struct {
		TotalResults int             `json:"totalResults"`
		Results      []SellerListing `json:"results"`
	} `json:"results"`
}

func (tcg *TCGClient) TCGInventoryListing(productId, size, page int, useDirect bool) ([]SellerListing, error) {
	var params sellerInventoryListingRequest
	params.Filters.Term.SellerStatus = "Live"
	if useDirect {
		params.Filters.Term.SellerStatus = "Live"
		params.Filters.Term.DirectSeller = true
		params.Filters.Term.DirectProduct = true
		params.Filters.Term.Language = []string{"English"}
		params.Filters.Range.DirectInventory.Gte = 1
	}
	params.Filters.Range.Quantity.Gte = 1
	params.Context.ShippingCountry = "US"
	params.Aggregations = []string{"listingType"}
	params.From = size * page
	params.Size = size
	params.Sort.Field = "price"
	params.Sort.Order = "asc"

	payload, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	link := fmt.Sprintf(SellerListingURL, productId)
	resp, err := tcg.client.Post(link, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response sellerInventoryListingResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), string(data))
	}
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("%s: %s", response.Errors[0].Code, response.Errors[0].Message)
	}
	if len(response.Results) == 0 {
		return nil, fmt.Errorf("emtpy results in response at page %d", page)
	}

	return response.Results[0].Results, nil
}

type CookieClient struct {
	client  *retryablehttp.Client
	authKey string
}

func NewCookieClient(authKey string) *CookieClient {
	tcg := CookieClient{}
	tcg.client = retryablehttp.NewClient()
	tcg.client.Logger = nil
	tcg.authKey = authKey
	return &tcg
}

type TCGUserData struct {
	UserName                string `json:"userName"`
	UserID                  int    `json:"userId"`
	UserKey                 string `json:"userKey"`
	IsSubscriber            bool   `json:"isSubscriber"`
	LastUsedShippingAddress struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Address1  string `json:"address1"`
		Address2  string `json:"address2"`
		City      string `json:"city"`
		State     string `json:"state"`
		Zipcode   string `json:"zipcode"`
		Country   string `json:"country"`
		Phone     string `json:"phone"`
	} `json:"lastUsedShippingAddress"`
	ShippingCountry     string `json:"shippingCountry"`
	CreatedAt           string `json:"createdAt"`
	ExternalUserID      string `json:"externalUserId"`
	IsGuest             any    `json:"isGuest"`
	IsValidated         bool   `json:"isValidated"`
	CartKey             string `json:"cartKey"`
	SaveForLaterKey     string `json:"saveForLaterKey"`
	ProductLineAffinity any    `json:"productLineAffinity"`
	Traits              struct {
		ProductLineAffinityMostViewed any `json:"product_line_affinity_most_viewed"`
		Apv                           int `json:"apv"`
	} `json:"traits"`
	SellerKeys []string `json:"sellerKeys"`
}

type TCGUserResponse struct {
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Results []TCGUserData `json:"results"`
}

const TCGUserDataURL = "https://mpapi.tcgplayer.com/v2/user?isGuest=false"

func (tcg *CookieClient) GetUserData() (*TCGUserData, error) {
	req, err := http.NewRequest(http.MethodGet, TCGUserDataURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", "TCGAuthTicket_Production="+tcg.authKey+";")

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response TCGUserResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), string(data))
	}
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("%s: %s", response.Errors[0].Code, response.Errors[0].Message)
	}
	if len(response.Results) == 0 {
		return nil, fmt.Errorf("emtpy results in user request")
	}

	return &response.Results[0], nil
}

const TCGCreateCartURL = "https://mpgateway.tcgplayer.com/v1/cart/create/usercart"

func TCGCreateCartKey(userId string) (string, error) {
	var params struct {
		ExternalUserId string `json:"externalUserId"`
	}
	params.ExternalUserId = userId

	payload, err := json.Marshal(&params)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, TCGCreateCartURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response TCGUserResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return "", fmt.Errorf("%s: %s", err.Error(), string(data))
	}
	if len(response.Errors) > 0 {
		return "", fmt.Errorf("%s: %s", response.Errors[0].Code, response.Errors[0].Message)
	}
	if len(response.Results) == 0 {
		return "", fmt.Errorf("emtpy results in user request")
	}

	return response.Results[0].CartKey, nil

}

const TCGBuylistUpdateQtyURL = "https://store.tcgplayer.com/buylist/updatequantity"

func (tcg *CookieClient) BuylistSetQuantity(skuId, qty int) (string, error) {
	payload := url.Values{}
	payload.Set("productConditionId", fmt.Sprint(skuId))
	payload.Set("reqQty", fmt.Sprint(qty))
	payload.Set("availQty", "0")
	payload.Set("overrideQty", "true")

	req, err := http.NewRequest(http.MethodPost, TCGBuylistUpdateQtyURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Cookie", "TCGAuthTicket_Production="+tcg.authKey+";")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("buylist not ok")
	}

	// Nothing interesting in the response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	response := string(data)

	return response, nil
}

const TCGBuylistOffersURL = "https://store.tcgplayer.com/buylist/viewtopsellerprices?productConditionId="

type TCGBuylistOffer struct {
	SkuId      int
	SellerName string
	Price      float64
	Quantity   int
}

func (tcg *CookieClient) BuylistViewOffers(skuId int) ([]TCGBuylistOffer, error) {
	req, err := http.NewRequest(http.MethodGet, TCGBuylistOffersURL+fmt.Sprint(skuId), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", "TCGAuthTicket_Production="+tcg.authKey+";")

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var offers []TCGBuylistOffer
	doc.Find(`tbody`).Find(`tr`).Each(func(i int, s *goquery.Selection) {
		sellerName := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
		offerPrice := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
		offerQty := strings.TrimSpace(s.Find("td:nth-child(3)").Text())

		price, _ := mtgmatcher.ParsePrice(offerPrice)
		qty, _ := strconv.Atoi(offerQty)

		if price == 0 || qty == 0 {
			return
		}

		offers = append(offers, TCGBuylistOffer{
			SkuId:      skuId,
			SellerName: sellerName,
			Price:      price,
			Quantity:   qty,
		})
	})

	if len(offers) == 0 {
		return nil, errors.New("no offers in buylist")
	}

	return offers, nil
}
