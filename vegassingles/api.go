package vegassingles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mtgban/go-mtgban/mtgban"
)

const (
	buylistURL = "https://api.hippolink.app/v1/public/stores/vegassingles/buylist/products"
	staticUA   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:135.0) Gecko/20100101 Firefox/135.0"

	// The most products the buylist serves on one page.
	pageSize = 250
)

// HLResponse is one page of the buylist, as Hippolink serves it.
type HLResponse struct {
	Products []HLProduct `json:"products"`
	Total    int         `json:"total"`
	Limit    int         `json:"limit"`
	Offset   int         `json:"offset"`
}

// HLProduct is one TCGplayer product in one finish the store buys.
type HLProduct struct {
	TCGplayerProductID int64         `json:"tcgplayer_product_id"`
	ProductName        string        `json:"product_name"`
	SetName            string        `json:"set_name"`
	SetCode            string        `json:"set_code"`
	CollectorNumber    string        `json:"collector_number"`
	Finish             string        `json:"finish"`
	IsFoil             bool          `json:"is_foil"`
	Treatments         []string      `json:"treatments"`
	Hotlist            bool          `json:"hotlist"`
	Conditions         []HLCondition `json:"conditions"`
}

// HLCondition is the store's offer for one TCGplayer sku of a product.
type HLCondition struct {
	Condition      string  `json:"condition"`
	TCGplayerSkuID int64   `json:"tcgplayer_sku_id"`
	Language       string  `json:"language"`
	MaxQty         *int    `json:"max_qty"`
	Cash           float64 `json:"cash"`
	Credit         float64 `json:"credit"`
}

// VSClient reads the Vegas Singles buylist.
type VSClient struct {
	client *http.Client
	// Held rather than referenced so a test can point the client at a
	// server it controls.
	buylistURL string
}

// NewVSClient returns a client for the Vegas Singles buylist.
func NewVSClient() *VSClient {
	vs := VSClient{}
	vs.client = mtgban.NewHTTPClient()
	vs.buylistURL = buylistURL
	return &vs
}

// GetBuylistPage returns the page of the buylist of game, in Hippolink's
// spelling, starting at offset.
func (vs *VSClient) GetBuylistPage(ctx context.Context, game string, offset int) (*HLResponse, error) {
	q := url.Values{}
	q.Set("game", game)
	q.Set("sort", "set_alpha")
	q.Set("limit", strconv.Itoa(pageSize))
	q.Set("offset", strconv.Itoa(offset))

	var response HLResponse
	err := vs.getJSON(ctx, vs.buylistURL+"?"+q.Encode(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (vs *VSClient) getJSON(ctx context.Context, link string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", staticUA)

	resp, err := vs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// A host that moved answers 200 with a page of HTML
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		return fmt.Errorf("answered with %q, not JSON", mediaType)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}
