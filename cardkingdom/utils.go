package cardkingdom

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"

	"github.com/hashicorp/go-cleanhttp"
)

type CookieClient struct {
	client  *http.Client
	session string
}

func NewCookieClient(session string) *CookieClient {
	ck := CookieClient{}
	ck.client = cleanhttp.DefaultClient()
	jar, _ := cookiejar.New(nil)
	ck.client.Jar = jar
	ck.session = session
	return &ck
}

type InventoryRequest struct {
	ProductID string `json:"product_id"`
	Style     string `json:"style"`
	Quantity  int    `json:"quantity"`
}

var condMap = map[string]string{
	"NM": "NM",
	"SP": "EX",
	"MP": "VG",
	"HP": "G",
}

const (
	ckInventoryAddURL = "https://www.cardkingdom.com/api/cart/add"
	ckBuylistAddURL   = "https://www.cardkingdom.com/api/sellcart/add"
)

func (ck *CookieClient) InventorySetQuantity(ckId, cond string, qty int) (string, error) {
	return ck.setQuantity(ckInventoryAddURL, ckId, cond, qty)
}

func (ck *CookieClient) BuylistSetQuantity(ckId, cond string, qty int) (string, error) {
	return ck.setQuantity(ckBuylistAddURL, ckId, "NM", qty)
}

func (ck *CookieClient) setQuantity(link, ckId, cond string, qty int) (string, error) {
	style, found := condMap[cond]
	if found {
		cond = style
	}

	payload := InventoryRequest{
		ProductID: ckId,
		Style:     cond,
		Quantity:  qty,
	}

	reqBody, err := json.Marshal(&payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, link, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Add("Cookie", "laravel_session="+ck.session+";")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	req.Header.Add("User-Agent", "curl/8.6.0")

	resp, err := ck.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Nothing interesting in the response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	response := string(data)

	if resp.StatusCode != http.StatusOK {
		err = errors.New("inventory not ok")
	}

	return response, err
}

func (ck *CookieClient) Get(link string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", "laravel_session="+ck.session+";")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	req.Header.Add("User-Agent", "curl/8.6.0")

	return ck.client.Do(req)
}
