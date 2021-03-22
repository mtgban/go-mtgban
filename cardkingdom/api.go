package cardkingdom

import (
	"encoding/json"
	"io/ioutil"

	http "github.com/hashicorp/go-retryablehttp"
)

type conditionValues struct {
	NMPrice string `json:"nm_price"`
	NMQty   int    `json:"nm_qty"`
	EXPrice string `json:"ex_price"`
	EXQty   int    `json:"ex_qty"`
	VGPrice string `json:"vg_price"`
	VGQty   int    `json:"vg_qty"`
	GOPrice string `json:"g_price"`
	GOQty   int    `json:"g_qty"`
}

type CKCard struct {
	Id              int             `json:"id"`
	SKU             string          `json:"sku"`
	URL             string          `json:"url"`
	Name            string          `json:"name"`
	Variation       string          `json:"variation"`
	Edition         string          `json:"edition"`
	IsFoil          string          `json:"is_foil"`
	SellPrice       string          `json:"price_retail"`
	SellQuantity    int             `json:"qty_retail"`
	BuyPrice        string          `json:"price_buy"`
	BuyQuantity     int             `json:"qty_buying"`
	ConditionValues conditionValues `json:"condition_values"`

	// Only from GetHotBuylist()
	HotPrice  string `json:"price"`
	ShortName string `json:"short_name,omitempty"`
}

const (
	ckPricelistURL  = "https://api.cardkingdom.com/api/v2/pricelist"
	ckHotBuylistURL = "https://api.cardkingdom.com/api/product/list/hotbuy"
)

type CKClient struct {
	client *http.Client
}

func NewCKClient() *CKClient {
	ck := CKClient{}
	ck.client = http.NewClient()
	ck.client.Logger = nil
	return &ck
}

func (ck *CKClient) GetPriceList() ([]CKCard, error) {
	resp, err := ck.client.Get(ckPricelistURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pricelist struct {
		Meta struct {
			CreatedAt string `json:"created_at"`
			BaseURL   string `json:"base_url"`
		} `json:"meta"`
		Data []CKCard `json:"data"`
	}
	err = json.Unmarshal(data, &pricelist)
	if err != nil {
		return nil, err
	}

	return pricelist.Data, nil
}

func (ck *CKClient) GetHotBuylist() ([]CKCard, error) {
	resp, err := ck.client.Get(ckHotBuylistURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pricelist struct {
		Data []CKCard `json:"list"`
	}
	err = json.Unmarshal(data, &pricelist)
	if err != nil {
		return nil, err
	}

	return pricelist.Data, nil
}
