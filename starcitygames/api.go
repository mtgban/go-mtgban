package starcitygames

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-cleanhttp"
)

const (
	scgInventoryURL = "https://lusearchapi-na.hawksearch.com/sites/starcitygames/?instockonly=Yes&mpp=1&product_type=Singles"
	scgBuylistURL   = "https://search.starcitygames.com/indexes/sell_list_products/search"

	DefaultRequestLimit = 200
)

type SCGClient struct {
	client *http.Client
	bearer string
}

func NewSCGClient(bearer string) *SCGClient {
	scg := SCGClient{}
	scg.client = cleanhttp.DefaultClient()
	scg.bearer = bearer
	return &scg
}

func (scg *SCGClient) NumberOfItems() (int, error) {
	resp, err := scg.client.Get(scgInventoryURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	pagination := doc.Find(`div[id="hawktoppager"]`).Find(`div[class="hawk-searchrange"] span`).Text()
	items := strings.Split(pagination, " of ")
	if len(items) > 1 {
		return strconv.Atoi(items[1])
	}

	return 0, fmt.Errorf("invalid pagination value: %s", pagination)
}

func (scg *SCGClient) GetPage(page int) (*http.Response, error) {
	u, err := url.Parse(scgInventoryURL)
	if err != nil {
		return nil, err
	}
	v := u.Query()
	v.Set("mpp", fmt.Sprint(DefaultRequestLimit))
	v.Set("pg", fmt.Sprint(page))
	u.RawQuery = v.Encode()

	return scg.client.Get(u.String())
}

type SCGSearchRequest struct {
	Q                string   `json:"q"`
	Filter           string   `json:"filter"`
	MatchingStrategy string   `json:"matchingStrategy"`
	Limit            int      `json:"limit"`
	Offset           int      `json:"offset"`
	Sort             []string `json:"sort"`
}

type SCGSearchResponse struct {
	Message            string    `json:"message,omitempty"`
	Code               string    `json:"code,omitempty"`
	Type               string    `json:"type,omitempty"`
	Link               string    `json:"link,omitempty"`
	Hits               []SCGCard `json:"hits"`
	Query              string    `json:"query"`
	ProcessingTimeMs   int       `json:"processingTimeMs"`
	Limit              int       `json:"limit"`
	Offset             int       `json:"offset"`
	EstimatedTotalHits int       `json:"estimatedTotalHits"`
}

type SCGCard struct {
	Name            string           `json:"name"`
	ID              int              `json:"id"`
	Subtitle        string           `json:"subtitle"`
	Sku             string           `json:"sku"`
	ProductType     string           `json:"product_type"`
	CardName        string           `json:"card_name"`
	Finish          string           `json:"finish"`
	Language        string           `json:"language"`
	CollectorNumber string           `json:"collector_number"`
	Rarity          string           `json:"rarity"`
	SetID           int              `json:"set_id"`
	SetName         string           `json:"set_name"`
	SetReleaseDate  int              `json:"set_release_date"`
	SetSymbol       string           `json:"set_symbol"`
	IsBuying        int              `json:"is_buying"`
	Hotlist         int              `json:"hotlist"`
	Variants        []SCGCardVariant `json:"variants"`
}

type SCGCardVariant struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	Subtitle           string  `json:"subtitle"`
	VariantName        string  `json:"variant_name"`
	VariantValue       string  `json:"variant_value"`
	Sku                string  `json:"sku"`
	IsBuying           int     `json:"is_buying"`
	Hotlist            float64 `json:"hotlist"`
	BuyPrice           float64 `json:"buy_price"`
	TradePrice         float64 `json:"trade_price"`
	BonusCalculationID int     `json:"bonus_calculation_id"`
}

func (scg *SCGClient) SearchAll(offset, limit int) (*SCGSearchResponse, error) {
	q := SCGSearchRequest{
		Filter:           "is_buying = 1 AND (product_type = \"Singles\") AND ((language = \"en\") OR (language = \"ja\"))",
		MatchingStrategy: "all",
		Limit:            limit,
		Offset:           offset,
		Sort:             []string{"name:asc", "set_name:asc", "finish:desc"},
	}
	payload, err := json.Marshal(&q)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, scgBuylistURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+scg.bearer)
	req.Header.Add("Content-Type", "application/json")

	resp, err := scg.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var search SCGSearchResponse
	err = json.Unmarshal(data, &search)
	if err != nil {
		return nil, err
	}

	if search.Message != "" {
		return nil, fmt.Errorf(search.Message)
	}

	return &search, nil
}
