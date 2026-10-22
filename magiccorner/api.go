package magiccorner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mtgban/go-mtgban/mtgban"
)

// MCEdition is a set as Magic Corner files it for selling.
type MCEdition struct {
	ID   int    `json:"Id"`
	Name string `json:"Espansione"`
	Code string `json:"ImageUrl"`
}

// MCCard is one card on sale, with the copies in stock in each condition,
// finish and language.
type MCCard struct {
	ID       int
	SKU      string
	Name     string
	Edition  string
	Code     string
	Rarity   string
	Extra    string
	OrigName string
	URL      string
	Variants []MCVariant
}

// MCVariant is the stock of a card in one condition, finish and language.
type MCVariant struct {
	ID        int     `json:"idproduct"`
	SKU       string  `json:"sku"`
	Language  string  `json:"flag"`
	Foil      string  `json:"p4"`
	Condition string  `json:"condizionishort"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// mcRaw is how the singles search wraps every field it returns.
type mcRaw[T any] struct {
	Raw T `json:"raw"`
}

type mcSingle struct {
	ID       mcRaw[float64]  `json:"idproduct"`
	SKU      mcRaw[string]   `json:"sku"`
	Model    mcRaw[string]   `json:"model"`
	ModelEn  mcRaw[string]   `json:"modelen"`
	Image    mcRaw[string]   `json:"image"`
	Category mcRaw[string]   `json:"category"`
	Icon     mcRaw[string]   `json:"icon"`
	URL      mcRaw[string]   `json:"url"`
	Rarity   mcRaw[[]string] `json:"p2"`
}

func (s *mcSingle) card() MCCard {
	card := MCCard{
		ID:       int(s.ID.Raw),
		SKU:      s.SKU.Raw,
		Name:     s.ModelEn.Raw,
		Edition:  s.Category.Raw,
		Code:     s.Icon.Raw,
		Extra:    s.Image.Raw,
		OrigName: s.Model.Raw,
		URL:      s.URL.Raw,
	}
	if len(s.Rarity.Raw) > 0 {
		card.Rarity = s.Rarity.Raw[0]
	}
	return card
}

type mcSearch struct {
	Query   string `json:"query"`
	Filters struct {
		All []map[string]any `json:"all"`
	} `json:"filters"`
	Page struct {
		Size    int `json:"size"`
		Current int `json:"current"`
	} `json:"page"`
}

type mcSearchRequest struct {
	JSON mcSearch `json:"json"`
}

type mcSearchResponse struct {
	Meta struct {
		Page struct {
			TotalPages int `json:"total_pages"`
		} `json:"page"`
	} `json:"meta"`
	Results []mcSingle `json:"results"`
}

type mcVariantsRequest struct {
	OnlyAvailable bool     `json:"onlyAvailable"`
	SKUs          []string `json:"SKUs"`
}

const (
	mcSearchBaseURL     = "https://www.cardgamecorner.com/webapi/"
	mcEditionsEndpt     = "mcexpansions/magic/it"
	mcSinglesEndpt      = "productssearch-singles"
	mcVariantsEndpt     = "productssearch-variants"
	mcSinglesPageSize   = 1000
	mcVariantsBatchSize = 1000

	mcHotBuylistURL      = "https://www.cardgamecorner.com/webapi/mcbuylist/magic/-/0"
	mcEditionBuylistURL  = "https://www.cardgamecorner.com/webapi/mclistboxes/magic/it"
	mcAdvancedBuylistURL = "https://www.cardgamecorner.com/webapi/mcadvsearch"

	mcPromoEditionID = 1113
)

// MCClient reads the Magic Corner storefront, which serves its catalog and
// its buylist through different endpoints.
type MCClient struct {
	client *http.Client
}

// NewMCClient returns a client.
func NewMCClient() *MCClient {
	mc := MCClient{}
	mc.client = mtgban.NewHTTPClient()
	return &mc
}

func (mc *MCClient) do(ctx context.Context, method, endpoint string, body, out any) error {
	var reqBody bytes.Buffer
	if body != nil {
		err := json.NewEncoder(&reqBody).Encode(body)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, mcSearchBaseURL+endpoint, &reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", endpoint, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetEditionList returns the editions Magic Corner sells, by id and name.
func (mc *MCClient) GetEditionList(ctx context.Context, addPromoEd bool) ([]MCEdition, error) {
	var editionList []MCEdition
	err := mc.do(ctx, http.MethodGet, mcEditionsEndpt, nil, &editionList)
	if err != nil {
		return nil, err
	}

	if addPromoEd {
		// This edition is not present in the normal callback
		editionList = append(editionList, MCEdition{
			ID:   mcPromoEditionID,
			Name: "Promo",
		})
	}

	return editionList, nil
}

// GetInventoryForEdition returns the cards of an edition that have any copy
// in stock, without their stock: GetVariants reads that by SKU.
func (mc *MCClient) GetInventoryForEdition(ctx context.Context, edition MCEdition) ([]MCCard, error) {
	// The loader holds no art series card
	if strings.Contains(edition.Name, "Art Series") {
		return nil, nil
	}

	var search mcSearch
	search.Filters.All = []map[string]any{
		{"game": "magic"},
		{"quantity": map[string]int{"from": 1}},
		{"idcategory": edition.ID},
	}
	search.Page.Size = mcSinglesPageSize

	var cards []MCCard
	for page := 1; ; page++ {
		search.Page.Current = page

		var response mcSearchResponse
		err := mc.do(ctx, http.MethodPost, mcSinglesEndpt, mcSearchRequest{JSON: search}, &response)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", edition.Name, err)
		}
		for i := range response.Results {
			cards = append(cards, response.Results[i].card())
		}

		if page >= response.Meta.Page.TotalPages || len(response.Results) == 0 {
			return cards, nil
		}
	}
}

// GetVariants returns the stock of the cards with the given SKUs, in every
// condition, finish and language. The search answers rows out of stock too,
// asked for available ones or not, so a caller still checks the quantity.
func (mc *MCClient) GetVariants(ctx context.Context, skus []string) ([]MCVariant, error) {
	var variants []MCVariant
	err := mc.do(ctx, http.MethodPost, mcVariantsEndpt, mcVariantsRequest{
		OnlyAvailable: true,
		SKUs:          skus,
	}, &variants)
	if err != nil {
		return nil, err
	}
	return variants, nil
}

// MCExpansion is a set as Magic Corner files it for buying, which does not
// always agree with MCEdition.
type MCExpansion struct {
	ID      int    `json:"Id"`
	Name    string `json:"Espansione"`
	Enabled bool   `json:"Enabled"`
}

// MCBuylistEditionResponse is what the buylist edition endpoint answers with.
type MCBuylistEditionResponse struct {
	Expansions []MCExpansion `json:"Expansions"`
}

// GetBuylistEditions returns the editions Magic Corner is buying from.
func (mc *MCClient) GetBuylistEditions(ctx context.Context) ([]MCExpansion, error) {
	link := mcEditionBuylistURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response MCBuylistEditionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}

	return response.Expansions, nil
}

// MCBuylistRequest is one page of a buylist query.
type MCBuylistRequest struct {
	Q              string  `json:"q"`
	Game           string  `json:"game"`
	Edition        int     `json:"edition"`
	Rarity         string  `json:"rarity"`
	Color          string  `json:"color"`
	Firstedition   string  `json:"firstedition"`
	Foil           string  `json:"foil"`
	Language       *string `json:"language"`
	Page           int     `json:"page"`
	Sort           int     `json:"sort"`
	IsBuyList      bool    `json:"isBuyList"`
	OnlyHotBuyList bool    `json:"onlyHotBuyList"`
	OnlyAvailable  bool    `json:"onlyAvailable"`
}

// MCBuylistResponse is what the buylist endpoint answers with.
type MCBuylistResponse struct {
	Result MCBuylistResult `json:"Result"`

	ID              int  `json:"Id"`
	Status          int  `json:"Status"`
	IsCanceled      bool `json:"IsCanceled"`
	IsCompleted     bool `json:"IsCompleted"`
	CreationOptions int  `json:"CreationOptions"`
	IsFaulted       bool `json:"IsFaulted"`
}

// MCBuylistResult is one page of buylist entries.
type MCBuylistResult struct {
	Products []MCProduct `json:"Products"`
	Total    int         `json:"Total"`
}

// MCProduct is one card Magic Corner is buying, with the price they pay.
type MCProduct struct {
	ID           string  `json:"Id"`
	Game         string  `json:"Game"`
	ModelEn      string  `json:"ModelEn"`
	Rarity       string  `json:"Rarity"`
	Category     string  `json:"Category"`
	Quantity     int     `json:"Quantity"`
	MinAcquisto  float64 `json:"MinAcquisto"`
	MaxAcquisto  float64 `json:"MaxAcquisto"`
	Language     int     `json:"Language"`
	SerialNumber int     `json:"SerialNumber"`
}

// GetBuylistForEdition returns one page of what Magic Corner buys in an
// edition.
func (mc *MCClient) GetBuylistForEdition(ctx context.Context, edition, page int) (*MCBuylistResult, error) {
	payload, err := json.Marshal(&MCBuylistRequest{
		IsBuyList: true,
		Game:      "magic",
		Page:      page,
		Edition:   edition,
		Sort:      5,
	})
	if err != nil {
		return nil, err
	}

	link := mcAdvancedBuylistURL
	if page > 1 {
		link = fmt.Sprintf("%s?p=%d", mcAdvancedBuylistURL, page)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, link, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response MCBuylistResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("%d: %v", resp.StatusCode, err)
	}

	return &response.Result, nil
}
