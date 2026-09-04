package cardtrader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	ctBlueprintsURL  = "https://api.cardtrader.com/api/v2/blueprints/export?expansion_id="
	ctExpansionsURL  = "https://api.cardtrader.com/api/v2/expansions"
	ctMarketplaceURL = "https://api.cardtrader.com/api/v2/marketplace/products"

	ctBulkCreateURL = "https://api.cardtrader.com/api/full/v1/products/bulk_create"
	ctBulkUpdateURL = "https://api.cardtrader.com/api/full/v1/products/bulk_update"
	ctBulkDeleteURL = "https://api.cardtrader.com/api/full/v1/products/bulk_destroy"

	ctProductsExport    = "https://api.cardtrader.com/api/v2/products/export"
	ctAddProductCart    = "https://api.cardtrader.com/api/v2/cart/add"
	ctRemoveProductCart = "https://api.cardtrader.com/api/v2/cart/remove"

	// MaxBulkUploadItems is how many listings one bulk request may carry
	MaxBulkUploadItems = 450
)

// The games Card Trader carries, as their catalog numbers them.
const (
	GameMagic             = 1
	GameYuGiOh            = 4
	GamePokemon           = 5
	GameFleshAndBlood     = 6
	GameDigimon           = 8
	GameDragonBallSuper   = 9
	GameVanguard          = 10
	GameMyHeroAcademia    = 14
	GameOnePiece          = 15
	GameLorcana           = 18
	GameStarWarsUnlimited = 20
	GameUnionArena        = 21
	GameRiftbound         = 22
	GameGundam            = 23
)

// The catalog categories. Card Trader splits every game into product types of
// its own, and a blueprint belongs to exactly one of them.
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

// The Lorcana categories.
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

	CategoryLorcanaTokens = 270
)

// The Riftbound categories.
const (
	CategoryRiftboundSingles = iota + 258
	CategoryRiftboundBoosterBoxes
	CategoryRiftboundBoosters
	CategoryRiftboundBundles
	CategoryRiftboundStarterDecks
	CategoryRiftboundBoxDisplays
	CategoryRiftboundPlaymats
	CategoryRiftboundAlbums
	CategoryRiftboundSleeves
	CategoryRiftboundDeckBoxes
	CategoryRiftboundMemorabilia
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	CategoryRiftboundCompleteSets
	CategoryRiftboundOversized
)

// The One Piece category ids are not contiguous with any other game's, so
// they are spelled out rather than derived from an iota base. Singles is
// the only one the scrapers test for; everything else lands on the sealed
// side by exclusion, DON!! cards included, where the product-map
// resolution drops what is not a real sealed product.
const (
	CategoryOnePieceSingles         = 192
	CategoryOnePieceBoosterBoxes    = 193
	CategoryOnePieceBoosters        = 194
	CategoryOnePieceStarterDecks    = 195
	CategoryOnePiecePlaymats        = 196
	CategoryOnePieceSleeves         = 197
	CategoryOnePieceDeckBoxes       = 198
	CategoryOnePieceAlbums          = 199
	CategoryOnePieceBundles         = 200
	CategoryOnePieceBoxSetsDisplays = 201
	CategoryOnePieceUncutSheets     = 253
	CategoryOnePieceDon             = 255
	CategoryOnePieceTins            = 256
	CategoryOnePieceMemorabilia     = 257
	CategoryOnePieceOversized       = 299
)

// The Yu-Gi-Oh categories.
const (
	CategoryYuGiOhSingles               = 44
	CategoryYuGiOhSleeves               = 45
	CategoryYuGiOhPlaymats              = 46
	CategoryYuGiOhStarterDecks          = 47
	CategoryYuGiOhDeckBoxes             = 49
	CategoryYuGiOhAlbums                = 50
	CategoryYuGiOhBooks                 = 51
	CategoryYuGiOhMemorabilia           = 52
	CategoryYuGiOhBoosters              = 53
	CategoryYuGiOhBoosterBoxes          = 54
	CategoryYuGiOhTins                  = 55
	CategoryYuGiOhEmptyTinsStorage      = 56
	CategoryYuGiOhDividers              = 57
	CategoryYuGiOhPreconstructedDecks   = 70
	CategoryYuGiOhBundles               = 72
	CategoryYuGiOhDice                  = 75
	CategoryYuGiOhOversized             = 76
	CategoryYuGiOhSpecialDeluxeEditions = 117
)

