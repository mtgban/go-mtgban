package tcgplayer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	tcgplayer "github.com/mtgban/go-tcgplayer"
)

const (
	defaultConcurrency = 8

	tcgLatestSalesURL = "https://mpapi.tcgplayer.com/v2/product/%s/latestsales"
)

func GetProductNumber(tcgp *tcgplayer.Product) string {
	for _, extData := range tcgp.ExtendedData {
		if extData.Name == "Number" {
			num := strings.TrimLeft(extData.Value, "0")
			num = strings.Split(num, "/")[0]
			return num
		}
	}
	return ""
}

func GetProductNameAndVariant(tcgp *tcgplayer.Product) (string, string) {
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

func isToken(tcgp *tcgplayer.Product) bool {
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

func EditionMap(tcg *tcgplayer.Client, category int) (map[int]tcgplayer.Group, error) {
	totals, err := tcg.TotalGroups(category)
	if err != nil {
		return nil, err
	}

	results := map[int]tcgplayer.Group{}
	for i := 0; i < totals; i += tcgplayer.MaxItemsInResponse {
		groups, err := tcg.ListAllCategoryGroups(category, i)
		if err != nil {
			return nil, err
		}

		for _, group := range groups {
			results[group.GroupID] = group
		}
	}

	return results, nil
}

var SKUConditionMap = map[int]string{
	1: "NM",
	2: "SP",
	3: "MP",
	4: "HP",
	5: "PO",
}

type latestSalesRequest struct {
	Variants    []int  `json:"variants"`
	Conditions  []int  `json:"conditions"`
	Languages   []int  `json:"languages"`
	ListingType string `json:"listingType"`
	Limit       int    `json:"limit"`
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

func LatestSales(tcgProductId string, flags ...bool) (*latestSalesResponse, error) {
	link := fmt.Sprintf(tcgLatestSalesURL, tcgProductId)

	var params latestSalesRequest
	params.ListingType = defaultListingTypeLatestSales
	params.Limit = defaultLimitLastestSales

	if len(flags) > 0 {
		foil := flags[0]
		if foil {
			params.Variants = []int{2, 133, 141}
		} else {
			params.Variants = []int{1, 132}
		}
	}

	anyLang := len(flags) > 1 && flags[1]
	if !anyLang {
		// 1 being English
		params.Languages = []int{1}
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

type SellerClient struct {
	client *retryablehttp.Client
}

func NewSellerClient() *SellerClient {
	tcg := SellerClient{}
	tcg.client = retryablehttp.NewClient()
	tcg.client.Logger = nil
	return &tcg
}

func (tcg *SellerClient) InventoryForSeller(sellerKeys []string, size, page int, useDirect bool, finishes []string, sets ...string) (*sellerInventoryResponse, error) {
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

func (tcg *SellerClient) InventoryListing(productId, size, page int, useDirect bool) ([]SellerListing, error) {
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
	client     *http.Client
	cookieLine string
}

func NewCookieClient(authKey string) *CookieClient {
	client := retryablehttp.NewClient()
	client.Logger = nil
	tcg := CookieClient{}
	tcg.cookieLine = "TCGAuthTicket_Production=" + authKey + ";"
	tcg.client = client.StandardClient()
	return &tcg
}

func NewCookieSetClient(cookies map[string]string) *CookieClient {
	client := retryablehttp.NewClient()
	client.Logger = nil
	tcg := CookieClient{}
	for name, value := range cookies {
		tcg.cookieLine += fmt.Sprintf("%s=%s; ", name, value)
	}
	tcg.client = client.StandardClient()
	return &tcg
}

type UserData struct {
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

type UserResponse struct {
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Results []UserData `json:"results"`
}

const tcgUserDataURL = "https://mpapi.tcgplayer.com/v2/user?isGuest=false"

func (tcg *CookieClient) Get(link string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", tcg.cookieLine)
	req.Header.Add("User-Agent", "curl/8.6.0")

	return tcg.client.Do(req)
}

func (tcg *CookieClient) Post(link, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, link, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", tcg.cookieLine)
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("User-Agent", "curl/8.6.0")

	return tcg.client.Do(req)
}

func (tcg *CookieClient) Delete(link string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", tcg.cookieLine)
	req.Header.Add("User-Agent", "curl/8.6.0")

	return tcg.client.Do(req)
}

func (tcg *CookieClient) GetUserData() (*UserData, error) {
	resp, err := tcg.Get(tcgUserDataURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response UserResponse
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

const tcgCreateCartURL = "https://mpgateway.tcgplayer.com/v1/cart/create/usercart"

func CreateCartKey(userId string) (string, error) {
	var params struct {
		ExternalUserId string `json:"externalUserId"`
	}
	params.ExternalUserId = userId

	payload, err := json.Marshal(&params)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, tcgCreateCartURL, bytes.NewReader(payload))
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

	var response UserResponse
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

func (tcg *CookieClient) EmptyCart(cartKey string) error {
	link := fmt.Sprintf("https://mpgateway.tcgplayer.com/v1/cart/%s/items/all", cartKey)

	resp, err := tcg.Delete(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

type AddressConfig struct {
	FirstName                 string `json:"firstName"`
	LastName                  string `json:"lastName"`
	AddressLine1              string `json:"addressLine1"`
	AddressLine2              string `json:"addressLine2"`
	City                      string `json:"city"`
	ZipCode                   string `json:"zipCode"`
	StateProvinceRegion       string `json:"stateProvinceRegion"`
	Phone                     string `json:"phone"`
	IsDefaultAddress          bool   `json:"isDefaultAddress"`
	SaveAddressOnPaymentSave  bool   `json:"saveAddressOnPaymentSave"`
	ExternalUserID            string `json:"externalUserId"`
	CountryCode               string `json:"countryCode"`
	ID                        int    `json:"id"`
	EasyPostShippingAddressID string `json:"easyPostShippingAddressId"`
	IsEasyPostVerified        bool   `json:"isEasyPostVerified"`
	CreatedAt                 string `json:"createdAt"`
	LastUsedAt                string `json:"lastUsedAt"`
}

const (
	addressUpdateURL = "https://mpgateway.tcgplayer.com/v2/useraddressbooks/update"
)

func (tcg *CookieClient) SetAddress(address AddressConfig) error {
	payload, err := json.Marshal(&address)
	if err != nil {
		return err
	}

	resp, err := tcg.Post(addressUpdateURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(http.StatusText(resp.StatusCode))
	}

	return nil
}
