package cardmarket

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	mkmProductBaseURL  = "https://api.cardmarket.com/ws/v2.0/output.json/products/"
	mkmArticlesBaseURL = "https://api.cardmarket.com/ws/v2.0/output.json/articles/"

	mkmPriceGuideURL  = "https://api.cardmarket.com/ws/v2.0/output.json/priceguide"
	mkmProductListURL = "https://api.cardmarket.com/ws/v2.0/output.json/productlist"
	mkmExpansionsURL  = "https://api.cardmarket.com/ws/v2.0/output.json/games/1/expansions"

	mkmMaxEntities = 1000
)

type MKMClient struct {
	client *retryablehttp.Client
}

func NewMKMClient(appToken, appSecret string) *MKMClient {
	tcg := MKMClient{}
	tcg.client = retryablehttp.NewClient()
	tcg.client.Logger = nil
	tcg.client.HTTPClient.Transport = &authTransport{
		Parent:    tcg.client.HTTPClient.Transport,
		AppToken:  appToken,
		AppSecret: appSecret,
	}
	return &tcg
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
		return "", err
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
		return "", err
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

func (mkm *MKMClient) MKMExpansions() (map[string]MKMExpansion, error) {
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
		return nil, err
	}

	editionMap := map[string]MKMExpansion{}
	for _, exp := range response.Expansions {
		editionMap[fmt.Sprint(exp.IdExpansion)] = exp
	}

	return editionMap, nil
}

type MKMProduct struct {
	IdProduct     int    `json:"idProduct"`
	IdMetaproduct int    `json:"idMetaproduct"`
	Name          string `json:"enName"`
	Website       string `json:"website"`
	Number        string `json:"number"`
	Expansion     struct {
		IdExpansion int    `json:"idExpansion"`
		Name        string `json:"enName"`
	} `json:"expansion"`
	PriceGuide    map[string]float64 `json:"priceGuide"`
	CountArticles int                `json:"countArticles"`
	CountFoils    int                `json:"countFoils"`
}

func (mkm *MKMClient) MKMProduct(id string) (*MKMProduct, error) {
	resp, err := mkm.client.Get(mkmProductBaseURL + id)
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
		return nil, err
	}

	return &response.Product, nil
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

func (mkm *MKMClient) MKMArticles(id string, anyLanguage bool) ([]MKMArticle, error) {
	u, err := url.Parse(mkmArticlesBaseURL + id)
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
			return nil, err
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
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
	return t.Parent.RoundTrip(req)
}
