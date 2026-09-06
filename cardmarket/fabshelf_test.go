package cardmarket

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

// fabShelfDatastore is the published Flesh and Blood datastore cut down to
// the printings these tests turn on, every row copied verbatim from it:
// the two foils of Will of Arcana, each a TCGplayer product of its own;
// Silver, sold plain alone; Aether Ashwing, whose fused row is plain and
// whose cold foil is filed under the front face; the Prism deck's two
// Heralds; the red and yellow Lead with Heart; the High Seas and Super
// Slam heroes the datastore files once under both faces; and the promos
// and the Silver Age deck card filed in sets of their own.
const fabShelfDatastore = `{
 "game": "fleshandblood",
 "sets": {
  "DYN": {"name": "Dynasty", "releaseDate": "2022-11-11"},
  "FAB": {"name": "Promos", "releaseDate": ""},
  "HER": {"name": "Hero Card Promos", "releaseDate": ""},
  "HS": {"name": "High Seas", "releaseDate": "2025-06-06"},
  "HVY": {"name": "Heavy Hitters", "releaseDate": "2024-02-02"},
  "PSM": {"name": "Blitz Deck: Monarch - Prism", "releaseDate": "2021-05-14"},
  "ROS": {"name": "Rosetta", "releaseDate": "2024-09-20"},
  "SAR": {"name": "Silver Age Chapter 2 - Arakni, Web of Deceit", "releaseDate": "2026-02-13"},
  "SUP": {"name": "Super Slam", "releaseDate": "2025-09-26"},
  "UPR": {"name": "Uprising", "releaseDate": "2022-07-01"}
 },
 "cards": [
  {"externalLinks": {"fabId": "PSM017", "tcgPlayerId": 238517}, "fabId": "PSM017", "finish": "Normal", "id": "psm017_238517", "name": "Herald of Ravages (Blue)", "number": "PSM017", "rarity": "Common", "setCode": "PSM"},
  {"externalLinks": {"fabId": "PSM018", "tcgPlayerId": 238518}, "fabId": "PSM018", "finish": "Normal", "id": "psm018_238518", "name": "Herald of Rebirth (Blue)", "number": "PSM018", "rarity": "Common", "setCode": "PSM"},
  {"externalLinks": {"tcgPlayerId": 275840}, "finish": "Normal", "id": "upr042-upr043_275840", "name": "Aether Ashwing // Ash", "number": "UPR042//UPR043", "rarity": "Token", "setCode": "UPR"},
  {"externalLinks": {"fabId": "UPR042"}, "fabId": "UPR042", "finish": "Normal", "id": "upr042", "name": "Aether Ashwing", "number": "UPR042", "rarity": "Token", "setCode": "UPR"},
  {"externalLinks": {"fabId": "UPR042"}, "fabId": "UPR042", "finish": "Cold Foil", "id": "upr042_cold", "name": "Aether Ashwing", "number": "UPR042", "rarity": "Token", "setCode": "UPR"},
  {"externalLinks": {"fabId": "DYN245", "tcgPlayerId": 453353}, "fabId": "DYN245", "finish": "Normal", "id": "dyn245_453353", "name": "Silver", "number": "DYN245", "rarity": "Common", "setCode": "DYN"},
  {"externalLinks": {"fabId": "HVY192", "tcgPlayerId": 533597}, "fabId": "HVY192", "finish": "Normal", "id": "hvy192_533597", "name": "Lead with Heart (Red)", "number": "HVY192", "rarity": "Common", "setCode": "HVY"},
  {"externalLinks": {"fabId": "HVY192", "tcgPlayerId": 533597}, "fabId": "HVY192", "finish": "Rainbow Foil", "id": "hvy192_533597_rainbow", "name": "Lead with Heart (Red)", "number": "HVY192", "rarity": "Common", "setCode": "HVY"},
  {"externalLinks": {"fabId": "HVY193", "tcgPlayerId": 533598}, "fabId": "HVY193", "finish": "Normal", "id": "hvy193_533598", "name": "Lead with Heart (Yellow)", "number": "HVY193", "rarity": "Common", "setCode": "HVY"},
  {"externalLinks": {"fabId": "HVY193", "tcgPlayerId": 533598}, "fabId": "HVY193", "finish": "Rainbow Foil", "id": "hvy193_533598_rainbow", "name": "Lead with Heart (Yellow)", "number": "HVY193", "rarity": "Common", "setCode": "HVY"},
  {"externalLinks": {"fabId": "ROS000", "tcgPlayerId": 577711}, "fabId": "ROS000", "finish": "Cold Foil", "id": "ros000_577711_cold", "name": "Will of Arcana", "number": "ROS000", "promoTypes": ["cold foil"], "rarity": "Fabled", "setCode": "ROS", "variant": "Cold Foil"},
  {"externalLinks": {"fabId": "ROS000", "tcgPlayerId": 578820}, "fabId": "ROS000", "finish": "Rainbow Foil", "id": "ros000_578820_rainbow", "name": "Will of Arcana", "number": "ROS000", "promoTypes": ["rainbow foil"], "rarity": "Fabled", "setCode": "ROS", "variant": "Rainbow Foil"},
  {"externalLinks": {"fabId": "SEA043", "tcgPlayerId": 624358}, "fabId": "SEA043", "finish": "Cold Foil", "id": "sea043_624358_cold", "name": "Gravy Bones, Shipwrecked Looter (Marvel)", "number": "SEA043", "rarity": "Marvel", "setCode": "HS"},
  {"externalLinks": {"tcgPlayerId": 638044}, "finish": "Normal", "id": "sea043-sea044_638044", "name": "Gravy Bones, Shipwrecked Looter // Gravy Bones", "number": "SEA043//SEA044", "rarity": "Basic", "setCode": "HS"},
  {"externalLinks": {"fabId": "SUP001", "tcgPlayerId": 641658}, "fabId": "SUP001", "finish": "Cold Foil", "id": "sup001_641658_cold", "name": "Tuffnut, Bumbling Hulkster (Marvel)", "number": "SUP001", "rarity": "Marvel", "setCode": "SUP"},
  {"externalLinks": {"tcgPlayerId": 656933}, "finish": "Normal", "id": "sup001-sup002_656933", "name": "Tuffnut, Bumbling Hulkster // Tuffnut", "number": "SUP001//SUP002", "rarity": "Basic", "setCode": "SUP"},
  {"externalLinks": {"fabId": "FAB298"}, "fabId": "FAB298", "finish": "Normal", "id": "fab298", "name": "Tide Flippers", "number": "FAB298", "rarity": "Promo", "setCode": "FAB"},
  {"externalLinks": {"fabId": "FAB305"}, "fabId": "FAB305", "finish": "Cold Foil", "id": "fab305_cold", "name": "Imperial Seal of Command", "number": "FAB305", "rarity": "Promo", "setCode": "FAB"},
  {"externalLinks": {"fabId": "HER069"}, "fabId": "HER069", "finish": "Cold Foil", "id": "her069_cold", "name": "Prism, Sculptor of Arc Light", "number": "HER069", "rarity": "Promo", "setCode": "HER"},
  {"externalLinks": {"fabId": "SAR033"}, "fabId": "SAR033", "finish": "Normal", "id": "sar033", "name": "Graphene Chelicera", "number": "SAR033", "rarity": "Basic", "setCode": "SAR"}
 ]
}`