// The Pokemon categories.
const (
	CategoryPokemonTins             = 59
	CategoryPokemonBoxSet           = 60
	CategoryPokemonMemorabilia      = 61
	CategoryPokemonSleeves          = 62
	CategoryPokemonPlaymats         = 63
	CategoryPokemonDeckBoxes        = 64
	CategoryPokemonAlbums           = 65
	CategoryPokemonBooster          = 66
	CategoryPokemonBoosterBox       = 67
	CategoryPokemonBundle           = 68
	CategoryPokemonPreconstructed   = 69
	CategoryPokemonSingles          = 73
	CategoryPokemonDividers         = 74
	CategoryPokemonOversized        = 78
	CategoryPokemonDice             = 86
	CategoryPokemonEmptyBoxsStorage = 118
	CategoryPokemonCompleteSet      = 136
	CategoryPokemonBlisters         = 190
)

// The Gundam category ids run 272-281 and then jump to 287: 282 is Lorcana's
// and 283-286 belong to Riftbound and Flesh and Blood, so the run is spelled
// out rather than derived from an iota base, the way One Piece's is.
const (
	CategoryGundamSingles      = 272
	CategoryGundamBoosterBoxes = 273
	CategoryGundamBoosters     = 274
	CategoryGundamStarterDecks = 275
	CategoryGundamPlaymats     = 276
	CategoryGundamSleeves      = 277
	CategoryGundamBundles      = 278
	CategoryGundamDeckBoxes    = 279
	CategoryGundamMemorabilia  = 280
	CategoryGundamBoxDisplays  = 281
	CategoryGundamDice         = 287
)

// The Flesh and Blood categories.
const (
	CategoryFleshAndBloodSingles             = 80
	CategoryFleshAndBloodBoosterBoxes        = 81
	CategoryFleshAndBloodBoosters            = 82
	CategoryFleshAndBloodPreconstructedDecks = 83
	CategoryFleshAndBloodDecksDisplays       = 84
	CategoryFleshAndBloodPlaymats            = 85
	CategoryFleshAndBloodBooksGuides         = 177
	CategoryFleshAndBloodBoxSets             = 179
	CategoryFleshAndBloodSleeves             = 183
	CategoryFleshAndBloodDice                = 212
	CategoryFleshAndBloodArtCardTokens       = 285
	CategoryFleshAndBloodCompleteSets        = 286
)

// Blueprint is Card Trader's catalog entry for a card or sealed product, the
// thing listings are sold against.
type Blueprint struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	GameID      int    `json:"game_id"`
	CategoryID  int    `json:"category_id"`
	ExpansionID int    `json:"expansion_id"`
	ScryfallID  string `json:"scryfall_id"`
	TCGplayerID int    `json:"tcg_player_id"`
	// Every Cardmarket product this blueprint sells as, the language
	// variants included: cardtrader is the one source linking the two
	// marketplaces' ids, which the sealed bridge in bantool leans on.
	CardMarketIDs []int `json:"card_market_ids"`

	Expansion struct {
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
	// Needed to extract year for certain sets
	Slug string `json:"slug"`
}

