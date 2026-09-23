// Package strikezone scrapes Strike Zone.
package strikezone

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The games this scraper covers, as the storefront names them.
const (
	GameMagic         = "Magic_the_Gathering"
	GameLorcana       = "Lorcana"
	GamePokemon       = "Pokemon_Tcg"
	GameYuGiOh        = "Yu_Gi_Oh"
	GameFleshAndBlood = "Flesh_and_Blood"
)

// szGames is what NewScraper is built through: it names the storefront
// category a game is filed under, and a game named nowhere here is not one
// Strike Zone is read for.
var szGames = map[mtgban.Game]string{
	mtgban.GameMagic:         GameMagic,
	mtgban.GameLorcana:       GameLorcana,
	mtgban.GamePokemon:       GamePokemon,
	mtgban.GameYuGiOh:        GameYuGiOh,
	mtgban.GameFleshAndBlood: GameFleshAndBlood,
}

const (
	defaultConcurrency = 8

	szInventoryURL = "http://shop.strikezoneonline.com/Category/%s_Singles.html"
	szBuylistURL   = "http://shop.strikezoneonline.com/BuyList/%s.html"

	modeRetail  = "retail"
	modeBuylist = "buylist"
)

// The pages the crawl must not enter: the re-sorted copies of every category,
// which would double its rows, and each game's sealed hubs, whose products
// are not singles. Gift_Sets and Collector_Tins hang off both the Pokemon
// and the Yu-Gi-Oh tree under one name.
var skipSuffixes = []string{
	"_ByTable.html",
	"_ByRarity.html",
	"_ByNumber.html",
	"Games.html",
	"Magic_Booster_Boxes.html",
	"Fat_Packs.html",
	"Gift_Sets_and_Secret_Lairs.html",
	"Preconstructed_Decks.html",
	"Pokemon_Booster_Boxes.html",
	"Pokemon_Booster_Packs.html",
	"Yugioh_Booster_Boxes.html",
	"Yu_Gi_Oh_Booster_Packs.html",
	"Collector_Tins.html",
	"Gift_Sets.html",
}

// Strikezone prices Strike Zone's singles, both what they sell and what they
// buy.
type Strikezone struct {
	logCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	maxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	disableRetail  bool
	disableBuylist bool

	backend *mtgmatcher.Backend
	game    mtgban.Game
	shelf   string
	client  *http.Client
}

// NewScraper returns a scraper for the datastore's game.
func NewScraper(b *mtgmatcher.Backend) (*Strikezone, error) {
	game, err := mtgban.GameOf(b)
	if err != nil {
		return nil, err
	}
	shelf, ok := szGames[game]
	if !ok {
		return nil, fmt.Errorf("unsupported game %q", game)
	}
	sz := Strikezone{}
	sz.inventory = mtgban.InventoryRecord{}
	sz.buylist = mtgban.BuylistRecord{}
	sz.maxConcurrency = defaultConcurrency
	sz.backend = b
	sz.game = game
	sz.shelf = shelf
	client := retryablehttp.NewClient()
	client.Logger = nil
	sz.client = client.StandardClient()
	return &sz, nil
}

func (sz *Strikezone) printf(format string, a ...any) {
	if sz.logCallback != nil {
		sz.logCallback("[SZ] "+format, a...)
	}
}

type respChan struct {
	cardID string
	inv    *mtgban.InventoryEntry
	bl     *mtgban.BuylistEntry
}

func (sz *Strikezone) getDoc(ctx context.Context, link string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := sz.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", link, resp.Status)
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

func absoluteURL(base, href string) string {
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if ref.IsAbs() {
		return ref.String()
	}
	b, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return b.ResolveReference(ref).String()
}

func (sz *Strikezone) keepLink(link, mode string) bool {
	basePath := "/Category/"
	if mode == modeBuylist {
		basePath = "/BuyList/"
	}
	if !strings.Contains(link, basePath) {
		return false
	}
	for _, suffix := range skipSuffixes {
		if strings.HasSuffix(link, suffix) {
			return false
		}
	}
	return true
}

func (sz *Strikezone) childLinks(doc *goquery.Document, pageURL, mode string) []string {
	var links []string
	seen := map[string]struct{}{}
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		abs := absoluteURL(pageURL, href)
		if abs == "" || !sz.keepLink(abs, mode) {
			return
		}
		if u, err := url.Parse(abs); err != nil || u.Host != "shop.strikezoneonline.com" {
			return
		}
		if _, dup := seen[abs]; dup {
			return
		}
		seen[abs] = struct{}{}
		links = append(links, abs)
	})
	return links
}

