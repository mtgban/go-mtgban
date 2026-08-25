// Package coolstuffinc scrapes Cool Stuff Inc, for singles and sealed
// product.
package coolstuffinc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	csiInventoryURL = "https://www.coolstuffinc.com/sq/?s="
)

// The games this scraper covers, as the storefront names them.
const (
	GameMagic             = "mtg"
	GameLorcana           = "lorcana"
	GameRiftbound         = "riftbound"
	GameYuGiOh            = "yugioh"
	GameDragonBallSuper   = "dbs"
	GameOnePiece          = "optcg"
	GameStarWarsUnlimited = "swu"
	GamePokemon           = "pokemon"
)

var deductions = []float64{1, 1, 0.75}

var availableMarketNames = []string{
	"Cool Stuff Inc", "Cool Stuff Inc (unique)",
}

var name2shorthand = map[string]string{
	"Cool Stuff Inc":          "CSI",
	"Cool Stuff Inc (unique)": "CSIUnique",
}

// Coolstuffinc prices Cool Stuff Inc's singles, both what they sell and what
// they buy.
type Coolstuffinc struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	// If set to true scrape will include all entries without a nonfoil NM price
	// but will be almost twice as slow
	IncludeOOS bool

	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	TargetEdition string

	DisableRetail  bool
	DisableBuylist bool

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *http.Client
	game   string
}

// buylistNumberWord matches the number-shaped words of a buylist note.
var buylistNumberWord = regexp.MustCompile(`(?i)^[A-Z]{0,4}\d+[a-z]?(?:/[A-Z]{0,4}\d+)?[,.]?$`)

var buylistReprintNote = regexp.MustCompile(`(?i)\breprints?\b`)

// buylistVariation builds everything the buylist knows about a printing
// beyond its name. The feed spends a free-text note on the qualifier that
// the sell listing spends its variation on - "Love Ball Foil", "Detective
// Pikachu Stamped" - and reading only the number field asked the two sides
// of one product different questions.
//
// The note's own numbers stay behind. They name other products the buyer
// might have meant ("Can be Pikachu 2, 19, 41, or 45"), while the collector
// number arrives in its own field and again in the product name. A note
// that says a printing reprints another stays behind whole, because every
// word after the number is then the name of the set being reprinted rather
// than of this one.
func buylistVariation(product CSIPriceEntry) string {
	if buylistReprintNote.MatchString(product.Notes) {
		return product.Number
	}
	words := []string{product.Number}
	for word := range strings.FieldsSeq(product.Notes) {
		if buylistNumberWord.MatchString(word) {
			continue
		}
		words = append(words, word)
	}
	return strings.TrimSpace(strings.Join(words, " "))
}

