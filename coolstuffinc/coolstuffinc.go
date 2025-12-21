package coolstuffinc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	csiInventoryURL = "https://www.coolstuffinc.com/sq/?s="

	GameMagic             = "mtg"
	GameLorcana           = "lorcana"
	GameYugioh            = "yugioh"
	GameDragonBallSuper   = "dbs"
	GameOnePiece          = "optcg"
	GameStarWarsUnlimited = "swu"
	GamePokemon           = "pokemon"
)

var deductions = []float64{1, 1, 0.75}

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
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
	relaxed  bool
}

func (csi *Coolstuffinc) printf(format string, a ...interface{}) {
	if csi.LogCallback != nil {
		csi.LogCallback("[CSI] "+format, a...)
	}
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
			link := "https://www.coolstuffinc.com/sq/" + result.PageId + "?page=" + fmt.Sprint(page)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
			if err != nil {
				continue
			}
			resp, err := csi.client.Do(req)
			if err != nil {
				continue
			}
			data, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
		if err != nil {
			csi.printf("newDoc - %s", err.Error())
			continue
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
				bundle := bundleStr == "Buy 1 get 3 free!"

				// Derive the condition portion
				conditions := strings.TrimLeft(fullRow, qtyStr+"+ ")
				conditions = strings.Split(conditions, "$")[0]
				conditions = strings.TrimSuffix(conditions, bundleStr)
				// From the sale text, there is a weird space
				conditions = strings.TrimSuffix(conditions, "Was ")

				isFoil := strings.HasPrefix(conditions, "Foil")

				if strings.Contains(conditions, "BGS") ||
					strings.Contains(conditions, "Non-Foil") ||
					strings.Contains(conditions, "Unique") {
					conditions = "Near Mint"
					relaxed = true
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
				if bundle {
					price /= 4
				}

				if price == 0.0 || qty == 0 {
					return
				}

				link := "https://www.coolstuffinc.com/p/" + pid
				if csi.Partner != "" {
					link += "?utm_referrer=" + csi.Partner
				}

				var cardId string
				switch csi.game {
				case GameMagic:
					theCard, err := preprocess(cardName, edition, notes, imgURL)
					if err != nil {
						return
					}
					// preprocess() might return something that derived foil status
					// from one of the fields (cardName in particular)
					theCard.Foil = theCard.Foil || isFoil

					cardId, err = mtgmatcher.Match(theCard)
					if errors.Is(err, mtgmatcher.ErrUnsupported) {
						return
					} else if err != nil {
						switch {
						// Ignore errors
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
								probes := alias.Probe()
								for _, probe := range probes {
									card, _ := mtgmatcher.GetUUID(probe)
									csi.printf("- %s", card)
								}
							}
						}
						return
					}

					// Sanity check, skip cards that do not have the right finish
					if strings.Contains(cardName, "Foil-etched") {
						co, err := mtgmatcher.GetUUID(cardId)
						if err != nil || !co.Etched {
							return
						}
					}
					if isFoil {
						co, err := mtgmatcher.GetUUID(cardId)
						if err != nil || (!co.Etched && !co.Foil) {
							return
						}
					}
				case GameLorcana:
					number := mtgmatcher.ExtractNumber(strings.Split(notes, "/")[0])

					cardId, err = mtgmatcher.SimpleSearch(cardName, number, isFoil)
					if errors.Is(err, mtgmatcher.ErrUnsupported) {
						return
					} else if err != nil {
						csi.printf("%v", err)
						csi.printf("%s %s %v", cardName, number, isFoil)

						var alias *mtgmatcher.AliasingError
						if errors.As(err, &alias) {
							probes := alias.Probe()
							csi.printf("%s got ids: %s", cardName, probes)
							for _, probe := range probes {
								co, _ := mtgmatcher.GetUUID(probe)
								csi.printf("%s: %s", probe, co)
							}
						}
						return
					}
				default:
					csi.printf("unsupported game")
					return
				}

				out := responseChan{
					cardId: cardId,
					invEntry: &mtgban.InventoryEntry{
						Conditions: conditions,
						Price:      price,
						Quantity:   qty,
						URL:        link,
						OriginalId: pid,
					},
					relaxed: relaxed,
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

	items := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < csi.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for itemName := range items {
				csi.printf("Processing %s", itemName)
				err := csi.processSearch(ctx, results, itemName)
				if err != nil {
					csi.printf("%v for %s", err, itemName)
				}
			}
			wg.Done()
		}()
	}
	go func() {
		for _, item := range itemNames {
			if csi.TargetEdition != "" && item != csi.TargetEdition {
				continue
			}
			items <- item
		}
		close(items)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		var err error
		if record.relaxed {
			err = csi.inventory.AddRelaxed(record.cardId, record.invEntry)
		} else {
			err = csi.inventory.Add(record.cardId, record.invEntry)
		}
		if err != nil {
			csi.printf("%s", err.Error())
		}
	}

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

		var cardId string
		if csi.game == GameMagic {
			theCard, err := PreprocessBuylist(product)
			if err != nil {
				continue
			}

			cardId, err = mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				csi.printf("error: %v", err)
				csi.printf("original: %q", product)
				csi.printf("preprocessed: %q", theCard)
				csi.printf("link: %q", link)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						csi.printf("- %s", card)
					}
				}
				continue
			}
		} else if csi.game == GameLorcana {
			number := strings.Split(strings.TrimLeft(product.Number, "0"), "/")[0]
			cardName := product.Name

			cardId, err = mtgmatcher.SimpleSearch(cardName, number, product.IsFoil == 1)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				csi.printf("%v", err)
				csi.printf("%+v", product)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					csi.printf("%s got ids: %s", cardName, probes)
					for _, probe := range probes {
						co, _ := mtgmatcher.GetUUID(probe)
						csi.printf("%s: %s", probe, co)
					}
				}
				continue
			}
		} else {
			return errors.New("unsupported game")
		}

		buyPrice, err := mtgmatcher.ParsePrice(product.Price)
		if err != nil {
			csi.printf("%s error: %s", product.Name, err.Error())
			continue
		}

		var priceRatio, sellPrice float64

		invCards := csi.inventory[cardId]
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

			err := csi.buylist.Add(cardId, &buyEntry)
			if err != nil {
				csi.printf("%s", err.Error())
				continue
			}
		}
	}

	csi.buylistDate = time.Now()

	return nil
}

func (csi *Coolstuffinc) SetConfig(opt mtgban.ScraperOptions) {
	csi.DisableRetail = opt.DisableRetail
	csi.DisableBuylist = opt.DisableBuylist
}

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

func (csi *Coolstuffinc) Inventory() mtgban.InventoryRecord {
	return csi.inventory
}

func (csi *Coolstuffinc) Buylist() mtgban.BuylistRecord {
	return csi.buylist
}

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
	}
	return
}
