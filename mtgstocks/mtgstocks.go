package mtgstocks

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
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
	cardId string
	entry  mtgban.InventoryEntry
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
	"Cevill, Bane of Monsters":    "Chevill, Bane of Monsters",
	"Frontland Felidar":           "Frondland Felidar",
	"Ragurin Crystal":             "Raugrin Crystal",
	"Bastion of Rememberance":     "Bastion of Remembrance",
	"Rograkh, Son of Gohgahh":     "Rograkh, Son of Rohgahh",
	"Swords of Plowshares":        "Swords to Plowshares",
	"Kedniss, Emberclaw Familiar": "Kediss, Emberclaw Familiar",

	"Battra, Terror of the City (JP Alternate Art)": "Dirge Bat (Godzilla)",
}

func (stks *MTGStocks) processEntry(channel chan<- responseChan, req requestChan) error {
	if req.interest.Percentage < 0 {
		return nil
	}

	edition := req.interest.Print.SetName

	fullName := req.interest.Print.Name
	fullName = strings.Replace(fullName, "[", "(", 1)
	fullName = strings.Replace(fullName, "]", ")", 1)

	if mtgmatcher.IsToken(fullName) ||
		strings.Contains(fullName, "Biography Card") ||
		strings.Contains(fullName, "Ultra Pro Puzzle Quest") ||
		strings.Contains(edition, "Oversize") {
		return nil
	}

	lutName, found := cardTable[fullName]
	if found {
		fullName = lutName
	}

	s := mtgmatcher.SplitVariants(fullName)

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

	if variant == "Welcome Back Promo Hangarback Walker Miscellaneous Promos" {
		cardName = "Hangarback Walker"
		edition = "PLGS"
	}

	lutName, found = cardTable[cardName]
	if found {
		cardName = lutName
	}

	switch edition {
	case "Revised Edition (Foreign White Border)":
		return nil
	case "Secret Lair Series":
		if cardName == "Thalia, Guardian of Thraben" && variant == "" {
			variant = "37"
		}
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
	case "Arena Promos":
		if cardName == "Underworld Dreams" {
			edition = "DCI"
		}
	case "WPN & Gateway Promos":
		if cardName == "Deathless Angel" {
			edition = "Rise of the Eldrazi Promos"
		}
	case "Judge Promos":
		switch cardName {
		case "Vampiric Tutor":
			if variant == "" {
				variant = "2000"
			}
		}
	case "Miscellaneous Promos":
		if variant == "Magic Scholarship" {
			edition = "Junior Super Series"
		}
	case "Unglued":
		if strings.HasSuffix(variant, "Right") {
			variant = "29"
		} else if strings.HasSuffix(variant, "Left") {
			variant = "28"
		}
	case "Ikoria: Lair of Behemoths: Extras":
		if variant == "JP Alternate Art" {
			variant = "Godzilla"
		}
		edition = "Ikoria: Lair of Behemoths"
	}

	theCard := &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      req.interest.Foil,
	}
	cardId, err := mtgmatcher.Match(theCard)
	if err != nil {
		stks.printf("%q", theCard)
		stks.printf("%q", req.interest.Print)
		alias, ok := err.(*mtgmatcher.AliasingError)
		if ok {
			probes := alias.Probe()
			for _, probe := range probes {
				card, _ := mtgmatcher.GetUUID(probe)
				stks.printf("- %s", card)
			}
		}
		return err
	}

	// Some slug strings are missing quotes and are plain numbers
	var slug string
	switch v := req.interest.Print.Slug.(type) {
	case string:
		slug = v
	case int:
		slug = fmt.Sprint(v)
	default:
		stks.printf("invalid data type used for %s", cardName)
	}
	out := responseChan{
		cardId: cardId,
		entry: mtgban.InventoryEntry{
			Price:      req.interest.PresentPrice,
			Quantity:   1,
			URL:        "https://www.mtgstocks.com/prints/" + slug,
			SellerName: req.name + " " + strings.Title(req.interest.InterestType),
		},
	}

	channel <- out

	return nil
}

func (stks *MTGStocks) scrape() error {
	averages, err := AverageInterests()
	if err != nil {
		return err
	}
	markets, err := MarketInterests()
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
		for _, interest := range averages.Foil {
			pages <- requestChan{
				name:     "Average",
				interest: interest,
			}
		}
		for _, interest := range averages.Normal {
			pages <- requestChan{
				name:     "Average",
				interest: interest,
			}
		}
		for _, interest := range markets.Foil {
			pages <- requestChan{
				name:     "Market",
				interest: interest,
			}
		}
		for _, interest := range markets.Normal {
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
		err := stks.inventory.Add(result.cardId, &result.entry)
		if err != nil {
			stks.printf("%s", err.Error())
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
