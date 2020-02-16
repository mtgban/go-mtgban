package miniaturemarket

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

type Miniaturemarket struct {
	LogCallback mtgban.LogCallbackFunc

	db        mtgjson.MTGDB
	inventory map[string][]mtgban.InventoryEntry
	buylist   map[string][]mtgban.BuylistEntry

	norm *mtgban.Normalizer
}

func NewScraper(db mtgjson.MTGDB) *Miniaturemarket {
	mm := Miniaturemarket{}
	mm.db = db
	mm.inventory = map[string][]mtgban.InventoryEntry{}
	mm.buylist = map[string][]mtgban.BuylistEntry{}
	mm.norm = mtgban.NewNormalizer()
	return &mm
}

const (
	maxConcurrency = 6
	firstPage      = 1
	lastPage       = 9
)

type resultChan struct {
	err   error
	cards []mtgban.BuylistEntry
}

func (mm *Miniaturemarket) printf(format string, a ...interface{}) {
	if mm.LogCallback != nil {
		mm.LogCallback(format, a...)
	}
}

func (mm *Miniaturemarket) processEntry(page int) (res resultChan) {
	mmClient := NewMMClient()
	buyback, err := mmClient.BuyBackPage(MMCategoryMtgSingles, page)
	if err != nil {
		res.err = err
		return
	}

	for _, card := range buyback {
		// This field is always "[Foil] Near Mint" or null for sealed
		if card.MtgCondition == "" ||
			card.MtgSet == "Bulk MTG" ||
			card.MtgRarity == "Sealed Product" {
			continue
		}

		cond := card.MtgCondition
		switch cond {
		case "Near Mint", "Foil Near Mint", "Foil Near MInt":
			cond = "NM"
		default:
			mm.printf("Unsupported %s condition", cond)
			continue
		}

		// This field is always "<name> - <set> (<condition>)"
		names := strings.Split(card.Name, " - ")
		cardName := names[0]

		// Skip non-singles cards
		if strings.Contains(cardName, "Token") ||
			strings.Contains(cardName, "Emblem") ||
			strings.Contains(cardName, "Oversized") ||
			cardName == "Experience Counter" || cardName == "Poison Counter" {
			continue
		}
		if strings.HasPrefix(cardName, "Mana Crypt") && strings.Contains(cardName, "(Media Insert)") && !strings.Contains(cardName, "(English)") {
			continue
		}

		// Needed to discern duplicates of this particular card
		if strings.HasPrefix(cardName, "Sorcerous Spyglass") {
			cardName += " (" + card.SKU + ")"
		}

		if strings.Contains(cardName, " #") {
			fields := strings.Split(cardName, " #")
			subfields := strings.Split(fields[1], " ")
			cardName = strings.Replace(cardName, "#"+subfields[0], "("+subfields[0]+")", 1)
		}

		if strings.Contains(cardName, " [") && strings.Contains(cardName, "]") {
			cardName = strings.Replace(cardName, "[", "(", 1)
			cardName = strings.Replace(cardName, "]", ")", 1)
		}

		lutName, found := cardTable[cardName]
		if found {
			cardName = lutName
		}

		// Skip foreign cards
		switch card.MtgSet {
		case "Legends":
			if strings.Contains(cardName, "Italian") {
				continue
			}
		case "Portal Three Kingdoms":
			if strings.Contains(cardName, "Chinese") || strings.Contains(cardName, "Japanese") {
				continue
			}
		case "Duel Decks: Jace vs. Chandra":
			if strings.Contains(cardName, "Japanese") {
				continue
			}
		}

		price, err := strconv.ParseFloat(card.Price, 64)
		if err != nil {
			res.err = err
			return
		}

		mCard := mmCard{
			Name:    cardName,
			Edition: card.MtgSet,
			Foil:    card.IsFoil,
		}

		cc, err := mm.convert(&mCard)
		if err != nil {
			mm.printf("%v", err)
			continue
		}

		out := mtgban.BuylistEntry{
			Card:       *cc,
			Conditions: cond,
			BuyPrice:   price,
			TradePrice: price * 1.3,
			Quantity:   0,
		}

		res.cards = append(res.cards, out)
	}

	return
}

func (mm *Miniaturemarket) parseBL() error {
	pages := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				results <- mm.processEntry(page)
			}
			wg.Done()
		}()
	}

	go func() {
		for i := firstPage; i <= lastPage; i++ {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			mm.printf("%v", result.err)
			continue
		}
		for _, card := range result.cards {
			err := mm.BuylistAdd(card)
			if err != nil {
				mm.printf(err.Error())
				continue
			}
		}
	}

	return nil
}

func (mm *Miniaturemarket) BuylistAdd(card mtgban.BuylistEntry) error {
	entries, found := mm.buylist[card.Id]
	if found {
		for _, entry := range entries {
			if entry.BuyPrice == card.BuyPrice {
				return fmt.Errorf("Attempted to add a duplicate buylist card:\n-new: %v\n-old: %v", card, entry)
			}
		}
	}

	mm.buylist[card.Id] = append(mm.buylist[card.Id], card)
	return nil
}

func (mm *Miniaturemarket) Buylist() (map[string][]mtgban.BuylistEntry, error) {
	if len(mm.buylist) > 0 {
		return mm.buylist, nil
	}

	mm.printf("Empty buylist, scraping started")

	err := mm.parseBL()
	if err != nil {
		return nil, err
	}

	return mm.buylist, nil
}

func (mm *Miniaturemarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Miniature Market"
	info.Shorthand = "MM"
	return
}
