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

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	msg, _ := d.Find(`div[data-controller="flash"]`).Attr("data-flash-type")
	if msg != "success" {
		return nil, fmt.Errorf("invalid credentials")
	}

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

type OrderItem struct {
	Id          int     `json:"id"`
	BlueprintId int     `json:"blueprint_id"`
	Quantity    int     `json:"quantity"`
	Description string  `json:"description"`
	PriceCents  float64 `json:"price_cents"`
	Properties  struct {
		Condition string `json:"condition"`
		Language  string `json:"mtg_language"`
		Foil      bool   `json:"mtg_foil"`
	} `json:"properties_hash"`

	Blueprint Blueprint `json:"blueprint"`
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

func (ct *Cardtrader) Activate(user, pass string) error {
	client, err := NewCTLoggedClient(user, pass)
	if err != nil {
		return err
	}

	ct.loggedClient = client

	return nil
}

func (ct *Cardtrader) Add(entry mtgban.InventoryEntry) error {
	id, err := strconv.Atoi(entry.InstanceId)
	if err != nil {
		return err
	}

	return ct.loggedClient.Add2Cart(id, entry.Quantity, entry.Bundle)
}
