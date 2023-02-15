package mythicmtg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	DefaultResultsPerPage = 128

	inventoryURL = "https://mythicmtg.com/magic-singles/?items_per_page=1&result_ids=pagination_contents&is_ajax=1"
)

type MythicMTGPage struct {
	Params struct {
		ItemsPerPage int `json:"items_per_page"`
		TotalItems   int `json:"total_items,string"`
	} `json:"ab__alp_params"`
	HTML struct {
		Contents string `json:"pagination_contents"`
	} `json:"html"`
	CurrentURL string `json:"current_url"`
}

type MythicClient struct {
	client *retryablehttp.Client
}

func NewMythicClient() *MythicClient {
	mc := MythicClient{}
	mc.client = retryablehttp.NewClient()
	mc.client.Logger = nil
	return &mc
}

func (mc *MythicClient) TotalItems() (int, error) {
	resp, err := mc.query(1, 1)
	if err != nil {
		return 0, err
	}
	return resp.Params.TotalItems, nil
}

func (mc *MythicClient) Products(page int) (io.Reader, error) {
	resp, err := mc.query(page, DefaultResultsPerPage)
	if err != nil {
		return nil, err
	}
	return strings.NewReader(resp.HTML.Contents), nil
}

func (mc *MythicClient) query(page, maxResults int) (*MythicMTGPage, error) {
	u, err := url.Parse(inventoryURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("page", fmt.Sprint(page))
	q.Set("items_per_page", fmt.Sprint(maxResults))
	u.RawQuery = q.Encode()

	resp, err := mc.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var search MythicMTGPage
	err = json.Unmarshal(data, &search)
	if err != nil {
		return nil, err
	}

	return &search, nil
}
