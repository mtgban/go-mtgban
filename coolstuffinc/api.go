package coolstuffinc

import (
	"encoding/json"
	"io"

	http "github.com/hashicorp/go-retryablehttp"
)

type CSICard struct {
	Id             int     `json:"id,string"`
	URL            string  `json:"url"`
	Name           string  `json:"name"`
	Variation      string  `json:"variation"`
	Edition        string  `json:"edition"`
	Language       string  `json:"language"`
	IsFoil         bool    `json:"is_foil,string"`
	PriceRetail    float64 `json:"price_retail,string"`
	QuantityRetail int     `json:"qty_retail,string"`
	PriceBuy       float64 `json:"price_buy,string"`
	QuantityBuy    int     `json:"qty_buying,string"`
}

const (
	csiPricelistURL = "https://www.coolstuffinc.com/gateway_json.php?k="
)

type CSIClient struct {
	client *http.Client
	key    string
}

func NewCSIClient(key string) *CSIClient {
	csi := CSIClient{}
	csi.client = http.NewClient()
	csi.client.Logger = nil
	csi.key = key
	return &csi
}

func (csi *CSIClient) GetPriceList() ([]CSICard, error) {
	resp, err := csi.client.Get(csiPricelistURL + csi.key)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pricelist struct {
		Meta struct {
			CreatedAt string `json:"created_at"`
		} `json:"meta"`
		Data []CSICard `json:"data"`
	}
	err = json.Unmarshal(data, &pricelist)
	if err != nil {
		return nil, err
	}

	return pricelist.Data, nil
}
