package cardmarket

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

const (
	mkmProductsBaseURL   = "https://api.cardmarket.com/ws/v2.0/output.json/products/"
	mkmArticlesBaseURL   = "https://api.cardmarket.com/ws/v2.0/output.json/articles/"
	mkmExpansionsBaseURL = "https://api.cardmarket.com/ws/v2.0/output.json/expansions/"

	mkmUserArticlesFormatURL = "https://api.cardmarket.com/ws/v2.0/output.json/users/%s/articles"

	mkmPriceGuideURL  = "https://api.cardmarket.com/ws/v2.0/output.json/priceguide"
	mkmProductListURL = "https://api.cardmarket.com/ws/v2.0/output.json/productlist"
	mkmExpansionsURL  = "https://api.cardmarket.com/ws/v2.0/output.json/games/1/expansions"

	mkmMaxEntities = 1000
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
	mkm.client.RetryWaitMin = 1 * time.Second
	mkm.client.RetryWaitMax = 5 * time.Second
	mkm.client.RetryMax = 15
	mkm.client.CheckRetry = customCheckRetry
	mkm.client.HTTPClient.Transport = &authTransport{
		Parent:    mkm.client.HTTPClient.Transport,
		AppToken:  appToken,
		AppSecret: appSecret,
		// Set a more conservative limit than defined below to avoid losing requests
		// https://www.cardmarket.com/en/Magic/News/Additional-Request-Limits-For-Our-API
		Limiter: rate.NewLimiter(9, 16),
	}
	return &mkm
}

// Implement our own retry policy to leverage the internal retry mechanism.
// The api seems to return a 200 status code with a plain-text error message
// "Too Many Requests" even when there just one in progress.
func customCheckRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	data, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(data))
	if string(data) == "Too Many Requests" {
		return true, errors.New(string(data))
	}
	return false, err
}

func (mkm *MKMClient) RequestNo() int {
	return mkm.client.HTTPClient.Transport.(*authTransport).RequestNo
}

func (mkm *MKMClient) MKMRawPriceGuide() (string, error) {
	resp, err := mkm.client.Get(mkmPriceGuideURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response struct {
		PriceGuideFile string `json:"priceguidefile"`
		MIME           string `json:"mime"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return "", errors.New(string(data))
	}

	return response.PriceGuideFile, nil
}

func (mkm *MKMClient) MKMRawProductList() (string, error) {
	resp, err := mkm.client.Get(mkmProductListURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response struct {
		ProductsFile string `json:"productsfile"`
		MIME         string `json:"mime"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return "", errors.New(string(data))
	}

	return response.ProductsFile, nil
}

type MKMExpansion struct {
	IdExpansion int    `json:"idExpansion"`
	Name        string `json:"enName"`
	SetCode     string `json:"abbreviation"`
	Icon        int    `json:"icon"`
	ReleaseDate string `json:"releaseDate"`
	IsReleased  bool   `json:"isReleased"`
}

func (mkm *MKMClient) MKMExpansions() (map[int]MKMExpansion, error) {
	resp, err := mkm.client.Get(mkmExpansionsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
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

	editionMap := map[int]MKMExpansion{}
	for _, exp := range response.Expansions {
		editionMap[exp.IdExpansion] = exp
	}

	return editionMap, nil
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

	data, err := ioutil.ReadAll(resp.Body)
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

	data, err := ioutil.ReadAll(resp.Body)
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
	}
	IsFoil    bool `json:"isFoil"`
	IsSigned  bool `json:"isSigned"`
	IsPlayset bool `json:"isPlayset"`
	IsAltered bool `json:"isAltered"`
}

func (mkm *MKMClient) MKMArticles(id int, anyLanguage bool) ([]MKMArticle, error) {
	u, err := url.Parse(mkmArticlesBaseURL + fmt.Sprint(id))
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if !anyLanguage {
		params.Set("idLanguage", "1")
	}
	params.Set("minCondition", "GD")
	params.Set("minUserScore", "3")
	params.Set("isSigned", "false")
	params.Set("isAltered", "false")

	return mkm.articles(u.String())
}

func (mkm *MKMClient) MKMUserArticles(user string) ([]MKMArticle, error) {
	return mkm.articles(fmt.Sprintf(mkmUserArticlesFormatURL, user))
}

func (mkm *MKMClient) articles(link string) ([]MKMArticle, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	params := u.Query()

	var i int
	var articles []MKMArticle
	var response struct {
		Articles []MKMArticle `json:"article"`
	}
	// Keep polling 1000 entities at a time
	for {
		params.Set("start", fmt.Sprint(i*mkmMaxEntities))
		params.Set("maxResults", fmt.Sprint(mkmMaxEntities))
		u.RawQuery = params.Encode()

		resp, err := mkm.client.Get(u.String())
		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		// No more data to read, break to avoid a "no data" unmarshal error
		if len(data) == 0 {
			break
		}

		err = json.Unmarshal(data, &response)
		if err != nil {
			return nil, errors.New(string(data))
		}

		// Stash the result
		articles = append(articles, response.Articles...)

		// No more entities left, we can break now
		if len(response.Articles) < mkmMaxEntities {
			break
		}

		// Next round
		i++
	}

	return articles, nil
}

type authTransport struct {
	Parent    http.RoundTripper
	AppToken  string
	AppSecret string

	// May be empty
	AccessToken       string
	AccessTokenSecret string

	Limiter *rate.Limiter

	RequestNo int
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	err := t.Limiter.Wait(context.Background())
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	// Items we need
	q := url.Values{}
	q.Set("oauth_consumer_key", t.AppToken)
	q.Set("oauth_nonce", uuid.New().String())
	q.Set("oauth_signature_method", "HMAC-SHA1")
	q.Set("oauth_timestamp", fmt.Sprintf("%d", timestamp))
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
		auth += key + "=\"" + val[0] + "\", "
	}
	auth += "oauth_signature=\"" + signature + "\""

	req.Header.Set("Authorization", auth)

	t.RequestNo++
	return t.Parent.RoundTrip(req)
}
