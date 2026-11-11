package cardgamecorner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mtgban/go-mtgban/mtgban"
)

// Edition is a set as Card Game Corner files it for selling.
type Edition struct {
	ID   int    `json:"Id"`
	Name string `json:"Espansione"`
}

// Card is one card on sale, with the copies in stock in each condition,
// finish and language.
type Card struct {
	ID       int
	SKU      string
	Name     string
	Edition  string
	Rarity   string
	Extra    string
	OrigName string
	URL      string
	Number   string
	Variants []Variant
}

// Variant is the stock of a card in one condition, finish and language.
type Variant struct {
	ID        int     `json:"idproduct"`
	SKU       string  `json:"sku"`
	Print     string  `json:"p1"`
	Language  string  `json:"flag"`
	Foil      string  `json:"p4"`
	Condition string  `json:"condizionishort"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// raw is how the singles search wraps every field it returns.
type raw[T any] struct {
	Raw T `json:"raw"`
}

type single struct {
	ID       raw[float64]  `json:"idproduct"`
	SKU      raw[string]   `json:"sku"`
	Model    raw[string]   `json:"model"`
	ModelEn  raw[string]   `json:"modelen"`
	Image    raw[string]   `json:"image"`
	Category raw[string]   `json:"category"`
	URL      raw[string]   `json:"url"`
	Number   raw[string]   `json:"serialnumber"`
	Rarity   raw[[]string] `json:"p2"`
}

func (s *single) card() Card {
	card := Card{
		ID:       int(s.ID.Raw),
		SKU:      s.SKU.Raw,
		Name:     s.ModelEn.Raw,
		Edition:  s.Category.Raw,
		Extra:    s.Image.Raw,
		OrigName: s.Model.Raw,
		URL:      s.URL.Raw,
		Number:   s.Number.Raw,
	}
	if len(s.Rarity.Raw) > 0 {
		card.Rarity = s.Rarity.Raw[0]
	}
	return card
}

type searchQuery struct {
	Query   string `json:"query"`
	Filters struct {
		All []map[string]any `json:"all"`
	} `json:"filters"`
	Page struct {
		Size    int `json:"size"`
		Current int `json:"current"`
	} `json:"page"`
}

type searchRequest struct {
	JSON searchQuery `json:"json"`
}

type searchResponse struct {
	Meta struct {
		Page struct {
			TotalPages int `json:"total_pages"`
		} `json:"page"`
	} `json:"meta"`
	Results []single `json:"results"`
}

type variantsRequest struct {
	OnlyAvailable bool     `json:"onlyAvailable"`
	SKUs          []string `json:"SKUs"`
}

const (
	searchBaseURL     = "https://www.cardgamecorner.com/webapi/"
	editionsEndpt     = "mcexpansions/%s/it"
	singlesEndpt      = "productssearch-singles"
	variantsEndpt     = "productssearch-variants"
	singlesPageSize   = 1000
	variantsBatchSize = 1000

	promoEditionID = 1113
)

// Client reads the Card Game Corner storefront's catalog and stock.
type Client struct {
	client *http.Client
	game   string
}

// NewClient returns a client for one of the storefront's games, named the
// way it names them (GameMagic, GamePokemon, ...).
func NewClient(game string) *Client {
	cgc := Client{}
	cgc.client = mtgban.NewHTTPClient()
	cgc.game = game
	return &cgc
}

func (cgc *Client) do(ctx context.Context, method, endpoint string, body, out any) error {
	var reqBody bytes.Buffer
	if body != nil {
		err := json.NewEncoder(&reqBody).Encode(body)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, searchBaseURL+endpoint, &reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cgc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", endpoint, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetEditionList returns the editions Card Game Corner sells, by id and name.
func (cgc *Client) GetEditionList(ctx context.Context, addPromoEd bool) ([]Edition, error) {
	var editionList []Edition
	err := cgc.do(ctx, http.MethodGet, fmt.Sprintf(editionsEndpt, cgc.game), nil, &editionList)
	if err != nil {
		return nil, err
	}

	if addPromoEd {
		// This edition is not present in the normal callback
		editionList = append(editionList, Edition{
			ID:   promoEditionID,
			Name: "Promo",
		})
	}

	return editionList, nil
}

// GetInventoryForEdition returns the cards of an edition that have any copy
// in stock, without their stock: GetVariants reads that by SKU.
func (cgc *Client) GetInventoryForEdition(ctx context.Context, edition Edition) ([]Card, error) {
	// The loader holds no art series card
	if strings.Contains(edition.Name, "Art Series") {
		return nil, nil
	}

	var search searchQuery
	search.Filters.All = []map[string]any{
		{"game": cgc.game},
		{"quantity": map[string]int{"from": 1}},
		{"idcategory": edition.ID},
	}
	search.Page.Size = singlesPageSize

	var cards []Card
	for page := 1; ; page++ {
		search.Page.Current = page

		var response searchResponse
		err := cgc.do(ctx, http.MethodPost, singlesEndpt, searchRequest{JSON: search}, &response)
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
func (cgc *Client) GetVariants(ctx context.Context, skus []string) ([]Variant, error) {
	var variants []Variant
	err := cgc.do(ctx, http.MethodPost, variantsEndpt, variantsRequest{
		OnlyAvailable: true,
		SKUs:          skus,
	}, &variants)
	if err != nil {
		return nil, err
	}
	return variants, nil
}
