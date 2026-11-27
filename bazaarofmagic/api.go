package bazaarofmagic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	baseURL = "https://www.bazaarofmagic.eu/en-WW"

	// singlesURL lists the singles of the sets named by sets[], and the
	// buylist takes the same filters under the Magic category.
	singlesURL = baseURL + "/query"
	buylistURL = baseURL + "/buylist/singles"
	sealedURL  = baseURL + "/c/magic-the-gathering/1000382"
	searchURL  = baseURL + "/search"

	magicCategory = "1000382"

	// Every listing stops at page 20 of 144, past which the site answers 404
	// whatever the total it reports.
	pageSize   = 144
	maxResults = 20 * pageSize

	// The site rate limits a bare client within a few dozen requests, so
	// every request wears a browser's User-Agent.
	staticUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:135.0) Gecko/20100101 Firefox/135.0"
)

// rarities split a set into slices that add back up to its total, which
// keeps each slice under maxResults.
var rarities = []string{"basic", "common", "uncommon", "rare", "mythic", "token", "special"}

// Set is one of the store's Magic expansions, by its own code and name.
type Set struct {
	Code string
	Name string
}

// Listing is one row of a listing.
type Listing struct {
	ID    string
	Title string
	URL   string
	// Finish is the store's finish code: 0 nonfoil, 1 foil, others a
	// special foil the title names.
	Finish string
	// Price is in euro; zero where the row says Bulk or nothing.
	Price float64
}

// newClient waits out the site's 429s, which it sends without a
// Retry-After, rather than giving up within seconds.
func newClient() *http.Client {
	return mtgban.NewHTTPClient(
		mtgban.WithHTTPBackoff(retryablehttp.RateLimitLinearJitterBackoff),
		mtgban.WithHTTPRetryWait(5*time.Second, 30*time.Second),
		mtgban.WithHTTPRetries(10),
	)
}

var totalRe = regexp.MustCompile(`[0-9.]+ - [0-9.]+ of ([0-9.]+)`)

func get(ctx context.Context, client *http.Client, link string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", staticUA)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", link, resp.Status)
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

// GetSets returns every Magic set the advanced search offers.
func GetSets(ctx context.Context, client *http.Client) ([]Set, error) {
	doc, err := get(ctx, client, searchURL)
	if err != nil {
		return nil, err
	}
	var sets []Set
	doc.Find(`select[name="sets[]"] option`).Each(func(_ int, s *goquery.Selection) {
		code, _ := s.Attr("value")
		if code == "" {
			return
		}
		sets = append(sets, Set{Code: code, Name: strings.TrimSpace(s.Text())})
	})
	if len(sets) == 0 {
		return nil, errors.New("no sets found in the search form")
	}
	return sets, nil
}

// SinglesQuery lists the in-stock singles of one set.
func SinglesQuery(code string) url.Values {
	v := url.Values{}
	v.Set("tab", "singles")
	v.Set("display", "stock")
	v.Set("sets[]", code)
	return v
}

// BuylistQuery lists what the store buys of one set.
func BuylistQuery(code string) url.Values {
	v := url.Values{}
	v.Set("expansion", magicCategory)
	v.Set("sets[]", code)
	return v
}

// SealedQuery lists the in-stock sealed Magic product.
func SealedQuery() url.Values {
	v := url.Values{}
	v.Set("display", "stock")
	return v
}

// GetListings returns every row of the listing at link with query v,
// slicing it by rarity, and then walking it by name from both ends, where
// one listing would run past maxResults. A listing still too long after
// that is returned as far as it reaches, with an error saying so.
func GetListings(ctx context.Context, client *http.Client, link string, v url.Values) ([]Listing, error) {
	rows, total, err := walk(ctx, client, link, v, "")
	if err != nil || total <= maxResults {
		return rows, err
	}

	seen := map[string]bool{}
	var out []Listing
	var short []string
	for _, rarity := range rarities {
		q := clone(v)
		q.Set(rarity, "1")
		rows, n, err := walk(ctx, client, link, q, "name")
		if err != nil {
			return nil, err
		}
		if n > maxResults {
			tail, _, err := walk(ctx, client, link, q, "name-desc")
			if err != nil {
				return nil, err
			}
			rows = append(rows, tail...)
		}
		var found int
		for _, row := range rows {
			if seen[row.ID] {
				continue
			}
			seen[row.ID] = true
			out = append(out, row)
			found++
		}
		if found < n {
			short = append(short, rarity)
		}
	}
	if len(out) < total {
		return out, fmt.Errorf("%s?%s: read %d of %d rows (over the cap: %s)", link, v.Encode(), len(out), total, strings.Join(short, ", "))
	}
	return out, nil
}

func clone(v url.Values) url.Values {
	out := url.Values{}
	for k, vals := range v {
		out[k] = append([]string(nil), vals...)
	}
	return out
}

// walk reads every page of one listing up to maxResults, returning the rows
// and the total the listing reports. An unsorted listing over maxResults
// returns no rows after its first page.
func walk(ctx context.Context, client *http.Client, link string, v url.Values, order string) ([]Listing, int, error) {
	q := clone(v)
	q.Set("view", "table")
	q.Set("items", strconv.Itoa(pageSize))
	switch order {
	case "name":
		q.Set("sort", "name")
	case "name-desc":
		q.Set("sort", "name")
		q.Set("arrange", "d")
	}

	var out []Listing
	var total int
	for page := 1; page*pageSize-pageSize < maxResults; page++ {
		q.Set("page", strconv.Itoa(page))
		doc, err := get(ctx, client, link+"?"+q.Encode())
		if err != nil {
			return nil, 0, err
		}
		if page == 1 {
			total = parseTotal(doc)
			// An unsorted listing over the cap is read again by rarity.
			if order == "" && total > maxResults {
				return nil, total, nil
			}
		}
		rows := parseRows(doc)
		out = append(out, rows...)
		if len(rows) < pageSize || page*pageSize >= total {
			break
		}
	}
	return out, total, nil
}

func parseTotal(doc *goquery.Document) int {
	m := totalRe.FindStringSubmatch(doc.Find(`div.pagination`).First().Text())
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.ReplaceAll(m[1], ".", ""))
	return n
}

// parseRows reads the rows of the table view.
func parseRows(doc *goquery.Document) []Listing {
	var out []Listing
	doc.Find(`table tbody tr`).Each(func(_ int, tr *goquery.Selection) {
		a := tr.Find(`a.image-hover`).First()
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		finish, _ := a.Attr("data-foil")
		out = append(out, Listing{
			ID:     path.Base(href),
			Title:  strings.TrimSpace(a.Text()),
			URL:    href,
			Finish: finish,
			Price:  parsePrice(tr.Find(`td.text-right`).First()),
		})
	})
	return out
}

// parsePrice reads a price cell. A discounted product carries its sale
// price beside the struck-through one, in either order; a buylist row says
// "Bulk" where the store takes the card only by the box.
func parsePrice(cell *goquery.Selection) float64 {
	spans := cell.Find("span")
	sale := spans.Filter(".sale")
	if sale.Length() == 0 {
		sale = spans.Not(".nonsale")
	}
	text := strings.TrimSpace(strings.ReplaceAll(sale.First().Text(), "€", ""))
	text = strings.ReplaceAll(text, ".", "")
	text = strings.Replace(text, ",", ".", 1)
	price, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0
	}
	return price
}
