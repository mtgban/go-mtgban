package manaleak

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	// The retail singles all file under the Wizards of the Coast brand page;
	// the buylist mirrors the category tree, where 416 is the singles.
	inventoryURL = "https://www.manaleak.com/wizards-of-the-coast"
	buylistURL   = "https://www.manaleak.com/index.php?route=buylist/category&path=416"

	// A product page, which the storefront serves without a redirect.
	productURL = "https://www.manaleak.com/index.php?route=product/product&product_id="

	// The widest page the storefront serves. The sort is pinned so the
	// pages keep their order while the crawl fans out over them.
	pageLimit = 100
	// The most pages a listing is walked over, far past any real listing.
	maxPages = 20000

	// The server answers 406 to a bare client, so every request wears a
	// browser's User-Agent and Accept.
	staticUA     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:135.0) Gecko/20100101 Firefox/135.0"
	staticAccept = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
)

// MLProduct is one listing row: a product in one finish, priced in pounds.
type MLProduct struct {
	Name       string
	SetName    string
	URL        string
	Price      float64
	OutOfStock bool

	// The storefront's own id, which both of its carts take
	ProductID string

	// The card behind the row, read from its image file: the newer sets
	// name theirs by TCGplayer product id, the older by multiverse id, and
	// the file name says which era it is.
	TCGProductID string
	MultiverseID string

	// AmbiguousID marks a TCGProductID read off a filename that carries
	// neither era's own marker, so match() confirms the name agrees before
	// trusting it.
	AmbiguousID bool

	// CodeID is the bare number of a product code, in whichever of the two
	// id spaces the storefront took it from; match() confirms the name
	// agrees in each.
	CodeID string

	// Shelf is the letters a product code opens with, which name the
	// storefront's own shelf of promos.
	Shelf string

	// Number is the index the product code ends in. Where the storefront
	// took it from the collector number, it picks the printing on a shelf
	// that holds many sets' promos.
	Number string

	// Foil is the finish the product code states, which a name does not
	// always repeat.
	Foil bool

	// FromPage marks a row whose image names no card and whose product
	// page names one, by a "mtg" product code.
	FromPage bool
}

// cardImage matches a card image path: "<folder>/<id>_200w-..." (TCGplayer
// id), "<folder>/<id>.full-..." (multiverse id), or the bare
// "<folder>/<id>-WxH..." form, which names neither era (see AmbiguousID).
// <folder> is "mtg/<set>/" or "mtg<set>/" glued into one path segment.
var cardImage = regexp.MustCompile(`/mtg[^/]*/(?:[^/]+/)?(\d+)(_200w|\.full|-\d+x\d+\.\w+)`)

// codeTCG and codeBare read a product page's code, "mtg" removed: a set code
// followed by a TCGplayer product id, or the bare number of an id. A trailing
// "-f" or "-jp-f" states the finish. Other shapes (a set code and a collector
// number, or the storefront's own "FNMP_145" index) carry no id and fall to the
// set under Categories.
var (
	codeTCG  = regexp.MustCompile(`^[A-Za-z0-9]*?[A-Za-z](\d{5,})(?:-.*)?$`)
	codeBare = regexp.MustCompile(`^(\d{4,})(?:-.*)?$`)
	codeIdx  = regexp.MustCompile(`^[A-Za-z0-9]*?[A-Za-z_](\d{1,4})[a-z]?(?:-.*)?$`)

	codeShelf = regexp.MustCompile(`^[A-Za-z]+`)
)

// maxTCGDigits is the width of a TCGplayer product id. A set code ending in a
// digit ("CMB1") leaves that digit in front of the id.
const maxTCGDigits = 6

// showingTotal is the listing's own count of everything it paginates.
var showingTotal = regexp.MustCompile(`Showing \d+ to \d+ of (\d+)`)

// readImage classifies src against cardImage and fills the id field it
// names.
func (p *MLProduct) readImage(src string) {
	m := cardImage.FindStringSubmatch(src)
	if m == nil {
		return
	}
	switch m[2] {
	case "_200w":
		p.TCGProductID = m[1]
	case ".full":
		p.MultiverseID = m[1]
	default:
		p.TCGProductID = m[1]
		p.AmbiguousID = true
	}
}

