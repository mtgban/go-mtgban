package mtgmatcher

import (
	"slices"
	"testing"
)

// A deck is shared verbatim by every sealed product that references it by
// name, so its own uuids are always the deck's default-language (English)
// ones. GetPicksForDeck substitutes a card's own foreignData uuid for the
// requesting product's language, but only when that foreign printing is one
// this datastore already has loaded under its own uuid - it never invents
// one that was never minted.
func deckLanguageBackend() *Backend {
	b := &Backend{
		UUIDs: map[string]*CardObject{
			"en-1": {Card: Card{
				UUID: "en-1", Name: "Carrion Feeder",
				ForeignData: []struct {
					Name        string            `json:"name"`
					Language    string            `json:"language"`
					Identifiers map[string]string `json:"identifiers"`
					Type        string            `json:"type"`
					UUID        string            `json:"uuid"`
				}{
					{Name: "カリオンフィーダー", Language: "Japanese", UUID: "ja-1"},
				},
			}},
			// en-2's Japanese foreignData entry points at a uuid this
			// datastore never loaded - e.g. Scryfall has no transcription
			// for it yet, or this build simply does not carry it.
			"en-2": {Card: Card{
				UUID: "en-2", Name: "Some Other Card",
				ForeignData: []struct {
					Name        string            `json:"name"`
					Language    string            `json:"language"`
					Identifiers map[string]string `json:"identifiers"`
					Type        string            `json:"type"`
					UUID        string            `json:"uuid"`
				}{
					{Name: "何か", Language: "Japanese", UUID: "ja-2-not-loaded"},
				},
			}},
			// en-3 carries no foreignData at all.
			"en-3": {Card: Card{UUID: "en-3", Name: "Plain Card"}},
			"ja-1": {Card: Card{UUID: "ja-1", Name: "カリオンフィーダー"}},
		},
		Sets: map[string]*Set{
			"SLD": {
				Code: "SLD",
				Decks: []struct {
					Code               string     `json:"code"`
					Commander          []DeckCard `json:"commander"`
					MainBoard          []DeckCard `json:"mainBoard"`
					DisplayCommander   []DeckCard `json:"displayCommander"`
					Planes             []DeckCard `json:"planes"`
					Schemes            []DeckCard `json:"schemes"`
					SideBoard          []DeckCard `json:"sideBoard"`
					Tokens             []DeckCard `json:"tokens"`
					Name               string     `json:"name"`
					SealedProductUUIDs []string   `json:"sealedProductUuids"`
				}{
					{
						Name: "Test Deck",
						MainBoard: []DeckCard{
							{UUID: "en-1", Count: 1},
							{UUID: "en-2", Count: 1},
							{UUID: "en-3", Count: 1},
						},
					},
				},
			},
		},
	}
	return b
}

func TestGetPicksForDeckWithoutLanguageKeepsDefaultUUIDs(t *testing.T) {
	b := deckLanguageBackend()

	picks, err := b.GetPicksForDeck("SLD", "Test Deck", "")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"en-1", "en-2", "en-3"}
	slices.Sort(picks)
	slices.Sort(want)
	if !slices.Equal(picks, want) {
		t.Errorf("GetPicksForDeck(..., \"\") = %v, want %v", picks, want)
	}
}

func TestGetPicksForDeckSubstitutesLoadedForeignUUIDs(t *testing.T) {
	b := deckLanguageBackend()

	picks, err := b.GetPicksForDeck("SLD", "Test Deck", "Japanese")
	if err != nil {
		t.Fatal(err)
	}

	// en-1 has a loaded Japanese printing and is substituted; en-2's
	// Japanese printing is not loaded, so it falls back to English; en-3
	// has no Japanese printing at all and is untouched.
	want := []string{"ja-1", "en-2", "en-3"}
	slices.Sort(picks)
	slices.Sort(want)
	if !slices.Equal(picks, want) {
		t.Errorf("GetPicksForDeck(..., \"Japanese\") = %v, want %v", picks, want)
	}
}

// The exact Junji Ito English/Japanese scenario: two products share the same
// deck object, and each must resolve it to its own language's uuids.
func TestGetPicksForDeckSameDeckDiffersByRequestedLanguage(t *testing.T) {
	b := deckLanguageBackend()

	english, err := b.GetPicksForDeck("SLD", "Test Deck", "")
	if err != nil {
		t.Fatal(err)
	}
	japanese, err := b.GetPicksForDeck("SLD", "Test Deck", "Japanese")
	if err != nil {
		t.Fatal(err)
	}

	if slices.Equal(english, japanese) {
		t.Fatalf("English and Japanese picks must differ, both were %v", english)
	}
	if !slices.Contains(english, "en-1") || slices.Contains(english, "ja-1") {
		t.Errorf("English picks = %v, want en-1 and not ja-1", english)
	}
	if !slices.Contains(japanese, "ja-1") || slices.Contains(japanese, "en-1") {
		t.Errorf("Japanese picks = %v, want ja-1 and not en-1", japanese)
	}
}

func TestForeignUUIDReturnsEmptyWhenNotLoaded(t *testing.T) {
	b := deckLanguageBackend()

	if got := b.foreignUUID("en-2", "Japanese"); got != "" {
		t.Errorf("foreignUUID(en-2, Japanese) = %q, want empty (ja-2-not-loaded is not in UUIDs)", got)
	}
}

func TestForeignUUIDReturnsEmptyWhenNoSuchLanguage(t *testing.T) {
	b := deckLanguageBackend()

	if got := b.foreignUUID("en-1", "Klingon"); got != "" {
		t.Errorf("foreignUUID(en-1, Klingon) = %q, want empty", got)
	}
}

func TestForeignUUIDReturnsEmptyForUnknownCard(t *testing.T) {
	b := deckLanguageBackend()

	if got := b.foreignUUID("does-not-exist", "Japanese"); got != "" {
		t.Errorf("foreignUUID(does-not-exist, Japanese) = %q, want empty", got)
	}
}

func TestForeignUUIDResolvesALoadedPrinting(t *testing.T) {
	b := deckLanguageBackend()

	if got := b.foreignUUID("en-1", "Japanese"); got != "ja-1" {
		t.Errorf("foreignUUID(en-1, Japanese) = %q, want ja-1", got)
	}
}
