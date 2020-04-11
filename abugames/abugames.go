package abugames

import (
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

const (
	maxConcurrency = 8
)

type ABUGames struct {
	LogCallback   mtgban.LogCallbackFunc
	InventoryDate time.Time
	BuylistDate   time.Time

	client *ABUClient

	db        mtgjson.MTGDB
	inventory map[string][]mtgban.InventoryEntry
	buylist   map[string]mtgban.BuylistEntry

	norm *mtgban.Normalizer
}

func NewScraper(db mtgjson.MTGDB) *ABUGames {
	abu := ABUGames{}
	abu.db = db
	abu.inventory = map[string][]mtgban.InventoryEntry{}
	abu.buylist = map[string]mtgban.BuylistEntry{}
	abu.norm = mtgban.NewNormalizer()
	abu.client = NewABUClient()
	return &abu
}

type resultChan struct {
	err       error
	inventory []mtgban.InventoryEntry
	buylist   []mtgban.BuylistEntry
}

func (abu *ABUGames) printf(format string, a ...interface{}) {
	if abu.LogCallback != nil {
		abu.LogCallback("[ABU] "+format, a...)
	}
}

func (abu *ABUGames) processEntry(page int) (res resultChan) {
	product, err := abu.client.GetProduct(page)
	if err != nil {
		res.err = err
		return
	}

	duplicate := map[string]bool{}

	for _, group := range product.Grouped.ProductId.Groups {
		for _, card := range group.Doclist.Cards {
			// Deprecated value
			if card.Condition == "SP" {
				continue
			}

			cond := card.Condition
			switch cond {
			case "NM", "HP":
			case "MINT":
				cond = "NM"
			case "PLD":
				cond = "SP"
			default:
				abu.printf("Unknown '%s' condition", cond)
				continue
			}

			isFoil := strings.Contains(strings.ToLower(card.DisplayTitle), " foil")

			if duplicate[card.Id] {
				abu.printf("Skipping duplicate card: %s (%s %q)", card.SimpleTitle, card.Edition, isFoil)
				continue
			}

			lang := ""
			if len(card.Language) > 0 {
				switch card.Language[0] {
				case "English":
					lang = "EN"
				case "French":
					lang = "FR"
				case "German":
					lang = "DE"
				case "Italian":
					lang = "IT"
				case "Spanish":
					lang = "ES"
				case "Portuguese":
					lang = "PT"
				case "Japanese":
					lang = "JP"
				case "Korean":
					lang = "KR"
				case "Chinese Simplified":
					lang = "CH"
				case "Russian":
					lang = "RU"
				default:
					lang = card.Language[0]
				}
			}

			if lang != "EN" || strings.Contains(card.Title, "Non-English") {
				continue
			}

			// Non-Singles magic cards
			switch card.Layout {
			case "Scheme", "Plane", "Phenomenon":
				continue
			}
			if strings.Contains(card.DisplayTitle, "Oversized") ||
				strings.Contains(card.DisplayTitle, "Charlie Brown") {
				continue
			}
			// Non-existing cards
			switch card.DisplayTitle {
			case "Steward of Valeron (Dengeki Character Festival) - FOIL",
				"Captain's Claws (Goldnight Castigator Shadow) - FOIL",
				"Island (Arena 1999 Urza Saga No Symbol) - FOIL",
				"Beast of Burden (Prerelease - No Expansion Symbol) - FOIL",
				"Hymn to Tourach (B - Mark Justice - 1996)",
				"Mountain (6th Edition 343 - Mark Le Pine - 1999)":
				continue
			}
			// Unique cards
			if strings.HasPrefix(card.Title, "ID#") {
				continue
			}

			number := strings.TrimLeft(card.Number, "0")

			// Drop any foil reference from the name (careful not to drop the Foil card)
			fullName := strings.TrimSpace(card.DisplayTitle)
			fullName = strings.Replace(fullName, " - Foil", "", 1)
			fullName = strings.Replace(fullName, " - FOIL", "", 1)
			fullName = strings.Replace(fullName, " FOIL", "", 1)
			fullName = strings.Replace(fullName, " Foil", "", 1)

			// Merge Prerelease and Promo Pack tags in the full name for later parsing
			fullName = strings.Replace(fullName, "- (Prerelease)", "(Prerelease)", 1)
			fullName = strings.Replace(fullName, "- (Promo Pack)", "(Promo Pack)", 1)

			// Fix some untagged prerelease cards
			if strings.HasSuffix(fullName, " - "+card.Edition) {
				fullName = strings.Replace(fullName, "- "+card.Edition, "(Prerelease)", 1)
			}

			layout := ""
			if card.Layout != "Normal" {
				layout = card.Layout
			}
			// Fix a card with wrong information
			if card.SimpleTitle == "Repudiate and Replicate" {
				layout = "Split"
			}

			switch card.Edition {
			case "Anthologies":
				if fullName == "Mountain (A)" {
					fullName = "Mountain (B)"
				} else if fullName == "Mountain (B)" {
					fullName = "Mountain (A)"
				}
			case "World Championship":
				switch {
				case strings.HasSuffix(fullName, "1996)"):
					if card.SimpleTitle == "Mishra's Factory" {
						number = "361"
					}
				case strings.HasSuffix(fullName, "1997)"):
					if card.SimpleTitle == "Pyroblast" {
						number = "262"
					}
				case strings.HasSuffix(fullName, "2000)"):
					switch card.SimpleTitle {
					case "Wrath of God":
						number = "54"
					case "Snake Basket":
						number = "312"
					case "Meekstone":
						number = "299"
					case "Sky Diamond":
						number = "311"
					}
				case strings.HasSuffix(fullName, "2003)"):
					if card.SimpleTitle == "Phantom Nishoba" {
						number = "190"
					}
				}
			}

			cardName := card.SimpleTitle
			name, found := cardTable[cardName]
			if found {
				cardName = name
			}

			aCard := abuCard{
				Name: cardName,
				Set:  card.Edition,
				Foil: isFoil,

				FullName: fullName,
				Number:   number,
				Layout:   layout,
				Id:       card.Id,
			}

			cc, err := abu.convert(&aCard)
			if err != nil {
				abu.printf("%v", err)
				continue
			}

			if card.SellQuantity > 0 && card.SellPrice > 0 {
				notes := "https://abugames.com/magic-the-gathering/singles?search=\"" + card.SimpleTitle + "\"&magic_edition=[\"" + card.Edition + "\"]"
				if isFoil {
					notes += "&card_style=[\"Foil\"]"
				} else {
					notes += "&card_style=[\"Normal\"]"
				}

				out := mtgban.InventoryEntry{
					Card:       *cc,
					Conditions: cond,
					Price:      card.SellPrice,
					Quantity:   card.SellQuantity,
					Notes:      notes,
				}
				res.inventory = append(res.inventory, out)
			}

			if card.BuyQuantity > 0 && card.BuyPrice > 0 && card.TradePrice > 0 && card.Condition == "NM" {
				var priceRatio, qtyRatio float64
				if card.SellPrice > 0 {
					priceRatio = card.BuyPrice / card.SellPrice * 100
				}
				if card.SellQuantity > 0 {
					qtyRatio = float64(card.BuyQuantity) / float64(card.SellQuantity) * 100
				}

				notes := "https://abugames.com/buylist/magic-the-gathering/singles?search=\"" + card.SimpleTitle + "\"&magic_edition=[\"" + card.Edition + "\"]"
				if isFoil {
					notes += "&card_style=[\"Foil\"]"
				} else {
					notes += "&card_style=[\"Normal\"]"
				}

				out := mtgban.BuylistEntry{
					Card:          *cc,
					Conditions:    cond,
					BuyPrice:      card.BuyPrice,
					TradePrice:    card.TradePrice,
					Quantity:      card.BuyQuantity,
					PriceRatio:    priceRatio,
					QuantityRatio: qtyRatio,
					Notes:         notes,
				}
				res.buylist = append(res.buylist, out)
			}

			duplicate[card.Id] = true
		}
	}

	return
}

// Scrape returns an array of Entry, containing pricing and card information
func (abu *ABUGames) scrape() error {
	product, err := abu.client.GetInfo()
	if err != nil {
		return err
	}

	count := product.Grouped.ProductId.Count
	abu.printf("Parsing %d entries", count)

	pages := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				results <- abu.processEntry(page)
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < count; i += maxEntryPerRequest {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			abu.printf("%v", result.err)
			continue
		}

		for i := range result.inventory {
			err = mtgban.InventoryAdd(abu.inventory, result.inventory[i])
			if err != nil {
				abu.printf(err.Error())
				continue
			}
		}
		for i := range result.buylist {
			err = mtgban.BuylistAdd(abu.buylist, result.buylist[i])
			if err != nil {
				abu.printf(err.Error())
				continue
			}
		}
	}

	abu.InventoryDate = time.Now()
	abu.BuylistDate = time.Now()

	return nil
}

