package cardtrader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	ctFilterURL     = "https://www.cardtrader.com/cards/%d/filter.json"
	ctBlueprintsURL = "https://api.cardtrader.com/api/full/v1/blueprints/export?category_id=1"

	ctExpansionsURL  = "https://api.cardtrader.com/api/full/v1/expansions"
	ctMarketplaceURL = "https://api.cardtrader.com/api/full/v1/marketplace/products?expansion_id="

	ctBulkCreateURL = "https://api.cardtrader.com/api/full/v1/products/bulk_create"
	ctBulkUpdateURL = "https://api.cardtrader.com/api/full/v1/products/bulk_update"

	MaxBulkUploadItems = 450
)

type Blueprint struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	CategoryId int    `json:"category_id"`
	GameId     int    `json:"game_id"`
	Slug       string `json:"slug"`
	ScryfallId string `json:"scryfall_id"`
	Expansion  struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"expansion"`
	// Returned by product
	Properties struct {
		Number   string `json:"collector_number"`
		Language string `json:"mtg_language"`
	} `json:"properties_hash"`
	// Returned by market
	FixedProperties struct {
		Number   string `json:"collector_number"`
		Language string `json:"mtg_language"`
	} `json:"fixed_properties"`
	ExpansionId int `json:"expansion_id"`
}

type Product struct {
	Id          int    `json:"id"`
	BlueprintId int    `json:"blueprint_id"`
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
		Cents    int    `json:"cents"`
		Currency string `json:"currency"`
	} `json:"price"`
	Expansion struct {
		Name string `json:"name"`
	} `json:"expansion"`
}

type BlueprintFilter struct {
	Blueprint Blueprint `json:"blueprint"`
	Products  []Product `json:"products"`
}

type Expansion struct {
	Id     int    `json:"id"`
	GameId int    `json:"game_id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
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

func (ct *CTAuthClient) Expansions() ([]Expansion, error) {
	resp, err := ct.client.Get(ctExpansionsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var out []Expansion
	err = json.Unmarshal(data, &out)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for expansions, got: %s", string(data))
	}

	return out, nil
}

// Returns all products from an Expansion, with the 15 cheapest listings per product
func (ct *CTAuthClient) ProductsForExpansion(id int) (map[int][]Product, error) {
	resp, err := ct.client.Get(ctMarketplaceURL + fmt.Sprint(id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var out map[int][]Product
	err = json.Unmarshal(data, &out)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for expansion %d, got: %s", id, string(data))
	}

	return out, nil
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
		return nil, fmt.Errorf("unmarshal error for blueprints, got: %s", string(data))
	}

	return blueprints, nil
}

// This is slightly different from the main Product type
type BulkProduct struct {
	// The id of the Product to edit
	Id int `json:"id,omitempty"`

	// The id of the Blueprint to put on sale
	BlueprintId int `json:"blueprint_id,omitempty"`

	// The price of the product, indicated in your current currency
	Price float64 `json:"price,omitempty"`

	// The quantity to be put up for sale
	Quantity int `json:"quantity,omitempty"`

	// A list of optional properties
	Properties struct {
		Condition string `json:"condition,omitempty"`
		Language  string `json:"mtg_language,omitempty"`
		Foil      bool   `json:"mtg_foil,omitempty"`
		Signed    bool   `json:"signed,omitempty"`
		Altered   bool   `json:"altered,omitempty"`
	} `json:"properties,omitempty"`
}

// Create new listings using the products slice, separating into multiple
// requests if there are more than MaxBulkUploadItems elements. A list of
// job ids is returned to monitor the execution status.
func (ct *CTAuthClient) BulkCreate(products []BulkProduct) ([]string, error) {
	return ct.bulkOperation(ctBulkCreateURL, products)
}

// Update existing listings using the products slice, separating into multiple
// requests if there are more than MaxBulkUploadItems elements. A list of
// job ids is returned to monitor the execution status.
func (ct *CTAuthClient) BulkUpdate(products []BulkProduct) ([]string, error) {
	return ct.bulkOperation(ctBulkUpdateURL, products)
}

func (ct *CTAuthClient) bulkOperation(link string, products []BulkProduct) ([]string, error) {
	var jobs []string
	var bulkUpload struct {
		Products []BulkProduct `json:"products"`
	}

	for i := 0; i < len(products); i += MaxBulkUploadItems {
		end := i + MaxBulkUploadItems
		if end > len(products) {
			end = len(products)
		}

		bulkUpload.Products = products[i:end]
		bodyBytes, err := json.Marshal(&bulkUpload)
		if err != nil {
			return nil, err
		}

		resp, err := ct.client.Post(link, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var jobResp struct {
			Job string `json:"job"`
		}
		err = json.Unmarshal(data, &jobResp)
		if err != nil {
			return nil, fmt.Errorf("unmarshal error for chunk %d, got: %s", i, string(data))
		}

		jobs = append(jobs, jobResp.Job)
	}

	return jobs, nil
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
		return nil, fmt.Errorf("unmarshal error for blueprint %d, got: %s", id, string(data))
	}

	if bf.Blueprint.Id == 0 {
		return nil, fmt.Errorf("empty blueprint for id %d", id)
	}

	return &bf, nil
}
