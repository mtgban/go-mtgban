package eudogames

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	colly "github.com/gocolly/colly/v2"
	queue "github.com/gocolly/colly/v2/queue"
	cleanhttp "github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

const (
	defaultConcurrency = 8

	modeInventory = "inventory"
	modeBuylist   = "buylist"
)

type Eudogames struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *Eudogames {
	eudo := Eudogames{}
	eudo.inventory = mtgban.InventoryRecord{}
	eudo.buylist = mtgban.BuylistRecord{}
	eudo.MaxConcurrency = defaultConcurrency
	return &eudo
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (eudo *Eudogames) printf(format string, a ...interface{}) {
	if eudo.LogCallback != nil {
		eudo.LogCallback("[EUDO] "+format, a...)
	}
}

func (eudo *Eudogames) scrape(mode string) error {
	channel := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("store.eudogames.com"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: eudo.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		//eudo.printf("Visiting %s", r.URL.String())
	})

	c.OnHTML(`li[class="next"] a[href]`, func(e *colly.HTMLElement) {
		link := e.Attr("href")
		u, err := url.Parse(link)
		if err != nil {
			return
		}

		q := u.Query()
		page := q.Get("page")
		if page == "" {
			q.Set("page", "1")
		}
		q.Set("available_only", "True")
		q.Set("finish", "all")
		q.Set("sortField", "title")

		link = u.String()
		err = c.Visit(e.Request.AbsoluteURL(link))
		if err != nil {
			if err != colly.ErrAlreadyVisited {
				//eudo.printf("error while linking %s: %s", e.Request.AbsoluteURL(link), err.Error())
			}
		}
	})

	c.OnHTML(`li[class="clearfix"]`, func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.ChildAttr(`a[class="product-header"]`, "href"))
		cardName := e.ChildText(`h3`)

		edition := e.ChildText(`tr[id="variant_id"] td[class="text variant text-wrap"]`)
		if edition == "" {
			edition = e.ChildText(`a[class="product-header"] b`)
		}

		variant := e.ChildText(`a[class="product-header"] i`)
		if variant != "" {
			variant = strings.Replace(variant, "(", "", 1)
			variant = strings.Replace(variant, ")", "", 1)

			if variant == "promo" {
				set, err := mtgmatcher.GetSet(edition)
				if err != nil {
					return
				}
				setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
				if setDate.After(mtgmatcher.PromosForEverybodyYay) {
					for _, card := range set.Cards {
						if card.Name == cardName {
							if card.HasFrameEffect(mtgjson.FrameEffectInverted) {
								variant = "Promo Pack"
								break
							}
							if card.BorderColor == mtgjson.BorderColorBorderless {
								variant = "Borderless"
								break
							}
						}
					}
				}
			}
		}

		e.ForEach(`table[class="product-stock available"] tr[id="variant_id"]`, func(_ int, elem *colly.HTMLElement) {
			meta := elem.ChildText(`td[class="text variant"]`)
			s := strings.Split(meta, ", ")
			if len(s) != 3 {
				log.Println(len(s), meta)
				return
			}

			conditions := s[0]
			switch conditions {
			case "condition: Near Mint":
				conditions = "NM"
			case "condition: Lightly Played":
				conditions = "SP"
			case "condition: Moderately Played":
				conditions = "MP"
			case "condition: Heavily Played":
				conditions = "HP"
			case "condition: Damaged":
				conditions = "PO"
			default:
				eudo.printf("Unsupported %s condition for %s %s", conditions, cardName, edition)
				return
			}

			isFoil := s[1] == "finish: Foil"

			if !strings.Contains(s[2], "English") {
				eudo.printf("%s %s %s", cardName, edition, meta)
				return
			}

			qtyStr := elem.ChildText(`td[class="number qty"]`)
			if mode == modeBuylist {
				qtyStr = strings.TrimPrefix(qtyStr, "Buying ")
			}
			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				eudo.printf("%s %s %v", cardName, edition, err)
				return
			}

			priceStr := elem.ChildText(`td[class="number price"]`)
			priceStr = strings.Replace(priceStr, "$", "", 1)
			priceStr = strings.Replace(priceStr, ",", "", 1)
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				eudo.printf("%s %s %v", cardName, edition, err)
				return
			}

			if price == 0.0 || qty == 0 {
				return
			}

			theCard := &mtgmatcher.Card{
				Name:      cardName,
				Variation: variant,
				Edition:   edition,
				Foil:      isFoil,
			}

			cardId, err := mtgmatcher.Match(theCard)
			if err != nil {
				switch edition {
				// No easy way to separate old variants, so lots of duplicate warnings
				case "Alliances",
					"Commander Anthology Volume II",
					"Fallen Empires",
					"Homelands":
				default:
					// Ignore aliasing errors
					_, ok := err.(*mtgmatcher.AliasingError)
					if !ok {
						eudo.printf("%v", err)
						eudo.printf("%q", theCard)
					}
				}
				return
			}

			var out responseChan
			if mode == modeInventory {
				out = responseChan{
					cardId: cardId,
					invEntry: &mtgban.InventoryEntry{
						Conditions: conditions,
						Price:      price,
						Quantity:   qty,
						URL:        link,
					},
				}
			} else {
				out = responseChan{
					cardId: cardId,
					buyEntry: &mtgban.BuylistEntry{
						BuyPrice:   price,
						TradePrice: price * 1.3,
						Quantity:   qty,
						URL:        link,
					},
				}
			}

			channel <- out
		})
	})

	q, _ := queue.New(
		eudo.MaxConcurrency,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	sets := mtgmatcher.GetSets()
	setNames := []string{}
	for _, set := range sets {
		setNames = append(setNames, set.Name)
	}

	for _, edition := range setNames {
		edition = strings.ToLower(edition)
		edition = strings.Replace(edition, " ", "-", -1)
		edition = strings.Replace(edition, ":", "", -1)

		if mode == modeInventory {
			q.AddURL("https://store.eudogames.com/catalogue/magic-card/" + edition + "/?available_only=True&finish=all&sortField=title")
		} else {
			q.AddURL("https://store.eudogames.com/catalogue/buylist/magic-card/" + edition + "/?available_only=True&finish=all&sortField=title")
		}
	}

	q.Run(c)

	go func() {
		c.Wait()
		close(channel)
	}()

	for res := range channel {
		var err error
		if mode == modeInventory {
			err = eudo.inventory.Add(res.cardId, res.invEntry)
		} else {
			err = eudo.buylist.Add(res.cardId, res.buyEntry)
		}
		if err != nil {
			eudo.printf("%v", err)
		}
	}

	if mode == modeInventory {
		eudo.inventoryDate = time.Now()
	} else {
		eudo.buylistDate = time.Now()
	}

	return nil
}

func (eudo *Eudogames) Inventory() (mtgban.InventoryRecord, error) {
	if len(eudo.inventory) > 0 {
		return eudo.inventory, nil
	}

	err := eudo.scrape(modeInventory)
	if err != nil {
		return nil, err
	}

	return eudo.inventory, nil
}

func (eudo *Eudogames) Buylist() (mtgban.BuylistRecord, error) {
	if len(eudo.buylist) > 0 {
		return eudo.buylist, nil
	}

	err := eudo.scrape(modeBuylist)
	if err != nil {
		return nil, err
	}

	return eudo.buylist, nil
}

func (eudo *Eudogames) Info() (info mtgban.ScraperInfo) {
	info.Name = "Eudo Games"
	info.Shorthand = "EUDO"
	info.InventoryTimestamp = eudo.inventoryDate
	info.BuylistTimestamp = eudo.buylistDate
	info.Grading = mtgban.DefaultGrading
	return
}
