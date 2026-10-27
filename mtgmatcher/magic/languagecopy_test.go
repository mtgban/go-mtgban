package magic

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A language copy goes by the uuid mtgjson gives that printing and holds the
// products mtgjson lists for it, leaving its English original untouched.
func TestLanguageCopyTakesMtgjsonPrinting(t *testing.T) {
	var cards []Card
	err := json.Unmarshal([]byte(`[
		{"name": "Carrion Feeder", "number": "1114", "uuid": "en-1114",
		 "identifiers": {"scryfallId": "sf-en-1114", "mtgjsonId": "en-1114", "tcgplayerProductId": "1114", "mcmId": "1114"},
		 "sourceProducts": {"nonfoil": ["english-product"]},
		 "foreignData": [{"language": "Japanese", "name": "屍肉喰らい", "uuid": "ja-1114",
		                  "identifiers": {"scryfallId": "sf-ja-1114", "multiverseId": "mv-ja-1114"},
		                  "sourceProducts": {"nonfoil": ["japanese-product"]}}]},
		{"name": "Carrion Feeder", "number": "1114★", "uuid": "en-1114_f",
		 "identifiers": {"scryfallId": "sf-en-1114", "mtgjsonId": "en-1114"},
		 "sourceProducts": {"nonfoil": ["english-product"]}},
		{"name": "Doomsday", "number": "1115", "uuid": "en-1115",
		 "identifiers": {"scryfallId": "sf-en-1115"},
		 "sourceProducts": {"nonfoil": ["english-product"]},
		 "foreignData": [{"language": "Japanese", "name": "終末", "uuid": "ja-1115",
		                  "identifiers": {"scryfallId": "sf-ja-1115"}}]},
		{"name": "Thoughtseize", "number": "1117", "uuid": "en-1117",
		 "identifiers": {"scryfallId": "sf-en-1117"},
		 "sourceProducts": {"nonfoil": ["english-product"]}}
	]`), &cards)
	if err != nil {
		t.Fatal(err)
	}
	// the alternative foil split off 1114 shares its foreignData, as the loader leaves it
	cards[1].ForeignData = cards[0].ForeignData
	cards[0].Variations = []string{"en-1114_f"}
	cards[1].Variations = []string{"en-1114"}
	sets := map[string]*Set{"SLD": {Code: "SLD", Cards: cards}}

	copies := duplicateCards(sets, "SLD", "JPN", []string{"1114", "1115", "1117"})

	want := []struct {
		uuid, product string
	}{
		{"ja-1114", "japanese-product"},
		// the alternative foil keeps the suffix telling it apart rather than
		// colliding with 1114's copy
		{"ja-1114_f", "japanese-product"},
		// mtgjson lists no product for this printing yet: the original's stand
		{"ja-1115", "english-product"},
		// no Japanese printing in mtgjson at all: the copy is named after its original
		{"en-1117_jpn", "english-product"},
	}
	if len(copies) != len(want) {
		t.Fatalf("got %d copies, want %d", len(copies), len(want))
	}
	for i, w := range want {
		if copies[i].UUID != w.uuid {
			t.Errorf("copy of %s has uuid %s, want %s", copies[i].Number, copies[i].UUID, w.uuid)
		}
		if got := copies[i].SourceProducts["nonfoil"]; !slices.Equal(got, []string{w.product}) {
			t.Errorf("copy %s holds %v, want %s", w.uuid, got, w.product)
		}
	}

	for _, original := range sets["SLD"].Cards {
		if id, found := original.Identifiers["originalScryfallId"]; found {
			t.Errorf("English %s took its copy's scryfall id %s", original.Number, id)
		}
		if !slices.Equal(original.SourceProducts["nonfoil"], []string{"english-product"}) {
			t.Errorf("English %s holds %v", original.Number, original.SourceProducts)
		}
	}
	if copies[0].Identifiers["originalScryfallId"] != "sf-ja-1114" {
		t.Errorf("copy's image id is %q", copies[0].Identifiers["originalScryfallId"])
	}
	if copies[0].Identifiers["scryfallId"] != "sf-ja-1114" {
		t.Errorf("copy's scryfall id is %q", copies[0].Identifiers["scryfallId"])
	}
	if copies[0].Identifiers["multiverseId"] != "mv-ja-1114" {
		t.Errorf("copy's multiverse id is %q", copies[0].Identifiers["multiverseId"])
	}
	// TCGplayer files the skus of every language under the original's uuid
	if copies[0].Identifiers["mtgjsonId"] != "en-1114" {
		t.Errorf("copy's mtgjson id is %q", copies[0].Identifiers["mtgjsonId"])
	}
	// TCGplayer and Cardmarket sell the copy under its original's product
	if copies[0].Identifiers["tcgplayerProductId"] != "1114" || copies[0].Identifiers["mcmId"] != "1114" {
		t.Errorf("copy's product ids are %q and %q", copies[0].Identifiers["tcgplayerProductId"], copies[0].Identifiers["mcmId"])
	}
	// a copy mtgjson knows nothing of keeps its original's ids
	if copies[3].Identifiers["scryfallId"] != "sf-en-1117" {
		t.Errorf("copy %s has scryfall id %q", copies[3].UUID, copies[3].Identifiers["scryfallId"])
	}
	// the copies are each other's finish twins, as their originals are
	if !slices.Equal(copies[0].Variations, []string{"ja-1114_f"}) || !slices.Equal(copies[1].Variations, []string{"ja-1114"}) {
		t.Errorf("copies' variations are %v and %v", copies[0].Variations, copies[1].Variations)
	}
	if !slices.Equal(cards[0].Variations, []string{"en-1114_f"}) {
		t.Errorf("English 1114's variations became %v", cards[0].Variations)
	}

	// each original's Japanese entry names the copy minted from it
	for i, want := range []string{"ja-1114", "ja-1114_f", "ja-1115"} {
		original := sets["SLD"].Cards[i]
		if got := original.ForeignData[0].UUID; got != want {
			t.Errorf("English %s names Japanese %s, want %s", original.Number, got, want)
		}
	}
}

