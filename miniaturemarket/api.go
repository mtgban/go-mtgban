package miniaturemarket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	http "github.com/hashicorp/go-retryablehttp"
)

type MMBuyBack struct {
	Id           string `json:"id"`
	Image        string `json:"image"`
	FullImage    string `json:"full_image"`
	Name         string `json:"name"`
	BuybackName  string `json:"buyback_name"`
	SKU          string `json:"sku"`
	Price        string `json:"price"`
	Note         string `json:"note"`
	IsFoil       bool   `json:"foil"`
	MtgSet       string `json:"mtg_set"`
	MtgRarity    string `json:"mtg_rarity"`
	MtgCondition string `json:"mtg_condition"`
}

type MMPagination struct {
	TotalResults   int `json:"totalResults"`
	Begin          int `json:"begin"`
	End            int `json:"end"`
	CurrentPage    int `json:"currentPage"`
	TotalPages     int `json:"totalPages"`
	PreviousPage   int `json:"previousPage"`
	NextPage       int `json:"nextPage"`
	PerPage        int `json:"perPage"`
	DefaultPerPage int `json:"defaultPerPage"`
}

type MMSearchSpring struct {
	Pagination MMPagination `json:"pagination"`
	Results    string       `json:"results"`
}

type MMPrivateInfoGroup struct {
	PID          string  `json:"pid"`
	SKU          string  `json:"sku"`
	Price        float64 `json:"price"`
	RegularPrice string  `json:"regular_price"`
	Cost         string  `json:"cost"`
	Name         string  `json:"name"`
	Image        string  `json:"image"`
	Stock        int     `json:"stock"`
	InStock      string  `json:"instock"`
	Default      string  `json:"default"`
}

type MMClient struct {
	client         *http.Client
	resultsPerPage int
}

const (
	mmBuyBackURL       = "https://www.miniaturemarket.com/buyback/data/products/"
	mmBuyBackSearchURL = "https://www.miniaturemarket.com/buyback/data/productsearch/"

	MMCategoryMtgSingles    = "1466"
	MMDefaultResultsPerPage = 30

	mmSearchSpringURL = `https://api.searchspring.net/api/search/search.json?format=json&websiteKey=6f9c319d45519a85863e68be9c3f5d81&filter.stock_status=In+Stock&bgfilter.category_hierarchy=Magic+The+Gathering%2FMTG+Singles`
)

func NewMMClient() *MMClient {
	mm := MMClient{}
	mm.client = http.NewClient()
	mm.client.Logger = nil
	mm.resultsPerPage = MMDefaultResultsPerPage
	return &mm
}

func (mm *MMClient) GetPagination(resultsPerPage int) (*MMPagination, error) {
	mm.resultsPerPage = resultsPerPage

	u, err := url.Parse(mmSearchSpringURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("resultLayout", "none")
	q.Set("resultsPerPage", fmt.Sprint(resultsPerPage))
	u.RawQuery = q.Encode()

	resp, err := mm.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchspring MMSearchSpring
	err = json.Unmarshal(data, &searchspring)
	if err != nil {
		return nil, err
	}

	return &searchspring.Pagination, nil
}

func (mm *MMClient) SearchSpringPage(page int, orderDesc bool) (*MMSearchSpring, error) {
	resultsPerPage := mm.resultsPerPage

	u, err := url.Parse(mmSearchSpringURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("page", fmt.Sprint(page))
	q.Set("resultsPerPage", fmt.Sprint(resultsPerPage))
	if orderDesc {
		q.Set("sort.name", "desc")
	}
	u.RawQuery = q.Encode()

	resp, err := mm.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchspring MMSearchSpring
	err = json.Unmarshal(data, &searchspring)
	if err != nil {
		return nil, err
	}

	return &searchspring, nil
}

func (mm *MMClient) BuyBackPage(category string, page int) ([]MMBuyBack, error) {
	resp, err := mm.client.PostForm(mmBuyBackURL, url.Values{
		"category": {category},
		"page":     {fmt.Sprintf("%d", page)},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var buyback []MMBuyBack
	err = json.Unmarshal(data, &buyback)
	if err != nil {
		return nil, err
	}

	return buyback, nil
}

func (mm *MMClient) BuyBackSearch(search string) ([]MMBuyBack, error) {
	resp, err := http.PostForm(mmBuyBackSearchURL, url.Values{
		"search": {search},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var buyback []MMBuyBack
	err = json.Unmarshal(data, &buyback)
	if err != nil {
		return nil, err
	}

	return buyback, nil
}
