package cardmarket

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	mkmProductsBaseURL   = "https://api.cardmarket.com/ws/v2.0/output.json/products/"
	mkmArticlesBaseURL   = "https://api.cardmarket.com/ws/v2.0/output.json/articles/"
	mkmExpansionsBaseURL = "https://api.cardmarket.com/ws/v2.0/output.json/expansions/"

	mkmPriceGuideURL  = "https://api.cardmarket.com/ws/v2.0/output.json/priceguide"
	mkmProductListURL = "https://api.cardmarket.com/ws/v2.0/output.json/productlist"
	mkmExpansionsURL  = "https://api.cardmarket.com/ws/v2.0/output.json/games/%d/expansions"

	MaxEntities = 100
)

type MKMClient struct {
	client *retryablehttp.Client
}

func NewMKMClient(appToken, appSecret string) *MKMClient {
	mkm := MKMClient{}
	mkm.client = retryablehttp.NewClient()
	mkm.client.Logger = nil
	// The api is very sensitive to multiple concurrent requests,
	// This backoff strategy lets the system chill out a bit before retrying
	mkm.client.Backoff = retryablehttp.LinearJitterBackoff
	mkm.client.RetryWaitMin = 2 * time.Second
	mkm.client.RetryWaitMax = 10 * time.Second
	mkm.client.RetryMax = 20
	mkm.client.HTTPClient.Transport = &authTransport{
		Parent:    mkm.client.HTTPClient.Transport,
		AppToken:  appToken,
		AppSecret: appSecret,
	}
	return &mkm
}

func (mkm *MKMClient) RequestNo() int {
	return mkm.client.HTTPClient.Transport.(*authTransport).RequestNo
}

type MKMExpansion struct {
	IdExpansion int    `json:"idExpansion"`
	Name        string `json:"enName"`
	SetCode     string `json:"abbreviation"`
	Icon        int    `json:"icon"`
	ReleaseDate string `json:"releaseDate"`
	IsReleased  bool   `json:"isReleased"`
}

func (mkm *MKMClient) Expansions(gameId int) ([]MKMExpansion, error) {
	resp, err := mkm.client.Get(fmt.Sprintf(mkmExpansionsURL, gameId))
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

type MKMProduct struct {
	IdProduct     int    `json:"idProduct"`
	IdMetaproduct int    `json:"idMetaproduct"`
	Name          string `json:"enName"`
	Website       string `json:"website"`
	Number        string `json:"number"`
	ExpansionName string `json:"expansionName"`
	Expansion     struct {
		IdExpansion int    `json:"idExpansion"`
		Name        string `json:"enName"`
	} `json:"expansion"`
	PriceGuide    map[string]float64 `json:"priceGuide"`
	CountArticles int                `json:"countArticles"`
	CountFoils    int                `json:"countFoils"`
}

func (mkm *MKMClient) MKMProduct(id int) (*MKMProduct, error) {
	resp, err := mkm.client.Get(mkmProductsBaseURL + fmt.Sprint(id))
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

func (mkm *MKMClient) MKMProductsInExpansion(id int) ([]MKMProduct, error) {
	resp, err := mkm.client.Get(mkmExpansionsBaseURL + fmt.Sprint(id) + "/singles")
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

type MKMArticle struct {
	IdArticle int `json:"idArticle"`
	IdProduct int `json:"idProduct"`
	Language  struct {
		IdLanguage   int    `json:"idLanguage"`
		LanguageName string `json:"languageName"`
	} `json:"language"`
	Comments       string  `json:"comments"`
	Price          float64 `json:"price"`
	IdCurrency     int     `json:"idCurrency"`
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
		IdUser   int    `json:"idUser"`
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

func (mkm *MKMClient) MKMSimpleArticles(id int, onlyEnglish bool, page, maxResults int) ([]MKMArticle, error) {
	options := map[string]string{
		"minCondition": "GD",
		"minUserScore": "3",
		"isSigned":     "false",
		"isAltered":    "false",
	}
	if onlyEnglish {
		options["idLanguage"] = "1"
	}

	return mkm.MKMArticles(id, options, page, maxResults)
}

// Note that page should start from 0
func (mkm *MKMClient) MKMArticles(id int, options map[string]string, page, maxResults int) ([]MKMArticle, error) {
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

	resp, err := mkm.client.Get(u.String())
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

	RequestNo int
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Items we need
	q := url.Values{}
	q.Set("oauth_consumer_key", t.AppToken)
	q.Set("oauth_nonce", uuid.New().String())
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
	authUrl := &url.URL{}
	*authUrl = *req.URL
	authUrl.RawQuery = ""

	// Message and key
	msg := fmt.Sprintf("%s&%s&%s", req.Method, url.QueryEscape(authUrl.String()), url.QueryEscape(queries))

	signkey := fmt.Sprintf("%s&%s", url.QueryEscape(t.AppSecret), url.QueryEscape(t.AccessTokenSecret))

	mac := hmac.New(sha1.New, []byte(signkey))
	mac.Write([]byte(msg))
	msgHash := mac.Sum(nil)
	signature := base64.StdEncoding.EncodeToString(msgHash)

	// Build the header
	auth := "OAuth realm=\"" + authUrl.String() + "\", "
	for key, val := range q {
		// Only keep oauth parameters here
		if !strings.HasPrefix(key, "oauth") {
			continue
		}
		auth += key + "=\"" + val[0] + "\", "
	}
	auth += "oauth_signature=\"" + signature + "\""

	req.Header.Set("Authorization", auth)

	t.RequestNo++
	return t.Parent.RoundTrip(req)
}
