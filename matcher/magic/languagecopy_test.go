package magic

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
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

	copies := duplicateCards(sets, "SLD", "SLD", "JPN", []string{"1114", "1115", "1117"})

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

	duplicate(sets, "Alternate Fourth Edition", "4ED", "4EDALT", "ALT", "1995-04-01")

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
		space matcher.IDSpace
	}{
		{"mtgjsonId", matcher.IDSpaceMTGJSON},
		{"scryfallId", matcher.IDSpaceScryfall},
		{"tcgplayerProductId", matcher.IDSpaceTCGplayer},
		{"multiverseId", matcher.IDSpaceMultiverse},
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
				if filing.space == matcher.IDSpaceTCGplayer || source.Identifiers[filing.tag] == id {
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
		validated, err := testBackend.ValidateID(matcher.InputCard{
			ID: original.UUID, Name: original.Name, Language: co.Language, Finish: original.Finish,
		}, matcher.IDValidationOptions{})
		if err != nil || validated != id {
			t.Errorf("%s #%s validated in %s as %s, %v", original.SetCode, original.Number, co.Language, validated, err)
		}
		if original.Etched {
			continue
		}
		matched, err := testBackend.Match(&matcher.InputCard{
			ID: original.UUID, Name: original.Name, Language: co.Language, Foil: original.Foil,
		})
		if err != nil || matched != id {
			t.Errorf("%s #%s matched in %s as %s, %v", original.SetCode, original.Number, co.Language, matched, err)
		}
	}
}

// Every language set holds each card of its base set printed in its language,
// as that printing, and is filed under its base where the site stacks sets.
func TestLanguageSets(t *testing.T) {
	realDatastore(t)

	topParent := func(code string) string {
		for {
			set, err := testBackend.GetSet(code)
			if err != nil || set.ParentCode == "" {
				return code
			}
			code = set.ParentCode
		}
	}

	for _, ls := range languageSets {
		code, language := ls.code, langs[ls.tag]
		base, err := testBackend.GetSet(ls.base)
		if err != nil {
			t.Fatal(err)
		}
		set, err := testBackend.GetSet(code)
		if err != nil {
			t.Fatal(err)
		}
		if set.Name != ls.name || set.ParentCode != ls.base || set.ReleaseDate != base.ReleaseDate ||
			set.Type != base.Type || topParent(code) != topParent(ls.base) {
			t.Errorf("%s is %q under %s, %s %s; want %q under %s, %s %s", code, set.Name,
				set.ParentCode, set.ReleaseDate, set.Type, ls.name, ls.base, base.ReleaseDate, base.Type)
		}

		// Its cards' art letters read like its base's
		if _, found := VariantsTable[base.Name]; found {
			if _, found := VariantsTable[set.Name]; !found {
				t.Errorf("VariantsTable has %q but not %q", base.Name, set.Name)
			}
		}

		var want int
		for _, card := range base.Cards {
			var printing string
			for _, fd := range card.ForeignData {
				if fd.Language == language {
					printing = fd.UUID
				}
			}
			if printing == "" || card.Language == language {
				continue
			}
			want++
			copied, err := testBackend.GetUUID(printing)
			if err != nil {
				t.Errorf("%s #%s has no %s printing in %s", ls.base, card.Number, language, code)
				continue
			}
			var name, scryfallID string
			for _, fd := range card.ForeignData {
				if fd.Language == language {
					name, scryfallID = fd.Name, fd.Identifiers["scryfallId"]
				}
			}
			if copied.SetCode != code || copied.Edition != set.Name || copied.Number != card.Number ||
				copied.Name != card.Name || copied.Language != language || copied.PrintedName != name ||
				copied.Identifiers["scryfallId"] != scryfallID || !strings.Contains(copied.Images["full"], scryfallID) ||
				copied.Identifiers["mtgjsonId"] != card.Identifiers["mtgjsonId"] ||
				!slices.Contains(copied.Printings, code) || !slices.Contains(card.Printings, code) {
				t.Errorf("%s #%s in %s is %s %s #%s %s %q, image %s", ls.base, card.Number, language,
					copied.Name, copied.SetCode, copied.Number, copied.Language, copied.PrintedName, copied.Images["full"])
			}

			id, err := testBackend.MatchIDLanguage(card.UUID, language)
			if err != nil || id != printing {
				t.Errorf("%s #%s in %s: MatchIDLanguage = %s %v", ls.base, card.Number, language, id, err)
			}
			// mtgjson lists no English entry, so nothing leads back to an English base
			wantBack := card.UUID
			if card.Language == "English" {
				wantBack = ""
			}
			id, err = testBackend.MatchIDLanguage(printing, card.Language)
			if id != wantBack || (err != nil) != (wantBack == "") {
				t.Errorf("%s #%s in %s: MatchIDLanguage back to %s = %s %v", ls.base, card.Number, language, card.Language, id, err)
			}
			in := matcher.InputCard{
				Name: card.Name, Edition: base.Name, Variation: card.Number,
				Language: strings.Fields(language)[0],
			}
			id, err = testBackend.Match(&in)
			if err != nil || id != printing {
				t.Errorf("%s #%s in %s: Match = %s %v", ls.base, card.Number, language, id, err)
			}
		}
		if want == 0 || len(set.Cards) != want {
			t.Errorf("%s holds %d cards, want the %d %s printings of %s", code, len(set.Cards), want, language, ls.base)
		}
	}
}

