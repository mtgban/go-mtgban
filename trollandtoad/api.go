package trollandtoad

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"

	http "github.com/hashicorp/go-retryablehttp"
)

type tatParam struct {
	Action     string `json:"action"`
	DeptCode   string `json:"deptCode"`
	CategoryId string `json:"catid"`
}

type TATBuyingOption struct {
	ProductId  int     `json:"productid"`
	Price      float64 `json:"saleprice"`
	Quantity   int     `json:"quantityonsite"`
	Conditions string  `json:"conditioncode"`
}

type TATEdition struct {
	DeptId       string `json:"dept_id"`
	CategoryId   string `json:"category_id"`
	CategoryName string `json:"category_name"`
}

type TATProduct struct {
	Product map[string]struct {
		Name      string `json:"name"`
		Edition   string `json:"catname"`
		Condition string `json:"condition"`
		BuyPrice  string `json:"buyprice"`
		Quantity  string `json:"buyqty"`
	} `json:"product"`
}

type TATClient struct {
	client *http.Client
}

const (
	tatInventoryURL = "https://www.trollandtoad.com/ajax/productAjax.php"
	tatBuylistURL   = "https://www2.trollandtoad.com/buylist/ajax_scripts/buylist.php"
)

func NewTATClient() *TATClient {
	tat := TATClient{}
	tat.client = http.NewClient()
	tat.client.Logger = nil
	return &tat
}

func (tat *TATClient) GetProductOptions(productId string) ([]TATBuyingOption, error) {
	resp, err := tat.client.PostForm(tatInventoryURL, url.Values{
		"productid": {productId},
		"action":    {"getBuyingOptions"},
	})
	if err != nil {
		return nil, err

	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var options []TATBuyingOption
	err = json.Unmarshal(data, &options)
	if err != nil {
		return nil, err
	}

	return options, nil
}

func (tat *TATClient) ListVintageEditions() ([]TATEdition, error) {
	return tat.listEditions("V")
}

func (tat *TATClient) ListModernEditions() ([]TATEdition, error) {
	return tat.listEditions("M")
}

func (tat *TATClient) listEditions(code string) ([]TATEdition, error) {
	param := tatParam{
		Action:   "getdeptcategorylist",
		DeptCode: code,
	}
	reqBody, err := json.Marshal(&param)
	if err != nil {
		return nil, err
	}

	resp, err := tat.client.Post(tatBuylistURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err

	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var editions []TATEdition
	err = json.Unmarshal(data, &editions)
	if err != nil {
		return nil, err
	}

	return editions, nil
}

func (tat *TATClient) ProductsForId(id string, code string) (*TATProduct, error) {
	param := tatParam{
		Action:     "getbuylist",
		DeptCode:   code,
		CategoryId: id,
	}
	reqBody, _ := json.Marshal(&param)

	resp, err := tat.client.Post(tatBuylistURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var products TATProduct
	err = json.Unmarshal(data, &products)
	if err != nil {
		return nil, err
	}

	return &products, nil
}
