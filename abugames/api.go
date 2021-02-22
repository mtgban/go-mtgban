package abugames

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	http "github.com/hashicorp/go-retryablehttp"
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
				Doclist struct {
					Cards []ABUCard `json:"docs"`
				} `json:"doclist"`
			} `json:"groups"`
		} `json:"product_id"`
	} `json:"grouped"`
}

const (
	maxEntryPerRequest = 40

	abuBaseUrl = "https://data.abugames.com/solr/nodes/select?q=*:*&fq=%2Bcategory%3A%22Magic%20the%20Gathering%20Singles%22%20%20-buy_price%3A0%20-buy_list_quantity%3A0%20%2Blanguage%3A(%22English%22%2C%20%22Italian%22%2C%20%22Japanese%22)%20%2Bdisplay_title%3A*&group=true&group.field=product_id&group.ngroups=true&group.limit=10&start=0&rows=0&wt=json"
)

type ABUClient struct {
	client *http.Client
}

func NewABUClient() *ABUClient {
	abu := ABUClient{}
	abu.client = http.NewClient()
	abu.client.Logger = nil
	return &abu
}

func (abu *ABUClient) sendRequest(url string) (*ABUProduct, error) {
	resp, err := abu.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var product ABUProduct
	err = json.Unmarshal(data, &product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (abu *ABUClient) GetInfo() (*ABUProduct, error) {
	return abu.sendRequest(abuBaseUrl)
}

func (abu *ABUClient) GetProduct(pageStart int) (*ABUProduct, error) {
	u, err := url.Parse(abuBaseUrl)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("rows", fmt.Sprintf("%d", maxEntryPerRequest))
	q.Set("start", fmt.Sprintf("%d", pageStart))
	u.RawQuery = q.Encode()

	return abu.sendRequest(u.String())
}
