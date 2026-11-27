package matcher

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
)

// sealedIDsBackend is an empty backend with the sets the sealed id tests
// file their products under.
func sealedIDsBackend() *Backend {
	return &Backend{
		UUIDs:          map[string]*CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*Set{
			"OGN": {Name: "Origins", Code: "OGN"},
			"SLD": {Name: "Secret Lair Drop", Code: "SLD"},
		},
	}
}

func sealedIDsEqual(a, b SealedIDs) bool {
	return maps.EqualFunc(a, b, slices.Equal)
}

func sealedLinksEqual(a, b SealedLinks) bool {
	return a.TcgPlayerID == b.TcgPlayerID &&
		slices.Equal(a.CardmarketIDs, b.CardmarketIDs) &&
		slices.Equal(a.CardTraderIDs, b.CardTraderIDs) &&
		slices.Equal(a.CSIIDs, b.CSIIDs) &&
		slices.Equal(a.SCGIDs, b.SCGIDs) &&
		slices.Equal(a.MiniatureMarketIDs, b.MiniatureMarketIDs)
}

func TestSealedLinksIDs(t *testing.T) {
	var links SealedLinks
	err := json.Unmarshal([]byte(`{"tcgPlayerId": 5, "cardmarketIds": [7, 9], "cardTraderIds": [501],
		"csiIds": ["a"], "scgIds": ["S1"], "miniaturemarketIds": ["M1", "M2"]}`), &links)
	if err != nil {
		t.Fatal(err)
	}
	want := SealedIDs{
		"tcgplayerProductId": {"5"},
		"mcmId":              {"7", "9"},
		"cardtraderId":       {"501"},
		"csiId":              {"a"},
		"scgId":              {"S1"},
		"miniaturemarketId":  {"M1", "M2"},
	}
	got := links.IDs()
	if !sealedIDsEqual(got, want) {
		t.Errorf("IDs() = %v, want %v", got, want)
	}

	var bare SealedLinks
	err = json.Unmarshal([]byte(`{"tcgPlayerId": 0}`), &bare)
	if err != nil {
		t.Fatal(err)
	}
	got = bare.IDs()
	if len(got) != 0 {
		t.Errorf("IDs() of a zero id = %v, want empty", got)
	}

	// Zero and empty values drop, and so does the name they leave empty
	zeros := SealedLinks{CardTraderIDs: []int{0}, SCGIDs: []string{"", "S1"}, MiniatureMarketIDs: []string{""}}
	got = zeros.IDs()
	want = SealedIDs{"scgId": {"S1"}}
	if !sealedIDsEqual(got, want) {
		t.Errorf("IDs() = %v, want %v", got, want)
	}
}

// TestSealedLinksTolerantLists pins that a store list the loader cannot read
// costs that list or that element, never the decode: a datastore's typo must
// not take a whole game's load down.
func TestSealedLinksTolerantLists(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  SealedLinks
	}{
		{`{"tcgPlayerId": 5}`, SealedLinks{TcgPlayerID: 5}},
		{`{"tcgPlayerId": 5, "csiIds": null, "cardmarketIds": null}`, SealedLinks{TcgPlayerID: 5}},
		// A numeric id written as a number, or a number written as a string
		{`{"csiIds": [123456]}`, SealedLinks{CSIIDs: []string{"123456"}}},
		{`{"cardmarketIds": ["7"]}`, SealedLinks{CardmarketIDs: []int{7}}},
		{`{"cardTraderIds": ["07"]}`, SealedLinks{CardTraderIDs: []int{7}}},
		// What is not a positive int is dropped from a numeric list
		{`{"cardmarketIds": [7.0]}`, SealedLinks{}},
		{`{"cardmarketIds": [7, 7.5, -1, 0, 99999999999999999999, "abc", "", "+8", "-8", " 8", null, true, {}, [8], "8"]}`,
			SealedLinks{CardmarketIDs: []int{7, 8}}},
		{`{"cardTraderIds": ["99999999999999999999", "1e3", 1e3]}`, SealedLinks{}},
		// What is not a list is ignored
		{`{"tcgPlayerId": 5, "cardTraderIds": "501"}`, SealedLinks{TcgPlayerID: 5}},
		{`{"cardTraderIds": 501}`, SealedLinks{}},
		{`{"scgIds": {"a": "S1"}}`, SealedLinks{}},
		{`{"miniaturemarketIds": true}`, SealedLinks{}},
		// A string list keeps a string as it is and a number in its plain
		// decimal spelling, and drops the rest
		{`{"scgIds": ["S1", "", 12, 1.5, -3, 1e3, true, null, ["S2"]]}`, SealedLinks{SCGIDs: []string{"S1", "12"}}},
		{`{"miniaturemarketIds": [" M1 ", "007"]}`, SealedLinks{MiniatureMarketIDs: []string{" M1 ", "007"}}},
	} {
		var got SealedLinks
		err := json.Unmarshal([]byte(tc.input), &got)
		if err != nil {
			t.Errorf("%s: %v", tc.input, err)
			continue
		}
		if !sealedLinksEqual(got, tc.want) {
			t.Errorf("%s: decoded %+v, want %+v", tc.input, got, tc.want)
		}
	}
}

