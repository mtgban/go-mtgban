package cardgameclub

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	colly "github.com/gocolly/colly/v2"
	queue "github.com/gocolly/colly/v2/queue"
	cleanhttp "github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

var inventoryURLs = []string{
	"https://www.cardgame-club.it/brand/magic-the-gathering/carte-singole-duel-decks-en.html",
	"https://www.cardgame-club.it/brand/magic-the-gathering/carte-singole-set-base-en.html",
	"https://www.cardgame-club.it/brand/magic-the-gathering/carte-singole-set-speciali-en.html",
	"https://www.cardgame-club.it/brand/magic-the-gathering/carte-singole-espansioni-en.html",
	"https://www.cardgame-club.it/brand/magic-the-gathering/carte-singole-set-introduttivi-en.html",
}

var cardTable = map[string]string{
	"Artiful Maneuver":      "Artful Maneuver",
	"Bersekers' Onslaught":  "Berserkers' Onslaught",
	"Ceantaur Omenreader":   "Centaur Omenreader",
	"Chronomatic Escape":    "Chronomantic Escape",
	"Death-Death":           "Life // Death",
	"Fastering March":       "Festering March",
	"Fire-Ice":              "Fire // Ice",
	"Force Savagery":        "Force of Savagery",
	"Gatecreeper":           "Gatecreeper Vine",
	"Guardian Shiel-Bearer": "Guardian Shield-Bearer",
	"Ichor Slik":            "Ichor Slick",
	"Lorescale Coalt":       "Lorescale Coatl",
	"Mistmeadow Skull":      "Mistmeadow Skulk",
	"Mystic Specutalion":    "Mystic Speculation",
	"Narset Trascendent":    "Narset Transcendent",
	"Night's Wisper":        "Night's Whisper",
	"Obliovion Ring":        "Oblivion Ring",
	"Oscuring Ether":        "Obscuring Aether",
	"Pyre Changer":          "Pyre Charger",
	"Pyromancer's Swalth":   "Pyromancer's Swath",
	"Seth's Tiger":          "Seht's Tiger",
	"Shielhide Dragon":      "Shieldhide Dragon",
	"Tainteed Wood":         "Tainted Wood",
	"Terramorphing Expanse": "Terramorphic Expanse",
	"Undyng Rage":           "Undying Rage",
	"Which's Mist":          "Witch's Mist",
}

type Cardgameclub struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int

	inventory    mtgban.InventoryRecord
	exchangeRate float64
}

func NewScraper() (*Cardgameclub, error) {
	cgc := Cardgameclub{}
	cgc.inventory = mtgban.InventoryRecord{}
	cgc.MaxConcurrency = defaultConcurrency
	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	cgc.exchangeRate = rate
	return &cgc, nil
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
}

func (cgc *Cardgameclub) printf(format string, a ...interface{}) {
	if cgc.LogCallback != nil {
		cgc.LogCallback("[CGC] "+format, a...)
	}
}

