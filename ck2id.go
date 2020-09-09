package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/kodabb/go-mtgmatcher/cklite"
	"github.com/kodabb/go-mtgmatcher/mtgmatcher"
)

type meta struct {
	Id  int    `json:"id,omitempty"`
	URL string `json:"url,omitempty"`
}

type ck2id struct {
	Normal *meta `json:"normal,omitempty"`
	Foil   *meta `json:"foil,omitempty"`
}

var UseOnline bool = false

func run() int {
	logger := log.New(os.Stderr, "", 0)

	allPrintingsReader, err := os.Open("allprintings.json")
	if err != nil {
		logger.Println(err)
		return 1
	}
	defer allPrintingsReader.Close()

	err = mtgmatcher.LoadDatastore(allPrintingsReader)
	if err != nil {
		logger.Println(err)
		return 1
	}

	var list *cklite.CKPriceList
	if UseOnline {
		list, err = cklite.GetPriceList()
		if err != nil {
			logger.Println(err)
			return 1
		}
	} else {
		pricelistReader, err := os.Open("pricelist.json")
		if err != nil {
			logger.Println(err)
			return 1
		}
		defer pricelistReader.Close()

		err = json.NewDecoder(pricelistReader).Decode(&list)
		if err != nil {
			logger.Println(err)
			return 1
		}
	}

	output := map[string]*ck2id{}

	for _, card := range list.Data {
		theCard, err := cklite.Preprocess(card)
		if err != nil {
			continue
		}

		cc, err := mtgmatcher.Match(theCard)
		if err != nil {
			logger.Println(err)
			logger.Println(theCard)
			logger.Println(card)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.Unmatch(probe)
					logger.Println("-", card)
				}
			}
			continue
		}

		id := strings.TrimSuffix(cc, "_f")
		_, found := output[id]
		if !found {
			output[id] = &ck2id{}
		}
		if !strings.HasSuffix(cc, "_f") {
			output[id].Normal = &meta{}
			output[id].Normal.Id = card.Id
			output[id].Normal.URL = "https://www.cardkingdom.com/" + card.URL
		} else {
			output[id].Foil = &meta{}
			output[id].Foil.Id = card.Id
			output[id].Foil.URL = "https://www.cardkingdom.com/" + card.URL
		}
	}

	enc := json.NewEncoder(os.Stdout)
	err = enc.Encode(output)
	if err != nil {
		logger.Println(err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run())
}