// A Secret Lair number listed for a Japanese copy brings its starred sibling.
func TestSLDJapaneseCopies(t *testing.T) {
	realDatastore(t)

	for _, uuid := range testBackend.SetUUIDs["SLD"] {
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.Number == "1858★jpn" {
			return
		}
	}
	t.Error("SLD 1858★ has no Japanese copy")
}

// Cards sharing a name list a duplicated set once, however many of them
// the set holds, and each of them is copied.
func TestDuplicateListsItsCodeOnce(t *testing.T) {
	day := time.Date(1995, 4, 1, 0, 0, 0, 0, time.UTC)
	plains := func(number string, printings ...string) Card {
		return Card{Name: "Plains", Number: number, UUID: "plains-" + number, Printings: printings,
			Language: "English", Identifiers: map[string]string{"tcgplayerProductId": "tcg-" + number, "mcmId": "mcm-" + number}}
	}
	sets := map[string]*Set{
		"LEA": {Code: "LEA", Name: "Alpha", ReleaseDateTime: day.AddDate(-2, 0, 0), Cards: []Card{plains("a1", "4ED", "LEA")}},
		"4ED": {Code: "4ED", Name: "Fourth Edition", ReleaseDateTime: day, Cards: []Card{
			plains("1", "4ED", "LEA"), plains("2", "4ED", "LEA"), plains("3", "4ED", "LEA"),
		}},
	}

	duplicate(sets, "Alternate Fourth Edition", "4ED", "ALT", "1995-04-01")

	for _, code := range []string{"LEA", "4ED"} {
		for _, card := range sets[code].Cards {
			want := []string{"4EDALT", "4ED", "LEA"}
			if !slices.Equal(card.Printings, want) {
				t.Errorf("%s %s printings %v, want %v", code, card.Number, card.Printings, want)
			}
		}
	}
	if len(sets["4EDALT"].Cards) != 3 {
		t.Errorf("4EDALT holds %d cards, want 3", len(sets["4EDALT"].Cards))
	}
	// a copy in its original's language is not sold under the original's products
	for i, card := range sets["4EDALT"].Cards {
		for _, tag := range []string{"tcgplayerProductId", "mcmId"} {
			if id, found := card.Identifiers[tag]; found {
				t.Errorf("4EDALT %s took %s %s", card.Number, tag, id)
			}
			if sets["4ED"].Cards[i].Identifiers[tag] == "" {
				t.Errorf("4ED %s lost its %s", sets["4ED"].Cards[i].Number, tag)
			}
		}
	}
}