func (cgc *Cardgameclub) scrape() error {
	results := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("www.cardgame-club.it"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: cgc.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		//cgc.printf("Visiting %s", r.URL.String())
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if !strings.HasPrefix(link, "https://www.cardgame-club.it/brand/magic-the-gathering/carte-singole-") {
			return
		}
		if !strings.Contains(link, "-en") {
			return
		}
		if strings.Contains(link, "en.html") {
			return
		}

		u, err := url.Parse(link)
		if err != nil {
			return
		}

		v := u.Query()
		page := v.Get("p")
		if page == "" {
			v.Set("p", "1")
		}
		if v.Get("dir") != "" {
			return
		}
		if v.Get("tipo") != "" {
			return
		}
		if v.Get("serie") != "" {
			return
		}
		if v.Get("price") != "" {
			return
		}
		if v.Get("rarita") != "" {
			return
		}
		if v.Get("mode") != "" {
			return
		}
		if v.Get("brand") != "" {
			return
		}
		if v.Get("order") != "" {
			return
		}

		u.RawQuery = v.Encode()

		err = c.Visit(u.String())
		if err != nil {
			if err != colly.ErrAlreadyVisited {
				//cgc.printf("error while linking %s: %s", e.Request.AbsoluteURL(link), err.Error())
			}
		} else {
			//cgc.printf("%s", u.String())
		}
	})

	// Callback for when a scraped page contains a form element
	c.OnHTML(`div[class="product-item-details"]`, func(e *colly.HTMLElement) {

		if e.ChildText(`span[class="square-check"]`) == "Esaurito" {
			return
		}

		title := e.ChildAttr(`span[class="product-subname"]`, "data-name")

		if title == "" {
			return
		}

		// Tokens
		if strings.HasSuffix(title, "comune (EN)") ||
			strings.Contains(title, "TOKEN") ||
			strings.Contains(title, "Emblem") ||
			strings.Contains(title, "Pedina") {
			return
		}

		edition := e.ChildAttr(`span[class="product-subname"]`, "data-category")
		// "data-quantity" is fake
		priceStr := e.ChildAttr(`span[class="product-subname"]`, "data-price")

		foil := strings.Contains(title, "foil")
		isNM := strings.Contains(title, "MINT")
		isMP := strings.Contains(title, "PLAYED") || strings.Contains(title, "GOOD")

		cardName := title
		for _, rarity := range []string{" rara", " non comune", " comune", " mitica"} {
			if strings.Contains(cardName, rarity) {
				cardName = strings.Split(title, rarity)[0]
				break
			}
		}
		num := mtgmatcher.ExtractNumber(cardName)

		firstPrefixSize := len(strings.Split(cardName, " / ")[0])
		totalPrefixSize := firstPrefixSize + 3 + firstPrefixSize + 1
		if len(cardName) > totalPrefixSize {
			cardName = cardName[totalPrefixSize:]
		}

		lutName, found := cardTable[cardName]
		if found {
			cardName = lutName
		}

		cardName = strings.Replace(cardName, "EN", "it", 1)

		conditions := ""
		if isNM {
			conditions = "NM"
		} else if isMP {
			conditions = "MP"
		} else {
			cgc.printf("Unsupported condition for %s", title)
			return
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			cgc.printf("%v", err)
			return
		}

		theCard := &mtgmatcher.Card{
			Name:      cardName,
			Variation: num,
			Edition:   edition,
			Foil:      foil,
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		} else if err != nil {
			cgc.printf("%v", err)
			cgc.printf("%v", theCard)
			cgc.printf("%v", title)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					cgc.printf("- %s", card)
				}
			}
			return
		}

		link := e.ChildAttr(`h2[class="product-name"] a[class="link-product-name"]`, "href")

		out := responseChan{
			cardId: cardId,
			invEntry: &mtgban.InventoryEntry{
				Conditions: conditions,
				Price:      price * cgc.exchangeRate,
				Quantity:   1,
				URL:        link,
			},
		}

		results <- out
	})

	q, _ := queue.New(
		cgc.MaxConcurrency,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	for _, link := range inventoryURLs {
		resp, err := cleanhttp.DefaultClient().Get(link)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return err
		}

		doc.Find(`div[class="box-category"]`).Each(func(i int, s *goquery.Selection) {
			editionUrl, _ := s.Find("a").Attr("href")
			q.AddURL(editionUrl)
		})
	}
	q.Run(c)

	go func() {
		c.Wait()
		close(results)
	}()

	lastTime := time.Now()

	for result := range results {
		// This scraper has visits duplicates but quantities are fake
		// So just merge the various entries
		err := cgc.inventory.AddRelaxed(result.cardId, result.invEntry)
		if err != nil {
			cgc.printf("%v", err)
			continue
		}
		// This would be better with a select, but for now just print a message
		// that we're still alive every minute
		if time.Now().After(lastTime.Add(60 * time.Second)) {
			card, _ := mtgmatcher.GetUUID(result.cardId)
			cgc.printf("Still going, last processed card: %s", card)
			lastTime = time.Now()
		}
	}

	cgc.inventoryDate = time.Now()

	return nil
}

func (cgc *Cardgameclub) Inventory() (mtgban.InventoryRecord, error) {
	if len(cgc.inventory) > 0 {
		return cgc.inventory, nil
	}

	err := cgc.scrape()
	if err != nil {
		return nil, err
	}

	return cgc.inventory, nil

}

func (cgc *Cardgameclub) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Game Club"
	info.Shorthand = "CGC"
	info.CountryFlag = "IT"
	info.InventoryTimestamp = cgc.inventoryDate
	return
}