func loadFabShelfDatastore(t *testing.T) {
	t.Helper()
	if err := mtgmatcher.LoadDatastore(strings.NewReader(fabShelfDatastore)); err != nil {
		t.Fatal(err)
	}
}

// TestFabShelves pins the sets a product is asked of, in order: a promo
// programme's own set before the one every programme was once filed in,
// the set an expansion names by name, and the set wearing the expansion's
// code when no name places it.
func TestFabShelves(t *testing.T) {
	loadFabShelfDatastore(t)
	for _, tt := range []struct {
		expansion, code string
		want            []string
	}{
		{"FAB Promos", "FAB", []string{"FAB"}},
		{"Hero Promos", "HER", []string{"HER"}},
		{"Heavy Hitters", "HVY", []string{"HVY"}},
		{"Silver Age Deck - Arakni, Web of Deceit", "SAR", []string{"SAR"}},
		{"Monarch - Prism Blitz Deck", "PSM", []string{"PSM"}},
		{"Nowhere Deck", "NOPE", nil},
	} {
		var got []string
		for _, sh := range fabShelves(&MKMProduct{ExpansionName: tt.expansion, ExpansionCode: tt.code}) {
			got = append(got, sh.set.Code)
		}
		if strings.Join(got, ",") != strings.Join(tt.want, ",") {
			t.Errorf("fabShelves(%q, %s) = %v, want %v", tt.expansion, tt.code, got, tt.want)
		}
	}
}