// TestSealedLinksStrictTcgPlayerID pins that tcgPlayerId accepts and refuses
// exactly what the plain int field every loader read it through did.
func TestSealedLinksStrictTcgPlayerID(t *testing.T) {
	type plainLinks struct {
		TcgPlayerID int `json:"tcgPlayerId"`
	}
	for _, input := range []string{
		`{"tcgPlayerId": 5}`,
		`{"tcgPlayerId": -3}`,
		`{"tcgPlayerId": null}`,
		`{"TCGPLAYERID": 4}`,
		`{}`,
		`null`,
		`{"tcgPlayerId": "5"}`,
		`{"tcgPlayerId": 5.5}`,
		`{"tcgPlayerId": 5.0}`,
		`{"tcgPlayerId": 1e3}`,
		`{"tcgPlayerId": 99999999999999999999}`,
		`{"tcgPlayerId": [5]}`,
		`{"tcgPlayerId": true}`,
		`"links"`,
		`[5]`,
	} {
		var plain plainLinks
		plainErr := json.Unmarshal([]byte(input), &plain)
		var got SealedLinks
		err := json.Unmarshal([]byte(input), &got)
		if (err == nil) != (plainErr == nil) {
			t.Errorf("%s: error %v, want %v", input, err, plainErr)
			continue
		}
		if err == nil && got.TcgPlayerID != plain.TcgPlayerID {
			t.Errorf("%s: tcgPlayerId %d, want %d", input, got.TcgPlayerID, plain.TcgPlayerID)
		}
	}
}

func TestAddSealedIdentifiers(t *testing.T) {
	b := sealedIDsBackend()
	b.AddSealed("ogn-box", "Booster Box", "OGN", "", SealedIDs{
		"mcmId":              {"7", "9", "7"},
		"tcgplayerProductId": {"5"},
	})

	co, err := b.GetUUID("ogn-box")
	if err != nil {
		t.Fatal(err)
	}
	got := co.Identifiers["mcmId"]
	if got != "7" {
		t.Errorf(`Identifiers["mcmId"] = %q, want "7"`, got)
	}
	got = co.Identifiers["tcgplayerProductId"]
	if got != "5" {
		t.Errorf(`Identifiers["tcgplayerProductId"] = %q, want "5"`, got)
	}
	// A repeated id is filed once
	ids := b.SealedIdentifiers("ogn-box", "mcmId")
	if !slices.Equal(ids, []string{"7", "9"}) {
		t.Errorf(`SealedIdentifiers("mcmId") = %v, want [7 9]`, ids)
	}
	ids = b.SealedIdentifiers("ogn-box", "scgId")
	if ids != nil {
		t.Errorf(`SealedIdentifiers("scgId") = %v, want nil`, ids)
	}
	ids = b.SealedIdentifiers("unknown", "mcmId")
	if ids != nil {
		t.Errorf(`SealedIdentifiers of an unknown uuid = %v, want nil`, ids)
	}

	products := b.Sets["OGN"].SealedProduct
	if len(products) != 1 {
		t.Fatalf("SealedProduct has %d entries, want 1", len(products))
	}
	if !maps.Equal(products[0].Identifiers, co.Identifiers) {
		t.Errorf("SealedProduct.Identifiers = %v, want %v", products[0].Identifiers, co.Identifiers)
	}
}

