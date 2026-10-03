package vocabulary

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestLoaderContracts holds every game in mtgmatcher.AllGames but Magic to
// what each loader must keep, one backend load per game. A game with no row
// in Games fails rather than skipping, so a new game cannot be left out. A
// job carries one game's datastore, so the others skip.
func TestLoaderContracts(t *testing.T) {
	var loaded int
	for _, game := range mtgmatcher.AllGames {
		// Magic's checks live in mtgmatcher/magic, which parses AllPrintings already.
		if game == mtgmatcher.GameMagic {
			continue
		}
		t.Run(string(game), func(t *testing.T) {
			variable := Games[game]
			if variable == "" {
				t.Fatalf("Games has no datastore variable for %s", game)
			}
			path := PathOf(game)
			if path == "" {
				t.Skipf("Need %s set to run this test", variable)
			}
			b, err := datastore.Read(game, path)
			if err != nil {
				t.Fatal(err)
			}
			loaded++

			t.Run("SetUUIDsIndexesRealCards", func(t *testing.T) {
				setUUIDsIndexRealCards(t, b)
			})
			t.Run("PlainNumberMatchesLoader", func(t *testing.T) {
				plainNumberMatchesLoader(t, b)
			})
			t.Run("ProductGroupHoldsOneCard", func(t *testing.T) {
				productGroupHoldsOneCard(t, b, path)
			})
			t.Run("ColorsAreLowerCase", func(t *testing.T) {
				colorsAreLowerCase(t, b)
			})
			t.Run("RaritiesAreNames", func(t *testing.T) {
				raritiesAreNames(t, b, path)
			})
			t.Run("PropertiesOrderEveryValue", func(t *testing.T) {
				propertiesOrderEveryValue(t, b, path)
			})
		})
	}
	if loaded == 0 {
		t.Skip("no game datastore is named in this run")
	}
}

// TestLoadersRefuse holds every loader but Magic's to refusing a file that is
// not its game's datastore: a payload outside the envelope, and one missing
// any of what the loader checks for. Each payload loads as written, so a
// refusal is its defect's.
func TestLoadersRefuse(t *testing.T) {
	const card = `{"id":"a","name":"A","setCode":"S","finish":"Normal"}`
	flat := func(game mtgmatcher.Game) string {
		return `{"game":"` + string(game) + `","sets":{"S":{"name":"S"}},"cards":[` + card + `]}`
	}
	flatDefects := func(game mtgmatcher.Game) [][2]string {
		return [][2]string{
			{`"game":"` + string(game) + `"`, `"game":"magic"`},
			{`{"S":{"name":"S"}}`, `{}`},
			{card, ``},
			{`"id":"a"`, `"id":""`},
			{`"name":"A"`, `"name":""`},
			{`"finish":"Normal"`, `"finish":""`},
		}
	}
	lorcanaCard := `{"id":1,"fullName":"A","setCode":"S"}`
	riftboundCard := `{"id":"a","name":"A","setCode":"S"}`
	type row struct {
		payload string
		defects [][2]string
	}
	rows := map[mtgmatcher.Game]row{
		mtgmatcher.GameLorcana: {
			`{"sets":{"S":{"name":"S"}},"cards":[` + lorcanaCard + `]}`,
			[][2]string{{`{"S":{"name":"S"}}`, `{}`}, {lorcanaCard, ``}, {`"id":1`, `"id":0`}, {`"fullName":"A"`, `"fullName":""`}},
		},
		mtgmatcher.GameRiftbound: {
			`{"pageProps":{"page":{"blades":[{"type":"riftboundCardGallery","sets":{"items":[{"id":"S","name":"S"}]},"cards":{"items":[` + riftboundCard + `]}}]}}}`,
			[][2]string{{`"riftboundCardGallery"`, `"masthead"`}, {`{"id":"S","name":"S"}`, ``}, {riftboundCard, ``}, {`"id":"a"`, `"id":""`}, {`"name":"A"`, `"name":""`}},
		},
	}
	for _, game := range []mtgmatcher.Game{mtgmatcher.GameOnePiece, mtgmatcher.GameYuGiOh, mtgmatcher.GameFleshAndBlood, mtgmatcher.GamePokemon, mtgmatcher.GameGundam, mtgmatcher.GamePalworld} {
		rows[game] = row{flat(game), flatDefects(game)}
	}
	for _, game := range mtgmatcher.AllGames {
		if game == mtgmatcher.GameMagic {
			continue
		}
		t.Run(string(game), func(t *testing.T) {
			row, found := rows[game]
			if !found {
				t.Fatalf("no payload for %s", game)
			}
			_, err := mtgmatcher.Open(game, strings.NewReader(`{"meta":{},"data":`+row.payload+`}`))
			if err != nil {
				t.Fatalf("the payload as written is refused: %v", err)
			}
			_, err = mtgmatcher.Open(game, strings.NewReader(row.payload))
			if err == nil {
				t.Error("a payload outside the envelope loads")
			}
			for _, defect := range row.defects {
				payload := strings.Replace(row.payload, defect[0], defect[1], 1)
				_, err = mtgmatcher.Open(game, strings.NewReader(`{"meta":{},"data":`+payload+`}`))
				if err == nil {
					t.Errorf("loads with %s replaced by [%s]", defect[0], defect[1])
				}
			}
		})
	}
}