// Product is one listing: a blueprint offered by a seller at a price, in a
// condition and a language.
type Product struct {
	ID          int    `json:"id"`
	BlueprintID int    `json:"blueprint_id"`
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

		RiftboundLanguage string `json:"riftbound_language,omitempty"`
		RiftboundFoil     bool   `json:"riftbound_foil,omitempty"`

		OnePieceLanguage string `json:"onepiece_language,omitempty"`
		OnePieceFoil     bool   `json:"onepiece_foil,omitempty"`

		// Gundam carries no foil property either, and needs none: every
		// printing of it is a product of its own, told apart by the
		// rarity its number or its version names.
		GundamLanguage string `json:"gundam_language,omitempty"`

		// Yu-Gi-Oh carries no foil property: the rarity is the finish.
		// Its treatment is the print run instead, which every listing
		// names through FirstEdition below.
		YuGiOhLanguage string `json:"yugioh_language,omitempty"`

		// The Flesh and Blood finish is a named treatment ("Regular",
		// "Rainbow Foil", "Cold Foil") rather than a boolean, and crosses
		// with the print run FirstEdition names.
		FabLanguage string `json:"fab_language,omitempty"`
		FabFoilNew  string `json:"fab_foil_new,omitempty"`

		// FirstEdition is the print run, which the games selling one
		// carry beside whatever else names their treatment.
		FirstEdition bool `json:"first_edition,omitempty"`

		// Pokemon names its treatment with two flags rather than one, and
		// needs to: a holo rare's own printing is already a foil one, so a
		// single bit cannot say whether the reverse holo beside it is the
		// one being priced. The print run is the FirstEdition flag above.
		PokemonLanguage string `json:"pokemon_language,omitempty"`
		PokemonReverse  bool   `json:"pokemon_reverse,omitempty"`
	} `json:"properties_hash"`
	User struct {
		Name        string `json:"username"`
		SinglesZero bool   `json:"can_sell_via_hub"`
		SealedZero  bool   `json:"can_sell_sealed_with_ct_zero"`
		CountryCode string `json:"country_code"`
		UserType    string `json:"user_type"`
	} `json:"user"`

	UserDataField string `json:"user_data_field"`
	Tag           string `json:"tag"`

	// There are 3 difference places the price can be found
	PriceCents    int     `json:"price_cents"`
	PriceCurrency string  `json:"price_currency"`
	Price         CTPrice `json:"price"`
	BuyerPrice    CTPrice `json:"buyer_price"`
}

// BlueprintError is what the catalog answers with when a blueprint cannot be
// served.
type BlueprintError struct {
	ErrorCode string   `json:"error_code"`
	Errors    []string `json:"errors"`
	Extra     struct {
		Message string `json:"message"`
	} `json:"extra"`
	RequestID string `json:"request_id"`
}

