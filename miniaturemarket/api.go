package miniaturemarket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-retryablehttp"
)

type MMProduct struct {
	UUID     string  `json:"uniqueId"`
	EntityId string  `json:"entity_id"`
	Edition  string  `json:"mtg_set"`
	Title    string  `json:"title"`
	URL      string  `json:"productUrl"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type MMSearchResponse struct {
	Response struct {
		NumberOfProducts int         `json:"numberOfProducts"`
		Products         []MMProduct `json:"products"`
	} `json:"response"`
}

type MMClient struct {
	client *http.Client
}

const (
	MMCategoryMtgSingles    = "1466"
	MMDefaultResultsPerPage = 32

	mmSearchURL = "https://search.unbxd.io/fb500edbf5c28edfa74cc90561fe33c3/prod-miniaturemarket-com811741582229555/category"
)

func NewMMClient() *MMClient {
	mm := MMClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	mm.client = client.StandardClient()
	return &mm
}

func (mm *MMClient) NumberOfProducts() (int, error) {
	response, err := mm.query(0, 0)
	if err != nil {
		return 0, err
	}
	return response.Response.NumberOfProducts, nil
}

func (mm *MMClient) GetInventory(start int) (*MMSearchResponse, error) {
	return mm.query(start, MMDefaultResultsPerPage)
}

func (mm *MMClient) query(start, maxResults int) (*MMSearchResponse, error) {
	u, err := url.Parse(mmSearchURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("format", "json")
	q.Set("version", "V2")
	q.Set("start", fmt.Sprint(start))
	q.Set("rows", fmt.Sprint(maxResults))
	q.Set("variants", "true")
	q.Set("variants.count", "10")
	q.Set("fields", "*")
	q.Set("facet.multiselect", "true")
	q.Set("selectedfacet", "true")
	q.Set("pagetype", "boolean")
	q.Set("p", `categoryPath:"Trading Card Games"`)
	q.Add("filter", `categoryPath1_fq:"Trading Card Games"`)
	q.Add("filter", `categoryPath2_fq:"Trading Card Games>Magic the Gathering"`)
	q.Add("filter", `manufacturer_uFilter:"Wizards of the Coast"`)
	u.RawQuery = q.Encode()

	resp, err := mm.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var search MMSearchResponse
	err = json.NewDecoder(resp.Body).Decode(&search)
	if err != nil {
		return nil, err
	}

	return &search, nil
}