// NewScraper returns a singles scraper for one game.
func NewScraper(game string) *Coolstuffinc {
	csi := Coolstuffinc{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	csi.client = client.StandardClient()
	csi.MaxConcurrency = defaultConcurrency
	csi.game = game
	return &csi
}

type responseChan struct {
	cardID   string
	invEntry *mtgban.InventoryEntry
	relaxed  bool
}

func (csi *Coolstuffinc) printf(format string, a ...any) {
	if csi.LogCallback != nil {
		csi.LogCallback("[CSI] "+format, a...)
	}
}

// saleTail is what the price column writes in front of a discounted price.
// The space is a non-breaking one, as the page spells it.
const saleTail = "Was\u00a0"

// offerCondition reads the condition out of one offer row's text. The row
// spells the quantity, the condition, whatever promotions it is running and
// the price, and only the condition is wanted.
//
// The promotions are tails on the condition, and a row may run several at
// once: the bundle flag sits in the condition column and the sale's "Was" in
// the price one, so cutting a single tail left the other row's flag glued to
// the condition and the whole offer was dropped as unparseable. Cutting
// until nothing more comes off reads them in whatever order the page lays
// them out, and a row running none is unchanged by the first pass.
func offerCondition(fullRow, qtyStr, bundleStr string) string {
	conditions := strings.TrimLeft(fullRow, qtyStr+"+ ")
	conditions = strings.Split(conditions, "$")[0]
	for {
		trimmed := strings.TrimSuffix(conditions, bundleStr)
		trimmed = strings.TrimSuffix(trimmed, saleTail)
		if trimmed == conditions {
			return conditions
		}
		conditions = trimmed
	}
}

// bundleRe matches the wording of the bundle promotion, whatever count it
// gives away: "Buy 1 get 3 free!" sells four copies for the listed price.
var bundleRe = regexp.MustCompile(`^Buy 1 get (\d+) free!$`)

// bundledCopies reads how many copies the listed price buys: one, unless the
// row runs the bundle promotion, whose price covers the bought copy and the
// free ones together. The wording carries the count, so a promotion the
// condition parser learned to cut is also the one the price is divided by,
// rather than only the one spelling the exact count the flag used to name.
//
// The price is the only thing the promotion changes. The count beside it is
// the same card-qty column every row on the page carries, promotion or not,
// capped at "20+" the way a stock figure is, so it goes out as the stock it
// reads as rather than divided to match the price.
func bundledCopies(bundleStr string) int {
	match := bundleRe.FindStringSubmatch(bundleStr)
	if match == nil {
		return 1
	}
	free, err := strconv.Atoi(match[1])
	if err != nil {
		return 1
	}
	return 1 + free
}

func (csi *Coolstuffinc) processSearch(ctx context.Context, results chan<- responseChan, itemName string) error {
	skipOOS := !csi.IncludeOOS
	switch itemName {
	case "Alpha", "Beta", "Unlimited Edition":
		skipOOS = false
	}
	result, err := Search(ctx, csi.game, itemName, skipOOS)
	if err != nil {
		return err
	}

	// result.PageId may be empty if the results have only one page
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

		doc.Find(`div[class="row product-search-row main-container"]`).Each(func(i int, s *goquery.Selection) {
			cardName := s.Find(`span[itemprop="name"]`).Text()

			pid, _ := s.Find(`span[class="rating-display "]`).Attr("data-pid")
			edition := itemName
			notes := s.Find(`div[class="large-8 medium-12 small- 12 product-notes"]`).Text()
			notes = strings.TrimPrefix(notes, "Notes: ")

			imgURL, _ := s.Find(`a[class="productLink"]`).Find("img").Attr("data-src")
			if imgURL == "" {
				imgURL, _ = s.Find(`a[class="productLink"]`).Find("img").Attr("src")
				if imgURL == "" {
					csi.printf("img not found %s %s", cardName, edition)
				}
			}

			s.Find(`div[itemprop="offers"]`).Each(func(i int, se *goquery.Selection) {
				var relaxed bool
				var graded bool
				fullRow := strings.TrimSpace(se.Text())

				switch {
				case strings.Contains(fullRow, "Out of Stock"),
					strings.Contains(fullRow, "not currently available"):
					return
				}

				qtyStr := se.Find(`span[class="card-qty"]`).Text()
				qtyStr = strings.TrimSpace(strings.TrimSuffix(qtyStr, "+"))
				// If preorder has no quantity,, set max allowed
				if qtyStr == "" && strings.Contains(notes, "Preorder") {
					qtyStr = "20"
				}

				qty, err := strconv.Atoi(qtyStr)
				if err != nil {
					csi.printf("%s", fullRow)
					csi.printf("%s %s %v", cardName, edition, err)
					return
				}

				bundleStr := se.Find(`div[class="b1-gx-free"]`).Text()
				bundleCopies := bundledCopies(bundleStr)

				conditions := offerCondition(fullRow, qtyStr, bundleStr)

				isFoil := strings.HasPrefix(conditions, "Foil")

				if strings.Contains(conditions, "BGS") ||
					strings.Contains(conditions, "Non-Foil") ||
					strings.Contains(conditions, "Unique") {
					conditions = "Near Mint"
					graded = true
				}

				// Sometimes etched cards have a Near Mint and Near Mint Foil condition
				// for the same card
				if strings.Contains(cardName, "Foil-etched") {
					relaxed = true
				}

				switch conditions {
				case "Near Mint", "Foil Near Mint":
					conditions = "NM"
				case "Played", "Foil Played":
					conditions = "MP"
				default:
					csi.printf("Unsupported '%s' condition for %s", conditions, cardName)
					return
				}
				if strings.Contains(cardName, "Signed by") {
					conditions = "HP"
				}

				priceStr := se.Find(`b[itemprop="price"]`).Text()
				price, err := strconv.ParseFloat(priceStr, 64)
				if err != nil {
					csi.printf("%v", err)
					return
				}
				if bundleCopies > 1 {
					price /= float64(bundleCopies)
				}

				if price == 0.0 || qty == 0 {
					return
				}

				link := "https://www.coolstuffinc.com/p/" + pid
				if csi.Partner != "" {
					link += "?utm_referrer=" + csi.Partner
				}

				var theCard *mtgmatcher.InputCard
				switch csi.game {
				case GameMagic:
					c, err := preprocess(cardName, edition, notes, imgURL)
					if err != nil {
						return
					}
					// preprocess() might return something that derived foil status
					// from one of the fields (cardName in particular)
					c.Foil = c.Foil || isFoil
					theCard = c
				case GameLorcana, GameRiftbound, GameOnePiece, GamePokemon, GameYuGiOh:
					theCard = &mtgmatcher.InputCard{Name: cardName, Edition: edition, Variation: notes, Foil: isFoil}
				default:
					csi.printf("unsupported game")
					return
				}

				cardID, err := mtgmatcher.Match(theCard)
				if errors.Is(err, mtgmatcher.ErrUnsupported) {
					return
				} else if err != nil {
					switch {
					// Ignore expected misses
					case theCard.IsBasicLand(),
						notes == "" && strings.Contains(edition, "The List"),
						strings.Contains(notes, "Preorder"):
					default:
						csi.printf("%v", err)
						csi.printf("%v", theCard)
						csi.printf("'%s' '%s' '%s'", cardName, edition, notes)
						csi.printf("- %s", link)

						var alias *mtgmatcher.AliasingError
						if errors.As(err, &alias) {
							for _, probe := range alias.Probe() {
								card, _ := mtgmatcher.GetUUID(probe)
								csi.printf("- %s", card)
							}
						}
					}
					return
				}

				// Magic-only finish sanity check: skip cards that do not have the
				// requested finish.
				if csi.game == GameMagic {
					if strings.Contains(cardName, "Foil-etched") {
						co, err := mtgmatcher.GetUUID(cardID)
						if err != nil || !co.Etched {
							return
						}
					}
					if isFoil {
						co, err := mtgmatcher.GetUUID(cardID)
						if err != nil || (!co.Etched && !co.Foil) {
							return
						}
					}
				}

				out := responseChan{
					cardID: cardID,
					invEntry: &mtgban.InventoryEntry{
						Conditions: conditions,
						Price:      price,
						Quantity:   qty,
						URL:        link,
						OriginalID: pid,
						SellerName: availableMarketNames[0],
					},
					relaxed: relaxed || graded,
				}

				if graded {
					out.invEntry.SellerName = availableMarketNames[1]
				}
				results <- out
			})
		})

		next, _ := doc.Find(`span[id="nextLink"]`).Find("a").Attr("href")
		if next == "" {
			break
		}
	}

	return nil
}