// TestFabBaseName pins what comes off a Cardmarket name to reach the card's
// own: treatment, art and label, but never a pitch color.
func TestFabBaseName(t *testing.T) {
	for _, tt := range []struct{ name, want string }{
		{"Silver (Rainbow Foil)", "Silver"},
		{"Bonebreaker Bellow (Yellow) (Extended Art Regular)", "Bonebreaker Bellow (Yellow)"},
		{"Tide Flippers (Cold Foil Golden)", "Tide Flippers"},
		{"Enigma, Ledger of Ancestry (Marvel)", "Enigma, Ledger of Ancestry"},
		{"Levels of Enlightenment (Alternate Art Rainbow Foil)", "Levels of Enlightenment"},
		{"Sink Below (Red)", "Sink Below (Red)"},
		{"Aether Ashwing // Ash (Cold Foil)", "Aether Ashwing // Ash"},
		{"Ironrot Chest (Artist Proof)", "Ironrot Chest"},
	} {
		if got := fabBaseName(tt.name); got != tt.want {
			t.Errorf("fabBaseName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestFabSameProduct pins which two products are one card sold twice: the
// treatments of one card, the unpitched listing beside the pitched one,
// the same listing under two numbers - and not two pitches of a card.
func TestFabSameProduct(t *testing.T) {
	for _, tt := range []struct {
		a, b MKMProduct
		want bool
	}{
		{MKMProduct{Name: "Silver (Regular)", Number: "245"}, MKMProduct{Name: "Silver (Rainbow Foil)", Number: "245"}, true},
		{MKMProduct{Name: "Ice Bolt (Regular)", Number: "135"}, MKMProduct{Name: "Ice Bolt (Blue) (Regular)", Number: "135"}, true},
		{MKMProduct{Name: "Engulfing Light (Red) (Regular)", Number: "014"}, MKMProduct{Name: "Engulfing Light (Red) (Regular)", Number: "009"}, true},
		{MKMProduct{Name: "Pleiades, Superstar (Regular)", Number: "009"}, MKMProduct{Name: "Pleiades, Superstar (Marvel)", Number: "009"}, true},
		{MKMProduct{Name: "Galaxxi Black (Cold Foil)", Number: "155"}, MKMProduct{Name: "Galaxxi Black"}, true},
		{MKMProduct{Name: "Bonebreaker Bellow (Red) (Regular)", Number: "016"}, MKMProduct{Name: "Bonebreaker Bellow (Yellow) (Regular)", Number: "020"}, false},
		{MKMProduct{Name: "Kunai of Retribution (Cold Foil Golden)", Number: "415"}, MKMProduct{Name: "Kunai of Retribution (Cold Foil Golden)", Number: "416"}, true},
		{MKMProduct{Name: "Herald of Ravages (Blue) (Regular)", Number: "017"}, MKMProduct{Name: "Herald of Rebirth (Blue) (Regular)", Number: "018"}, false},
	} {
		if got := fabSameProduct(&tt.a, &tt.b); got != tt.want {
			t.Errorf("fabSameProduct(%q, %q) = %v, want %v", tt.a.Name, tt.b.Name, got, tt.want)
		}
	}
}

// TestMatchProductFinishes pins where a product naming its treatment lands:
// on the printing sold in that treatment wherever the shelves keep it -
// under another TCGplayer id, under the front face of a fused card, on
// the programme's own set - and on the card's plain printing only when no
// shelf carries the treatment.
func TestMatchProductFinishes(t *testing.T) {
	loadFabShelfDatastore(t)
	mkm := &Index{gameID: GameFleshAndBlood}
	for _, tt := range []struct {
		expansion, code, name, number, want string
	}{
		{"Rosetta", "ROS", "Will of Arcana (Rainbow Foil)", "000", "ros000_578820_rainbow"},
		{"Rosetta", "ROS", "Will of Arcana (Cold Foil)", "000", "ros000_577711_cold"},
		{"Dynasty", "DYN", "Silver (Regular)", "245", "dyn245_453353"},
		{"Dynasty", "DYN", "Silver (Rainbow Foil)", "245", "dyn245_453353"},
		{"Uprising", "UPR", "Aether Ashwing // Ash (Regular)", "042", "upr042-upr043_275840"},
		{"Uprising", "UPR", "Aether Ashwing // Ash (Cold Foil)", "042", "upr042_cold"},
		{"FAB Promos", "FAB", "Tide Flippers (Cold Foil Golden)", "298", "fab298"},
		{"FAB Promos", "FAB", "Imperial Seal of Command (Cold Foil)", "305", "fab305_cold"},
		{"Hero Promos", "HER", "Prism, Sculptor of Arc Light (Cold Foil)", "069", "her069_cold"},
		{"Silver Age Deck - Arakni, Web of Deceit", "SAR", "Graphene Chelicera (Regular)", "033", "sar033"},
		{"High Seas", "SEA", "Gravy Bones, Shipwrecked Looter (Regular)", "043", "sea043-sea044_638044"},
		{"High Seas", "SEA", "Gravy Bones, Shipwrecked Looter (Marvel)", "043", "sea043_624358_cold"},
		{"Heavy Hitters", "HVY", "Lead With Heart (Yellow) (Rainbow Foil)", "193", "hvy193_533598_rainbow"},
	} {
		product := MKMProduct{Name: tt.name, Number: tt.number, ExpansionName: tt.expansion, ExpansionCode: tt.code}
		if got := mkm.matchProduct(&product); got != tt.want {
			t.Errorf("matchProduct(%q, %q %s) = %q, want %q", tt.expansion, tt.name, tt.number, got, tt.want)
		}
	}
}

// TestResolveProductBridgeFinish pins the bridge's say against the
// product's: an id whose card carries the named treatment under another
// row is answered with that row, and an id whose card was never sold in
// it comes to rest on the card's plain printing, by name and so held
// back for the collector to fold beside the plain product.
func TestResolveProductBridgeFinish(t *testing.T) {
	loadFabShelfDatastore(t)
	mkm := &Index{gameID: GameFleshAndBlood, TCGBridge: map[int]int{1: 577711, 2: 453353, 3: 275840}}
	for _, tt := range []struct {
		product MKMProduct
		want    string
		byName  bool
	}{
		{MKMProduct{IDProduct: 1, Name: "Will of Arcana (Rainbow Foil)", Number: "000", ExpansionName: "Rosetta"}, "ros000_578820_rainbow", true},
		{MKMProduct{IDProduct: 2, Name: "Silver (Rainbow Foil)", Number: "245", ExpansionName: "Dynasty"}, "dyn245_453353", true},
		{MKMProduct{IDProduct: 3, Name: "Aether Ashwing // Ash (Cold Foil)", Number: "042", ExpansionName: "Uprising"}, "upr042_cold", true},
		{MKMProduct{IDProduct: 3, Name: "Aether Ashwing // Ash (Regular)", Number: "042", ExpansionName: "Uprising"}, "upr042-upr043_275840", false},
	} {
		got, _, byName, err := mkm.resolveProduct(&tt.product)
		if err != nil || got != tt.want || byName != tt.byName {
			t.Errorf("resolveProduct(%q) = %q byName=%v err=%v, want %q byName=%v", tt.product.Name, got, byName, err, tt.want, tt.byName)
		}
	}
}

// TestDisownBridged pins that a bridge landing on a card another priced
// product of the shelf names is given back to the name - the two Heralds
// the bridge crossed, the red and yellow Lead with Heart it swapped - and
// that a misspelt listing keeps its id when the spelling the shelf also
// sells under went unpriced.
func TestDisownBridged(t *testing.T) {
	loadFabShelfDatastore(t)
	mkm := &Index{gameID: GameFleshAndBlood}
	ravages := &MKMProduct{Name: "Herald of Ravages (Blue) (Regular)", Number: "017", ExpansionName: "Monarch - Prism Blitz Deck"}
	rebirth := &MKMProduct{Name: "Herald of Rebirth (Blue) (Regular)", Number: "018", ExpansionName: "Monarch - Prism Blitz Deck"}
	results := []resolved{
		{product: ravages, cardID: "psm018_238518", cardIDFoil: "psm018_238518"},
		{product: rebirth, cardID: "psm018_238518", cardIDFoil: "psm018_238518", byName: true},
	}
	mkm.disownBridged(results)
	if results[0].cardID != "psm017_238517" || !results[0].byName {
		t.Errorf("Herald of Ravages kept %q byName=%v, want psm017_238517 by name", results[0].cardID, results[0].byName)
	}

	red := &MKMProduct{Name: "Lead With Heart (Red) (Regular)", Number: "192", ExpansionName: "Heavy Hitters"}
	yellow := &MKMProduct{Name: "Lead With Heart (Yellow) (Regular)", Number: "193", ExpansionName: "Heavy Hitters"}
	results = []resolved{
		{product: red, cardID: "hvy193_533598", cardIDFoil: "hvy193_533598"},
		{product: yellow, cardID: "hvy192_533597", cardIDFoil: "hvy192_533597"},
	}
	mkm.disownBridged(results)
	if results[0].cardID != "hvy192_533597" || results[1].cardID != "hvy193_533598" {
		t.Errorf("Lead with Heart landed red on %q and yellow on %q, want each on its own", results[0].cardID, results[1].cardID)
	}

	typo := &MKMProduct{Name: "Herald of Rebirht (Blue) (Regular)", Number: "018", ExpansionName: "Monarch - Prism Blitz Deck"}
	results = []resolved{
		{product: typo, cardID: "psm018_238518", cardIDFoil: "psm018_238518"},
		{product: rebirth, err: errNoPrinting},
	}
	mkm.disownBridged(results)
	if results[0].cardID != "psm018_238518" || results[0].byName {
		t.Errorf("the misspelt listing was disowned to %q byName=%v, want its id kept", results[0].cardID, results[0].byName)
	}
}

// TestTwinsAmongFaces pins that a product naming one face of a fused card
// another product already holds is its twin, and refused quietly.
func TestTwinsAmongFaces(t *testing.T) {
	loadFabShelfDatastore(t)
	adult := &MKMProduct{Name: "Tuffnut, Bumbling Hulkster (Regular)", Number: "001", ExpansionName: "Super Slam"}
	young := &MKMProduct{Name: "Tuffnut (Regular)", Number: "002", ExpansionName: "Super Slam"}
	results := []resolved{
		{product: adult, cardID: "sup001-sup002_656933", cardIDFoil: "sup001-sup002_656933", byName: true},
		{product: young, err: errNoPrinting},
	}
	twinsAmong(results, fabSameProduct, fabFaceOf)
	if !errors.Is(results[1].err, errTwin) {
		t.Errorf("the young hero's refusal is %v, want errTwin beside the fused card", results[1].err)
	}

	collector := namedLast{add: func(responseChan) {}, twin: fabSameProduct, face: fabFaceOf}
	collector.collect(responseChan{cardID: "sup001-sup002_656933", product: adult})
	collector.collect(responseChan{cardID: "sup001-sup002_656933", product: young})
	if collector.twins != 1 {
		t.Errorf("the collector held %d twins, want the young hero to give way", collector.twins)
	}
}
