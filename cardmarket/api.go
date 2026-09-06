package cardmarket

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	mkmProductsBaseURL   = "https://apiv2.cardmarket.com/ws/v2.0/output.json/products/"
	mkmArticlesBaseURL   = "https://apiv2.cardmarket.com/ws/v2.0/output.json/articles/"
	mkmExpansionsBaseURL = "https://apiv2.cardmarket.com/ws/v2.0/output.json/expansions/"

	mkmPriceGuideURL  = "https://apiv2.cardmarket.com/ws/v2.0/output.json/priceguide"
	mkmProductListURL = "https://apiv2.cardmarket.com/ws/v2.0/output.json/productlist"
	mkmExpansionsURL  = "https://apiv2.cardmarket.com/ws/v2.0/output.json/games/%d/expansions"

	// MaxEntities is how many results one request may ask for
	MaxEntities = 100
)

// MKMClient reads Cardmarket's API, which signs every request with an app
// token and secret.
type MKMClient struct {
	client *http.Client
	auth   *authTransport
}

// NewMKMClient returns a client signing with the given app credentials.
func NewMKMClient(appToken, appSecret string) *MKMClient {
	mkm := MKMClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	// The api is very sensitive to multiple concurrent requests,
	// This backoff strategy lets the system chill out a bit before retrying
	client.Backoff = retryablehttp.LinearJitterBackoff
	client.RetryWaitMin = 2 * time.Second
	client.RetryWaitMax = 10 * time.Second
	client.RetryMax = 20

	auth := &authTransport{
		Parent:    client.HTTPClient.Transport,
		AppToken:  appToken,
		AppSecret: appSecret,
	}

	client.HTTPClient.Transport = auth
	mkm.auth = auth
	mkm.client = client.StandardClient()
	return &mkm
}

// RequestNo returns how many requests the client has made, which matters
// against Cardmarket's daily allowance.
func (mkm *MKMClient) RequestNo() int {
	return int(mkm.auth.RequestNo.Load())
}

// MKMExpansion is a set as Cardmarket files it.
type MKMExpansion struct {
	IDExpansion int    `json:"idExpansion"`
	Name        string `json:"enName"`
	SetCode     string `json:"abbreviation"`
	Icon        int    `json:"icon"`
	ReleaseDate string `json:"releaseDate"`
	IsReleased  bool   `json:"isReleased"`
}

// Expansions returns every expansion of one game.
func (mkm *MKMClient) Expansions(ctx context.Context, gameID int) ([]MKMExpansion, error) {
	link := fmt.Sprintf(mkmExpansionsURL, gameID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := mkm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Expansions []MKMExpansion `json:"expansion"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, errors.New(string(data))
	}

	return response.Expansions, nil
}

// MKMProduct is one catalog entry, a card or a sealed item.
type MKMProduct struct {
	IDProduct     int    `json:"idProduct"`
	IDMetaproduct int    `json:"idMetaproduct"`
	Name          string `json:"enName"`
	Website       string `json:"website"`
	Number        string `json:"number"`
	ExpansionName string `json:"expansionName"`
	// ExpansionCode is the marketplace's own abbreviation of the
	// expansion, read off the id map rather than the product
	ExpansionCode string `json:"-"`
	Expansion     struct {
		IDExpansion int    `json:"idExpansion"`
		Name        string `json:"enName"`
	} `json:"expansion"`
	PriceGuide    map[string]float64 `json:"priceGuide"`
	CountArticles int                `json:"countArticles"`
	CountFoils    int                `json:"countFoils"`
}

// MKMProduct returns one catalog entry by id.
func (mkm *MKMClient) MKMProduct(ctx context.Context, id int) (*MKMProduct, error) {
	link := mkmProductsBaseURL + fmt.Sprint(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := mkm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Product MKMProduct `json:"product"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, errors.New(string(data))
	}

	return &response.Product, nil
}

