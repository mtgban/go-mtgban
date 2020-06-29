package mtgstocks

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

const (
	defaultConcurrency = 8
)

type MTGStocks struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord
}

type requestChan struct {
	name     string
	interest Interest
}

type responseChan struct {
	card  mtgdb.Card
	entry mtgban.InventoryEntry
}

func (stks *MTGStocks) printf(format string, a ...interface{}) {
	if stks.LogCallback != nil {
		stks.LogCallback("[STKS] "+format, a...)
	}
}

func NewScraper() *MTGStocks {
	stks := MTGStocks{}
	stks.inventory = mtgban.InventoryRecord{}
	stks.marketplace = map[string]mtgban.InventoryRecord{}
	stks.MaxConcurrency = defaultConcurrency
	return &stks
}

var cardTable = map[string]string{
	"Cevill, Bane of Monsters": "Chevill, Bane of Monsters",
	"Frontland Felidar":        "Frondland Felidar",
	"Ragurin Crystal":          "Raugrin Crystal",
	"Bastion of Rememberance":  "Bastion of Remembrance",
}

func (stks *MTGStocks) processEntry(channel chan<- responseChan, req requestChan) error {
	if req.interest.Percentage < 0 {
		return nil
	}

	edition := req.interest.Print.SetName

	fullName := req.interest.Print.Name
	fullName = strings.Replace(fullName, "[", "(", 1)
	fullName = strings.Replace(fullName, "]", ")", 1)

	if strings.Contains(fullName, "Token") ||
		strings.Contains(fullName, "Oversize") ||
		strings.Contains(fullName, "Biography Card") ||
		strings.Contains(fullName, "Blank Card") ||
		strings.Contains(fullName, "Decklist") ||
		strings.Contains(edition, "Oversize") {
		return nil
	}

	s := mtgdb.SplitVariants(fullName)

	variant := ""
	cardName := s[0]
	if len(s) > 1 {
		variant = strings.Join(s[1:], " ")
	}

	s = strings.Split(cardName, " - ")
	cardName = s[0]
	if len(s) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += s[1]
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	switch edition {
	case "Arabian Nights":
		if variant == "Version 2" {
			variant = "dark"
		} else if variant == "Version 1" {
			variant = "light"
		}
	case "Prerelease Cards":
		variant = edition
	case "JSS/MSS Promos":
		edition = "Junior Super Series"
	case "Media Promos":
		if variant == "" {
			variant = "Book"
		}
	default:
		if strings.HasSuffix(edition, "Promos") {
			if variant != "" {
				variant += " "
			}
			variant += edition
			if strings.Contains(variant, "J20") {
				variant += " 2020"
			} else if strings.Contains(variant, "J18") {
				variant += " 2018"
			}

		}
	}

	theCard := &mtgdb.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      req.interest.Foil,
	}
	cc, err := theCard.Match()
	if err != nil {
		stks.printf("%q", theCard)
		stks.printf("%q", req.interest.Print)
		return err
	}

	out := responseChan{
		card: *cc,
		entry: mtgban.InventoryEntry{
			Price:      req.interest.PresentPrice,
			Quantity:   1,
			URL:        fmt.Sprintf("http://store.stksplayer.com/product.aspx?id=%d", req.interest.Print.Id),
			SellerName: req.name + " " + strings.Title(req.interest.InterestType),
		},
	}

	channel <- out

	return nil
}

func (stks *MTGStocks) scrape() error {
	interests, err := GetInterests()
	if err != nil {
		return err
	}

	pages := make(chan requestChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < stks.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := stks.processEntry(channel, page)
				if err != nil {
					stks.printf("%s", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, interest := range interests.Average.Foil {
			pages <- requestChan{
				name:     "Average",
				interest: interest,
			}
		}
		for _, interest := range interests.Average.Normal {
			pages <- requestChan{
				name:     "Average",
				interest: interest,
			}
		}
		for _, interest := range interests.Market.Foil {
			pages <- requestChan{
				name:     "Market",
				interest: interest,
			}
		}
		for _, interest := range interests.Market.Normal {
			pages <- requestChan{
				name:     "Market",
				interest: interest,
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := stks.inventory.Add(&result.card, &result.entry)
		if err != nil {
			stks.printf(err.Error())
			continue
		}
	}

	stks.inventoryDate = time.Now()

	return nil
}

func (stks *MTGStocks) Inventory() (mtgban.InventoryRecord, error) {
	if len(stks.inventory) > 0 {
		return stks.inventory, nil
	}

	err := stks.scrape()
	if err != nil {
		return nil, err
	}

	return stks.inventory, nil
}

func (stks *MTGStocks) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(stks.inventory) == 0 {
		_, err := stks.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := stks.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range stks.inventory {
		for i := range stks.inventory[card] {
			if stks.inventory[card][i].SellerName == sellerName {
				if stks.inventory[card][i].Price == 0 {
					continue
				}
				if stks.marketplace[sellerName] == nil {
					stks.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				stks.marketplace[sellerName][card] = append(stks.marketplace[sellerName][card], stks.inventory[card][i])
			}
		}
	}

	if len(stks.marketplace[sellerName]) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
	}
	return stks.marketplace[sellerName], nil
}

func (stks *MTGStocks) Info() (info mtgban.ScraperInfo) {
	info.Name = "MTGStocks"
	info.Shorthand = "STKS"
	info.InventoryTimestamp = stks.inventoryDate
	info.MetadataOnly = true
	return
}
