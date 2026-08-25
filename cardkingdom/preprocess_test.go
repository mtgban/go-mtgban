package cardkingdom

import (
	"encoding/json"
	"errors"
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

// TestPreprocessListAngelToken pins the sku fixup for the one Angel token
// The List carries twice. Its sku names the Forgotten Realms printing and
// its bare number reaches the Guilds of Ravnica one, which is the wrong
// card at the right number.
func TestPreprocessListAngelToken(t *testing.T) {
	theCard, err := Preprocess(cardkingdom.Product{
		SKU:     "MTAFR-001",
		Name:    "Angel Token // Spirit Token",
		Edition: "Mystery Booster/The List",
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	cardID, err := mtgmatcher.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	const want = "ba22fdaf-8d82-5f14-a5f0-3e5908f04d8c"
	if cardID != want {
		co, _ := mtgmatcher.GetUUID(cardID)
		t.Errorf("Match(%v) = %s (%v), want the Forgotten Realms Angel", theCard, cardID, co)
	}
}

// TestPreprocessEmblems pins the three shapes CK spells an emblem in, each
// against the uuid the datastore files at the address the sku names: the
// planeswalker left in the variation, the planeswalker abbreviated into a
// parenthetical on a card shared with another token, and the Mythic Edition
// numbering that diverges from mtgjson's so only the name can carry the row.
func TestPreprocessEmblems(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		product cardkingdom.Product
		name    string
		uuid    string
	}{
		{
			desc: "the planeswalker rides in the variation",
			product: cardkingdom.Product{
				SKU:       "TDKA-003",
				Name:      "Emblem",
				Edition:   "Dark Ascension",
				Variation: "Sorin, Lord of Innistrad",
			},
			name: "Sorin, Lord of Innistrad Emblem",
			uuid: "1d0792f5-6ed6-5385-9a96-87b472909d1c",
		},
		{
			desc: "a parenthetical short name against the full one",
			product: cardkingdom.Product{
				SKU:     "TC14-035",
				Name:    "Emblem (Nixilis) - Zombie (Black) Token",
				Edition: "Commander 2014",
			},
			name: "Ob Nixilis of the Black Oath Emblem",
			uuid: "5f895769-4c30-5d4e-a7ad-51ebb7cbf63e",
		},
		{
			// CK numbers this G6 printing 006A, so the number cannot
			// reach it and the respelled name has to.
			desc: "a number the set does not carry",
			product: cardkingdom.Product{
				SKU:       "TMED-006A",
				Name:      "Emblem",
				Edition:   "Masterpiece Series: Mythic Edition",
				Variation: "Ral",
			},
			name: "Ral, Izzet Viceroy Emblem",
			uuid: "2bcf14e3-ff8f-58b7-acbd-9f96dd49ee6d",
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
			if cardID != tt.uuid {
				co, _ := mtgmatcher.GetUUID(cardID)
				t.Errorf("Match(%v) = %s (%v), want %s", theCard, cardID, co, tt.uuid)
			}
		})
	}
}

// TestPreprocessSplitCard pins the split cards a T-prefixed set code used to
// sweep into the double-faced token split, which renamed them after their
// first face and lost the row.
func TestPreprocessSplitCard(t *testing.T) {
	theCard, err := Preprocess(cardkingdom.Product{
		SKU:     "TSR-186",
		Name:    "Rough // Tumble",
		Edition: "Time Spiral Remastered",
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	if theCard.Name != "Rough // Tumble" {
		t.Errorf("Preprocess name = %q, want the whole split card", theCard.Name)
	}
	cardID, err := mtgmatcher.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	const want = "609b3e64-4e46-595c-a99d-bcbb04691d4f"
	if cardID != want {
		co, _ := mtgmatcher.GetUUID(cardID)
		t.Errorf("Match(%v) = %s (%v), want the Time Spiral Remastered split card", theCard, cardID, co)
	}
}

func TestPreprocessTokenFoilRefused(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		product cardkingdom.Product
	}{
		{
			// The sheet holds one nonfoil emblem, so the foil row would
			// be served as the price of the plain one
			desc: "a foil-wrapped code whose sheet was never sold foil",
			product: cardkingdom.Product{
				SKU:     "FTNEO-019",
				Name:    "Tezzeret, Betrayer of Flesh Emblem",
				Edition: "Kamigawa: Neon Dynasty",
				IsFoil:  true,
			},
		},
		{
			desc: "the same for a surge-foil wrapping",
			product: cardkingdom.Product{
				SKU:     "SFT40K-011",
				Name:    "Arco-Flagellant Token // Soldier Token",
				Edition: "Warhammer 40,000",
				IsFoil:  true,
			},
		},
		{
			// No real card is named Astartes Warrior, so the sheet files
			// the token without the suffix the row spells it with
			desc: "a token the datastore files without a Token suffix",
			product: cardkingdom.Product{
				SKU:     "SFT40K-012",
				Name:    "Astartes Warrior Token // Spawn Token",
				Edition: "Warhammer 40,000",
				IsFoil:  true,
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			theCard, err := Preprocess(tt.product)
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
				t.Errorf("Preprocess(%v) = %v, %v, want ErrUnsupported", tt.product, theCard, err)
			}
		})
	}
}
