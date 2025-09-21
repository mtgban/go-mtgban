package cardtrader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	ctFilterURL = "https://www.cardtrader.com/cards/%d/filter.json"

	ctBlueprintsURL  = "https://api.cardtrader.com/api/v2/blueprints/export?expansion_id="
	ctExpansionsURL  = "https://api.cardtrader.com/api/v2/expansions"
	ctMarketplaceURL = "https://api.cardtrader.com/api/v2/marketplace/products"

	ctBulkCreateURL = "https://api.cardtrader.com/api/full/v1/products/bulk_create"
	ctBulkUpdateURL = "https://api.cardtrader.com/api/full/v1/products/bulk_update"
	ctBulkDeleteURL = "https://api.cardtrader.com/api/full/v1/products/bulk_destroy"

	ctProductsExport    = "https://api.cardtrader.com/api/v2/products/export"
	ctAddProductCart    = "https://api.cardtrader.com/api/v2/cart/add"
	ctRemoveProductCart = "https://api.cardtrader.com/api/v2/cart/remove"

	MaxBulkUploadItems = 450

	GameIdMagic             = 1
	GameIdYuGiOh            = 4
	GameIdPokemon           = 5
	GameIdFleshAndBlood     = 5
	GameIdDigimon           = 8
	GameIdDragonBallSuper   = 9
	GameIdVanguard          = 10
	GameIdMyHeroAcademia    = 14
	GameIdOnePiece          = 15
	GameIdLorcana           = 18
	GameIdStarWarsUnlimited = 20
)

const (
	CategoryMagicSingles = iota + 1
	CategoryMagicTokens
	CategoryMagicOversized
	CategoryMagicBoosterBoxes
	CategoryMagicBoosters
	CategoryMagicCompleteSets
	CategoryMagicStarterDecks
	CategoryMagicEmptyPackaging
	CategoryMagicBooks
	CategoryMagicBoxDisplays
	_
	CategoryMagicSleeves
	CategoryMagicBoxedSet
	_
	CategoryMagicAlbums
	CategoryMagicDeckBoxes
	CategoryMagicPreconstructedDecks
	CategoryMagicMemorabilia
	CategoryMagicPlaymats
	CategoryMagicLifeCounter
	CategoryMagicCardStorage
	CategoryMagicDice
	CategoryMagicBundles
	CategoryMagicTournamentPrereleasePacks
	CategoryMagicDividers
	CategoryMagicBinderPages
	_
	CategoryMagicGamingStones
)

const (
	CategoryLorcanaSingles = iota + 214
	CategoryLorcanaBoosterBoxes
	CategoryLorcanaBoosters
	CategoryLorcanaBundles
	CategoryLorcanaBoxDisplays
	CategoryLorcanaStarterDecks
	CategoryLorcanaPlaymats
	CategoryLorcanaAlbums
	CategoryLorcanaSleeves
	CategoryLorcanaDeckBoxes
	_
	_
	_
	_
	_
	CategoryLorcanaMemorabilia
	CategoryLorcanaOversized
	CategoryLorcanaCompleteSets
)

type Blueprint struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	CategoryId int    `json:"category_id"`
	GameId     int    `json:"game_id"`
	Slug       string `json:"slug"`
	ScryfallId string `json:"scryfall_id"`
	Expansion  struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"expansion"`
	// Returned by product
	Properties struct {
		Number   string `json:"collector_number"`
		Language string `json:"mtg_language"`
	} `json:"properties_hash"`
	// Returned by market
	FixedProperties struct {
		Number   string `json:"collector_number"`
		Language string `json:"mtg_language"`
	} `json:"fixed_properties"`
	ExpansionId int `json:"expansion_id"`
}

type Product struct {
	Id          int    `json:"id"`
	BlueprintId int    `json:"blueprint_id"`
	Quantity    int    `json:"quantity"`
	Description string `json:"description"`
	OnVacation  bool   `json:"on_vacation"`
	Bundle      bool   `json:"bundle"`
	Properties  struct {
		Condition string `json:"condition"`
		Number    string `json:"collector_number"`
		Altered   bool   `json:"altered"`
		Signed    bool   `json:"signed"`

		MTGLanguage string `json:"mtg_language,omitempty"`
		MTGFoil     bool   `json:"mtg_foil,omitempty"`

		LorcanaLanguage string `json:"lorcana_language,omitempty"`
		LorcanaFoil     bool   `json:"lorcana_foil,omitempty"`
	} `json:"properties_hash"`
	User struct {
		Name        string `json:"username"`
		SinglesZero bool   `json:"can_sell_via_hub"`
		SealedZero  bool   `json:"can_sell_sealed_with_ct_zero"`
		CountryCode string `json:"country_code"`
		UserType    string `json:"user_type"`
	} `json:"user"`
	Price CTPrice `json:"price"`

	UserDataField string `json:"user_data_field"`
	Tag           string `json:"tag"`
	PriceCents    int    `json:"price_cents"`
	PriceCurrency string `json:"price_currency"`
}

type BlueprintError struct {
	ErrorCode string   `json:"error_code"`
	Errors    []string `json:"errors"`
	Extra     struct {
		Message string `json:"message"`
	} `json:"extra"`
	RequestId string `json:"request_id"`
}

type BlueprintFilter struct {
	Blueprint Blueprint `json:"blueprint"`
	Products  []Product `json:"products"`
}

type Expansion struct {
	Id     int    `json:"id"`
	GameId int    `json:"game_id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
}

