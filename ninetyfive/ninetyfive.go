package ninetyfive

import (
	"errors"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	GameMagic   = "MTG"
	GameLorcana = "LRC"
)

type Ninetyfive struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	client *NFClient
	game   string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
	buylistDate   time.Time
	buylist       mtgban.BuylistRecord
}

func NewScraper(game string) (*Ninetyfive, error) {
	nf := Ninetyfive{}
	nf.inventory = mtgban.InventoryRecord{}
	nf.buylist = mtgban.BuylistRecord{}
	nf.client = NewNFClient()
	nf.MaxConcurrency = defaultConcurrency
	nf.game = game
	return &nf, nil
}

func (nf *Ninetyfive) printf(format string, a ...interface{}) {
	if nf.LogCallback != nil {
		nf.LogCallback("[95] "+format, a...)
	}
}

func (nf *Ninetyfive) processPrices(allCards NFCard, allPrices NFPrice, mode string) error {
	for key, items := range allPrices {
		if allCards[key].SetSupertype != nf.game {
			continue
		}
		for sku, priceSet := range items {
			var quantity int
			var priceStr string
			var lang string
			if mode == "retail" {
				priceStr = priceSet.Price
				quantity = priceSet.Quan
				fields := strings.Split(sku, "_")
				if len(fields) > 3 {
					lang = fields[3]
				}
			} else if mode == "buylist" {
				priceStr = priceSet.BuyPrice
				quantity = priceSet.QuantityBuy
				lang = sku
			}

			price, err := mtgmatcher.ParsePrice(priceStr)
			if err != nil {
				continue
			}

			if quantity == 0 || price == 0 {
				continue
			}

			foil := strings.HasSuffix(sku, "_true") || allCards[key].DedFoil == "yes"

			var cardId string
			if nf.game == GameMagic {
				theCard, err := preprocess(allCards, key, lang, foil)
				if err != nil {
					continue
				}

				cardId, err = mtgmatcher.Match(theCard)
			} else if nf.game == GameLorcana {
				cardId, err = mtgmatcher.SimpleSearch(allCards[key].CardName, allCards[key].CardNum, foil)
			} else {
				err = errors.New("unsupported game")
			}
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				var alias *mtgmatcher.AliasingError

				nf.printf("%v", err)
				nf.printf("%s: %q", key, allCards[key])
				if alias != nil {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						nf.printf("- %s", card)
					}
				}
				continue
			}

			link := "https://shop.95gamecenter.com/app.php"
			if mode == "retail" {
				var cond string
				switch {
				// _MT_ case is skipped
				case strings.Contains(sku, "_MT_"):
					continue
				case strings.Contains(sku, "_NM_"):
					cond = "NM"
				case strings.Contains(sku, "_LP_"):
					cond = "SP"
				case strings.Contains(sku, "_MP_"):
					cond = "MP"
				case strings.Contains(sku, "_HP_"):
					cond = "HP"
				case strings.Contains(sku, "_D_"):
					cond = "PO"
				default:
					nf.printf("unknown SKU format: %s", sku)
					continue
				}

				err = nf.inventory.Add(cardId, &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      price,
					Quantity:   quantity,
					URL:        link,
					OriginalId: key,
					InstanceId: sku,
				})
			} else if mode == "buylist" {
				idsToAdd := []string{cardId}
				// Buylist for the foil version of the card is the same
				cardFoilId, err := mtgmatcher.MatchId(cardId, true)
				if err != nil && cardFoilId != "" && cardFoilId != cardId {
					idsToAdd = append(idsToAdd, cardFoilId)
				}

				for _, id := range idsToAdd {
					invCards := nf.inventory[id]
					var sellPrice float64
					var priceRatio float64
					for _, invCard := range invCards {
						sellPrice = invCard.Price
						break
					}
					if sellPrice > 0 {
						priceRatio = price / sellPrice * 100
					}

					err = nf.buylist.Add(id, &mtgban.BuylistEntry{
						BuyPrice:   price,
						PriceRatio: priceRatio,
						Quantity:   quantity,
						URL:        link,
						OriginalId: key,
					})
				}
			}
			// Todo check codes are correct
			if err != nil && allCards[key].SetCode != "MB1" && allCards[key].SetCode != "pLIST" {
				nf.printf("%s: %s", key, err.Error())
			}
		}
	}

	nf.inventoryDate = time.Now()
	nf.buylistDate = time.Now()

	return nil
}

func (nf *Ninetyfive) getAllCards() (NFCard, error) {
	list, err := nf.client.getIndexList()
	if err != nil {
		return nil, err
	}

	pages := make(chan string)
	channel := make(chan NFCard)
	var wg sync.WaitGroup

	for i := 0; i < nf.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				cards, err := nf.client.getCards(page)
				if err != nil {
					nf.printf("%s: %s", page, err.Error())
					continue
				}
				channel <- cards
			}
			wg.Done()
		}()
	}

	go func() {
		for _, page := range list[1:] {
			pages <- page
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	allCards := NFCard{}
	for record := range channel {
		maps.Copy(allCards, record)
	}

	return allCards, nil
}

func (nf *Ninetyfive) scrape(mode string) error {
	allCards, err := nf.getAllCards()
	if err != nil {
		return err
	}
	nf.printf("Loaded %d cards", len(allCards))

	var allPrices NFPrice
	if mode == "retail" {
		allPrices, err = nf.client.getPrices()
	} else {
		allPrices, err = nf.client.getBuyPrices()
	}
	if err != nil {
		return err
	}
	nf.printf("Loaded %d prices", len(allPrices))

	return nf.processPrices(allCards, allPrices, mode)
}

func (nf *Ninetyfive) Inventory() (mtgban.InventoryRecord, error) {
	if len(nf.inventory) > 0 {
		return nf.inventory, nil
	}

	err := nf.scrape("retail")
	if err != nil {
		return nil, err
	}

	return nf.inventory, nil
}

func (nf *Ninetyfive) Buylist() (mtgban.BuylistRecord, error) {
	if len(nf.buylist) > 0 {
		return nf.buylist, nil
	}

	err := nf.scrape("buylist")
	if err != nil {
		return nil, err
	}

	return nf.buylist, nil
}

func (nf *Ninetyfive) Info() (info mtgban.ScraperInfo) {
	info.Name = "95mtg"
	info.Shorthand = "95"
	info.InventoryTimestamp = &nf.inventoryDate
	info.BuylistTimestamp = &nf.buylistDate
	switch nf.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	}
	return
}
