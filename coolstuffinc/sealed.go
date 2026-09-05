package coolstuffinc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Sealed prices Cool Stuff Inc's sealed product.
type Sealed struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	productMap map[string]string

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	DisableRetail  bool
	DisableBuylist bool

	client *http.Client
	game   string
}

// NewScraperSealed returns a sealed scraper for one game.
func NewScraperSealed(game string) *Sealed {
	csi := Sealed{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	csi.client = client.StandardClient()
	csi.MaxConcurrency = defaultConcurrency

	csi.productMap = map[string]string{}
	if game == GameMagic {
		for _, uuid := range mtgmatcher.GetSealedUUIDs() {
			co, err := mtgmatcher.GetUUID(uuid)
			if err != nil {
				continue
			}
			id, found := co.Identifiers["csiId"]
			if !found {
				continue
			}
			csi.productMap[id] = co.UUID
		}
	}
	csi.game = game
	return &csi
}

func (csi *Sealed) printf(format string, a ...any) {
	if csi.LogCallback != nil {
		csi.LogCallback("[CSISealed] "+format, a...)
	}
}

const sealedURL = "https://www.coolstuffinc.com/sq/2293832?page=1&sb=price|desc"

func (csi *Sealed) numOfPages(ctx context.Context) (int, error) {
	link := sealedURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return 0, err
	}
	resp, err := csi.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	text := doc.Find(".search-result-links").Text()
	text = strings.TrimPrefix(strings.Split(text, " Results")[0], "1 - ")

	fields := strings.Split(text, " of ")
	if len(fields) != 2 {
		return 0, errors.New("unknown page format")
	}

	resultsPerPage, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, err
	}

	resultsTotal, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, err
	}

	return resultsTotal/resultsPerPage + 1, nil
}

func (csi *Sealed) processSealedPage(ctx context.Context, channel chan<- responseChan, page int) error {
	csi.printf("Processing page %d", page)

	u, err := url.Parse(sealedURL)
	if err != nil {
		return err
	}

	v := u.Query()
	v.Set("page", fmt.Sprint(page))
	u.RawQuery = v.Encode()

	link := u.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := csi.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	doc.Find(".main-container").Each(func(i int, s *goquery.Selection) {
		productName := s.Find(`span[itemprop="name"]`).Text()
		path, _ := s.Find(`a[class="productLink"]`).Attr("href")

		csiID := strings.TrimPrefix(path, "/p/")

		uuid, found := csi.productMap[csiID]
		if !found {
			return
		}

		qtyStr := s.Find(`span[class="card-qty"]`).Text()
		qtyStr = strings.TrimSuffix(qtyStr, "+")
		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			qty = 20
		}

		priceStr := s.Find(`b[itemprop="price"]`).First().Text()
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			csi.printf("%s: %s", productName, err.Error())
			return
		}

		link := "https://coolstuffinc.com" + path
		if csi.Partner != "" {
			link += "?utm_referrer=" + csi.Partner
		}

		out := responseChan{
			cardID: uuid,
			invEntry: &mtgban.InventoryEntry{
				Price:    price,
				Quantity: qty,
				URL:      link,
			},
		}

		channel <- out
	})

	return nil
}

func (csi *Sealed) scrape(ctx context.Context) error {
	totalPages, err := csi.numOfPages(ctx)
	if err != nil {
		return err
	}
	csi.printf("Found %d pages", totalPages)

	pageNums := make([]int, totalPages)
	for i := range pageNums {
		pageNums[i] = i + 1
	}

	mtgban.WorkerPool(ctx, csi.MaxConcurrency, pageNums,
		func(ctx context.Context, page int, results chan<- responseChan) error {
			return csi.processSealedPage(ctx, results, page)
		},
		func(record responseChan) {
			err := csi.inventory.Add(record.cardID, record.invEntry)
			if err != nil {
				csi.printf("%s", err.Error())
			}
		},
		csi.printf,
	)

	csi.inventoryDate = time.Now()

	return nil
}

