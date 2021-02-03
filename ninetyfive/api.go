package ninetyfive

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	NFDefaultResultsPerPage = 400

	nfRetailURL  = "https://95mtg.com/api/products/?search=qnt:1;language_id:6;prices.price:0-649995;cmc:0-1000000;card.power:0-99;card.toughness:0-99;category_id:1;name:;foil:0|1;signed:0&searchJoin=and&perPage=30&page=1&orderBy=name&sortedBy=asc"
	nfBuylistURL = "https://95mtg.com/api/buylists/?search=foil:0|1&searchJoin=and&perPage=1&page=1&orderBy=card.name&sortedBy=asc"
)

type NFSearchResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Results struct {
		Data []NFProduct `json:"data"`
		Meta struct {
			Pagination struct {
				Total       int `json:"total"`
				Count       int `json:"count"`
				PerPage     int `json:"per_page"`
				CurrentPage int `json:"current_page"`
				TotalPages  int `json:"total_pages"`
			} `json:"pagination"`
		} `json:"meta"`
	} `json:"results"`
	Errors   []string `json:"errors"`
	Redirect string   `json:"redirect"`
}

type NFProduct struct {
	Language struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"language"`
	Foil       int      `json:"foil"`
	Condition  string   `json:"condition"`
	Conditions []string `json:"conditions"`
	Price      int      `json:"price"`
	Currency   struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"currency"`
	Quantity int    `json:"qnt"`
	Card     NFCard `json:"card"`
	// Present in retail but not buylist
	Set NFSet `json:"set"`
}

type NFCard struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	// Present in buylist but not retail
	Set    NFSet  `json:"set"`
	Number int    `json:"number"`
	Layout string `json:"layout"`
}

type NFSet struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type NFClient struct {
	client *retryablehttp.Client
}

func NewNFClient() *NFClient {
	nf := NFClient{}
	nf.client = retryablehttp.NewClient()
	nf.client.Logger = nil
	nf.client.HTTPClient.Transport.(*http.Transport).ForceAttemptHTTP2 = true
	return &nf
}

func (nf *NFClient) RetailTotals() (int, error) {
	resp, err := nf.query(nfRetailURL, 0, 1)
	if err != nil {
		return 0, err
	}
	return resp.Results.Meta.Pagination.Total, nil
}

func (nf *NFClient) GetRetail(start int) ([]NFProduct, error) {
	resp, err := nf.query(nfRetailURL, start, NFDefaultResultsPerPage)
	if err != nil {
		return nil, err
	}
	return resp.Results.Data, nil
}

func (nf *NFClient) BuylistTotals() (int, error) {
	resp, err := nf.query(nfBuylistURL, 0, 1)
	if err != nil {
		return 0, err
	}
	return resp.Results.Meta.Pagination.Total, nil
}

func (nf *NFClient) GetBuylist(start int) ([]NFProduct, error) {
	resp, err := nf.query(nfBuylistURL, start, NFDefaultResultsPerPage)
	if err != nil {
		return nil, err
	}
	return resp.Results.Data, nil
}

func (nf *NFClient) query(searchURL string, start, maxResults int) (*NFSearchResponse, error) {
	u, err := url.Parse(searchURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("page", fmt.Sprint(start))
	q.Set("perPage", fmt.Sprint(maxResults))
	u.RawQuery = q.Encode()

	resp, err := nf.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var search NFSearchResponse
	err = json.Unmarshal(data, &search)
	if err != nil {
		return nil, err
	}

	return &search, nil
}
