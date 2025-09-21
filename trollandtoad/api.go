package trollandtoad

import (
	"bytes"
	"encoding/json"
	"net/url"

	http "github.com/hashicorp/go-retryablehttp"
)

type tntParam struct {
	Action     string `json:"action"`
	DeptCode   string `json:"deptCode"`
	CategoryId string `json:"catid"`
}

type TNTBuyingOption struct {
	ProductId  int     `json:"productid"`
	Price      float64 `json:"saleprice"`
	Quantity   int     `json:"quantityonsite"`
	Conditions string  `json:"conditioncode"`
}

type TNTEdition struct {
	DeptId       string `json:"dept_id"`
	CategoryId   string `json:"category_id"`
	CategoryName string `json:"category_name"`
}

type TNTProduct struct {
	Product map[string]struct {
		Name      string `json:"name"`
		Edition   string `json:"catname"`
		Condition string `json:"condition"`
		BuyPrice  string `json:"buyprice"`
		Quantity  string `json:"buyqty"`
	} `json:"product"`
}

type TNTClient struct {
	client *http.Client
}

const (
	tntInventoryURL = "https://www.trollandtoad.com/ajax/productAjax.php"
	tntBuylistURL   = "https://www2.trollandtoad.com/buylist/ajax_scripts/buylist.php"
)

func NewTNTClient() *TNTClient {
	tnt := TNTClient{}
	tnt.client = http.NewClient()
	tnt.client.Logger = nil
	return &tnt
}

func (tnt *TNTClient) GetProductOptions(productId string) ([]TNTBuyingOption, error) {
	resp, err := tnt.client.PostForm(tntInventoryURL, url.Values{
		"productid": {productId},
		"action":    {"getBuyingOptions"},
	})
	if err != nil {
		return nil, err

	}
	defer resp.Body.Close()

	var options []TNTBuyingOption
	err = json.NewDecoder(resp.Body).Decode(&options)
	if err != nil {
		return nil, err
	}

	return options, nil
}

func (tnt *TNTClient) ListVintageEditions() ([]TNTEdition, error) {
	return tnt.listEditions("V")
}

func (tnt *TNTClient) ListModernEditions() ([]TNTEdition, error) {
	return tnt.listEditions("M")
}

func (tnt *TNTClient) listEditions(code string) ([]TNTEdition, error) {
	param := tntParam{
		Action:   "getdeptcategorylist",
		DeptCode: code,
	}
	reqBody, err := json.Marshal(&param)
	if err != nil {
		return nil, err
	}

	resp, err := tnt.client.Post(tntBuylistURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err

	}
	defer resp.Body.Close()

	var editions []TNTEdition
	err = json.NewDecoder(resp.Body).Decode(&editions)
	if err != nil {
		return nil, err
	}

	return editions, nil
}

func (tnt *TNTClient) ProductsForId(id string, code string) (*TNTProduct, error) {
	param := tntParam{
		Action:     "getbuylist",
		DeptCode:   code,
		CategoryId: id,
	}
	reqBody, _ := json.Marshal(&param)

	resp, err := tnt.client.Post(tntBuylistURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var products TNTProduct
	err = json.NewDecoder(resp.Body).Decode(&products)
	if err != nil {
		return nil, err
	}

	return &products, nil
}
