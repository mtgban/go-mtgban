package abugames

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"

	http "github.com/hashicorp/go-retryablehttp"
	"github.com/kodabb/go-mtgban/mtgban"
)

type abuJSON struct {
	Grouped struct {
		ProductId struct {
			Count  int `json:"ngroups"`
			Groups []struct {
				Doclist struct {
					Cards []struct {
						Id           string `json:"id"`
						DisplayTitle string `json:"display_title"`
						SimpleTitle  string `json:"simple_title"`

						Edition   string `json:"magic_edition_sort"`
						Condition string `json:"condition"`
						Layout    string `json:"layout"`

						Rarity    string   `json:"rarity"`
						Language  []string `json:"language"`
						Title     string   `json:"title"`
						CardStyle []string `json:"card_style"`
						Number    string   `json:"card_number"`
						Artist    []string `json:"artist"`

						SellPrice    float64 `json:"price"`
						SellQuantity int     `json:"quantity"`
						BuyQuantity  int     `json:"buy_list_quantity"`
						BuyPrice     float64 `json:"buy_price"`
						TradePrice   float64 `json:"trade_price"`
					} `json:"docs"`
				} `json:"doclist"`
			} `json:"groups"`
		} `json:"product_id"`
	} `json:"grouped"`
}

var l = log.New(os.Stderr, "", 0)

const (
	maxConcurrency     = 8
	maxEntryPerRequest = 40
	abuURL             = "https://data.abugames.com/solr/nodes/select?q=*:*&fq=%2Bcategory%3A%22Magic%20the%20Gathering%20Singles%22%20%20-buy_price%3A0%20-buy_list_quantity%3A0%20%2Blanguage%3A(%22English%22)%20%2Bdisplay_title%3A*&group=true&group.field=product_id&group.ngroups=true&group.limit=10&start=0&rows=0&wt=json"
)

type ABUGames struct {
	inventory []mtgban.Entry
	buylist   []mtgban.Entry
}

func NewVendor() mtgban.Scraper {
	abu := ABUGames{}
	return &abu
}

type resultChan struct {
	err       error
	inventory []ABUCard
	buylist   []ABUCard
}

func (abu *ABUGames) processEntry(page int) (res resultChan) {
	u, err := url.Parse(abuURL)
	if err != nil {
		res.err = err
		return
	}

	q := u.Query()
	q.Set("rows", fmt.Sprintf("%d", maxEntryPerRequest))
	q.Set("start", fmt.Sprintf("%d", page))
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		res.err = err
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var db abuJSON
	err = json.Unmarshal(data, &db)
	if err != nil {
		res.err = err
		return
	}

	for _, group := range db.Grouped.ProductId.Groups {
		for _, card := range group.Doclist.Cards {
			if card.Condition != "NM" {
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
			// Duplicate cards
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
			// Duplicate "Living Twister (Promo Pack) - FOIL"
			if card.Id == "1604919" {
				continue
			}

			isFoil := strings.Contains(strings.ToLower(card.DisplayTitle), " foil")

			// Some cards lack attribution
			artist := ""
			if len(card.Artist) > 0 {
				artist = card.Artist[0]
			}

			number := strings.TrimLeft(card.Number, "0")

			// Drop any foil reference from the name (careful not to drop the Foil card)
			fullName := strings.TrimSpace(card.DisplayTitle)
			fullName = strings.Replace(fullName, " - Foil", "", -1)
			fullName = strings.Replace(fullName, " - FOIL", "", -1)
			fullName = strings.Replace(fullName, " FOIL", "", 1)
			fullName = strings.Replace(fullName, " Foil", "", 1)

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

			if card.Edition == "World Championship" {
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

			if card.BuyQuantity > 0 && card.BuyPrice > 0 {
				cc := ABUCard{
					Name:      card.SimpleTitle,
					Set:       card.Edition,
					Foil:      isFoil,
					Condition: card.Condition,

					BuyPrice:     card.BuyPrice,
					BuyQuantity:  card.BuyQuantity,
					TradePricing: card.TradePrice,

					FullName: fullName,
					Artist:   artist,
					Number:   number,
					Layout:   layout,
					Id:       card.Id,
				}
				res.buylist = append(res.buylist, cc)
			}

			if card.SellQuantity > 0 && card.SellPrice > 0 {
				cc := ABUCard{
					Name:      card.SimpleTitle,
					Set:       card.Edition,
					Foil:      isFoil,
					Condition: card.Condition,

					SellPrice:    card.SellPrice,
					SellQuantity: card.SellQuantity,

					FullName: fullName,
					Artist:   artist,
					Number:   number,
					Layout:   layout,
					Id:       card.Id,
				}

				res.inventory = append(res.inventory, cc)
			}
		}
	}

	return
}

// Scrape returns an array of Entry, containing pricing and card information
func (abu *ABUGames) Scrape() ([]mtgban.Entry, error) {
	resp, err := http.Get(abuURL)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var header abuJSON
	err = json.Unmarshal(data, &header)
	if err != nil {
		return nil, err
	}
	count := header.Grouped.ProductId.Count

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
			l.Println(result.err)
			continue
		}
		for i := range result.buylist {
			//abu.inventory = append(abu.inventory, &result.inventory[i])
			abu.buylist = append(abu.buylist, &result.buylist[i])
		}
	}

	return abu.buylist, nil
}
