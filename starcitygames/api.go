package starcitygames

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	GameMagic         = 1
	GameFleshAndBlood = 2
	GameLorcana       = 3
)

type SCGClient struct {
	client *http.Client
	apiKey string
}

func NewSCGClient(apiKey string) *SCGClient {
	scg := SCGClient{}
	cli := retryablehttp.NewClient()
	cli.Logger = nil
	cli.RetryMax = 10
	cli.RetryWaitMin = 2 * time.Second
	scg.client = cli.StandardClient()
	scg.apiKey = apiKey
	return &scg
}

// Hit is the minimal product shape consumed by preprocess; the catalog scraper
// synthesizes one from a CatalogProduct (see catalogHit).
type Hit struct {
	Name                string    `json:"name"`
	ID                  int       `json:"id"`
	Subtitle            string    `json:"subtitle"`
	ProductType         string    `json:"product_type"`
	Finish              any       `json:"finish"`
	FinishPricingTypeID int       `json:"finish_pricing_type_id"`
	CardStyleID         int       `json:"card_style_id"`
	Language            string    `json:"language"`
	Rarity              any       `json:"rarity"`
	IsBuying            int       `json:"is_buying"`
	Hotlist             int       `json:"hotlist"`
	BorderColor         string    `json:"border_color"`
	CollectorNumber     string    `json:"collector_number"`
	GameID              int       `json:"game_id"`
	SetID               int       `json:"set_id"`
	SetName             string    `json:"set_name"`
	SetReleaseDate      int       `json:"set_release_date"`
	SetSymbol           string    `json:"set_symbol"`
	Variants            []Variant `json:"variants"`
	Image               string    `json:"image"`
	WizardsCode         string    `json:"wizards_code"`
}

type Variant struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Subtitle     string  `json:"subtitle"`
	Sku          string  `json:"sku"`
	IsBuying     int     `json:"is_buying"`
	Hotlist      float64 `json:"hotlist"`
	VariantName  string  `json:"variant_name"`
	VariantValue string  `json:"variant_value"`
	BuyPrice     float64 `json:"buy_price"`
	TradePrice   float64 `json:"trade_price"`
}

const (
	BaseProductURL    = "https://starcitygames.com"
	PartnerProductURL = "https://goto.starcitygames.com/c/%s/3052179/37198"
)

func SCGProductURL(URLDetail, variantSKU []string, affiliate string) string {
	if len(URLDetail) == 0 {
		return ""
	}

	link := BaseProductURL + URLDetail[0]
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}

	v := u.Query()
	if len(variantSKU) > 0 {
		v.Set("sku", variantSKU[0])
	}
	u.RawQuery = v.Encode()

	if affiliate == "" {
		return u.String()
	}

	q := url.Values{}
	q.Set("u", u.String())

	link = fmt.Sprintf(PartnerProductURL, affiliate)
	u, err = url.Parse(link)
	if err != nil {
		return ""
	}
	u.RawQuery = q.Encode()

	return u.String()
}
