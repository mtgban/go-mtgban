package magiccorner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	http "github.com/hashicorp/go-retryablehttp"
	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

type mcJSON struct {
	Error string `json:"Message"`
	Data  []struct {
		Name     string `json:"NomeEn"`
		Set      string `json:"Category"`
		Code     string `json:"Icon"`
		Rarity   string `json:"Rarita"`
		Extra    string `json:"Image"`
		OrigName string `json:"NomeIt"`
		Variants []struct {
			Id        int     `json:"IdProduct"`
			Language  string  `json:"Lingua"`
			Foil      string  `json:"Foil"`
			Condition string  `json:"CondizioniShort"`
			Quantity  int     `json:"DispoWeb"`
			Price     float64 `json:"Price"`
		} `json:"Varianti"`
	} `json:"d"`
}

type mcParam struct {
	SearchField   string `json:"f"`
	IdCategory    string `json:"IdCategory"`
	UIc           string `json:"UIc"`
	OnlyAvailable bool   `json:"SoloDispo"`
	IsBuy         bool   `json:"IsVendita"`
}

type mcBlob struct {
	Data string `json:"d"`
}

type mcEdition struct {
	Id   int    `json:"Id"`
	Set  string `json:"Espansione"`
	Code string `json:"ImageUrl"`
}

type mcEditionParam struct {
	UIc string `json:"UIc"`
}

const (
	maxConcurrency = 7

	mcReinassanceId       = 73
	mcRevisedEUFBBId      = 1041
	mcPromoEditionId      = 1113
	mcMerfolksVsGoblinsId = 1116

	mcNumberNotAvailable = "n/a"
	mcBaseURL            = "https://www.magiccorner.it/12/modules/store/mcpub.asmx/"
	mcEditionsEndpt      = "espansioni"
	mcCardsEndpt         = "carte"
)

type Magiccorner struct {
	LogCallback   mtgban.LogCallbackFunc
	InventoryDate time.Time

	httpClient *http.Client
	norm       *mtgban.Normalizer

	db           mtgjson.MTGDB
	exchangeRate float64

	inventory map[string][]mtgban.InventoryEntry
}

func NewScraper(db mtgjson.MTGDB) (*Magiccorner, error) {
	mc := Magiccorner{}
	mc.db = db
	mc.inventory = map[string][]mtgban.InventoryEntry{}
	mc.norm = mtgban.NewNormalizer()
	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	mc.exchangeRate = rate
	mc.httpClient = http.NewClient()
	mc.httpClient.Logger = nil
	return &mc, nil
}

type resultChan struct {
	err   error
	cards []mtgban.InventoryEntry
}

func (mc *Magiccorner) printf(format string, a ...interface{}) {
	if mc.LogCallback != nil {
		mc.LogCallback("[MC] "+format, a...)
	}
}