func (abu *ABUGames) Inventory() (map[string][]mtgban.InventoryEntry, error) {
	if len(abu.inventory) > 0 {
		return abu.inventory, nil
	}

	start := time.Now()
	abu.printf("Inventory scraping started at %s", start)

	err := abu.scrape()
	if err != nil {
		return nil, err
	}
	abu.printf("Inventory scraping took %s", time.Since(start))

	return abu.inventory, nil

}

func (abu *ABUGames) Buylist() (map[string]mtgban.BuylistEntry, error) {
	if len(abu.buylist) > 0 {
		return abu.buylist, nil
	}

	start := time.Now()
	abu.printf("Buylist scraping started at %s", start)

	err := abu.scrape()
	if err != nil {
		return nil, err
	}
	abu.printf("Buylist scraping took %s", time.Since(start))

	return abu.buylist, nil
}

// Purely estimated
func (abu *ABUGames) Grading(entry mtgban.BuylistEntry) (grade map[string]float64) {
	grade = map[string]float64{
		"SP": 0.70, "MP": 0.6, "HP": 0.4,
	}
	return
}

func (abu *ABUGames) Info() (info mtgban.ScraperInfo) {
	info.Name = "ABU Games"
	info.Shorthand = "ABU"
	info.InventoryTimestamp = abu.InventoryDate
	info.BuylistTimestamp = abu.BuylistDate
	return
}
