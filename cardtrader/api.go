package cardtrader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	http "github.com/hashicorp/go-retryablehttp"
)

const ctInventoryURL = "https://www.cardtrader.com/cards/%d/filter.json"

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

type BlueprintFilter struct {
	Blueprint Blueprint `json:"blueprint"`
	Products  []struct {
		//Image      string `json:"image"`
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
	} `json:"products"`
}

type CTClient struct {
	client *http.Client
}

func NewCTClient() *CTClient {
	ct := CTClient{}
	ct.client = http.NewClient()
	ct.client.Logger = nil
	return &ct
}

func (ct *CTClient) GetBlueprints(categoryId int) (*BlueprintFilter, error) {
	resp, err := ct.client.Post(fmt.Sprintf(ctInventoryURL, categoryId), "application/json", nil)
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
