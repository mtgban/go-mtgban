package cardtrader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type CTLoggedClient struct {
	client *http.Client
}

func NewCTLoggedClient(user, pass string) (*CTLoggedClient, error) {
	ct := CTLoggedClient{}
	ct.client = cleanhttp.DefaultClient()

	jar, _ := cookiejar.New(nil)
	ct.client.Jar = jar

	token, err := ct.NewToken()
	if err != nil {
		return nil, err
	}

	resp, err := ct.client.PostForm("https://www.cardtrader.com/users/sign_in?locale=en", url.Values{
		"utf8":               {"✓"},
		"authenticity_token": {token},
		"user[email]":        {user},
		"user[password]":     {pass},
		"user[remember_me]":  {"true"},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &ct, nil
}

func (ct *CTLoggedClient) NewToken() (string, error) {
	resp, err := ct.client.Get("https://www.cardtrader.com/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	token, ok := doc.Find(`meta[name="csrf-token"]`).Attr("content")
	if !ok {
		return "", fmt.Errorf("html token node not found")
	}

	return token, nil
}

func (ct *CTLoggedClient) Add2Cart(productId int, qty int, bundle bool) error {
	u, err := url.Parse(fmt.Sprintf("https://www.cardtrader.com/cart/add/%d.json", productId))
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Set("quantity", fmt.Sprint(qty))
	if bundle {
		q.Set("future_order", "1")
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return err
	}

	token, err := ct.NewToken()
	if err != nil {
		return err
	}
	req.Header.Add("X-CSRF-Token", token)

	resp, err := ct.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response struct {
		DeltaChangedQuantity int `json:"delta_changed_quantity"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	if response.DeltaChangedQuantity != qty {
		return fmt.Errorf("different delta")
	}

	return nil
}

// A variant of Product
type OrderItem struct {
	Id            int    `json:"id"`
	BlueprintId   int    `json:"blueprint_id"`
	Quantity      int    `json:"quantity"`
	Description   string `json:"description"`
	PriceCents    int    `json:"price_cents"`
	PriceCurrency string `json:"price_currency"`
	Properties    struct {
		Condition string `json:"condition"`
		Language  string `json:"mtg_language"`
		Foil      bool   `json:"mtg_foil"`
	} `json:"properties_hash"`
}

func (ct *CTLoggedClient) GetItemsForOrder(orderId int) ([]OrderItem, error) {
	resp, err := ct.client.Get(fmt.Sprintf("https://www.cardtrader.com/orders/%d.json", orderId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var purchase struct {
		OrderItems []OrderItem `json:"order_items"`
	}
	err = json.Unmarshal(data, &purchase)
	if err != nil {
		return nil, err
	}

	return purchase.OrderItems, nil
}

func (ct *CardtraderMarket) Activate(user, pass string) error {
	client, err := NewCTLoggedClient(user, pass)
	if err != nil {
		return err
	}

	ct.loggedClient = client

	return nil
}

func (ct *CardtraderMarket) Add(entry mtgban.InventoryEntry) error {
	id, err := strconv.Atoi(entry.InstanceId)
	if err != nil {
		return err
	}

	return ct.loggedClient.Add2Cart(id, entry.Quantity, entry.Bundle)
}

func ConvertItems(blueprints map[int]*Blueprint, products []OrderItem, rates ...float64) mtgban.InventoryRecord {
	inventory := mtgban.InventoryRecord{}
	for _, product := range products {
		bp, found := blueprints[product.BlueprintId]
		if !found {
			continue
		}
		theCard, err := Preprocess(bp)
		if err != nil {
			continue
		}
		theCard.Foil = product.Properties.Foil

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			continue
		}

		price := float64(product.PriceCents) / 100.0

		currency := product.PriceCurrency
		if currency == "EUR" && len(rates) > 0 && rates[0] != 0 {
			price *= rates[0]
		}

		quantity := product.Quantity

		conds, found := condMap[product.Properties.Condition]
		if !found {
			continue
		}

		err = inventory.AddRelaxed(cardId, &mtgban.InventoryEntry{
			Price:      price,
			Quantity:   quantity,
			Conditions: conds,
			OriginalId: fmt.Sprint(product.BlueprintId),
			InstanceId: fmt.Sprint(product.Id),
		})
	}

	return inventory
}