// Every vendor sells the foreign white borders apart from Revised, so they
// share none of its cards' vendor ids, and the sealed products mtgjson files
// under FWB stay there.
func TestForeignWhiteBorderIsItsOwn(t *testing.T) {
	realDatastore(t)
	for _, code := range []string{"FWB", "FWBFR", "FWBDE"} {
		set, err := testBackend.GetSet(code)
		if err != nil {
			t.Fatal(err)
		}
		for _, card := range set.Cards {
			for _, tag := range []string{
				"tcgplayerProductId", "mcmId", "mcmMetaId", "cardKingdomId", "cardsphereId", "deckboxId", "multiverseId",
			} {
				id, found := card.Identifiers[tag]
				if found {
					t.Errorf("%s #%s took Revised's %s %s", code, card.Number, tag, id)
				}
			}
		}
	}
	set, _ := testBackend.GetSet("FWB")
	if len(set.SealedProduct) == 0 {
		t.Error("FWB lost its sealed products")
	}
}

// A vendor names a card of these sets by one Scryfall id in every language,
// mtgjson's own or another printing's (Card Trader's blueprints), and the
// listing's language picks the printing.
func TestLanguageSetsFromVendorIDs(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct{ scryfallID, language, want string }{
		// FBB Badlands, mtgjson's French scan
		{"2fb67912-3d3d-40d4-8aa0-141a8025b037", "", "e8cb1f71-56a7-5cbf-8f93-99cf78386545"},
		{"2fb67912-3d3d-40d4-8aa0-141a8025b037", "Italian", "e8cb1f71-56a7-5cbf-8f93-99cf78386545"},
		{"2fb67912-3d3d-40d4-8aa0-141a8025b037", "French", "77f91bd3-4a2b-55f1-a9e9-7359771d0582"},
		{"2fb67912-3d3d-40d4-8aa0-141a8025b037", "German", "6610e802-6f8a-5180-ac9b-537dc63736a7"},
		// 4BB Counterspell, mtgjson's Spanish scan
		{"79976df3-e989-4f99-8534-dca962f85e3d", "Japanese", "b40a855d-6dcb-5225-8888-421f680a03af"},
		{"79976df3-e989-4f99-8534-dca962f85e3d", "Spanish", "70d94ab6-6a48-5c29-91ae-a7625bff07d9"},
		// REN Yotian Soldier, by the German scan Card Trader's blueprint names
		{"790ca247-bcd1-40a3-9b3e-720fc5fcc37d", "French", "e10242cb-58d6-565d-b9ff-5bd64055ff0f"},
		{"790ca247-bcd1-40a3-9b3e-720fc5fcc37d", "German", "1be6bdc9-a884-5c89-a990-bba633a4a288"},
	} {
		in := matcher.InputCard{ID: tt.scryfallID, Language: tt.language}
		id, err := testBackend.Match(&in)
		if err != nil || id != tt.want {
			t.Errorf("Scryfall %s in %q = %s %v, want %s", tt.scryfallID, tt.language, id, err, tt.want)
		}
	}
}
