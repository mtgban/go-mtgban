package cardkingdom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
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
	ckSealedListURL = "https://api.cardkingdom.com/api/sealed_pricelist"

	ckBackupURL = "https://mtgban.com/api/cardkingdom/pricelist.json"
	ckUserAgent = "MTGBAN/CK"
)

type CKClient struct {
	client *http.Client
}

func NewCKClient() *CKClient {
	ck := CKClient{}
	ck.client = cleanhttp.DefaultClient()
	return &ck
}

func (ck *CKClient) GetPriceList() ([]CKCard, error) {
	return ck.getList(ckPricelistURL)
}

func (ck *CKClient) GetSealedList() ([]CKCard, error) {
	return ck.getList(ckSealedListURL)
}

func (ck *CKClient) getList(link string) ([]CKCard, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", ckUserAgent)

	resp, err := ck.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return ck.getList(ckBackupURL)
	}

	data, err := io.ReadAll(resp.Body)
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
		return nil, fmt.Errorf("unmarshal error for list, got: %s", string(data))
	}

	return pricelist.Data, nil
}

func (ck *CKClient) GetHotBuylist() ([]CKCard, error) {
	resp, err := ck.client.Get(ckHotBuylistURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
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