func (sz *Strikezone) processRow(mode string, channel chan<- respChan, el *goquery.Selection, edition string) error {
	var cardName, pathURL, notes, cond, qty, price string

	cardName = strings.TrimSpace(el.Find("td:nth-child(1)").Text())
	if cardName == "" || cardName == "Name" {
		// No error as empty page may not have anything to process
		return nil
	}

	pathURL, _ = el.Find("a").Attr("href")

	// The columns and card construction differ per game; the match and error
	// handling below are shared.
	var theCard *mtgmatcher.InputCard
	switch sz.game {
	case mtgban.GameMagic:
		if mode == modeRetail {
			notes = strings.TrimSpace(el.Find("td:nth-child(4)").Text())
			cond = strings.TrimSpace(el.Find("td:nth-child(5)").Text())
			qty = strings.TrimSpace(el.Find("td:nth-child(6)").Text())
			price = strings.TrimSpace(el.Find("td:nth-child(7)").Text())
		} else if mode == modeBuylist {
			notes = strings.TrimSpace(el.Find("td:nth-child(4)").Text())
			cond = notes
			qty = strings.TrimSpace(el.Find("td:nth-child(5)").Text())
			price = strings.TrimSpace(el.Find("td:nth-child(6)").Text())
		}

		c, err := preprocess(sz.backend, cardName, edition, notes)
		if err != nil {
			return nil
		}
		theCard = c
	case mtgban.GameLorcana:
		notes = strings.TrimSpace(el.Find("td:nth-child(2)").Text())
		cond = strings.TrimSpace(el.Find("td:nth-child(4)").Text())
		qty = strings.TrimSpace(el.Find("td:nth-child(5)").Text())
		price = strings.TrimSpace(el.Find("td:nth-child(6)").Text())

		foil := strings.Contains(strings.ToLower(cond), "foil")
		theCard = &mtgmatcher.InputCard{Name: cardName, Edition: edition, Variation: notes, Foil: foil}
	case mtgban.GamePokemon, mtgban.GameYuGiOh, mtgban.GameFleshAndBlood:
		number := strings.TrimSpace(el.Find("td:nth-child(2)").Text())
		cond = strings.TrimSpace(el.Find("td:nth-child(4)").Text())
		qty = strings.TrimSpace(el.Find("td:nth-child(5)").Text())
		price = strings.TrimSpace(el.Find("td:nth-child(6)").Text())

		c, err := preprocessDetails(sz.game, cardName, edition, number, cond)
		if err != nil {
			return nil
		}
		theCard = c
	default:
		return nil
	}

	cardID, err := sz.backend.Match(theCard)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return nil
	} else if err != nil {
		// Skip errors from these sets, there is not enough information
		switch edition {
		case "Secret Lair", "The List", "Mystery Booster":
			return nil
		}
		sz.printf("%q", theCard)
		sz.printf("%s|%s|%s", cardName, edition, notes)

		var alias *mtgmatcher.AliasingError
		if errors.As(err, &alias) {
			for _, probe := range alias.Probe() {
				card, _ := sz.backend.GetUUID(probe)
				sz.printf("- %s", card)
			}
		}
		return err
	}

	if sz.game == mtgban.GameMagic {
		co, coErr := sz.backend.GetUUID(cardID)
		if coErr == nil && (namesAbsentTreatment(theCard.Variation, co) ||
			wearsUnnamedTextured(sz.backend, theCard.Variation, co)) {
			return nil
		}
	}

	cardPrice, err := mtgmatcher.ParsePrice(price)
	if err != nil || cardPrice <= 0 {
		return err
	}

	quantity, err := strconv.Atoi(qty)
	if err != nil || quantity <= 0 {
		return err
	}

	switch {
	case strings.Contains(cond, "Mint"):
		cond = "NM"
	case strings.Contains(cond, "Light"):
		cond = "SP"
	case strings.Contains(cond, "Medium"):
		cond = "MP"
	case strings.Contains(cond, "Heavy"):
		cond = "HP"
	default:
		return fmt.Errorf("unsupported %s condition", cond)
	}

	if mode == modeRetail {
		channel <- respChan{
			cardID: cardID,
			inv: &mtgban.InventoryEntry{
				Conditions: cond,
				Price:      cardPrice,
				Quantity:   quantity,
				URL:        "http://shop.strikezoneonline.com" + pathURL,
			},
		}
	} else if mode == modeBuylist {
		var sellPrice, priceRatio float64

		invCards := sz.inventory[cardID]
		for _, invCard := range invCards {
			if invCard.Conditions == "NM" {
				sellPrice = invCard.Price
				break
			}
		}

		if sellPrice > 0 {
			priceRatio = cardPrice / sellPrice * 100
		}

		// Some buy pages return wrong results if they have a comma
		cardName = url.QueryEscape(strings.Replace(cardName, ",", "", -1))
		link := "http://shop.strikezoneonline.com/TUser?MC=CUSTS&MF=B&BUID=637&ST=D&M=B&CMD=Search&T=" + cardName

		channel <- respChan{
			cardID: cardID,
			bl: &mtgban.BuylistEntry{
				Conditions: cond,
				BuyPrice:   cardPrice,
				Quantity:   quantity,
				PriceRatio: priceRatio,
				URL:        link,
			},
		}
	}
	return nil
}