// MKMProductsInExpansion returns every catalog entry in one expansion.
func (mkm *MKMClient) MKMProductsInExpansion(ctx context.Context, id int) ([]MKMProduct, error) {
	link := mkmExpansionsBaseURL + fmt.Sprint(id) + "/singles"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := mkm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Expansion MKMExpansion `json:"expansion"`
		Single    []MKMProduct `json:"single"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, errors.New(string(data))
	}

	return response.Single, nil
}

// MKMArticle is one seller's listing on a product.
type MKMArticle struct {
	IDArticle int `json:"idArticle"`
	IDProduct int `json:"idProduct"`
	Language  struct {
		IDLanguage   int    `json:"idLanguage"`
		LanguageName string `json:"languageName"`
	} `json:"language"`
	Comments       string  `json:"comments"`
	Price          float64 `json:"price"`
	IDCurrency     int     `json:"idCurrency"`
	CurrencyCode   string  `json:"currencyCode"`
	Count          int     `json:"count"`
	InShoppingCart bool    `json:"inShoppingCart"`
	Condition      string  `json:"condition"`
	Product        struct {
		Name      string `json:"enName"`
		Expansion string `json:"expansion"`
		Number    string `json:"nr"`
	} `json:"product"`
	Seller struct {
		IDUser   int    `json:"idUser"`
		Username string `json:"username"`
		Address  struct {
			Country string `json:"country"`
		} `json:"address"`
	}
	IsFoil    bool `json:"isFoil"`
	IsSigned  bool `json:"isSigned"`
	IsPlayset bool `json:"isPlayset"`
	IsAltered bool `json:"isAltered"`
}

// MKMSimpleArticles returns the listings on a product without the seller
// details the full call carries, which is all a price needs.
func (mkm *MKMClient) MKMSimpleArticles(ctx context.Context, id int, onlyEnglish bool, page, maxResults int) ([]MKMArticle, error) {
	options := map[string]string{
		"minCondition": "GD",
		"minUserScore": "3",
		"isSigned":     "false",
		"isAltered":    "false",
	}
	if onlyEnglish {
		options["idLanguage"] = "1"
	}

	return mkm.MKMArticles(ctx, id, options, page, maxResults)
}

// MKMArticles returns the listings on a product, one page at a time. Pages
// start at zero.
func (mkm *MKMClient) MKMArticles(ctx context.Context, id int, options map[string]string, page, maxResults int) ([]MKMArticle, error) {
	u, err := url.Parse(mkmArticlesBaseURL + fmt.Sprint(id))
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, value := range options {
		params.Set(key, value)
	}
	params.Set("start", fmt.Sprint(page*maxResults))
	params.Set("maxResults", fmt.Sprint(maxResults))
	u.RawQuery = params.Encode()

	link := u.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := mkm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// No more data to read, break to avoid a "no data" unmarshal error
	if len(data) == 0 {
		return nil, nil
	}

	var response struct {
		ErrorDescription string       `json:"mkm_error_description"`
		Articles         []MKMArticle `json:"article"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, errors.New(string(data))
	}

	if response.ErrorDescription != "" {
		return nil, errors.New(response.ErrorDescription)
	}

	return response.Articles, nil
}

type authTransport struct {
	Parent    http.RoundTripper
	AppToken  string
	AppSecret string

	// May be empty
	AccessToken       string
	AccessTokenSecret string

	RequestNo atomic.Int64
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Generate nonce
	rawID := make([]byte, 16)
	_, err := rand.Read(rawID)
	if err != nil {
		return nil, fmt.Errorf("unable to generate nonce: %w", err)
	}

	nonce := base64.RawStdEncoding.EncodeToString(rawID)

	// Items we need
	q := url.Values{}
	q.Set("oauth_consumer_key", t.AppToken)
	q.Set("oauth_nonce", nonce)
	q.Set("oauth_signature_method", "HMAC-SHA1")
	q.Set("oauth_timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	q.Set("oauth_token", t.AccessToken)
	q.Set("oauth_version", "1.0")

	for key, value := range req.URL.Query() {
		q.Set(key, value[0])
	}
	// MKM expects path-encoded queries because javascript, but q.Encode() uses
	// the query-encoding, so perform the only replacement that matters
	queries := strings.Replace(q.Encode(), "+", "%20", -1)

	// Duplicate request url and drop query parameters
	authURL := &url.URL{}
	*authURL = *req.URL
	authURL.RawQuery = ""

	// Message and key
	msg := fmt.Sprintf("%s&%s&%s", req.Method, url.QueryEscape(authURL.String()), url.QueryEscape(queries))

	signkey := fmt.Sprintf("%s&%s", url.QueryEscape(t.AppSecret), url.QueryEscape(t.AccessTokenSecret))

	mac := hmac.New(sha1.New, []byte(signkey))
	mac.Write([]byte(msg))
	msgHash := mac.Sum(nil)
	signature := base64.StdEncoding.EncodeToString(msgHash)

	// Build the header
	auth := "OAuth realm=\"" + authURL.String() + "\", "
	for key, val := range q {
		// Only keep oauth parameters here
		if !strings.HasPrefix(key, "oauth") {
			continue
		}
		auth += key + "=\"" + val[0] + "\", "
	}
	auth += "oauth_signature=\"" + signature + "\""

	req.Header.Set("Authorization", auth)

	t.RequestNo.Add(1)
	return t.Parent.RoundTrip(req)
}