// TestSealedIdentifiersIsACopy pins that a caller writing into the answer
// cannot change what the backend holds.
func TestSealedIdentifiersIsACopy(t *testing.T) {
	b := sealedIDsBackend()
	b.AddSealed("ogn-box", "Booster Box", "OGN", "", SealedIDs{"mcmId": {"7", "9"}})

	first := b.SealedIdentifiers("ogn-box", "mcmId")
	first[0] = "changed"
	_ = append(first[:1], "appended")

	second := b.SealedIdentifiers("ogn-box", "mcmId")
	if !slices.Equal(second, []string{"7", "9"}) {
		t.Errorf("SealedIdentifiers after a write into the first answer = %v, want [7 9]", second)
	}
}

// TestAddSealedOnACardUUID pins that ids filed on a uuid a card already holds
// are not stored for it: the card keeps the uuid, and no sealed id answers.
func TestAddSealedOnACardUUID(t *testing.T) {
	b := sealedIDsBackend()
	b.UUIDs["ogn-1"] = &CardObject{
		Card: Card{UUID: "ogn-1", Name: "Fixture Card", SetCode: "OGN"},
	}
	b.AddSealed("ogn-1", "Booster Box", "OGN", "", SealedIDs{"mcmId": {"7"}, "csiId": {"C1"}})

	_, found := b.sealedIDs["ogn-1"]
	if found {
		t.Errorf("sealedIDs holds %v for a card's uuid, want nothing", b.sealedIDs["ogn-1"])
	}
	ids := b.SealedIdentifiers("ogn-1", "mcmId")
	if ids != nil {
		t.Errorf(`SealedIdentifiers("mcmId") = %v, want nil`, ids)
	}
	co := b.UUIDs["ogn-1"]
	if co.Sealed || co.Name != "Fixture Card" {
		t.Errorf("the uuid answers %+v, want the card", co.Card)
	}
	productMap := b.BuildSealedProductMap("mcmId")
	if len(productMap) != 0 {
		t.Errorf("BuildSealedProductMap(mcmId) = %v, want empty", productMap)
	}
}

func TestBuildSealedProductMapEveryID(t *testing.T) {
	b := sealedIDsBackend()
	// "07" spells 7 again and must not file the product twice under it
	b.AddSealed("ogn-box", "Booster Box", "OGN", "", SealedIDs{"mcmId": {"7", "x", "9", "07"}})
	b.SortSealed()

	productMap := b.BuildSealedProductMap("mcmId")
	for _, id := range []int{7, 9} {
		got := productMap[id]
		if !slices.Equal(got, []string{"ogn-box"}) {
			t.Errorf("productMap[%d] = %v, want [ogn-box]", id, got)
		}
	}
	if len(productMap) != 2 {
		t.Errorf("productMap = %v, want two keys", productMap)
	}
}

func TestBuildSealedIDMap(t *testing.T) {
	b := sealedIDsBackend()
	b.AddSealed("ogn-box", "Booster Box", "OGN", "", SealedIDs{"csiId": {"a", "shared"}})
	b.AddSealed("ogn-bundle", "Bundle", "OGN", "", SealedIDs{"csiId": {"shared"}})
	b.AddSealed("ogn-alpha", "Alpha Display", "OGN", "", SealedIDs{"csiId": {"shared"}})
	b.SortSealed()

	idMap := b.BuildSealedIDMap("csiId")
	got := idMap["a"]
	if !slices.Equal(got, []string{"ogn-box"}) {
		t.Errorf(`idMap["a"] = %v, want [ogn-box]`, got)
	}
	want := []string{"ogn-alpha", "ogn-box", "ogn-bundle"}
	got = idMap["shared"]
	if !slices.Equal(got, want) {
		t.Errorf(`idMap["shared"] = %v, want %v`, got, want)
	}
	scgMap := b.BuildSealedIDMap("scgId")
	if len(scgMap) != 0 {
		t.Errorf("BuildSealedIDMap(scgId) = %v, want empty", scgMap)
	}
}

