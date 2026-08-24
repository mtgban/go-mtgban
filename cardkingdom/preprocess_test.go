package cardkingdom

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/mtgban/go-cardkingdom"
	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

func TestMain(m *testing.M) {
	allprintingsPath := os.Getenv("ALLPRINTINGS5_PATH")
	if allprintingsPath == "" {
		log.Fatalln("Need ALLPRINTINGS5_PATH variable set to run tests")
	}

	allPrintingsReader, err := datastore.Open(allprintingsPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer allPrintingsReader.Close()

	ds, err := magic.Load(allPrintingsReader)
	if err != nil {
		log.Fatalln(err)
	}
	mtgmatcher.SetGlobalDatastore(ds)

	mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0))

	os.Exit(m.Run())
}

var PriceListTest = `
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

			cardID, err := mtgmatcher.Match(theCard)
			if err != nil {
				t.Errorf("FAIL: unxpected Match error: %s", err)
				return
			}

			if cardID != priceListResults[idx] {
				co, _ := mtgmatcher.GetUUID(cardID)
				t.Errorf("FAIL %s: Expected '%s' got '%s' (%s)", test.Name, priceListResults[idx], cardID, co)
				return
			}
			t.Log("PASS:", product.Name)
		})
	}
}

// TestPreprocessTokens pins the two token paths a silent revert would take
// back to the old behavior: the double-faced split must not double the
// " Token" suffix the kept face already carries, and a sku code the
// datastore does not carry must reach the filing set once its treatment
// wrapping is stripped.
func TestPreprocessTokens(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		product cardkingdom.Product
		name    string
		setCode string
	}{
		{
			desc: "a dfc split keeps one Token suffix",
			product: cardkingdom.Product{
				SKU:     "TMID-0001",
				Name:    "Human Token // Wolf Token",
				Edition: "Innistrad: Midnight Hunt Tokens",
			},
			name:    "Human Token",
			setCode: "TMID",
		},
		{
			// The surge-foil wrapping is one the older per-treatment
			// strips never knew, so only the wrapping loop reaches it.
			desc: "a surge-foil-wrapped token code reaches the filing set",
			product: cardkingdom.Product{
				SKU:     "SFTWHO-0034",
				Name:    "Alien Token",
				Edition: "Doctor Who",
				IsFoil:  true,
			},
			name:    "Alien Token",
			setCode: "TWHO",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			theCard, err := Preprocess(tt.product)
			if err != nil {
				t.Fatalf("Preprocess(%v) = %v", tt.product, err)
			}
			if theCard.Name != tt.name {
				t.Errorf("Preprocess name = %q, want %q", theCard.Name, tt.name)
			}
			cardID, err := mtgmatcher.Match(theCard)
			if err != nil {
				t.Fatalf("Match(%v) = %v", theCard, err)
			}
			co, err := mtgmatcher.GetUUID(cardID)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", cardID, err)
			}
			if co.SetCode != tt.setCode {
				t.Errorf("Match(%v) = %s (%v), want a %s printing", theCard, cardID, co, tt.setCode)
			}
		})
	}
}