type CTClient struct {
	client *retryablehttp.Client
}

type CTAuthClient struct {
	client *retryablehttp.Client
}

type authTransport struct {
	Parent http.RoundTripper
	Token  string
}

func NewCTAuthClient(token string) *CTAuthClient {
	ct := CTAuthClient{}
	ct.client = retryablehttp.NewClient()
	ct.client.Logger = nil
	ct.client.HTTPClient.Transport = &authTransport{
		Parent: ct.client.HTTPClient.Transport,
		Token:  token,
	}
	return &ct
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	return t.Parent.RoundTrip(req)
}

func (ct *CTAuthClient) Expansions() ([]Expansion, error) {
	resp, err := ct.client.Get(ctExpansionsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out []Expansion
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for expansions, got: %w", err)
	}

	return out, nil
}

// Returns all products from an Expansion, with the 25 cheapest listings per product
func (ct *CTAuthClient) ProductsForExpansion(id int) (map[int][]Product, error) {
	resp, err := ct.client.Get(fmt.Sprintf("%s?expansion_id=%d", ctMarketplaceURL, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out map[int][]Product
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for expansion %d, got: %w", id, err)
	}

	return out, nil
}

// Returns all products from a given blueprint id, with the 25 cheapest listings
func (ct *CTAuthClient) ProductsForBlueprint(id int) ([]Product, error) {
	resp, err := ct.client.Get(fmt.Sprintf("%s?blueprint_id=%d", ctMarketplaceURL, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out map[int][]Product
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for blueprint %d, got: %w", id, err)
	}

	return out[id], nil
}

func (ct *CTAuthClient) Blueprints(expansionId int) ([]Blueprint, error) {
	resp, err := ct.client.Get(ctBlueprintsURL + fmt.Sprint(expansionId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var blueprints []Blueprint
	err = json.Unmarshal(data, &blueprints)
	if err != nil {
		var blueprintError BlueprintError
		bpErr := json.Unmarshal(data, &blueprintError)
		if bpErr == nil {
			return nil, fmt.Errorf("%s", blueprintError.Extra.Message)
		}
		return nil, fmt.Errorf("unmarshal error for blueprints (from edition id %d), got: %s", expansionId, string(data))
	}

	return blueprints, nil
}

// This is slightly different from the main Product type
type BulkProduct struct {
	// The id of the Product to edit
	Id int `json:"id,omitempty"`

	// The id of the Blueprint to put on sale
	BlueprintId int `json:"blueprint_id,omitempty"`

	// The price of the product, indicated in your current currency
	Price float64 `json:"price,omitempty"`

	// The quantity to be put up for sale
	Quantity int `json:"quantity,omitempty"`

	// A public-facing description field
	Description *string `json:"description,omitempty"`

	// A secondary internal-only field
	UserDataField *string `json:"user_data_field,omitempty"`

	// A field visible to the vendor only
	Tag *string `json:"tag"`

	// A list of optional properties
	Properties struct {
		Condition string `json:"condition,omitempty"`
		Language  string `json:"mtg_language,omitempty"`
		Foil      bool   `json:"mtg_foil,omitempty"`
		Signed    bool   `json:"signed,omitempty"`
		Altered   bool   `json:"altered,omitempty"`
	} `json:"properties,omitempty"`
}

func (ct *CTAuthClient) ProductsExport() ([]Product, error) {
	resp, err := ct.client.Get(ctProductsExport)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var products []Product
	err = json.NewDecoder(resp.Body).Decode(&products)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return products, nil
}

// Create new listings using the products slice, separating into multiple
// requests if there are more than MaxBulkUploadItems elements. A list of
// job ids is returned to monitor the execution status.
func (ct *CTAuthClient) BulkCreate(products []BulkProduct) ([]string, error) {
	return ct.bulkOperation(ctBulkCreateURL, products)
}

// Update existing listings using the products slice, separating into multiple
// requests if there are more than MaxBulkUploadItems elements. A list of
// job ids is returned to monitor the execution status.
func (ct *CTAuthClient) BulkUpdate(products []BulkProduct) ([]string, error) {
	return ct.bulkOperation(ctBulkUpdateURL, products)
}

// Delete existing listings using the products slice, separating into multiple
// requests if there are more than MaxBulkUploadItems elements. A list of
// job ids is returned to monitor the execution status.
func (ct *CTAuthClient) BulkDelete(products []BulkProduct) ([]string, error) {
	return ct.bulkOperation(ctBulkDeleteURL, products)
}

func (ct *CTAuthClient) bulkOperation(link string, products []BulkProduct) ([]string, error) {
	var jobs []string
	var bulkUpload struct {
		Products []BulkProduct `json:"products"`
	}

	for i := 0; i < len(products); i += MaxBulkUploadItems {
		end := i + MaxBulkUploadItems
		if end > len(products) {
			end = len(products)
		}

		bulkUpload.Products = products[i:end]
		bodyBytes, err := json.Marshal(&bulkUpload)
		if err != nil {
			return nil, err
		}

		resp, err := ct.client.Post(link, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}

		var jobResp struct {
			Job string `json:"job"`
		}
		err = json.NewDecoder(resp.Body).Decode(&jobResp)
		if err != nil {
			return nil, fmt.Errorf("unmarshal error for chunk %d, got: %w", i, err)
		}

		jobs = append(jobs, jobResp.Job)
	}

	return jobs, nil
}

type ctProductCart struct {
	ProductId int  `json:"product_id"`
	Quantity  int  `json:"quantity"`
	ViaZero   bool `json:"via_cardtrader_zero"`
}

type CTPrice struct {
	Cents    int    `json:"cents"`
	Currency string `json:"currency"`
}

type CTCartResponse struct {
	ID       int `json:"id"`
	Subcarts []struct {
		ID     int `json:"id"`
		Seller struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		} `json:"seller"`
		ViaCardtraderZero bool `json:"via_cardtrader_zero"`

		CartItems []struct {
			Quantity      int    `json:"quantity"`
			PriceCents    int    `json:"price_cents"`
			PriceCurrency string `json:"price_currency"`
			Product       struct {
				ID     int    `json:"id"`
				NameEn string `json:"name_en"`
			} `json:"product"`
		} `json:"cart_items"`
	} `json:"subcarts"`

	Subtotal                         CTPrice `json:"subtotal"`
	Total                            CTPrice `json:"total"`
	SafeguardFeeAmount               CTPrice `json:"safeguard_fee_amount"`
	CtZeroFeeAmount                  CTPrice `json:"ct_zero_fee_amount"`
	PaymentMethodFeePercentageAmount CTPrice `json:"payment_method_fee_percentage_amount"`
	PaymentMethodFeeFixedAmount      CTPrice `json:"payment_method_fee_fixed_amount"`
	ShippingCost                     CTPrice `json:"shipping_cost"`

	ErrorCode string `json:"error_code"`
	Extra     struct {
		Message string `json:"message"`
	} `json:"extra"`
	RequestID string `json:"request_id"`
}

func (ct *CTAuthClient) AddProductToCart(productId, quantity int, zero bool) (*CTCartResponse, error) {
	product := ctProductCart{
		ProductId: productId,
		Quantity:  quantity,
		ViaZero:   zero,
	}
	return ct.addremoveCart(product, ctAddProductCart)
}

func (ct *CTAuthClient) RemoveProductFromCart(productId, quantity int) (*CTCartResponse, error) {
	product := ctProductCart{
		ProductId: productId,
		Quantity:  quantity,
	}
	return ct.addremoveCart(product, ctRemoveProductCart)
}

func (ct *CTAuthClient) addremoveCart(product ctProductCart, link string) (*CTCartResponse, error) {
	bodyBytes, err := json.Marshal(&product)
	if err != nil {
		return nil, err
	}

	resp, err := ct.client.Post(link, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var products CTCartResponse
	err = json.NewDecoder(resp.Body).Decode(&products)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if products.ErrorCode != "" {
		return nil, errors.New(products.Extra.Message)
	}

	return &products, nil
}

func NewCTClient() *CTClient {
	ct := CTClient{}
	ct.client = retryablehttp.NewClient()
	ct.client.Logger = nil
	return &ct
}

func (ct *CTClient) ProductsForBlueprint(id int) (*BlueprintFilter, error) {
	resp, err := ct.client.Post(fmt.Sprintf(ctFilterURL, id), "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var bf BlueprintFilter
	err = json.NewDecoder(resp.Body).Decode(&bf)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for blueprint %d, got: %w", id, err)
	}

	if bf.Blueprint.Id == 0 {
		return nil, fmt.Errorf("empty blueprint for id %d", id)
	}

	return &bf, nil
}
