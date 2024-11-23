package starcitygames

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	scgInventoryURL = "https://essearchapi-na.hawksearch.com/api/v2/search"
	scgBuylistURL   = "https://search.starcitygames.com/indexes/sell_list_products_v2/search"

	scgSettingsURL = "https://sellyourcards.starcitygames.com/api/settings"

	maxResultsPerPage = 96

	GameMagic         = 1
	GameFleshAndBlood = 2
	GameLorcana       = 3
)

type SCGClient struct {
	client *http.Client
	guid   string
	bearer string

	SealedMode bool
}

func NewSCGClient(guid, bearer string) *SCGClient {
	scg := SCGClient{}
	cli := retryablehttp.NewClient()
	cli.Logger = nil
	cli.RetryMax = 10
	cli.RetryWaitMin = 2 * time.Second
	scg.client = cli.StandardClient()
	scg.guid = guid
	scg.bearer = bearer
	return &scg
}

// https://bridgeline.atlassian.net/wiki/spaces/HSKB/pages/3462479664/Hawksearch+v4.0+-+Search+API
type scgRetailRequest struct {
	Keyword         string              `json:"Keyword"`
	FacetSelections map[string][]string `json:"FacetSelections"`
	PageNo          int                 `json:"PageNo"`
	MaxPerPage      int                 `json:"MaxPerPage"`
	ClientGUID      string              `json:"clientguid"`
}

type scgSealedFacetSelection struct {
	VariantInStockOnly []string `json:"variant_instockonly"`
	ProductType        []string `json:"product_type"`
	Game               string   `json:"game"`
}

type scgRetailSealedRequest struct {
	Keyword         string                  `json:"Keyword"`
	FacetSelections scgSealedFacetSelection `json:"FacetSelections"`
	PageNo          int                     `json:"PageNo"`
	MaxPerPage      int                     `json:"MaxPerPage"`
	ClientGUID      string                  `json:"clientguid"`
}

type scgRetailResponse struct {
	Pagination struct {
		NofResults  int `json:"NofResults"`
		CurrentPage int `json:"CurrentPage"`
		MaxPerPage  int `json:"MaxPerPage"`
		NofPages    int `json:"NofPages"`
	} `json:"Pagination"`
	Results []scgRetailResult `json:"Results"`
}

type scgRetailResult struct {
	Document struct {
		Subtitle            []string `json:"subtitle"`
		UniqueID            []int    `json:"unique_id"`
		CardName            []string `json:"card_name"`
		Language            []string `json:"language"`
		Set                 []string `json:"set"`
		CollectorNumber     []string `json:"collector_number"`
		Finish              []string `json:"finish"`
		ProductType         []string `json:"product_type"`
		URLDetail           []string `json:"url_detail"`
		ItemDisplayName     []string `json:"item_display_name"`
		HawkChildAttributes []struct {
			Price           []string `json:"price"`
			VariantSKU      []string `json:"variant_sku"`
			Qty             []int    `json:"qty"`
			VariantLanguage []string `json:"variant_language"`
			Condition       []string `json:"condition"`
		} `json:"hawk_child_attributes"`
	} `json:"Document"`
	IsVisible bool `json:"IsVisible"`
}

func (scg *SCGClient) sendRetailRequest(game, page int) (*scgRetailResponse, error) {
	gameStr := "Magic: The Gathering"
	switch game {
	case GameFleshAndBlood:
		gameStr = "Flesh and Blood"
	case GameLorcana:
		gameStr = "Lorcana"
	}

	var payload []byte
	var err error
	if scg.SealedMode {
		q := scgRetailSealedRequest{
			ClientGUID: scg.guid,
			MaxPerPage: maxResultsPerPage,
			PageNo:     page,
			FacetSelections: scgSealedFacetSelection{
				VariantInStockOnly: []string{"Yes"},
				ProductType:        []string{"Sealed"},
				Game:               gameStr,
			},
		}
		payload, err = json.Marshal(&q)
	} else {
		q := scgRetailRequest{
			ClientGUID: scg.guid,
			MaxPerPage: maxResultsPerPage,
			PageNo:     page,
			FacetSelections: map[string][]string{
				"variant_instockonly": {"Yes"},
				"product_type":        {"Singles"},
				"game":                {gameStr},
			},
		}
		payload, err = json.Marshal(&q)
	}
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, scgInventoryURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-HawkSearch-IgnoreTracking", "true")
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

	var search scgRetailResponse
	err = json.Unmarshal(data, &search)
	if err != nil {
		return nil, err
	}

	return &search, nil
}

func (scg *SCGClient) NumberOfPages(game int) (int, error) {
	response, err := scg.sendRetailRequest(game, 0)
	if err != nil {
		return 0, err
	}
	return response.Pagination.NofPages, nil
}

func (scg *SCGClient) GetPage(game, page int) ([]scgRetailResult, error) {
	response, err := scg.sendRetailRequest(game, page)
	if err != nil {
		return nil, err
	}
	return response.Results, nil
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
	ProductType     string           `json:"product_type"`
	Finish          string           `json:"finish"`
	Language        string           `json:"language"`
	Rarity          string           `json:"rarity"`
	IsBuying        int              `json:"is_buying"`
	Hotlist         int              `json:"hotlist"`
	BorderColor     string           `json:"border_color"`
	CollectorNumber string           `json:"collector_number"`
	SetID           int              `json:"set_id"`
	SetName         string           `json:"set_name"`
	SetReleaseDate  int              `json:"set_release_date"`
	SetSymbol       string           `json:"set_symbol"`
	Variants        []SCGCardVariant `json:"variants"`
}

type SCGCardVariant struct {
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

type Settings struct {
	CardRarities []struct {
		ID               int       `json:"id"`
		Name             string    `json:"name"`
		GameID           int       `json:"game_id"`
		Abbr             string    `json:"abbr"`
		ExternalRarityID int       `json:"external_rarity_id"`
		SortOrder        int       `json:"sort_order"`
		CreatedAt        time.Time `json:"created_at"`
		UpdatedAt        time.Time `json:"updated_at"`
	} `json:"cardRarities"`
}

func SearchSettings() (*Settings, error) {
	resp, err := cleanhttp.DefaultClient().Get(scgSettingsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var search Settings
	err = json.Unmarshal(data, &search)
	if err != nil {
		return nil, err
	}

	return &search, nil
}

func (scg *SCGClient) SearchAll(game, offset, limit int, rarity string) (*SCGSearchResponse, error) {
	filter := `game_id = %d AND price_category_id = %s AND is_buying = 1 AND NOT primary_status IN ["do_not_show", "buying_in_bulk"]`
	mode := "1"
	if scg.SealedMode {
		mode = "2"
	}
	query := fmt.Sprintf(filter, game, mode)

	if rarity != "" {
		query += ` AND rarity = "` + rarity + `"`
	}

	q := SCGSearchRequest{
		Filter:           query,
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

const baseURL = "https://starcitygames.com"

func SCGProductURL(URLDetail, variantSKU []string, affiliate string) string {
	if len(URLDetail) == 0 {
		return ""
	}

	link := baseURL + URLDetail[0]
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}

	v := u.Query()
	if len(variantSKU) > 0 {
		v.Set("sku", variantSKU[0])
	}
	if affiliate != "" {
		v.Set("aff", affiliate)
	}
	u.RawQuery = v.Encode()

	return u.String()
}
