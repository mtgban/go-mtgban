package sealed_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/sealed"
)

// languageBackend holds one English deck shared by an English and a Japanese
// product, the shape of the Junji Ito Secret Lair: the deck lists English
// cards, and mtgjson names the Japanese product only on the Japanese printing.
func languageBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()

	var set mtgmatcher.Set
	err := json.Unmarshal([]byte(`{
		"code": "SLD",
		"decks": [{"name": "Special Guest", "mainBoard": [
			{"uuid": "en-1", "count": 1}, {"uuid": "en-2", "count": 1}]}],
		"sealedProduct": [
			{"uuid": "english-product", "contents": {"deck": [{"name": "Special Guest", "set": "sld"}]}},
			{"uuid": "japanese-product", "contents": {"deck": [{"name": "Special Guest", "set": "sld"}]}}
		]}`), &set)
	if err != nil {
		t.Fatal(err)
	}

	var cards []mtgmatcher.Card
	err = json.Unmarshal([]byte(`[
		{"uuid": "en-1", "sourceProducts": {"nonfoil": ["english-product"]},
		 "foreignData": [{"language": "Japanese", "uuid": "ja-1",
		                  "sourceProducts": {"nonfoil": ["japanese-product"]}}]},
		{"uuid": "en-2", "sourceProducts": {"nonfoil": ["english-product"]},
		 "foreignData": [{"language": "Japanese", "uuid": "ja-2",
		                  "sourceProducts": {"nonfoil": ["japanese-product"]}}]},
		{"uuid": "ja-1", "language": "Japanese", "sourceProducts": {"nonfoil": ["japanese-product"]}}
	]`), &cards)
	if err != nil {
		t.Fatal(err)
	}

	b := &mtgmatcher.Backend{
		Sets:  map[string]*mtgmatcher.Set{"SLD": &set},
		UUIDs: map[string]*mtgmatcher.CardObject{},
	}
	for _, card := range cards {
		b.UUIDs[card.UUID] = &mtgmatcher.CardObject{Card: card}
	}
	return b
}

// Each product answers with the printings it holds. ja-2 is not loaded, so
// the Japanese product keeps the English card the deck lists.
func TestDeckResolvesToTheProductsPrinting(t *testing.T) {
	b := languageBackend(t)

	for product, want := range map[string][]string{
		"english-product":  {"en-1", "en-2"},
		"japanese-product": {"ja-1", "en-2"},
	} {
		decklist, err := sealed.ProductDecklist(b, "SLD", product)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(decklist, want) {
			t.Errorf("%s decklist %v, want %v", product, decklist, want)
		}

		picks, err := sealed.ProductPicks(b, "SLD", product)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(picks, want) {
			t.Errorf("%s picks %v, want %v", product, picks, want)
		}

		counts, err := sealed.ProductCounts(b, "SLD", product)
		if err != nil {
			t.Fatal(err)
		}
		var counted []string
		for _, count := range counts {
			counted = append(counted, count.UUID)
		}
		if !slices.Equal(counted, want) {
			t.Errorf("%s counts %v, want %v", product, counted, want)
		}
	}
}
