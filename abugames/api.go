package abugames

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-retryablehttp"
)

type ABUCard struct {
	Id           string `json:"id"`
	DisplayTitle string `json:"display_title"`
	SimpleTitle  string `json:"simple_title"`

	Edition   string `json:"magic_edition_sort"`
	Condition string `json:"condition"`
	Layout    string `json:"layout"`

	Rarity   string   `json:"rarity"`
	Language []string `json:"language"`
	Title    string   `json:"title"`
	Number   string   `json:"card_number"`

	SellPrice    float64 `json:"price"`
	SellQuantity int     `json:"quantity"`
	BuyQuantity  int     `json:"buy_list_quantity"`
	BuyPrice     float64 `json:"buy_price"`
	TradePrice   float64 `json:"trade_price"`
}

type ABUProduct struct {
	Grouped struct {
		ProductId struct {
			Count  int `json:"ngroups"`
			Groups []struct {
				GroupValue string `json:"groupValue"`
				Doclist    struct {
					Docs []ABUCard `json:"docs"`
				} `json:"doclist"`
			} `json:"groups"`
		} `json:"product_id"`
	} `json:"grouped"`
}

type ABUResponse struct {
	Response struct {
		NumFound      int       `json:"numFound"`
		Start         int       `json:"start"`
		NumFoundExact bool      `json:"numFoundExact"`
		Docs          []ABUCard `json:"docs"`
	} `json:"response"`
}

const (
	maxEntryPerRequest = 200

	abuBaseUrl = `https://data.abugames.com/solr/nodes/select?q=*:*&group=true&group.field=product_id&group.ngroups=true&group.limit=10&start=0&rows=0&wt=json&fq=%2Bcategory%3A%22Magic%20the%20Gathering%20Singles%22%20%2Blanguage%3A(%22English%22%20OR%20%22Italian%22%20OR%20%22Japanese%22%20OR%20%22Phyrexian%22)%20-offline_item%3Atrue%20-magic_features%3A(%22Actual%20Picture%20Card%22)`

	// This URL will include pics, but queries will be slower
	abuBaseUrlFull = `https://data.abugames.com/solr/nodes/select?q=*:*&group=true&group.field=product_id&group.ngroups=true&group.limit=10&start=0&rows=0&wt=json&fq=%2Bcategory%3A%22Magic%20the%20Gathering%20Singles%22%20%2Blanguage%3A(%22English%22%20OR%20%22Italian%22%20OR%20%22Japanese%22%20OR%20%22Phyrexian%22)%20-offline_item%3Atrue%20`

	abuBaseSealedUrl = `https://data.abugames.com/solr/nodes/select?q=*:*&fq=%2Bcategory%3A%22Magic%20the%20Gathering%20Sealed%20Product%22%20-offline_item%3Atrue%20OR%20-title%3A%22STORE%22%20OR%20-title%3A%22AUCTION%22%20OR%20-title%3A%22OVERSTOCK%22%20%2Blanguage_magic_sealed_product%3A(%22English%22)&sort=display_title%20asc&wt=json&start=0&rows=0`
)

type ABUClient struct {
	client        *http.Client
	authorization string
}

func NewABUClient() *ABUClient {
	abu := ABUClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	abu.client = client.StandardClient()
	return &abu
}

func NewABUClientWithBearer(token string) *ABUClient {
	abu := NewABUClient()
	abu.authorization = token
	return abu
}

func (abu *ABUClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if abu.authorization != "" {
		req.Header.Set("Authorization", "Bearer "+abu.authorization)
	}

	return abu.client.Do(req)
}

func (abu *ABUClient) Post(url, contentType string, reader io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	if abu.authorization != "" {
		req.Header.Set("Authorization", "Bearer "+abu.authorization)
	}

	return abu.client.Do(req)
}

func (abu *ABUClient) sendRequest(url string) (*ABUProduct, error) {
	resp, err := abu.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var product ABUProduct
	err = json.NewDecoder(resp.Body).Decode(&product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (abu *ABUClient) sendSealedRequest(url string) (*ABUResponse, error) {
	resp, err := abu.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response ABUResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// Use the URL as is, with just one row requested to fetch the number of items
func (abu *ABUClient) GetTotalItems(extra string) (int, error) {
	link := abuBaseUrl
	if extra != "" {
		link = abuBaseUrlFull + url.QueryEscape(extra)
	}
	product, err := abu.sendRequest(link)
	if err != nil {
		return 0, err
	}
	return product.Grouped.ProductId.Count, nil
}

func (abu *ABUClient) GetTotalSealedItems() (int, error) {
	response, err := abu.sendSealedRequest(abuBaseSealedUrl)
	if err != nil {
		return 0, err
	}
	return response.Response.NumFound, nil
}

func (abu *ABUClient) GetProduct(extra string, pageStart int) (*ABUProduct, error) {
	link := abuBaseUrl
	if extra != "" {
		link = abuBaseUrlFull + url.QueryEscape(extra)
	}
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("rows", fmt.Sprintf("%d", maxEntryPerRequest))
	q.Set("start", fmt.Sprintf("%d", pageStart))
	u.RawQuery = q.Encode()

	return abu.sendRequest(u.String())
}

func (abu *ABUClient) GetSealedProduct(pageStart int) (*ABUResponse, error) {
	u, err := url.Parse(abuBaseSealedUrl)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("rows", fmt.Sprintf("%d", maxEntryPerRequest))
	q.Set("start", fmt.Sprintf("%d", pageStart))
	u.RawQuery = q.Encode()

	return abu.sendSealedRequest(u.String())
}

type CartRequest struct {
	ItemId   string `json:"item_id"`
	Quantity int    `json:"quantity"`
	// Ignored on buylist
	Call string `json:"call,omitempty"`
}

type CartResponse struct {
	BuyList string `json:"buyList"`
	NqData  struct {
		Maxqty int `json:"maxqty"`
	} `json:"nqData"`
	ConditionRowID int `json:"condition_row_id"`
	Resp           struct {
		Exception any   `json:"exception"`
		Headers   []any `json:"headers"`
		Original  any   `json:"original"`
	} `json:"resp"`

	Message    string `json:"message"`
	Code       string `json:"code"`
	StatusCode int    `json:"status_code"`
}

const (
	abuInventoryAddURL = "https://api.abugames.com/cart/item"
	abuBuylistAddURL   = "https://api.abugames.com/buy-list-cart/item"
)

func (abu *ABUClient) SetCartInventory(abuId string, qty int) (*CartResponse, error) {
	return abu.setCart(abuInventoryAddURL, abuId, qty)
}

func (abu *ABUClient) SetCartBuylist(abuId string, qty int) (*CartResponse, error) {
	return abu.setCart(abuBuylistAddURL, abuId, qty)
}

func (abu *ABUClient) setCart(link, abuId string, qty int) (*CartResponse, error) {
	payload := CartRequest{
		ItemId:   abuId,
		Quantity: qty,
		Call:     "add",
	}

	reqBody, err := json.Marshal(&payload)
	if err != nil {
		return nil, err
	}

	resp, err := abu.Post(link, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CartResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	if len(response.Message) > 0 {
		return nil, errors.New(response.Message)
	}

	return &response, nil
}
