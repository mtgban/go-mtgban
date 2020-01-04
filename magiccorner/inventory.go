package magiccorner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	http "github.com/hashicorp/go-retryablehttp"
	"github.com/kodabb/go-mtgban/mtgban"
)

type mcJSON struct {
	Data []struct {
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

var l = log.New(os.Stderr, "", 0)

const (
	mcReinassanceId       = 73
	mcRevisedEUFBBId      = 1041
	mcPromoEditionId      = 1113
	mcMerfolksVsGoblinsId = 1116

	maxConcurrency = 8

	mcNumberNotAvailable = "n/a"
	mcBaseURL            = "https://www.magiccorner.it/12/modules/store/mcpub.asmx/"
	mcEditionsEndpt      = "espansioni"
	mcCardsEndpt         = "carte"
)

type Magiccorner struct {
	LogCallback mtgban.LogCallbackFunc
	inventory   []mtgban.Entry
}

func NewVendor() mtgban.Scraper {
	mc := Magiccorner{}
	return &mc
}

type resultChan struct {
	err   error
	cards []MCCard
}

func (mc *Magiccorner) printf(format string, a ...interface{}) {
	if mc.LogCallback != nil {
		mc.LogCallback(format, a...)
	}
}

func (mc *Magiccorner) processEntry(edition mcEdition) (res resultChan) {
	// This breaks on the main website too
	if edition.Id == mcMerfolksVsGoblinsId {
		return
	}

	langCode := 0
	if edition.Id == mcPromoEditionId {
		langCode = 72
	}
	param := mcParam{
		// The last field before || is the language
		// 0 - any language, 72 - english only
		SearchField: fmt.Sprintf("%d|0|0|0|0|%d||true|0|", edition.Id, langCode),

		// The edition/category id
		IdCategory: fmt.Sprintf("%d", edition.Id),

		// No idea what these fields are for
		UIc:           "it",
		OnlyAvailable: true,
		IsBuy:         false,
	}
	reqBody, _ := json.Marshal(&param)

	resp, err := http.Post(mcBaseURL+mcCardsEndpt, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		res.err = err
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		res.err = err
		return
	}
	defer resp.Body.Close()

	var db mcJSON
	err = json.Unmarshal(data, &db)
	if err != nil {
		res.err = err
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
			mc.printf("Processing id %d - %s (%s, code: %s)...\n", edition.Id, edition.Set, extra, card.Code)
			printed = true
		}

		// Trust the collector number for a few selected cases
		// Fixup set code as needed
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
		case "UST", "E01", "RNA", "GRN", "ELD":
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

		for _, v := range card.Variants {
			// Skip duplicate cards
			if duplicate[v.Id] {
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

			isFoil := v.Foil == "Foil"

			cc := MCCard{
				Name: card.Name,
				Set:  card.Set,
				Foil: isFoil,

				Pricing:   v.Price,
				Qty:       v.Quantity,
				Condition: v.Condition,

				Id:     v.Id,
				Number: number,

				setCode: card.Code,
				extra:   extra,
				orig:    card.OrigName,
			}

			res.cards = append(res.cards, cc)

			duplicate[v.Id] = true
		}
	}

	return
}

// Scrape returns an array of Entry, containing pricing and card information
func (mc *Magiccorner) Scrape() ([]mtgban.Entry, error) {
	// Retrieve the edition ids
	resp, err := http.Post(mcBaseURL+mcEditionsEndpt, "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var blob mcBlob
	err = json.Unmarshal(data, &blob)
	if err != nil {
		return nil, err
	}

	// There is json in this json
	dec := json.NewDecoder(strings.NewReader(blob.Data))
	_, err = dec.Token()
	if err != nil {
		return nil, err
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
		// This edition is not present in the normal callback
		pages <- mcEdition{
			Id:  mcPromoEditionId,
			Set: "Promo",
		}
		for dec.More() {
			var edition mcEdition
			err := dec.Decode(&edition)
			if err != nil {
				l.Println(err)
				break
			}
			//if false {
			//if edition.Set == "Trono di Eldraine" {
			if edition.Set != "Theros: Oltre la Morte" && edition.Set != "Fallen Empires" && edition.Set != "Rinascimento" {
				pages <- edition
			}
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
		for i := range result.cards {
			mc.inventory = append(mc.inventory, &result.cards[i])
		}
	}

	return mc.inventory, nil
}