func TestAddSealedNoIDs(t *testing.T) {
	b := sealedIDsBackend()
	b.AddSealed("ogn-box", "Booster Box", "OGN", "", nil)
	b.AddSealed("ogn-bundle", "Bundle", "OGN", "", SealedIDs{"mcmId": {""}})

	for _, uuid := range []string{"ogn-box", "ogn-bundle"} {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.Identifiers != nil {
			t.Errorf("%s Identifiers = %v, want nil", uuid, co.Identifiers)
		}
		got := b.SealedIdentifiers(uuid, "mcmId")
		if got != nil {
			t.Errorf("%s SealedIdentifiers = %v, want nil", uuid, got)
		}
	}
	for _, product := range b.Sets["OGN"].SealedProduct {
		if product.Identifiers != nil {
			t.Errorf("SealedProduct %s Identifiers = %v, want nil", product.UUID, product.Identifiers)
		}
	}
}

func TestSealedIDsOfAMagicProduct(t *testing.T) {
	// Magic's sealed products arrive with Identifiers filled and no
	// AddSealed call, so each name answers its one id.
	b := sealedIDsBackend()
	b.UUIDs["magic-box"] = &CardObject{
		Card: Card{
			UUID:        "magic-box",
			Name:        "Magic Booster Box",
			SetCode:     "OGN",
			Identifiers: map[string]string{"csiId": "x", "mcmId": "3"},
		},
		Sealed: true,
	}
	b.AllSealedUUIDs = append(b.AllSealedUUIDs, "magic-box")

	got := b.SealedIdentifiers("magic-box", "csiId")
	if !slices.Equal(got, []string{"x"}) {
		t.Errorf(`SealedIdentifiers("csiId") = %v, want [x]`, got)
	}
	got = b.SealedIdentifiers("magic-box", "scgId")
	if got != nil {
		t.Errorf(`SealedIdentifiers("scgId") = %v, want nil`, got)
	}
	got = b.BuildSealedIDMap("csiId")["x"]
	if !slices.Equal(got, []string{"magic-box"}) {
		t.Errorf(`BuildSealedIDMap("csiId")["x"] = %v, want [magic-box]`, got)
	}
	got = b.BuildSealedProductMap("mcmId")[3]
	if !slices.Equal(got, []string{"magic-box"}) {
		t.Errorf(`BuildSealedProductMap("mcmId")[3] = %v, want [magic-box]`, got)
	}
}

func TestBuildSealedIDMapNoSecretLairPath(t *testing.T) {
	b := sealedIDsBackend()
	b.AddSealed("sld-drop", "Fixture Drop", "SLD", "", SealedIDs{"tcgplayerProductId": {"5"}})
	b.AddSealed("sld-drop-foil", "Fixture Drop Foil", "SLD", "", nil)
	b.SortSealed()

	idMap := b.BuildSealedIDMap("tcgplayerProductId")
	got := idMap["5"]
	if !slices.Equal(got, []string{"sld-drop"}) {
		t.Errorf(`BuildSealedIDMap["5"] = %v, want [sld-drop]`, got)
	}

	// The foil drop shares its parent's listing, foil last
	productMap := b.BuildSealedProductMap("tcgplayerProductId")
	want := []string{"sld-drop", "sld-drop-foil"}
	got = productMap[5]
	if !slices.Equal(got, want) {
		t.Errorf("BuildSealedProductMap[5] = %v, want %v", got, want)
	}
}
