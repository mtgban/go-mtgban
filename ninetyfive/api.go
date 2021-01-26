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
				Links       struct {
					Previous string `json:"previous"`
					Next     string `json:"next"`
				} `json:"links"`
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
	Conditions []string `json:"conditions"`
	Price      int      `json:"price"`
	Currency   struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"currency"`
	Quantity int    `json:"qnt"`
	Card     NFCard `json:"card"`
}

type NFCard struct {
	Name string `json:"name"`
	Set  struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"set"`
	Number int    `json:"number"`
	Layout string `json:"layout"`
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
