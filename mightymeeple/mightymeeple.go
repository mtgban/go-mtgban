package mightymeeple

import (
	"errors"
	"fmt"
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

type Mightymeeple struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *Mightymeeple {
	meeple := Mightymeeple{}
	meeple.inventory = mtgban.InventoryRecord{}
	meeple.buylist = mtgban.BuylistRecord{}
	meeple.MaxConcurrency = defaultConcurrency
	return &meeple
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (meeple *Mightymeeple) printf(format string, a ...interface{}) {
	if meeple.LogCallback != nil {
		meeple.LogCallback("[MEEPLE] "+format, a...)
	}
}

func (meeple *Mightymeeple) scrape(mode string) error {
	channel := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("mightymeeple.com"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: meeple.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		//meeple.printf("Visiting %s", r.URL.String())
	})

	c.OnHTML(`li[class="page-item"] a[class="page-link"]`, func(e *colly.HTMLElement) {
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
				//meeple.printf("error while linking %s: %s", e.Request.AbsoluteURL(link), err.Error())
			}
		}
	})

	c.OnHTML(`div[class="page_inner"] div[class="container"]`, func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.ChildAttr(`div a`, "href"))
		cardName := e.ChildText(`h5`)
		cardName = mtgmatcher.SplitVariants(cardName)[0]

		edition := e.ChildText(`strong`)

		variant := e.ChildText(`i`)
		if variant != "" {
			variant = strings.Replace(variant, "(", "", 1)
			variant = strings.Replace(variant, ")", "", 1)

			variant = strings.TrimSuffix(variant, ", vanity")
			variant = strings.TrimPrefix(variant, "inverted, ")

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
			} else if variant == "borderless" && edition == "Ikoria: Lair of Behemoths" {
				switch cardName {
				case "Narset of the Ancient Way", "Lukka, Coppercoat Outcast":
				default:
					variant = "godzilla"
				}
			}
		}

		e.ForEach(`div table[class="table table-borderless table-sm"] tr[id="variant_id"]`, func(_ int, elem *colly.HTMLElement) {
			meta := elem.ChildText(`td[class="variant"]`)
			s := strings.Split(meta, ", ")
			if len(s) != 3 {
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
				meeple.printf("Unsupported %s condition for %s %s", conditions, cardName, edition)
				return
			}

			isFoil := s[1] == "finish: Foil"

			if !strings.Contains(s[2], "English") {
				meeple.printf("%s %s %s", cardName, edition, meta)
				return
			}

			qtyStr := elem.ChildText(`td[class="qty number"]`)
			if mode == modeBuylist {
				qtyStr = strings.TrimPrefix(qtyStr, "Buying ")
			}
			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				meeple.printf("%s %s %v", cardName, edition, err)
				return
			}

			priceStr := elem.ChildText(`td[class="price number"]`)
			price, err := mtgmatcher.ParsePrice(priceStr)
			if err != nil {
				meeple.printf("%s %s %v", cardName, edition, err)
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
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				return
			} else if err != nil {
				switch edition {
				// No easy way to separate old variants, so lots of duplicate warnings
				case "Alliances",
					"Commander Anthology Volume II",
					"Fallen Empires",
					"Homelands":
				default:
					// Ignore aliasing errors
					var alias *mtgmatcher.AliasingError
					if errors.As(err, &alias) {
						return
					}

					meeple.printf("%v", err)
					meeple.printf("%q", theCard)
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
		meeple.MaxConcurrency,
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
			q.AddURL("https://mightymeeple.com/catalogue/magic-card/" + edition + "/?available_only=True&finish=all&sortField=title")
		} else {
			q.AddURL("https://mightymeeple.com/catalogue/buylist/magic-card/" + edition + "/?available_only=True&finish=all&sortField=title")
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
			err = meeple.inventory.AddRelaxed(res.cardId, res.invEntry)
		} else {
			err = meeple.buylist.AddRelaxed(res.cardId, res.buyEntry)
		}
		if err != nil {
			meeple.printf("%v", err)
		}
	}

	if mode == modeInventory {
		meeple.inventoryDate = time.Now()
	} else {
		meeple.buylistDate = time.Now()
	}

	return nil
}

func (meeple *Mightymeeple) Inventory() (mtgban.InventoryRecord, error) {
	if len(meeple.inventory) > 0 {
		return meeple.inventory, nil
	}

	err := meeple.scrape(modeInventory)
	if err != nil {
		return nil, err
	}

	return meeple.inventory, nil
}

func (meeple *Mightymeeple) Buylist() (mtgban.BuylistRecord, error) {
	if len(meeple.buylist) > 0 {
		return meeple.buylist, nil
	}

	err := meeple.scrape(modeBuylist)
	if err != nil {
		return nil, err
	}

	return meeple.buylist, nil
}

func (meeple *Mightymeeple) Info() (info mtgban.ScraperInfo) {
	info.Name = "Mighty Meeple"
	info.Shorthand = "MEEPLE"
	info.InventoryTimestamp = meeple.inventoryDate
	info.BuylistTimestamp = meeple.buylistDate
	info.Grading = mtgban.DefaultGrading
	return
}