// A language copy's TCGplayer ids and the ids it shares with its English
// original lead to English printings, as does every id an English printing
// carries, and an id of the copy's own leads to it.
func TestLanguageCopyIdentifiers(t *testing.T) {
	realDatastore(t)

	for _, filing := range []struct {
		tag   string
		space mtgmatcher.IDSpace
	}{
		{"mtgjsonId", mtgmatcher.IDSpaceMTGJSON},
		{"scryfallId", mtgmatcher.IDSpaceScryfall},
		{"tcgplayerProductId", mtgmatcher.IDSpaceTCGplayer},
		{"multiverseId", mtgmatcher.IDSpaceMultiverse},
	} {
		for _, uuid := range testBackend.AllUUIDs {
			co, err := testBackend.GetUUID(uuid)
			if err != nil {
				t.Fatal(err)
			}
			id := co.Identifiers[filing.tag]
			if id == "" {
				continue
			}
			want := co.Language
			if co.Language != "English" {
				source, err := testBackend.GetUUID(co.Identifiers["mtgjsonId"])
				if err != nil || source.Language != "English" {
					continue
				}
				if filing.space == mtgmatcher.IDSpaceTCGplayer || source.Identifiers[filing.tag] == id {
					want = "English"
				}
			}
			holder, err := testBackend.GetUUID(testBackend.ConvertID(filing.space, id))
			if err != nil {
				t.Fatal(err)
			}
			if holder.Language != want {
				t.Errorf("%s %s of %s %s #%s converts to %s %s #%s", filing.space, id,
					co.Language, co.SetCode, co.Number, holder.Language, holder.SetCode, holder.Number)
			}
		}
	}
}

// Every language copy is found from its original in the copy's language and
// the original's finish, as is a listing naming the original's id in it.
func TestMatchIDLanguage(t *testing.T) {
	realDatastore(t)

	for _, uuid := range testBackend.AllUUIDs {
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		original, err := testBackend.GetUUID(co.Identifiers["mtgjsonId"])
		if err != nil || original.Language == co.Language {
			continue
		}
		id, err := testBackend.MatchIDLanguage(original.UUID, co.Language)
		if err != nil {
			t.Errorf("%s #%s has no %s printing: %v", original.SetCode, original.Number, co.Language, err)
			continue
		}
		printing, _ := testBackend.GetUUID(id)
		if printing.Language != co.Language || printing.Identifiers["mtgjsonId"] != original.UUID ||
			printing.Foil != original.Foil || printing.Etched != original.Etched {
			t.Errorf("%s #%s in %s is %s %s #%s", original.SetCode, original.Number, co.Language,
				printing.Language, printing.SetCode, printing.Number)
		}

		// a listing naming the original's id in the copy's language lands on it
		validated, err := testBackend.ValidateID(mtgmatcher.InputCard{
			ID: original.UUID, Name: original.Name, Language: co.Language, Finish: original.Finish,
		}, mtgmatcher.IDValidationOptions{})
		if err != nil || validated != id {
			t.Errorf("%s #%s validated in %s as %s, %v", original.SetCode, original.Number, co.Language, validated, err)
		}
		if original.Etched {
			continue
		}
		matched, err := testBackend.Match(&mtgmatcher.InputCard{
			ID: original.UUID, Name: original.Name, Language: co.Language, Foil: original.Foil,
		})
		if err != nil || matched != id {
			t.Errorf("%s #%s matched in %s as %s, %v", original.SetCode, original.Number, co.Language, matched, err)
		}
	}
}