// readPage fills in a row whose image named no card from its product page:
// the set under Categories becomes the row's set, and the product code its id
// when it carries one. Sealed product and repacks wear no "mtg" code.
func (p *MLProduct) readPage(category, code string) {
	code, ok := strings.CutPrefix(code, "mtg")
	if !ok {
		return
	}
	p.SetName = category
	p.FromPage = true
	p.Foil = strings.HasSuffix(code, "-f")
	p.Shelf = codeShelf.FindString(code)
	m := codeTCG.FindStringSubmatch(code)
	if m != nil {
		// A 5-digit id behind such a digit reads as 6; the name check refuses it.
		p.TCGProductID = m[1][max(0, len(m[1])-maxTCGDigits):]
		p.AmbiguousID = true
		return
	}
	m = codeBare.FindStringSubmatch(code)
	if m != nil {
		p.CodeID = m[1]
		return
	}
	m = codeIdx.FindStringSubmatch(code)
	if m != nil {
		p.Number = strings.TrimLeft(m[1], "0")
	}
}

// idGuess is one id the row may name a card by, and whether match() must
// see the name agree before trusting it.
type idGuess struct {
	space   mtgmatcher.IDSpace
	id      string
	confirm bool
}

// idGuesses lists the ids the row names a card by, most trusted first. A
// bare product code is tried in both id spaces.
func (p MLProduct) idGuesses() []idGuess {
	switch {
	case p.TCGProductID != "":
		return []idGuess{{mtgmatcher.IDSpaceTCGplayer, p.TCGProductID, p.AmbiguousID}}
	case p.MultiverseID != "":
		return []idGuess{{mtgmatcher.IDSpaceMultiverse, p.MultiverseID, false}}
	case p.CodeID != "":
		return []idGuess{
			{mtgmatcher.IDSpaceTCGplayer, p.CodeID, true},
			{mtgmatcher.IDSpaceMultiverse, p.CodeID, true},
		}
	}
	return nil
}

// MLClient reads the Manaleak storefront.
type MLClient struct {
	client *http.Client
}

// NewMLClient returns a client for the storefront.
func NewMLClient() *MLClient {
	ml := MLClient{}
	ml.client = mtgban.NewHTTPClient()
	return &ml
}

func pageURL(base string, page int) string {
	u, _ := url.Parse(base)
	q := u.Query()
	q.Set("limit", strconv.Itoa(pageLimit))
	q.Set("page", strconv.Itoa(page))
	q.Set("sort", "pd.name")
	q.Set("order", "ASC")
	u.RawQuery = q.Encode()
	return u.String()
}

func (ml *MLClient) get(ctx context.Context, u string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", staticUA)
	req.Header.Set("Accept", staticAccept)

	resp, err := ml.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// GetPage fetches one listing page and answers its rows and the listing's
// own total, off which the caller sizes the fan-out.
func (ml *MLClient) GetPage(ctx context.Context, base string, page int) ([]MLProduct, int, error) {
	doc, err := ml.get(ctx, pageURL(base, page))
	if err != nil {
		return nil, 0, err
	}

	total := 0
	if m := showingTotal.FindStringSubmatch(doc.Text()); m != nil {
		total, _ = strconv.Atoi(m[1])
	}

	return parseListing(doc), total, nil
}

// GetProduct fetches the product page of a row whose image names no card, and
// fills in the set and code it carries.
func (ml *MLClient) GetProduct(ctx context.Context, product MLProduct) (MLProduct, error) {
	doc, err := ml.get(ctx, productURL+url.QueryEscape(product.ProductID))
	if err != nil {
		return product, err
	}
	category, code := parseProduct(doc)
	product.readPage(category, code)
	return product, nil
}

// parseProduct reads a product page's first category and its product code.
// The brand link beside the categories is the one wearing an itemprop.
func parseProduct(doc *goquery.Document) (category, code string) {
	desc := doc.Find(".product-options .description")
	category = strings.TrimSpace(desc.Find("a:not([itemprop])").First().Text())
	code = strings.TrimSpace(desc.Find(`[itemprop="model"]`).First().Text())
	return category, code
}

func parseListing(doc *goquery.Document) []MLProduct {
	var products []MLProduct
	doc.Find("div.product-list-item").Each(func(_ int, row *goquery.Selection) {
		var product MLProduct

		name := row.Find(".caption .name a").First()
		product.Name = strings.TrimSpace(name.Text())
		product.URL, _ = name.Attr("href")
		u, err := url.Parse(product.URL)
		if err == nil {
			product.ProductID = u.Query().Get("product_id")
		}

		// The set rides in the row's description as a link on the retail
		// side; the buylist fills the description with rules text instead,
		// and its rows lean on the image id alone.
		product.SetName = strings.TrimSpace(row.Find(".caption .description a").First().Text())

		price := strings.TrimSpace(row.Find(".caption .price").First().Text())
		price = strings.TrimPrefix(price, "£")
		product.Price, _ = strconv.ParseFloat(strings.ReplaceAll(price, ",", ""), 64)

		product.OutOfStock = row.HasClass("outofstock")

		src, _ := row.Find(".image img").First().Attr("data-src")
		product.readImage(src)

		products = append(products, product)
	})
	return products
}
