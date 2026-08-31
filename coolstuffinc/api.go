package coolstuffinc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
)

// CSICard is one card in the price list.
type CSICard struct {
	ID             int     `json:"id,string"`
	URL            string  `json:"url"`
	Name           string  `json:"name"`
	ScryfallID     string  `json:"scryfallid"`
	Variation      string  `json:"variation"`
	Edition        string  `json:"edition"`
	Language       string  `json:"language"`
	IsFoil         bool    `json:"is_foil,string"`
	PriceRetail    float64 `json:"price_retail,string"`
	QuantityRetail int     `json:"qty_retail,string"`
	PriceBuy       float64 `json:"price_buy,string"`
	QuantityBuy    int     `json:"qty_buying,string"`
}

const (
	csiPricelistURL = "https://www.coolstuffinc.com/gateway_json.php?k="

	csiBuylistURL  = "https://www.coolstuffinc.com/GeneratedFiles/SellList/Section-%s.json"
	csiBuylistLink = "https://www.coolstuffinc.com/main_selllist.php?s="
)

// csiSearchURL is a variable so a test can point the searches at a server
// of its own.
var csiSearchURL = "https://www.coolstuffinc.com/sq/"

// csiClient is shared, and retries. Every request in this file stands for
// a whole run of rows rather than a page of them - an edition's singles, a
// sealed query, the buylist, the edition map - so a connection dropped
// once discards all of it silently. A client built per call cannot pool
// anything either, and the one it was built from disables keep-alives, so
// each request paid for a fresh handshake against a storefront being asked
// for hundreds of editions at a time.
var csiClient = newCSIHTTPClient()

func newCSIHTTPClient() *http.Client {
	client := retryablehttp.NewClient()
	client.Logger = nil
	return client.StandardClient()
}

// CSIClient reads Cool Stuff Inc's price list, which needs a key.
type CSIClient struct {
	client *http.Client
	key    string
}

// NewCSIClient returns a client using the given key.
func NewCSIClient(key string) *CSIClient {
	csi := CSIClient{}
	csi.client = csiClient
	csi.key = key
	return &csi
}

// GetPriceList returns the whole price list in one call.
func (csi *CSIClient) GetPriceList(ctx context.Context) ([]CSICard, error) {
	link := csiPricelistURL + csi.key
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := csi.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pricelist struct {
		Meta struct {
			CreatedAt string `json:"created_at"`
		} `json:"meta"`
		Data []CSICard `json:"data"`
	}
	err = json.Unmarshal(data, &pricelist)
	if err != nil {
		return nil, err
	}

	return pricelist.Data, nil
}

// CSIPriceEntry is one card in the buylist feed.
type CSIPriceEntry struct {
	PID         string `json:"PID"`
	Name        string `json:"Name"`
	ItemSet     string `json:"ItemSet"`
	Notes       string `json:"Notes"`
	Price       string `json:"Price"`
	Number      string `json:"Number"`
	RarityName  string `json:"RarityName"`
	IsFoil      int    `json:"isFoil"`
	CreditPrice string `json:"CreditPrice"`
}

// GetBuylist returns what Cool Stuff Inc is buying for one game.
func GetBuylist(ctx context.Context, game string) ([]CSIPriceEntry, error) {
	link := fmt.Sprintf(csiBuylistURL, game)

	// The sell list is a large uncompressed download that occasionally
	// truncates mid-stream (unexpected EOF), so retry the whole fetch.
	const attempts = 3
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		var entries []CSIPriceEntry
		entries, err = fetchBuylist(ctx, link)
		if err == nil {
			return entries, nil
		}
		if attempt == attempts {
			break
		}
		// Back off before the next attempt, bailing out if the caller
		// gives up in the meantime
		select {
		case <-ctx.Done():
			return nil, err
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}

	return nil, err
}

func fetchBuylist(ctx context.Context, link string) ([]CSIPriceEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}

	// Disable gzip compression
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := csiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("unexpected %d status code", resp.StatusCode)
	}

	var entries []CSIPriceEntry
	err = json.NewDecoder(resp.Body).Decode(&entries)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// LoadBuylistEditions returns the edition-to-id map the storefront links are
// built from.
func LoadBuylistEditions(ctx context.Context, game string) (map[string]string, error) {
	link := csiBuylistLink + game
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := csiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	edition2id := map[string]string{}

	doc.Find(`option`).Each(func(_ int, s *goquery.Selection) {
		ed := s.Text()
		if ed == "" {
			return
		}
		id, found := s.Attr("value")
		if !found || id == "" {
			return
		}
		_, found = edition2id[ed]
		if found {
			return
		}

		edition2id[ed] = id
	})

	return edition2id, nil
}

// SearchResult is one hit from the storefront's search.
type SearchResult struct {
	PageID string
	Data   []byte
}

// Search resolves an item name to its id and returns the first page of
// results, narrowed to the given rarity tiers.
func Search(ctx context.Context, game, itemName string, skipOOS bool, rarities []string) (*SearchResult, error) {
	v := url.Values{}
	v.Set("name", "")
	v.Set("f[Artist][]", "")
	v.Add("f[Cost][]", "")
	v.Add("f[Cost][]", "")
	v.Set("f[Number][]", "")
	v.Set("f[Type][]", "")
	v.Set("f[Card+Text][]", "")
	v.Set("notes", "")
	v.Set("sign-Cost", "<")
	v.Set("sign-Power", "<")
	v.Set("f[Power][]", "")
	v.Set("sign-Toughness", "<")
	v.Set("f[Toughness][]", "")
	v.Set("sign-Loyalty", "<")
	v.Set("f[Loyalty][]", "")
	v.Set("signprice", "<")
	v.Set("price", "")
	if skipOOS {
		// This excludes all cards that lack a NM copy
		v.Set("options[instock]", "1")
	}
	// Naming every tier the game has but the sealed ones keeps sealed out
	// of a singles search, since a rarity constraint of any kind excludes
	// the rows that carry no rarity at all. An empty list asks for
	// everything, which costs a sealed row the condition parser then
	// refuses - never a card.
	for _, rarity := range rarities {
		v.Add("f[Rarity][]", rarity)
	}
	v.Set("f[ItemSet][]", itemName)
	v.Set("s", game)
	v.Set("page", "1")
	v.Set("resultsPerPage", "50")
	v.Set("submit", "Search")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, csiSearchURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "curl/8.6.0")

	resp, err := csiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	nextLink, _ := doc.Find(`span[id="nextLink"]`).Find("a").Attr("href")
	u, err := url.Parse(nextLink)
	if err != nil {
		return nil, err
	}

	clean := strings.Split(strings.TrimPrefix(u.Path, "/sq/"), "&")[0]

	return &SearchResult{
		PageID: clean,
		Data:   data,
	}, nil
}
