package cardkingdom

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/mtgban/go-cardkingdom"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestMain(m *testing.M) {
	allprintingsPath := os.Getenv("ALLPRINTINGS5_PATH")
	if allprintingsPath == "" {
		log.Fatalln("Need ALLPRINTINGS5_PATH variable set to run tests")
	}

	allPrintingsReader, err := os.Open(allprintingsPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer allPrintingsReader.Close()

	err = mtgmatcher.LoadDatastore(allPrintingsReader)
	if err != nil {
		log.Fatalln(err)
	}

	mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0))

	os.Exit(m.Run())
}

var PriceListTest string = `
[
    {
      "id": 320768,
      "sku": "PTLE-0305",
      "scryfall_id": null,
      "url": "mtg/promotional/enlightened-tutor-commanders-bundle-promo",
      "name": "Enlightened Tutor",
      "variation": "Commander's Bundle Promo",
      "edition": "Promotional",
      "is_foil": "false"
    }
]
`

var priceListResults = []string{
	"3b00adaa-962a-5316-9b9d-e12e0284f87f",
	"b30a3061-ce20-54eb-b25c-7520aa76f8b7",
}

func TestPreprocess(t *testing.T) {
	var products []cardkingdom.Product
	err := json.NewDecoder(strings.NewReader(PriceListTest)).Decode(&products)
	if err != nil {
		t.Errorf("FAIL: cannot umarshal products: %s", err)
		return
	}

	for i, product := range products {
		test := product
		idx := i
		t.Run(fmt.Sprint(test.Name), func(t *testing.T) {
			t.Parallel()

			theCard, err := Preprocess(test)
			if err != nil {
				t.Errorf("FAIL: unxpected Preprocess error: %s", err)
				return
			}

			cardId, err := mtgmatcher.Match(theCard)
			if err != nil {
				t.Errorf("FAIL: unxpected Match error: %s", err)
				return
			}

			if cardId != priceListResults[idx] {
				co, _ := mtgmatcher.GetUUID(cardId)
				t.Errorf("FAIL %s: Expected '%s' got '%s' (%s)", test.Name, priceListResults[idx], cardId, co)
				return
			}
			t.Log("PASS:", product.Name)
		})
	}
}
