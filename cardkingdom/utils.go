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

type CartRequest struct {
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

func (ck *CookieClient) SetCartInventory(ckId, cond string, qty int) (string, error) {
	return ck.setCart(ckInventoryAddURL, ckId, cond, qty)
}

func (ck *CookieClient) SetCartBuylist(ckId string, qty int) (string, error) {
	return ck.setCart(ckBuylistAddURL, ckId, "NM", qty)
}

func (ck *CookieClient) setCart(link, ckId, cond string, qty int) (string, error) {
	style, found := condMap[cond]
	if found {
		cond = style
	}

	payload := CartRequest{
		ProductID: ckId,
		Style:     cond,
		Quantity:  qty,
	}

	reqBody, err := json.Marshal(&payload)
	if err != nil {
		return "", err
	}

	resp, err := ck.Post(link, "application/json", bytes.NewReader(reqBody))
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
		err = errors.New("cart not ok")
	}

	return response, err
}

func (ck *CookieClient) Get(link string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", "laravel_session="+ck.session+";")
	req.Header.Add("User-Agent", "curl/8.6.0")

	return ck.client.Do(req)
}

func (ck *CookieClient) Post(url, contentType string, reader io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", "laravel_session="+ck.session+";")
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("User-Agent", "curl/8.6.0")

	return ck.client.Do(req)
}
