package starcitygames

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
)

type SCGCard struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Subtitle  string `json:"subtitle"`
	Condition string `json:"condition"`
	Foil      bool   `json:"foil"`
	Language  string `json:"language"`
	Price     string `json:"price"`
	Rarity    string `json:"rarity"`
	Image     string `json:"image"`
}

type SCGSearch struct {
	Ok      bool        `json:"ok"`
	Msg     string      `json:"msg"`
	Edition string      `json:"search"`
	Results [][]SCGCard `json:"results"`
}

const (
	scgInventoryURL = "https://lusearchapi-na.hawksearch.com/sites/starcitygames/?instockonly=Yes&mpp=1&product_type=Singles"
	scgBuylistURL   = "https://old.starcitygames.com/buylist/search?search-type=category&id="

	scgDefaultPages = 200
)

type SCGClient struct {
	client *retryablehttp.Client
}

func NewSCGClient() *SCGClient {
	scg := SCGClient{}
	scg.client = retryablehttp.NewClient()
	scg.client.Logger = nil
	return &scg
}

func (scg *SCGClient) NumberOfItems() (int, error) {
	resp, err := scg.client.Get(scgInventoryURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	pagination := doc.Find(`div[id="hawktoppager"]`).Find(`div[class="hawk-searchrange"] span`).Text()
	items := strings.Split(pagination, " of ")
	if len(items) > 1 {
		return strconv.Atoi(items[1])
	}

	return 0, fmt.Errorf("invalid pagination value: %s", pagination)
}

func (scg *SCGClient) GetPage(page int) (*http.Response, error) {
	u, err := url.Parse(scgInventoryURL)
	if err != nil {
		return nil, err
	}
	v := u.Query()
	v.Set("mpp", fmt.Sprint(scgDefaultPages))
	v.Set("pg", fmt.Sprint(page))
	u.RawQuery = v.Encode()

	return scg.client.Get(u.String())
}

func (scg *SCGClient) SearchProduct(product string) (*SCGSearch, error) {
	resp, err := scg.client.Get(scgBuylistURL + product)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var search SCGSearch
	err = json.Unmarshal(data, &search)
	if err != nil {
		return nil, err
	}
	if !search.Ok {
		return nil, fmt.Errorf("%s", search.Msg)
	}
	if search.Results == nil {
		return nil, fmt.Errorf("product %s not found", product)
	}

	return &search, nil
}
