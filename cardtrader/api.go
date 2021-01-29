package cardtrader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	ctFilterURL     = "https://www.cardtrader.com/cards/%d/filter.json"
	ctBlueprintsURL = "https://api.cardtrader.com/api/full/v1/blueprints/export?category_id=1"
)

type Blueprint struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	CategoryId  int    `json:"category_id"`
	GameId      int    `json:"game_id"`
	Expansion   struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"expansion"`
	Properties struct {
		Number   string `json:"collector_number"`
		Language string `json:"mtg_language"`
	} `json:"properties_hash"`
}

type Product struct {
	Id          int    `json:"id"`
	Quantity    int    `json:"quantity"`
	Description string `json:"description"`
	OnVacation  bool   `json:"on_vacation"`
	Bundle      bool   `json:"bundle"`
	Properties  struct {
		Condition string `json:"condition"`
		Language  string `json:"mtg_language"`
		Number    string `json:"collector_number"`
		Foil      bool   `json:"mtg_foil"`
		Altered   bool   `json:"altered"`
		Signed    bool   `json:"signed"`
	} `json:"properties_hash"`
	User struct {
		Name string `json:"username"`
		Zero bool   `json:"can_sell_via_hub"`
	} `json:"user"`
	Price struct {
		Cents int `json:"cents"`
	} `json:"price"`
}

type BlueprintFilter struct {
	Blueprint Blueprint `json:"blueprint"`
	Products  []Product `json:"products"`
}

type CTClient struct {
	client *retryablehttp.Client
}

type CTAuthClient struct {
	client *retryablehttp.Client
}

type authTransport struct {
	Parent http.RoundTripper
	Token  string
}

func NewCTAuthClient(token string) *CTAuthClient {
	ct := CTAuthClient{}
	ct.client = retryablehttp.NewClient()
	ct.client.Logger = nil
	ct.client.HTTPClient.Transport = &authTransport{
		Parent: ct.client.HTTPClient.Transport,
		Token:  token,
	}
	return &ct
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	return t.Parent.RoundTrip(req)
}

func (ct *CTAuthClient) Blueprints() ([]Blueprint, error) {
	resp, err := ct.client.Get(ctBlueprintsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var blueprints []Blueprint
	err = json.Unmarshal(data, &blueprints)
	if err != nil {
		return nil, err
	}

	return blueprints, nil
}

func NewCTClient() *CTClient {
	ct := CTClient{}
	ct.client = retryablehttp.NewClient()
	ct.client.Logger = nil
	return &ct
}

func (ct *CTClient) ProductsForBlueprint(id int) (*BlueprintFilter, error) {
	resp, err := ct.client.Post(fmt.Sprintf(ctFilterURL, id), "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var bf BlueprintFilter
	err = json.Unmarshal(data, &bf)
	if err != nil {
		return nil, err
	}

	return &bf, nil
}
