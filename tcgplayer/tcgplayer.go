package tcgplayer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/PuerkitoBio/goquery"
	cleanhttp "github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
)

const (
	defaultConcurrency = 8
	defaultAPIRetry    = 5

	pagesPerRequest = 50
	tcgBaseURL      = "https://shop.tcgplayer.com/productcatalog/product/getpricetable?productId=0&gameName=magic&useV2Listings=true&page=0&pageSize=0&sortValue=price"

	tcgApiVersion    = "v1.37.0"
	tcgApiProductURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/product/"
	tcgApiPricingURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/sku/"
	tcgApiBuylistURL = "https://api.tcgplayer.com/" + tcgApiVersion + "/pricing/buy/product/"
	tcgApiSKUURL     = "https://api.tcgplayer.com/" + tcgApiVersion + "/catalog/products/%s/skus"
	tcgApiSearchURL  = "https://api.tcgplayer.com/" + tcgApiVersion + "/catalog/categories/1/search"
)

type requestChan struct {
	TCGProductId string
	UUID         string
	retry        int
}

type responseChan struct {
	cardId string
	entry  mtgban.InventoryEntry
	bl     mtgban.BuylistEntry
}

func getListingsNumber(client *http.Client, productId string) (int, error) {
	u, _ := url.Parse(tcgBaseURL)
	q := u.Query()
	q.Set("productId", productId)
	q.Set("pageSize", fmt.Sprintf("%d", 1))
	q.Set("page", fmt.Sprintf("%d", 1))
	u.RawQuery = q.Encode()

	resp, err := client.Get(u.String())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	viewingResults := doc.Find("span[class='sort-toolbar__total-item-count']").Text()
	results := strings.Fields(viewingResults)
	if len(results) < 3 {
		return 0, fmt.Errorf("unknown pagination for %d: %q", productId, viewingResults)
	}
	entriesNum, err := strconv.Atoi(results[3])
	if err != nil {
		return 0, err
	}

	return entriesNum, nil
}

type authTransport struct {
	Parent    http.RoundTripper
	PublicId  string
	PrivateId string
	token     string
	expires   time.Time
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token == "" || t.expires.After(time.Now()) {
		if t.PublicId == "" || t.PrivateId == "" {
			return nil, fmt.Errorf("missing public or private id")
		}
		params := url.Values{}
		params.Set("grant_type", "client_credentials")
		params.Set("client_id", t.PublicId)
		params.Set("client_secret", t.PrivateId)
		body := strings.NewReader(params.Encode())

		r, err := http.NewRequest("POST", "https://api.tcgplayer.com/token", body)
		if err != nil {
			return nil, err
		}
		r.Header.Add("Content-Type", "application/json")

		client := cleanhttp.DefaultClient()
		resp, err := client.Do(r)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var response struct {
			AccessToken string        `json:"access_token"`
			Expires     time.Duration `json:"expires"`
		}
		err = json.Unmarshal(data, &response)
		if err != nil {
			return nil, err
		}

		t.token = response.AccessToken
		t.expires = time.Now().Add(response.Expires)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.token))
	return t.Parent.RoundTrip(req)
}