// setUUIDsIndexRealCards pins the index GetUUIDsInSet answers from, which
// the site's edition-only searches (s:CODE) seed their candidates from
// alone: an empty one answers nothing for every set of the game.
func setUUIDsIndexRealCards(t *testing.T, b *mtgmatcher.Backend) {
	if len(b.SetUUIDs) == 0 {
		t.Fatal("SetUUIDs is empty; IndexSetUUIDs was not called, or the datastore has no cards")
	}
	checked := 0
	for _, code := range b.AllSets {
		for _, uuid := range b.SetUUIDs[code] {
			co, err := b.GetUUID(uuid)
			if err != nil {
				t.Errorf("SetUUIDs[%s] holds %s, which GetUUID does not know: %v", code, uuid, err)
				continue
			}
			if co.SetCode != code {
				t.Errorf("SetUUIDs[%s] holds %s, whose SetCode is %q", code, uuid, co.SetCode)
			}
			if co.Sealed {
				t.Errorf("SetUUIDs[%s] holds the sealed uuid %s", code, uuid)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no uuid in any SetUUIDs bucket; the index is present but empty")
	}
}

// plainNumberMatchesLoader pins the game's PlainNumber rule to what the
// loader stored on each card. The two spelling a number differently finds
// nothing and raises nothing, which a caller reads as "no such card".
func plainNumberMatchesLoader(t *testing.T, b *mtgmatcher.Backend) {
	var seen, folded int
	for _, code := range b.GetAllSets() {
		set, err := b.GetSet(code)
		if err != nil {
			continue
		}
		for _, card := range set.Cards {
			plain := b.PlainNumber(card.Number)
			if plain != card.PlainNumber {
				t.Errorf("%s %q: the rule folds to %q, the card carries %q",
					code, card.Number, plain, card.PlainNumber)
			}
			if len(plain) > len(card.Number) {
				t.Errorf("%s: PlainNumber %q is wider than Number %q",
					code, plain, card.Number)
			}
			again := b.PlainNumber(plain)
			if again != plain {
				t.Errorf("%s %q: folding %q again gives %q",
					code, card.Number, plain, again)
			}
			if plain != card.Number {
				folded++
			}
			seen++
		}
	}
	if seen == 0 {
		t.Fatal("no cards to check")
	}
	// A datastore publishing only bare ordinals would leave the checks above
	// passing while testing nothing.
	if folded == 0 {
		t.Errorf("no number of %d reduced to anything shorter", seen)
	}
}

// printing is what a datastore says a printing is a printing of.
type printing struct{ name, number, setCode, variant string }

// productGroupHoldsOneCard pins that a card answers only with printings the
// datastore published as that same card. A fold on the wrong key prices one
// card's printing under another's finish, and every count still adds up.
//
// The product id has to answer first: an upstream id names a card, and 331
// Flesh and Blood fabIds and 687 Pokemon tcgdexIds are carried by more than
// one product. Reading the fabId first folds 345 printings onto another card.
//
// The published side is read beside the loader, because once folded a
// card's printings share one Card and could only be compared with itself.
func productGroupHoldsOneCard(t *testing.T, b *mtgmatcher.Backend, path string) {
	published := publishedPrintings(t, path)
	if len(published) == 0 {
		t.Fatal("the datastore publishes no printings")
	}
	for _, uuid := range slices.Sorted(maps.Keys(published)) {
		this := published[uuid]
		co := b.UUIDs[uuid]
		if co == nil {
			// The loader skips a printing of a set the datastore does not hold.
			if b.Sets[this.setCode] != nil {
				t.Errorf("%s (%s %s %s %q) is published but not loaded",
					uuid, this.name, this.setCode, this.number, this.variant)
			}
			continue
		}
		for _, finish := range slices.Sorted(maps.Keys(co.FoilUUIDs)) {
			other := co.FoilUUIDs[finish]
			that, found := published[other]
			if !found {
				t.Errorf("%s answers %s with %s, which the datastore does not publish",
					uuid, finish, other)
				continue
			}
			if that != this {
				t.Errorf("%s (%s %s %s %q) answers %s with %s (%s %s %s %q), a different card",
					uuid, this.name, this.setCode, this.number, this.variant,
					finish, other, that.name, that.setCode, that.number, that.variant)
			}
		}
	}
}

// publishedPrintings reads every printing a built datastore publishes, by
// uuid: each id a card nests under printings, or the card's own id where it
// is a printing by itself.
func publishedPrintings(t *testing.T, path string) map[string]printing {
	t.Helper()
	f, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	raw, err := datastore.Payload(f)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	err = json.Unmarshal(raw, &data)
	if err != nil {
		t.Fatal(err)
	}

	out := map[string]printing{}
	for _, card := range cardsOf(data) {
		this := printing{
			name:    textOf(card["name"]),
			number:  textOf(card["number"]),
			setCode: textOf(card["setCode"]),
			variant: textOf(card["variant"]),
		}
		var ids []string
		for _, nested := range objects(card["printings"]) {
			ids = append(ids, textOf(nested["id"]))
		}
		if len(ids) == 0 {
			ids = append(ids, textOf(card["id"]))
		}
		for _, id := range ids {
			out[id] = this
		}
	}
	return out
}

// textOf reads a field as the string it holds, empty where it holds none.
func textOf(value any) string {
	held, ok := value.(string)
	if !ok {
		return ""
	}
	return held
}

// raritiesAreNames holds every card to carrying its rarity spelled by
// RarityName, and RarityLabel to giving back the words the datastore
// published it as, which the rules that read a rarity word by word rely on.
func raritiesAreNames(t *testing.T, b *mtgmatcher.Backend, path string) {
	for _, set := range b.Sets {
		for _, card := range set.Cards {
			if card.Rarity != mtgmatcher.RarityName(card.Rarity) {
				t.Fatalf("%s carries the rarity %q", card.UUID, card.Rarity)
			}
		}
	}
	for _, published := range publishedProperties(t, path)["rarity"] {
		name := mtgmatcher.RarityName(published)
		label := b.RarityLabel(name)
		if name != published && label != published {
			t.Errorf("RarityLabel(%q) = %q, want %q", name, label, published)
		}
	}
}

// colorsAreLowerCase holds every game to Magic's spelling of a colour name,
// on its cards and on the sets that list them.
func colorsAreLowerCase(t *testing.T, b *mtgmatcher.Backend) {
	for code, set := range b.Sets {
		colors := slices.Clone(set.Colors)
		for _, card := range set.Cards {
			colors = append(colors, card.Colors...)
		}
		for _, color := range colors {
			if color != strings.ToLower(color) {
				t.Errorf("set %s lists the colour %q", code, color)
				break
			}
		}
	}
}

// propertiesOrderEveryValue holds a datastore's properties to ordering every
// value its cards load with: "rarity" each card's rarity, every other
// property its colours. A value no list names still loads, sorted last by
// name, so nothing else notices an order going stale. It also holds the
// backend to the order the lists give, which a loader reading a property
// under the wrong name loses without an error.
func propertiesOrderEveryValue(t *testing.T, b *mtgmatcher.Backend, path string) {
	properties := publishedProperties(t, path)
	if len(properties) == 0 {
		t.Skipf("%s publishes no properties", path)
	}
	// Folded as the loaders fold them: Lorcana's "Super Rare" is "superrare".
	fold := func(value string) string {
		return strings.ReplaceAll(strings.ToLower(value), " ", "")
	}
	listed := map[string]map[string]bool{}
	for name, order := range properties {
		field := "colour"
		if name == "rarity" {
			field = "rarity"
		}
		if listed[field] == nil {
			listed[field] = map[string]bool{}
		}
		for _, value := range order {
			listed[field][fold(value)] = true
		}
	}
	unlisted := map[string]int{}
	for _, set := range b.Sets {
		for _, card := range set.Cards {
			if listed["rarity"] != nil && card.Rarity != "" && !listed["rarity"][fold(card.Rarity)] {
				unlisted["rarity "+card.Rarity]++
			}
			for _, color := range card.Colors {
				if listed["colour"] != nil && !listed["colour"][fold(color)] {
					unlisted["colour "+color]++
				}
			}
		}
	}
	for _, value := range slices.Sorted(maps.Keys(unlisted)) {
		t.Errorf("%s, on %d cards, is in no list of the datastore's properties", value, unlisted[value])
	}

	// A loader reading a property by a name the datastore does not publish
	// loads all the same, with no order and no colours
	rarities, found := properties["rarity"]
	if found && !slices.Equal(b.Rarities, mtgmatcher.RarityNames(rarities)) {
		t.Errorf("rarities are ordered %v, not as the properties list them", b.Rarities)
	}
	// Every game publishes its colours under one property of its own name
	var colorKeys []string
	for name := range properties {
		if name != "rarity" {
			colorKeys = append(colorKeys, name)
		}
	}
	if len(colorKeys) > 1 {
		slices.Sort(colorKeys)
		t.Fatalf("the properties list colours under %v, not one name", colorKeys)
	}
	var colors []string
	if len(colorKeys) == 1 {
		colors = mtgmatcher.ColorNames(properties[colorKeys[0]])
	}
	coloured := false
	for code, set := range b.Sets {
		for _, card := range set.Cards {
			coloured = coloured || len(card.Colors) > 0
		}
		want := mtgmatcher.ColorsOf(set.Cards, colors)
		if colors != nil && !slices.Equal(set.Colors, want) {
			t.Errorf("%s lists its colours %v, not as the properties order them, %v", code, set.Colors, want)
		}
	}
	if colors != nil && !coloured {
		t.Error("no card carries a colour, though the properties list them")
	}
}

// publishedProperties reads the orders a built datastore publishes beside
// its cards.
func publishedProperties(t *testing.T, path string) map[string][]string {
	t.Helper()
	f, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	raw, err := datastore.Payload(f)
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Properties map[string][]string `json:"properties"`
	}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		t.Fatal(err)
	}
	return data.Properties
}