func (csi *Sealed) parseBL(ctx context.Context) error {
	products, err := GetBuylist(ctx, csi.game)
	if err != nil {
		return err
	}
	csi.printf("Found %d products", len(products))

	for _, product := range products {
		if product.RarityName != "Box" {
			continue
		}

		// Build link early to help debug
		u, _ := url.Parse(csiBuylistLink)
		v := url.Values{}
		v.Set("s", csi.game)
		v.Set("a", "1")
		v.Set("name", product.Name)
		u.RawQuery = v.Encode()
		link := u.String()

		// Magic products carry the catalog id the datastore knows; the
		// other games resolve the offer by name, unique or nothing,
		// skipping the language variants the datastores never carry.
		uuid, found := csi.productMap[product.PID]
		if !found {
			if csi.game == GameMagic {
				continue
			}
			if mtgmatcher.SealedIsLanguageVariant(product.Name) {
				continue
			}
			resolved, err := mtgmatcher.ResolveSealed(product.Name)
			if err != nil {
				continue
			}
			uuid = resolved
		}

		buyPrice, err := mtgmatcher.ParsePrice(product.Price)
		if err != nil {
			csi.printf("%s error: %s", product.Name, err.Error())
			continue
		}

		var priceRatio, sellPrice float64

		invCards := csi.inventory[uuid]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = buyPrice / sellPrice * 100
		}

		buyEntry := mtgban.BuylistEntry{
			BuyPrice:   buyPrice,
			PriceRatio: priceRatio,
			URL:        link,
		}

		err = csi.buylist.Add(uuid, &buyEntry)
		if err != nil {
			csi.printf("%s", err.Error())
			continue
		}
	}

	csi.buylistDate = time.Now()

	return nil
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (csi *Sealed) SetConfig(opt mtgban.ScraperOptions) {
	csi.DisableRetail = opt.DisableRetail
	csi.DisableBuylist = opt.DisableBuylist
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (csi *Sealed) Load(ctx context.Context) error {
	// The saved-query page and the buylist exist for Magic alone; the
	// other games ride the same set-facet search the singles use, with
	// the sealed-name resolver telling the sealed rows apart from the
	// card ones.
	if csi.game != GameMagic {
		var errs []error
		if !csi.DisableRetail {
			if err := csi.scrapeBysets(ctx); err != nil {
				errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
			}
		}
		if !csi.DisableBuylist {
			if err := csi.parseBL(ctx); err != nil {
				errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
			}
		}
		return errors.Join(errs...)
	}

	var errs []error

	if !csi.DisableRetail {
		err := csi.scrape(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		}
	}

	if !csi.DisableBuylist {
		err := csi.parseBL(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

// scrapeBysets walks every set of the game through the same search the
// singles scraper uses; the sealed rows resolve through the sealed-name
// resolver, the card rows resolve to nothing and drop out.
func (csi *Sealed) scrapeBysets(ctx context.Context) error {
	// One search per sealed-product word: the store's own names carry
	// them ("Booster Box", "Illumineer's Trove"), and a handful of
	// searches covers the whole sealed catalog where the set facets do
	// not - lorcana sealed carries no ItemSet at all
	queries := []string{
		"booster", "deck", "trove", "gift", "bundle",
		"case", "box", "kit", "vault", "collection",
	}

	// The queries overlap ("Booster Box" answers booster and box both);
	// the first sighting of a product wins
	seen := map[string]bool{}
	mtgban.WorkerPool(ctx, csi.MaxConcurrency, queries,
		func(ctx context.Context, query string, channel chan<- responseChan) error {
			err := csi.processSealedSearch(ctx, channel, query)
			if err != nil {
				csi.printf("%s: %s", query, err.Error())
			}
			return nil
		},
		func(record responseChan) {
			key := record.cardID + "|" + record.invEntry.OriginalID
			if seen[key] {
				return
			}
			seen[key] = true
			err := csi.inventory.Add(record.cardID, record.invEntry)
			if err != nil {
				csi.printf("%v", err)
			}
		},
		csi.printf,
	)

	csi.inventoryDate = time.Now()

	return nil
}

// searchSealed runs a product-name search without the singles' rarity
// facets, which sealed products have none of and would be filtered out
// by. The name route rather than the set facet: lorcana sealed carries
// no ItemSet at all, where riftbound's does.
func searchSealed(ctx context.Context, game, query string) (*SearchResult, error) {
	v := url.Values{}
	v.Set("name", query)
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
	// No instock option either: combined with a name search it returns
	// nothing at all, and the offer rows carry their own stock state
	// No rarity facets: sealed products carry no rarity, and any rarity
	// constraint excludes them outright
	v.Set("f[Rarity][]", "")
	v.Set("f[ItemSet][]", "")
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

// processSealedSearch pages through one query's results, pricing every
// row the sealed-name resolver recognizes. English only: language-variant
// names are skipped before resolution.
func (csi *Sealed) processSealedSearch(ctx context.Context, channel chan<- responseChan, query string) error {
	result, err := searchSealed(ctx, csi.game, query)
	if err != nil {
		return err
	}

	for page := 1; ; page++ {
		data := result.Data

		if page > 1 {
			link := "https://www.coolstuffinc.com/sq/" + result.PageID + "?page=" + fmt.Sprint(page)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
			if err != nil {
				return err
			}
			resp, err := csi.client.Do(req)
			if err != nil {
				return err
			}
			data, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}

		rows := doc.Find(`div[class="row product-search-row main-container"]`)
		rows.Each(func(i int, s *goquery.Selection) {
			productName := strings.TrimSpace(s.Find(`span[itemprop="name"]`).Text())
			if csi.game == GameYuGiOh {
				// The storefront leads its yugioh sealed listings with the
				// game's own name, which the canonical names never carry.
				productName = strings.TrimPrefix(productName, "Yu-Gi-Oh!")
				productName = strings.TrimPrefix(strings.TrimSpace(productName), "- ")
			}
			if productName == "" || mtgmatcher.SealedIsLanguageVariant(productName) {
				return
			}
			uuid, err := mtgmatcher.ResolveSealed(productName)
			if err != nil {
				// A card row, or a product the datastore does not carry
				return
			}

			pid, _ := s.Find(`span[class="rating-display "]`).Attr("data-pid")
			link := "https://www.coolstuffinc.com/p/" + pid
			if csi.Partner != "" {
				link += "?utm_referrer=" + csi.Partner
			}

			// The stock state is schema markup on the row; the row's
			// visible text mentions being out of stock even on in-stock
			// rows, through template markup, and must not be trusted
			availability, _ := s.Find(`[itemprop="availability"]`).Attr("content")
			if !strings.Contains(availability, "InStock") {
				return
			}

			// Sealed rows carry no quantity span; an in-stock row
			// without one sells at least a single copy
			qty := 1
			qtyStr := s.Find(`span[class="card-qty"]`).Text()
			qtyStr = strings.TrimSpace(strings.TrimSuffix(qtyStr, "+"))
			if qtyStr != "" {
				var err error
				qty, err = strconv.Atoi(qtyStr)
				if err != nil || qty == 0 {
					return
				}
			}

			// The price rides in the content attribute, the visible text
			// being nothing but a formatting placeholder
			priceStr, _ := s.Find(`[itemprop="price"]`).Attr("content")
			if priceStr == "" {
				priceStr = strings.TrimSpace(s.Find(`b[itemprop="price"]`).Text())
			}
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil || price == 0 {
				return
			}

			channel <- responseChan{
				cardID: uuid,
				invEntry: &mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      price,
					Quantity:   qty,
					URL:        link,
					OriginalID: pid,
				},
			}
		})

		// A short page is the last one; asking past it returns the same
		// page over again rather than an empty one
		if rows.Length() < 25 || result.PageID == "" {
			break
		}
	}

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (csi *Sealed) Inventory() mtgban.InventoryRecord {
	return csi.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (csi *Sealed) Buylist() mtgban.BuylistRecord {
	return csi.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (csi *Sealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cool Stuff Inc"
	info.Shorthand = "CSISealed"
	switch csi.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	case GameRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameOnePiece:
		info.Game = mtgban.GameOnePiece
	case GamePokemon:
		info.Game = mtgban.GamePokemon
	case GameYuGiOh:
		info.Game = mtgban.GameYuGiOh
	case GameGundam:
		info.Game = mtgban.GameGundam
	case GamePalworld:
		info.Game = mtgban.GamePalworld
	}
	info.InventoryTimestamp = &csi.inventoryDate
	info.BuylistTimestamp = &csi.buylistDate
	info.SealedMode = true
	info.CreditMultiplier = 1.25
	return
}