func (sz *Strikezone) parseRows(doc *goquery.Document, pageURL, mode string, channel chan<- respChan) {
	edition := strings.TrimSpace(doc.Find("h1").First().Text())
	edition = strings.TrimSuffix(edition, " Buy Lists")
	edition = strings.TrimPrefix(edition, "Singles ")

	sz.printf("Parsing %s", edition)

	// Only the Magic categories render the denser rtti table; every
	// other game lists retail and buylist alike in the generic one.
	tableRowName := "table.rtti tr"
	if mode == modeBuylist || sz.game != mtgban.GameMagic {
		tableRowName = "table.ItemTable tr"
	}

	doc.Find(tableRowName).Each(func(_ int, el *goquery.Selection) {
		err := sz.processRow(mode, channel, el, edition)
		if err != nil {
			cardName := strings.TrimSpace(el.Find("td:nth-child(1)").Text())
			sz.printf("cannot process %s %s (%s): %s", mode, cardName, edition, err.Error())
			sz.printf("-> %s", pageURL)
		}
	})
}

// processPage fetches one category page, emits its rows, then walks any
// deeper category links the same way mtgseattle recurses into subcategories.
func (sz *Strikezone) processPage(ctx context.Context, channel chan<- respChan, pageURL, mode string, visited *sync.Map) error {
	if _, loaded := visited.LoadOrStore(pageURL, true); loaded {
		return nil
	}

	doc, err := sz.getDoc(ctx, pageURL)
	if err != nil {
		return err
	}

	sz.parseRows(doc, pageURL, mode, channel)

	for _, child := range sz.childLinks(doc, pageURL, mode) {
		if err := sz.processPage(ctx, channel, child, mode, visited); err != nil {
			sz.printf("%v", err)
		}
	}
	return nil
}

func (sz *Strikezone) scrape(ctx context.Context, mode string) error {
	var link string
	if mode == modeRetail {
		link = fmt.Sprintf(szInventoryURL, sz.shelf)
		// The storefront files the Flesh and Blood singles under a bare
		// name no other game shares, instead of its own prefixed one.
		if sz.game == mtgban.GameFleshAndBlood {
			link = "http://shop.strikezoneonline.com/Category/Singles.html"
		}
	} else if mode == modeBuylist {
		link = fmt.Sprintf(szBuylistURL, sz.shelf)
	}
	sz.printf("Visiting %s", link)

	doc, err := sz.getDoc(ctx, link)
	if err != nil {
		return err
	}

	links := sz.childLinks(doc, link, mode)
	sz.printf("Found %d categories", len(links))

	consume := func(resp respChan) {
		if resp.inv != nil {
			err := sz.inventory.Add(resp.cardID, resp.inv)
			if err != nil {
				sz.printf("%v", err)
			}
		}
		if resp.bl != nil {
			err := sz.buylist.Add(resp.cardID, resp.bl)
			if err != nil {
				sz.printf("%v", err)
			}
		}
	}

	// Hub rows first (the entry URL was scraped for rows as well as links).
	hubCh := make(chan respChan)
	var hubWG sync.WaitGroup
	hubWG.Add(1)
	go func() {
		defer hubWG.Done()
		for resp := range hubCh {
			consume(resp)
		}
	}()
	sz.parseRows(doc, link, mode, hubCh)
	close(hubCh)
	hubWG.Wait()

	visited := &sync.Map{}
	// The hub is already loaded; keep workers from fetching it again when a
	// child page links back.
	visited.Store(link, true)

	mtgban.WorkerPool(ctx, sz.maxConcurrency, links,
		func(ctx context.Context, page string, results chan<- respChan) error {
			return sz.processPage(ctx, results, page, mode, visited)
		},
		consume,
		sz.printf,
	)

	if mode == modeRetail {
		sz.inventoryDate = time.Now()
	} else if mode == modeBuylist {
		sz.buylistDate = time.Now()
	}

	return nil
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (sz *Strikezone) SetConfig(opt mtgban.ScraperOptions) {
	sz.disableRetail = opt.DisableRetail
	sz.disableBuylist = opt.DisableBuylist
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (sz *Strikezone) Load(ctx context.Context) error {
	var errs []error

	if !sz.disableRetail {
		err := sz.scrape(ctx, modeRetail)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		}
	}

	if !sz.disableBuylist {
		err := sz.scrape(ctx, modeBuylist)
		if err != nil {
			errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

// Inventory returns what Load collected. See mtgban.Seller.
func (sz *Strikezone) Inventory() mtgban.InventoryRecord {
	return sz.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (sz *Strikezone) Buylist() mtgban.BuylistRecord {
	return sz.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (sz *Strikezone) Info() (info mtgban.ScraperInfo) {
	info.Name = "Strike Zone"
	info.Shorthand = "SZ"
	info.InventoryTimestamp = &sz.inventoryDate
	info.BuylistTimestamp = &sz.buylistDate
	info.Game = sz.game
	return
}