// Expansion is a set as Card Trader files it, which does not always agree with
// the game's own sets.
type Expansion struct {
	ID     int    `json:"id"`
	GameID int    `json:"game_id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
}

// CTAuthClient reads the full Card Trader API, which needs a token and can see
// your own listings as well as the public catalog.
type CTAuthClient struct {
	client *http.Client
}

type authTransport struct {
	Parent http.RoundTripper
	Token  string
}

// NewCTAuthClient returns a client authenticated with the given token.
func NewCTAuthClient(token string) *CTAuthClient {
	ct := CTAuthClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	// A full catalog walk gets rate limited partway through; back off for
	// longer than the default to wait a 429 out rather than fail on it.
	client.RetryMax = 10
	client.RetryWaitMax = 90 * time.Second
	client.HTTPClient.Transport = &authTransport{
		Parent: client.HTTPClient.Transport,
		Token:  token,
	}
	ct.client = client.StandardClient()
	return &ct
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	return t.Parent.RoundTrip(req)
}

// Expansions returns every expansion in the catalog, across all games.
func (ct *CTAuthClient) Expansions(ctx context.Context) ([]Expansion, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ctExpansionsURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := ct.client.Do(req)
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

// ProductsForExpansion returns every product in an expansion, each with its 25
// cheapest listings.
func (ct *CTAuthClient) ProductsForExpansion(ctx context.Context, id int) (map[int][]Product, error) {
	link := fmt.Sprintf("%s?expansion_id=%d", ctMarketplaceURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := ct.client.Do(req)
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

// ProductsForBlueprint returns every product sold against one blueprint, each
// with its 25 cheapest listings.
func (ct *CTAuthClient) ProductsForBlueprint(ctx context.Context, id int) ([]Product, error) {
	link := fmt.Sprintf("%s?blueprint_id=%d", ctMarketplaceURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := ct.client.Do(req)
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

// Blueprints returns every catalog entry in one expansion.
func (ct *CTAuthClient) Blueprints(ctx context.Context, expansionID int) ([]Blueprint, error) {
	link := ctBlueprintsURL + fmt.Sprint(expansionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := ct.client.Do(req)
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
		return nil, fmt.Errorf("unmarshal error for blueprints (from edition id %d), got: %s", expansionID, string(data))
	}

	return blueprints, nil
}

// GetOrderProducts returns what an order contained.
func (ct *CTAuthClient) GetOrderProducts(ctx context.Context, orderID int) ([]Product, error) {
	link := fmt.Sprintf("https://api.cardtrader.com/api/v2/orders/%d", orderID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := ct.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var order struct {
		OrderItems []Product `json:"order_items"`
	}
	err = json.NewDecoder(resp.Body).Decode(&order)
	if err != nil {
		return nil, err
	}

	return order.OrderItems, nil
}

// BulkProduct is a listing as the bulk endpoints take it, which differs
// slightly from the Product they return.
type BulkProduct struct {
	// The id of the Product to edit
	ID int `json:"id,omitempty"`

	// The id of the Blueprint to put on sale
	BlueprintID int `json:"blueprint_id,omitempty"`

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
	} `json:"properties"`
}

// ProductsExport returns your own listings.
func (ct *CTAuthClient) ProductsExport(ctx context.Context) ([]Product, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ctProductsExport, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := ct.client.Do(req)
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

// BulkCreate creates new listings, splitting into several requests past
// MaxBulkUploadItems. It returns the job ids to watch for completion.
func (ct *CTAuthClient) BulkCreate(ctx context.Context, products []BulkProduct) ([]string, error) {
	return ct.bulkOperation(ctx, ctBulkCreateURL, products)
}

// BulkUpdate updates existing listings, splitting into several requests past
// MaxBulkUploadItems. It returns the job ids to watch for completion.
func (ct *CTAuthClient) BulkUpdate(ctx context.Context, products []BulkProduct) ([]string, error) {
	return ct.bulkOperation(ctx, ctBulkUpdateURL, products)
}

// BulkDelete removes listings, splitting into several requests past
// MaxBulkUploadItems. It returns the job ids to watch for completion.
func (ct *CTAuthClient) BulkDelete(ctx context.Context, products []BulkProduct) ([]string, error) {
	return ct.bulkOperation(ctx, ctBulkDeleteURL, products)
}

func (ct *CTAuthClient) bulkOperation(ctx context.Context, link string, products []BulkProduct) ([]string, error) {
	var jobs []string
	var bulkUpload struct {
		Products []BulkProduct `json:"products"`
	}

	for i := 0; i < len(products); i += MaxBulkUploadItems {
		end := min(i+MaxBulkUploadItems, len(products))

		bulkUpload.Products = products[i:end]
		bodyBytes, err := json.Marshal(&bulkUpload)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, link, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := ct.client.Do(req)
		if err != nil {
			return nil, err
		}

		var jobResp struct {
			Job string `json:"job"`
		}
		err = json.NewDecoder(resp.Body).Decode(&jobResp)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("unmarshal error for chunk %d, got: %w", i, err)
		}

		jobs = append(jobs, jobResp.Job)
	}

	return jobs, nil
}

type ctProductCart struct {
	ProductID int  `json:"product_id"`
	Quantity  int  `json:"quantity"`
	ViaZero   bool `json:"via_cardtrader_zero"`
}

// CTPrice is an amount with the currency it was quoted in, which is not always
// euro.
type CTPrice struct {
	Cents    int    `json:"cents"`
	Currency string `json:"currency"`
}

// CTCartResponse is what the cart endpoints answer with.
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

// AddProductToCart puts a quantity of a listing into the cart, on the Zero
// storefront when asked.
func (ct *CTAuthClient) AddProductToCart(ctx context.Context, productID, quantity int, zero bool) (*CTCartResponse, error) {
	product := ctProductCart{
		ProductID: productID,
		Quantity:  quantity,
		ViaZero:   zero,
	}
	return ct.addremoveCart(ctx, product, ctAddProductCart)
}

// RemoveProductFromCart takes a quantity of a listing back out of the cart.
func (ct *CTAuthClient) RemoveProductFromCart(ctx context.Context, productID, quantity int) (*CTCartResponse, error) {
	product := ctProductCart{
		ProductID: productID,
		Quantity:  quantity,
	}
	return ct.addremoveCart(ctx, product, ctRemoveProductCart)
}

func (ct *CTAuthClient) addremoveCart(ctx context.Context, product ctProductCart, link string) (*CTCartResponse, error) {
	bodyBytes, err := json.Marshal(&product)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, link, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ct.client.Do(req)
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
