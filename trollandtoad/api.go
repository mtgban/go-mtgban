package trollandtoad

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	http "github.com/hashicorp/go-retryablehttp"
)

type tatParam struct {
	Action     string `json:"action"`
	DeptCode   string `json:"deptCode"`
	CategoryId string `json:"catid"`
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
	tatBuylistURL   = "https://www2.trollandtoad.com/buylist/ajax_scripts/buylist.php"
)

func NewTATClient() *TATClient {
	tat := TATClient{}
	tat.client = http.NewClient()
	tat.client.Logger = nil
	return &tat
}

func (tat *TATClient) ListEditions() ([]TATEdition, error) {
	param := tatParam{
		Action:   "getdeptcategorylist",
		DeptCode: "M",
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

	data, err := ioutil.ReadAll(resp.Body)
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

func (tat *TATClient) ProductsForId(id string) (*TATProduct, error) {
	param := tatParam{
		Action:     "getbuylist",
		DeptCode:   "M",
		CategoryId: id,
	}
	reqBody, _ := json.Marshal(&param)

	resp, err := tat.client.Post(tatBuylistURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
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