func (csi *Coolstuffinc) scrape(ctx context.Context) error {
	link := csiInventoryURL + csi.game
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

	var itemNames []string
	doc.Find(`fieldset`).Each(func(i int, s *goquery.Selection) {
		title := s.Find(`h2[class="mb10"] b`).Text()
		if title != "Item Set" {
			return
		}
		s.Find(`div[class="toggleTable"]`).Find("li").Each(func(j int, se *goquery.Selection) {
			itemName, _ := se.Find(`input[type="checkbox"]`).Attr("value")
			switch {
			case strings.Contains(itemName, "Bulk"),
				strings.Contains(itemName, "Random Lots"),
				strings.Contains(itemName, "Relic Token"),
				itemName == "Magic":
				return
			}

			itemNames = append(itemNames, itemName)
		})
	})
	// Sort for predictable results
	sort.Strings(itemNames)

	csi.printf("Found %d items", len(itemNames))

	start := time.Now()

	if csi.TargetEdition != "" {
		filtered := itemNames[:0]
		for _, item := range itemNames {
			if item == csi.TargetEdition {
				filtered = append(filtered, item)
			}
		}
		itemNames = filtered
	}

	mtgban.WorkerPool(ctx, csi.MaxConcurrency, itemNames,
		func(ctx context.Context, itemName string, results chan<- responseChan) error {
			csi.printf("Processing %s", itemName)
			return csi.processSearch(ctx, results, itemName)
		},
		func(record responseChan) {
			var err error
			if record.relaxed {
				err = csi.inventory.AddRelaxed(record.cardID, record.invEntry)
			} else {
				err = csi.inventory.Add(record.cardID, record.invEntry)
			}
			if err != nil {
				csi.printf("%s", err.Error())
			}
		},
		csi.printf,
	)

	csi.printf("This operation took %v", time.Since(start))

	csi.inventoryDate = time.Now()

	return nil
}

