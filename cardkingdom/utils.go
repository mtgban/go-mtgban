package cardkingdom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

type CookieClient struct {
	client    *http.Client
	session   string
	clearance string
}

func NewCookieClient(session, clearance string) *CookieClient {
	ck := CookieClient{}
	ck.client = cleanhttp.DefaultClient()
	jar, _ := cookiejar.New(nil)
	ck.client.Jar = jar
	ck.session = session
	ck.clearance = clearance
	return &ck
}

func NewCookieJarClient(jar http.CookieJar) *CookieClient {
	ck := CookieClient{}
	ck.client = cleanhttp.DefaultClient()
	ck.client.Jar = jar
	return &ck
}

type CartRequest struct {
	ProductID string `json:"product_id"`
	Style     string `json:"style"`
	Quantity  int    `json:"quantity"`
}

type CartResponse struct {
	AutoValidateCart  bool      `json:"auto_validate_cart"`
	ForceValidateCart bool      `json:"force_validate_cart"`
	HasPriceChange    bool      `json:"has_price_change"`
	HasQuantityChange bool      `json:"has_quantity_change"`
	ID                int       `json:"id"`
	ItemCount         int       `json:"item_count"`
	LastAccess        time.Time `json:"last_access"`
	LineitemCount     int       `json:"lineitem_count"`
	Lineitems         []struct {
		Product struct {
			Model           string `json:"model"`
			Width           any    `json:"width"`
			Height          any    `json:"height"`
			Depth           any    `json:"depth"`
			Weight          int    `json:"weight"`
			IsShiny         bool   `json:"is_shiny"`
			ProductSlug     string `json:"product_slug"`
			CategorySlug    string `json:"category_slug"`
			URI             string `json:"uri"`
			MaxQtyAvailable int    `json:"max_qty_available"`
			PriceBuy        string `json:"price_buy"`
			PriceSale       any    `json:"price_sale"`
			OrderLimit      any    `json:"order_limit"`
			BuyLimit        int    `json:"buy_limit"`
			BorderClass     string `json:"border_class"`
			Rarity          string `json:"rarity"`
			ShortName       string `json:"short_name"`
			IsActive        bool   `json:"is_active"`
		} `json:"product"`
		Title                      string `json:"title"`
		Edition                    string `json:"edition"`
		Rarity                     string `json:"rarity"`
		Variation                  string `json:"variation"`
		Weight                     int    `json:"weight"`
		AllowInternationalShipping bool   `json:"allow_international_shipping"`
		ID                         int    `json:"id"`
		IsPresale                  bool   `json:"is_presale"`
		ShipDate                   string `json:"ship_date"`
		IsShiny                    bool   `json:"is_shiny"`
		DefaultImage               string `json:"default_image"`
		Name                       string `json:"name"`
		Price                      string `json:"price"`
		PriceAfterCoupon           string `json:"price_after_coupon"`
		OriginalPrice              string `json:"original_price"`
		ProductID                  int    `json:"product_id"`
		Style                      string `json:"style"`
		Total                      string `json:"total"`
		Qty                        int    `json:"qty"`
		OriginalQuantity           int    `json:"original_quantity"`
		IsBuying                   bool   `json:"is_buying"`
		IsSelling                  int    `json:"is_selling"`
		IsTaxable                  int    `json:"is_taxable"`
		CouponDiscount             string `json:"coupon_discount"`
	} `json:"lineitems"`
	PremiumOffer               string `json:"premium_offer"`
	Status                     string `json:"status"`
	Subtotal                   string `json:"subtotal"`
	SubtotalStorecredit        string `json:"subtotal_storecredit"`
	Type                       string `json:"type"`
	NeedsCartMergeNotification bool   `json:"needs_cart_merge_notification"`
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

	ckInventoryEmptyURL = "https://www.cardkingdom.com/cart/empty"
	ckBuylistEmptyURL   = "https://www.cardkingdom.com/sellcart/empty_cart"

	staticUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:145.0) Gecko/20100101 Firefox/145.0"
)

func (ck *CookieClient) SetCartInventory(ctx context.Context, ckId, cond string, qty int) (*CartResponse, error) {
	return ck.setCart(ctx, ckInventoryAddURL, ckId, cond, qty)
}

func (ck *CookieClient) SetCartBuylist(ctx context.Context, ckId string, qty int) (*CartResponse, error) {
	return ck.setCart(ctx, ckBuylistAddURL, ckId, "NM", qty)
}

func (ck *CookieClient) EmptyCartInventory(ctx context.Context, cartToken string) error {
	return ck.emptyCart(ctx, ckInventoryEmptyURL, cartToken)
}

func (ck *CookieClient) EmptyCartBuylist(ctx context.Context, cartToken string) error {
	return ck.emptyCart(ctx, ckBuylistEmptyURL, cartToken)
}

func (ck *CookieClient) emptyCart(ctx context.Context, link, cartToken string) error {
	v := url.Values{}
	v.Set("_token", cartToken)

	resp, err := ck.Post(ctx, link, "application/x-www-form-urlencoded", strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("cart not ok")
	}

	return nil
}

func (ck *CookieClient) setCart(ctx context.Context, link, ckId, cond string, qty int) (*CartResponse, error) {
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
		return nil, err
	}

	resp, err := ck.Post(ctx, link, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("cart not ok")
	}

	var cartResponse CartResponse
	err = json.NewDecoder(resp.Body).Decode(&cartResponse)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &cartResponse, nil
}

func (ck *CookieClient) Get(ctx context.Context, link string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	if ck.session != "" && ck.clearance != "" {
		req.Header.Set("Cookie", "cf_clearance="+ck.clearance+"; laravel_session="+ck.session+";")
	}
	req.Header.Set("User-Agent", staticUA)

	return ck.client.Do(req)
}

func (ck *CookieClient) Post(ctx context.Context, link, contentType string, reader io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, link, reader)
	if err != nil {
		return nil, err
	}
	if ck.session != "" && ck.clearance != "" {
		req.Header.Set("Cookie", "cf_clearance="+ck.clearance+"; laravel_session="+ck.session+";")
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", staticUA)

	return ck.client.Do(req)
}
