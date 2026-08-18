// Package miniaturemarket scrapes Miniature Market, which stocks sealed
// product only.
package miniaturemarket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type Miniaturemarket struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
	productMap    map[string]string
	game          string
}

const (
	GameMagic     = "magic"
	GameLorcana   = "lorcana"
	GameRiftbound = "riftbound"
	GameOnePiece  = "onepiece"
)

// gameWidgets are the CMS navigation ids behind each game's storefront
// category, read off the category pages; the widget serves the paginated
// product listing the scraper walks.
var gameWidgets = map[string]string{
	GameMagic:     "be53d253d6bc3258a8160556dda3e9b2",
	GameLorcana:   "4e0223a87610176ef0d24ef6d2dcde3a",
	GameRiftbound: "019be122ca9779e5af00a663d064f775",
	GameOnePiece:  "f7ac67a9aa8d255282de7d11391e1b69",
}

func NewScraperSealed(game string) *Miniaturemarket {
	mm := Miniaturemarket{}
	mm.inventory = mtgban.InventoryRecord{}
	mm.MaxConcurrency = defaultConcurrency
	mm.productMap = map[string]string{}
	mm.game = game
	return &mm
}

const defaultConcurrency = 6

func (mm *Miniaturemarket) mainURL() string {
	return "https://www.miniaturemarket.com/widgets/cms/navigation/" + gameWidgets[mm.game] + "?filter-inStock=1&no-aggregations=1&order=name-asc&p=1"
}

type respChan struct {
	cardID   string
	invEntry *mtgban.InventoryEntry
}

func (mm *Miniaturemarket) printf(format string, a ...interface{}) {
	if mm.LogCallback != nil {
		mm.LogCallback("[MMSealed] "+format, a...)
	}
}

func (mm *Miniaturemarket) processPage(ctx context.Context, channel chan<- respChan, page int) error {
	u, err := url.Parse(mm.mainURL())
	if err != nil {
		return err
	}
	v := u.Query()
	v.Set("p", fmt.Sprint(page))
	u.RawQuery = v.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return err
	}
	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		mm.printf("newDoc - %s", err.Error())
		return err
	}

	doc.Find(`div[class="product-info"]`).Each(func(i int, s *goquery.Selection) {
		id, _ := s.Find(`input[name="product-id"]`).Attr("value")
		uuid, found := mm.productMap[id]
		if !found {
			if mm.game == GameMagic {
				return
			}
			// The other games' datastores carry no miniaturemarket ids:
			// resolve the product by its listed name, English only,
			// unique or nothing. A failing name retries without its
			// trailing decoration ("(New Arrival)"), which resolution
			// rightly refuses to see past on its own.
			name := strings.TrimSpace(s.Find(`a.product-name`).Text())
			if name == "" || mtgmatcher.SealedIsLanguageVariant(name) {
				return
			}
			resolved, err := mtgmatcher.ResolveSealed(name)
			if err != nil {
				if idx := strings.LastIndexByte(name, '('); idx > 0 {
					resolved, err = mtgmatcher.ResolveSealed(name[:idx])
				}
				if err != nil {
					return
				}
			}
			uuid = resolved
		}

		link, _ := s.Find(`a.product-name`).Attr("href")
		if mm.Affiliate != "" {
			link += "?utm_source=" + mm.Affiliate + "&utm_medium=feed&utm_campaign=mtg_singles"
		}

		priceStr := s.Find(`.product-price`).Text()
		price, err := mtgmatcher.ParsePrice(priceStr)
		if err != nil {
			mm.printf("uuid %s - %s", uuid, err.Error())
			return
		}

		channel <- respChan{
			cardID: uuid,
			invEntry: &mtgban.InventoryEntry{
				Price: price,
				URL:   link,
			},
		}
	})

	return nil
}

func (mm *Miniaturemarket) NumberOfProducts(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mm.mainURL(), http.NoBody)
	if err != nil {
		return 0, err
	}
	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		mm.printf("newDoc - %s", err.Error())
		return 0, err
	}

	// A catalog that fits one page renders no pagination at all
	href, _ := doc.Find("a.page-link").Last().Attr("href")
	if href == "" {
		return 1, nil
	}
	u, err := url.Parse(href)
	if err != nil {
		return 0, err
	}
	num := u.Query().Get("p")
	if num == "" {
		return 1, nil
	}
	return strconv.Atoi(num)
}

func (mm *Miniaturemarket) Load(ctx context.Context) error {
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil || co.Identifiers["miniaturemarketId"] == "" {
			continue
		}
		mm.productMap[co.Identifiers["miniaturemarketId"]] = uuid
	}
	mm.printf("Loaded %d sealed products", len(mm.productMap))
	if mm.game != GameMagic {
		mm.printf("Resolving %s products by name", mm.game)
	}

	totalProducts, err := mm.NumberOfProducts(ctx)
	if err != nil {
		return err
	}
	mm.printf("Parsing %d items", totalProducts)

	pageNums := make([]int, totalProducts)
	for i := range pageNums {
		pageNums[i] = i
	}

	mtgban.WorkerPool(ctx, mm.MaxConcurrency, pageNums,
		func(ctx context.Context, page int, results chan<- respChan) error {
			return mm.processPage(ctx, results, page)
		},
		func(record respChan) {
			err := mm.inventory.AddRelaxed(record.cardID, record.invEntry)
			if err != nil {
				mm.printf("%v", err)
			}
		},
		mm.printf,
	)

	mm.inventoryDate = time.Now()

	return nil
}

func (mm *Miniaturemarket) Inventory() mtgban.InventoryRecord {
	return mm.inventory
}

func (mm *Miniaturemarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Miniature Market"
	info.Shorthand = "MMSealed"
	info.InventoryTimestamp = &mm.inventoryDate
	info.SealedMode = true
	info.NoQuantityInventory = true
	switch mm.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	case GameRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameOnePiece:
		info.Game = mtgban.GameOnePiece
	}
	return
}