func (csi *Coolstuffinc) parseBL(ctx context.Context) error {
	edition2id, err := LoadBuylistEditions(ctx, csi.game)
	if err != nil {
		return err
	}
	csi.printf("Loaded %d editions", len(edition2id))

	products, err := GetBuylist(ctx, csi.game)
	if err != nil {
		return err
	}
	csi.printf("Found %d products", len(products))

	for _, product := range products {
		if product.RarityName == "Box" {
			continue
		}

		// Filter by set if needed
		if csi.TargetEdition != "" && product.ItemSet != csi.TargetEdition {
			continue
		}

		// Build link early to help debug
		u, _ := url.Parse(csiBuylistLink)
		v := url.Values{}
		v.Set("s", csi.game)
		v.Set("a", "1")
		v.Set("name", product.Name)
		v.Set("f[]", fmt.Sprint(product.IsFoil))

		id, found := edition2id[product.ItemSet]
		if found {
			v.Set("is[]", id)
		}
		u.RawQuery = v.Encode()
		link := u.String()

		var theCard *mtgmatcher.InputCard
		switch csi.game {
		case GameMagic:
			c, err := PreprocessBuylist(product)
			if err != nil {
				continue
			}
			theCard = c
		case GamePokemon:
			theCard = &mtgmatcher.InputCard{Name: product.Name, Edition: product.ItemSet, Variation: buylistVariation(product), Foil: product.IsFoil == 1}
		case GameLorcana, GameRiftbound, GameOnePiece, GameYuGiOh:
			theCard = &mtgmatcher.InputCard{Name: product.Name, Edition: product.ItemSet, Variation: product.Number, Foil: product.IsFoil == 1}
		default:
			return errors.New("unsupported game")
		}

		cardID, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			csi.printf("error: %v", err)
			csi.printf("original: %q", product)
			csi.printf("preprocessed: %q", theCard)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				for _, probe := range alias.Probe() {
					co, _ := mtgmatcher.GetUUID(probe)
					csi.printf("- %s", co)
				}
			}
			continue
		}

		buyPrice, err := mtgmatcher.ParsePrice(product.Price)
		if err != nil {
			csi.printf("%s error: %s", product.Name, err.Error())
			continue
		}

		var priceRatio, sellPrice float64

		invCards := csi.inventory[cardID]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = buyPrice / sellPrice * 100
		}

		for i, deduction := range deductions {
			buyEntry := mtgban.BuylistEntry{
				Conditions: mtgban.DefaultGradeTags[i],
				BuyPrice:   buyPrice * deduction,
				PriceRatio: priceRatio,
				URL:        link,
				CustomFields: map[string]string{
					"originalProduct": fmt.Sprintf("%q", product),
				},
			}

			err := csi.buylist.Add(cardID, &buyEntry)
			if err != nil {
				csi.printf("%s", err.Error())
				continue
			}
		}
	}

	csi.buylistDate = time.Now()

	return nil
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (csi *Coolstuffinc) SetConfig(opt mtgban.ScraperOptions) {
	csi.DisableRetail = opt.DisableRetail
	csi.DisableBuylist = opt.DisableBuylist
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (csi *Coolstuffinc) Load(ctx context.Context) error {
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

// Inventory returns what Load collected. See mtgban.Seller.
func (csi *Coolstuffinc) Inventory() mtgban.InventoryRecord {
	return csi.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (csi *Coolstuffinc) Buylist() mtgban.BuylistRecord {
	return csi.buylist
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (csi *Coolstuffinc) MarketNames() []string {
	if csi.Info().Game != mtgban.GameMagic {
		return availableMarketNames[:1]
	}
	return availableMarketNames
}

// InfoForScraper describes one of the sub-scrapers named above.
func (csi *Coolstuffinc) InfoForScraper(name string) mtgban.ScraperInfo {
	info := csi.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (csi *Coolstuffinc) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cool Stuff Inc"
	info.Shorthand = "CSI"
	info.InventoryTimestamp = &csi.inventoryDate
	info.BuylistTimestamp = &csi.buylistDate
	info.CreditMultiplier = 1.25
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
	}
	return
}