func (mc *Magiccorner) processEntry(edition mcEdition) (res resultChan) {
	// This breaks on the main website too
	if edition.Id == mcMerfolksVsGoblinsId {
		return
	}

	// The last field before || is the language
	// 0 - any language, 72 - english only
	langCode := 0
	if edition.Id == mcPromoEditionId {
		langCode = 72
	}
	param := mcParam{
		// Search string for Id and Language
		SearchField: fmt.Sprintf("%d|0|0|0|0|%d||true|0|", edition.Id, langCode),

		// The edition/category id
		IdCategory: fmt.Sprintf("%d", edition.Id),

		// Returns entries with available quantity
		OnlyAvailable: true,

		// No idea what these fields are for
		UIc:   "it",
		IsBuy: false,
	}
	reqBody, _ := json.Marshal(&param)

	resp, err := mc.httpClient.Post(mcBaseURL+mcCardsEndpt, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		res.err = fmt.Errorf("%s - %v", edition.Set, err)
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		res.err = fmt.Errorf("%s - %d: %v", edition.Set, resp.StatusCode, err)
		return
	}
	defer resp.Body.Close()

	var db mcJSON
	err = json.Unmarshal(data, &db)
	if err != nil {
		res.err = fmt.Errorf("%s - %d: %v", edition.Set, resp.StatusCode, err)
		return
	}
	if db.Error != "" {
		res.err = fmt.Errorf("%s - %d: %v", edition.Set, resp.StatusCode, db.Error)
		return
	}

	printed := false

	// Keep track of the processed ids, and don't add duplicates
	duplicate := map[int]bool{}

	// Fixup a couple of completely wrong set codes
	codeOverride := ""
	switch edition.Set {
	case "Ravnica Allegiance: Guild Kits":
		codeOverride = "GK2"
	case "Guilds of Ravnica: Guild Kits":
		codeOverride = "GK1"
	}

	for _, card := range db.Data {
		// Override set code when necessary
		if codeOverride != "" {
			card.Code = codeOverride
		}

		// Grab the image url and keep only the image name
		extra := strings.TrimSuffix(path.Base(card.Extra), path.Ext(card.Extra))

		if !printed {
			mc.printf("Processing id %d - %s (%s, code: %s)", edition.Id, edition.Set, extra, card.Code)
			printed = true
		}

		// Trust the collector number for a few selected cases
		// Fixup set codes as needed.
		tagToDrop := ""
		number := mcNumberNotAvailable
		switch card.Code {
		case "UMA":
			if strings.HasPrefix(extra, "PUMA") {
				card.Code = "PUMA"
				tagToDrop = "PUMA"
			}
		case "1338", "1339":
			tagToDrop = extra[:2]
		case "UST", "E01", "RNA", "GRN", "ELD", "THB":
			tagToDrop = card.Code
		default:
			switch {
			case strings.HasPrefix(extra, "SLD"):
				card.Code = "SLD"
				tagToDrop = "SLD"
			case strings.HasPrefix(extra, "p2018PRWK"):
				card.Code = "PRWK"
				tagToDrop = "p2018PRWK"
			case strings.HasPrefix(extra, "p2019prwk"):
				card.Code = "PRW2"
				tagToDrop = "p2019prwk"
			}
		}

		// Drop the anything preceding the number and the leading zeros
		if tagToDrop != "" {
			number = strings.Replace(extra, tagToDrop, "", 1)
			number = strings.TrimLeft(number, "0")
		}

		// Untangle the schemes from "Archenemy: Nicol Bolas"
		// Reset number to unknown because they are offset
		if card.Code == "E01" {
			n, _ := strconv.Atoi(number)
			if n > 106 {
				card.Code = "OE01"
				number = mcNumberNotAvailable
			}
		}

		// Skip lands, too many and without a simple solution
		switch card.Name {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			continue
		}

		for _, v := range card.Variants {
			// Skip duplicate cards
			if duplicate[v.Id] {
				mc.printf("Skipping duplicate card: %s (%s %s)", card.Name, card.Set, v.Foil)
				continue
			}

			// Only keep English cards and a few other exceptions
			switch v.Language {
			case "EN":
			case "JP":
				if edition.Set != "War of the Spark: Japanese Alternate-Art Planeswalkers" {
					continue
				}
			case "IT":
				if edition.Id != mcRevisedEUFBBId && edition.Id != mcReinassanceId {
					continue
				}
			default:
				continue
			}

			if v.Quantity < 1 {
				continue
			}

			// Skip any token or similar cards
			if strings.Contains(card.Name, "Token") ||
				strings.Contains(card.Name, "token") ||
				strings.Contains(card.Name, "Art Series") ||
				strings.Contains(card.Name, "Checklist") ||
				strings.Contains(card.Name, "Check List") ||
				strings.Contains(card.Name, "Check-List") ||
				strings.Contains(card.Name, "Emblem") ||
				card.Name == "Punch Card" ||
				card.Name == "The Monarch" ||
				card.Name == "Spirit" ||
				card.Name == "City's Blessing" {
				continue
			}
			// Circle of Protection: Red in Revised EU FWB???
			if v.Id == 223958 ||
				// Excruciator RAV duplicate card
				v.Id == 108840 {
				continue
			}

			cond := v.Condition
			switch cond {
			case "NM/M":
				cond = "NM"
			case "SP", "HP":
			case "D":
				cond = "PO"
			default:
				mc.printf("Unknown '%s' condition", cond)
				continue
			}

			isFoil := v.Foil == "Foil"

			name := card.Name
			lutName, found := cardTable[card.Name]
			if found {
				name = lutName
			}

			mcCard := MCCard{
				Name: name,
				Set:  card.Set,
				Foil: isFoil,

				Id:     v.Id,
				Number: number,

				setCode: card.Code,
				extra:   extra,
				orig:    card.OrigName,
			}

			cc, err := mc.Convert(&mcCard)
			if err != nil {
				mc.printf("%v", err)
				continue
			}

			out := mtgban.InventoryEntry{
				Card:       *cc,
				Conditions: cond,
				Price:      v.Price * mc.exchangeRate,
				Quantity:   v.Quantity,
			}

			res.cards = append(res.cards, out)

			duplicate[v.Id] = true
		}
	}

	return
}

// Scrape returns an array of Entry, containing pricing and card information
func (mc *Magiccorner) scrape() error {
	// Retrieve the edition ids
	param := mcEditionParam{
		UIc: "it",
	}
	reqBody, _ := json.Marshal(&param)
	resp, err := mc.httpClient.Post(mcBaseURL+mcEditionsEndpt, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var blob mcBlob
	err = json.Unmarshal(data, &blob)
	if err != nil {
		return err
	}

	// There is json in this json
	dec := json.NewDecoder(strings.NewReader(blob.Data))
	_, err = dec.Token()
	if err != nil {
		return err
	}

	pages := make(chan mcEdition)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				results <- mc.processEntry(page)
			}
			wg.Done()
		}()
	}

	go func() {
		for dec.More() {
			var edition mcEdition
			err := dec.Decode(&edition)
			if err != nil {
				mc.printf("%v", err)
				break
			}
			pages <- edition
		}
		// This edition is not present in the normal callback
		pages <- mcEdition{
			Id:  mcPromoEditionId,
			Set: "Promo",
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			mc.printf("%v", result.err)
			continue
		}
		for _, card := range result.cards {
			err = mtgban.InventoryAdd(mc.inventory, card)
			if err != nil {
				mc.printf(err.Error())
				continue
			}
		}
	}

	mc.InventoryDate = time.Now()

	return nil
}

func (mc *Magiccorner) Inventory() (map[string][]mtgban.InventoryEntry, error) {
	if len(mc.inventory) > 0 {
		return mc.inventory, nil
	}

	mc.printf("Empty inventory, scraping started")

	err := mc.scrape()
	if err != nil {
		return nil, err
	}

	return mc.inventory, nil

}

func (mc *Magiccorner) Info() (info mtgban.ScraperInfo) {
	info.Name = "Magic Corner"
	info.Shorthand = "MC"
	return
}
