package purplemana

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/go-cleanhttp"
)

type Product struct {
	ID              int     `json:"id"`
	Condition       string  `json:"condition"`
	SellerPayout    float64 `json:"seller_payout"`
	CatalogProducts struct {
		ID                 int    `json:"id"`
		Name               string `json:"name"`
		Variant            string `json:"variant"`
		SetName            string `json:"set_name"`
		TcgplayerID        int    `json:"tcgplayer_id"`
		CollectorCode      string `json:"collector_code"`
		FrontImageThumbURL string `json:"front_image_thumb_url"`
	} `json:"catalog_products"`
}

const hotlistURLAPI = "https://www.purplemana.com/api/trpc/orders.getHotlist"

func GetHotList() ([]Product, error) {
	resp, err := cleanhttp.DefaultClient().Get(hotlistURLAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var hotlist struct {
		Result struct {
			Data struct {
				JSON []Product `json:"json"`
			} `json:"data"`
		} `json:"result"`
	}
	err = json.Unmarshal(data, &hotlist)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for list, got: %s", string(data))
	}

	return hotlist.Result.Data.JSON, nil
}
