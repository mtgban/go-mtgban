// Package miniaturemarket scrapes Miniature Market, which stocks sealed
// product only.
package miniaturemarket

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Miniaturemarket prices Miniature Market's sealed product; they carry no
// singles.
type Miniaturemarket struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
	productMap    map[string]string
	game          string
}

// The games this scraper covers, as their storefront widget names them.
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

// NewScraperSealed returns a sealed scraper for one game.
func NewScraperSealed(game string) *Miniaturemarket {
	mm := Miniaturemarket{}
	mm.inventory = mtgban.InventoryRecord{}
	mm.MaxConcurrency = defaultConcurrency
	mm.productMap = map[string]string{}
	mm.game = game
	return &mm
}

const defaultConcurrency = 6

// The decorations miniaturemarket writes into a One Piece product name: a
// storefront prefix, availability and pack-count parentheticals, and the
// set code in brackets where the canonical names spell no code at all.
var (
	onePieceDecorations = regexp.MustCompile(`\s*\((?:Preorder|\d+ Packs?)\)`)
	onePieceSetCode     = regexp.MustCompile(`\s*\[([A-Z]+)-?(\d+)\]`)
)

// sealedName rewrites a storefront listing toward the shape its game's
// canonical names use, where the two differ by more than the trailing
// decoration the resolve retry already sees past.
//
// One Piece names arrive as "One Piece TCG: BLUE Kuzan [ST-33] - Starter
// Deck (Preorder)" while the canon says "Starter Deck 33: BLUE Kuzan": the
// prefix and parentheticals only decorate, and the bracket code either
// carries the deck number the canon leads with, or restates a set the rest
// of the name already spells.
func sealedName(game, name string) string {
	if game != GameOnePiece {
		return name
	}

	name = strings.TrimPrefix(name, "One Piece TCG: ")
	name = onePieceDecorations.ReplaceAllString(name, "")

	match := onePieceSetCode.FindStringSubmatch(name)
	if match != nil && match[1] == "ST" {
		name = onePieceSetCode.ReplaceAllString(name, "")
		name = strings.Replace(name, " - Starter Deck", "", 1)
		name = "Starter Deck " + match[2] + ": " + strings.TrimSpace(name)
	} else {
		name = onePieceSetCode.ReplaceAllString(name, "")
	}

	return strings.TrimSpace(name)
}

// resolveListing names the sealed product a storefront listing prices, or
// says why none was found. Every listing this scraper leaves behind leaves it
// here, which is why the reason is returned rather than swallowed: a run that
// resolved nothing at all used to look exactly like a run with nothing to
// resolve.
//
// Magic routes through the id the datastore records; the other games' data
// carries no miniaturemarket ids, so the product is resolved by its listed
// name, English only, unique or nothing. A failing name retries without its
// trailing decoration ("(New Arrival)"), which resolution rightly refuses to
// see past on its own.
func (mm *Miniaturemarket) resolveListing(id, listed string) (string, string) {
	if uuid, found := mm.productMap[id]; found {
		return uuid, ""
	}
	if mm.game == GameMagic {
		return "", "no datastore id"
	}
	name := strings.TrimSpace(sealedName(mm.game, listed))
	if name == "" {
		return "", "unnamed listing"
	}
	if mtgmatcher.SealedIsLanguageVariant(name) {
		return "", "language variant"
	}
	uuid, err := mtgmatcher.ResolveSealed(name)
	if err != nil {
		if idx := strings.LastIndexByte(name, '('); idx > 0 {
			uuid, err = mtgmatcher.ResolveSealed(name[:idx])
		}
		if err != nil {
			return "", err.Error()
		}
	}
	return uuid, ""
}

func (mm *Miniaturemarket) mainURL() string {
	return "https://www.miniaturemarket.com/widgets/cms/navigation/" + gameWidgets[mm.game] + "?filter-inStock=1&no-aggregations=1&order=name-asc&p=1"
}

type respChan struct {
	cardID   string
	invEntry *mtgban.InventoryEntry

	// drop names why a listing was not priced and repeats the name it was
	// listed under. A record carrying one prices nothing; it is there so
	// the run can say what it left behind instead of dropping it in
	// silence, which is what a run pricing none of a game looked like.
	drop string
	name string
}

func (mm *Miniaturemarket) printf(format string, a ...any) {
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
		listed := strings.TrimSpace(s.Find(`a.product-name`).Text())
		id, _ := s.Find(`input[name="product-id"]`).Attr("value")
		uuid, drop := mm.resolveListing(id, listed)
		if drop != "" {
			channel <- respChan{drop: drop, name: listed}
			return
		}

		link, _ := s.Find(`a.product-name`).Attr("href")
		if mm.Affiliate != "" {
			link += "?utm_source=" + mm.Affiliate + "&utm_medium=feed&utm_campaign=mtg_singles"
		}

		priceStr := s.Find(`.product-price`).Text()
		price, err := mtgmatcher.ParsePrice(priceStr)
		if err != nil {
			channel <- respChan{drop: "unparseable price", name: listed}
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

// NumberOfProducts returns how many products the widget holds, which is what
// the page walk is sized against.
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

// Load fetches everything this scraper offers. See mtgban.Scraper.
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

	// The consumer runs on one goroutine, so the tally needs no locking.
	var listed, priced int
	dropped := map[string]int{}
	mtgban.WorkerPool(ctx, mm.MaxConcurrency, pageNums,
		func(ctx context.Context, page int, results chan<- respChan) error {
			return mm.processPage(ctx, results, page)
		},
		func(record respChan) {
			listed++
			if record.drop != "" {
				dropped[record.drop]++
				// The names a resolver turned down are the whole reason a
				// run prices a fraction of a catalog, so say which name and
				// which refusal, the way the other sealed scrapers do. The
				// products carrying no id of ours are the ordinary case for
				// Magic and are counted rather than listed.
				if record.drop != "no datastore id" {
					mm.printf("%q: %s", record.name, record.drop)
				}
				return
			}
			priced++
			err := mm.inventory.AddRelaxed(record.cardID, record.invEntry)
			if err != nil {
				mm.printf("%v", err)
			}
		},
		mm.printf,
	)
	mm.printf("Priced %d of %d listings", priced, listed)
	for _, reason := range slices.Sorted(maps.Keys(dropped)) {
		mm.printf("Dropped %d listings: %s", dropped[reason], reason)
	}

	mm.inventoryDate = time.Now()

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mm *Miniaturemarket) Inventory() mtgban.InventoryRecord {
	return mm.inventory
}

// Info describes this scraper. See mtgban.Scraper.
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
