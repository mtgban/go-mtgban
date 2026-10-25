package sealed_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/matcher/sealed"
)

// languageBackend holds one English deck shared by an English and a Japanese
// product, the shape of the Junji Ito Secret Lair: the deck lists English
// cards, and only the Japanese product names a language. Of the Japanese
// printings foreignData names, the datastore loads ja-1 and ja-3_f, the copy
// of the ★ alternative foil en-3_f. The deck names en-3_f directly, where a
// datastore would reach it from en-3 by finish.
func languageBackend(t *testing.T) *matcher.Backend {
	t.Helper()

	var set matcher.Set
	err := json.Unmarshal([]byte(`{
		"code": "SLD",
		"decks": [{"name": "Special Guest", "mainBoard": [
			{"uuid": "en-1", "count": 1}, {"uuid": "en-2", "count": 1},
			{"uuid": "en-3_f", "count": 1, "isFoil": true}]}],
		"sealedProduct": [
			{"uuid": "english-product", "contents": {"deck": [{"name": "Special Guest", "set": "sld"}]}},
			{"uuid": "japanese-product", "language": "Japanese",
			 "contents": {"deck": [{"name": "Special Guest", "set": "sld"}]}}
		]}`), &set)
	if err != nil {
		t.Fatal(err)
	}

	var cards []matcher.Card
	err = json.Unmarshal([]byte(`[
		{"uuid": "en-1", "language": "English", "foreignData": [{"language": "Japanese", "uuid": "ja-1"}]},
		{"uuid": "en-2", "language": "English", "foreignData": [{"language": "Japanese", "uuid": "ja-2"}]},
		{"uuid": "ja-1", "language": "Japanese"},
		{"uuid": "en-3_f", "language": "English", "foreignData": [{"language": "Japanese", "uuid": "ja-3_f"}]},
		{"uuid": "ja-3_f", "language": "Japanese"}
	]`), &cards)
	if err != nil {
		t.Fatal(err)
	}

	foils := []string{"en-3_f", "ja-3_f"}

	b := &matcher.Backend{
		Sets:  map[string]*matcher.Set{"SLD": &set},
		UUIDs: map[string]*matcher.CardObject{},
	}
	for _, card := range cards {
		b.UUIDs[card.UUID] = &matcher.CardObject{Card: card, Foil: slices.Contains(foils, card.UUID)}
	}
	return b
}

// A card is taken in each language asked for that the datastore loads, and
// one loaded in none of them stays the printing the deck lists.
func TestDeckCardsInLanguages(t *testing.T) {
	b := languageBackend(t)

	for _, tt := range []struct {
		languages []string
		want      []string
	}{
		{nil, []string{"en-1", "en-2", "en-3_f"}},
		{[]string{"Japanese"}, []string{"ja-1", "en-2", "ja-3_f"}},
		{[]string{"English", "Japanese"}, []string{"en-1", "ja-1", "en-2", "en-3_f", "ja-3_f"}},
		{[]string{"Korean"}, []string{"en-1", "en-2", "en-3_f"}},
	} {
		got, err := sealed.DeckCards(b, "SLD", "Special Guest", tt.languages...)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("DeckCards(%q) = %v, want %v", tt.languages, got, tt.want)
		}
	}
}

// Each product opens the shared deck in the language it names.
func TestProductOpensItsDecksInItsLanguage(t *testing.T) {
	b := languageBackend(t)

	for product, want := range map[string][]string{
		"english-product":  {"en-1", "en-2", "en-3_f"},
		"japanese-product": {"ja-1", "en-2", "ja-3_f"},
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
