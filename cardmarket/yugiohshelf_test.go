package cardmarket

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// yugiohShelfDatastore is the published Yu-Gi-Oh datastore cut down to the
// printings these tests turn on: Tri-Horned Dragon in both runs of the set
// the storefront names without its article, its European print in the set
// the catalog keeps for that run, and a Mega-Tin card numbered with the
// region infix the storefront leaves out.
const yugiohShelfDatastore = `{
 "game": "yugioh",
 "sets": {
  "LOB": {"abbreviation": "LOB", "name": "The Legend of Blue Eyes White Dragon", "releaseDate": "2002-03-08"},
  "LOB-EN": {"abbreviation": "LOB-EN", "name": "Legend of Blue Eyes White Dragon (Worldwide English)", "releaseDate": "2002-03-08"},
  "MP18": {"abbreviation": "MP18", "name": "2018 Mega-Tins Mega Pack", "releaseDate": "2018-08-30"}
 },
 "cards": [
  {"finish": "1st Edition", "id": "lob-000_22538_1e", "name": "Tri-Horned Dragon", "number": "LOB-000", "rarity": "Secret Rare", "setCode": "LOB"},
  {"finish": "Unlimited", "id": "lob-000_22538_unl", "name": "Tri-Horned Dragon", "number": "LOB-000", "rarity": "Secret Rare", "setCode": "LOB"},
  {"finish": "Unlimited", "id": "lob-en000_1", "name": "Tri-Horned Dragon", "number": "LOB-EN000", "rarity": "Secret Rare", "setCode": "LOB-EN"},
  {"finish": "1st Edition", "id": "mp18-en065_1", "name": "Topologic Bomber Dragon", "number": "MP18-EN065", "rarity": "Prismatic Secret Rare", "setCode": "MP18"}
 ]
}`

// TestMatchYugiohShelves pins that the name path reaches the sets the
// storefront names its own way, reads the run off the version index, puts
// the region infix back on a number, follows a European number to the set
// kept for that print, and says which kind of miss a miss is.
func TestMatchYugiohShelves(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(yugiohShelfDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Index{gameID: GameYuGiOh}
	for _, tt := range []struct {
		expansion, name, number, want string
		err                           error
	}{
		{"Legend of Blue Eyes White Dragon", "Tri-Horned Dragon (V.1 - Secret Rare)", "000", "lob-000_22538_unl", nil},
		{"Legend of Blue Eyes White Dragon", "Tri-Horned Dragon (V.2 - Secret Rare)", "000", "lob-000_22538_1e", nil},
		{"Legend of Blue Eyes White Dragon", "Tri-Horned Dragon (V.4 - Secret Rare)", "EN000", "lob-en000_1", nil},
		{"Legend of Blue Eyes White Dragon", "Tri-Horned Dragon (V.3 - Secret Rare)", "A000", "", errForeign},
		{"2018 Mega-Tin Mega Pack", "Topologic Bomber Dragon", "065", "mp18-en065_1", nil},
		{"2018 Mega-Tin Mega Pack", "Gouki Re-Match", "070", "", errNoPrinting},
		{"Legend of Blue Eyes White Dragon (LDD)", "Tri-Horned Dragon", "000", "", errForeign},
	} {
		product := MKMProduct{Name: tt.name, Number: tt.number, ExpansionName: tt.expansion}
		got, err := mkm.matchYugioh(&product)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("%q in %q (%s) = %q, %v; want %q, %v", tt.name, tt.expansion, tt.number, got, err, tt.want, tt.err)
		}
	}
}

func TestYugiohSameProduct(t *testing.T) {
	for _, tt := range []struct {
		a, b MKMProduct
		want bool
	}{
		{MKMProduct{Name: "Feral Imp (V.1 - Common)", Number: "001"}, MKMProduct{Name: "Feral Imp (V.2 - Common)", Number: "001"}, true},
		{MKMProduct{Name: "Feral Imp (V.3 - Common)", Number: "EN001"}, MKMProduct{Name: "Feral Imp (V.2 - Common)", Number: "001"}, true},
		{MKMProduct{Name: "Tri-Horned Dragon", Number: ""}, MKMProduct{Name: "Tri-Horned Dragon", Number: ""}, true},
		{MKMProduct{Name: "Harpie Lady (V.1 - Common)", Number: "008"}, MKMProduct{Name: "Harpie Lady Sisters (V.1 - Super Rare)", Number: "009"}, false},
		{MKMProduct{Name: "Griggle (V.2 - Common)", Number: "016"}, MKMProduct{Name: "Rescue-ACE Monitor", Number: "279"}, false},
	} {
		if got := yugiohSameProduct(&tt.a, &tt.b); got != tt.want {
			t.Errorf("%q/%q vs %q/%q = %v, want %v", tt.a.Name, tt.a.Number, tt.b.Name, tt.b.Number, got, tt.want)
		}
	}
}

func TestYugiohEditions(t *testing.T) {
	for _, tt := range []struct {
		expansion string
		want      string
	}{
		{"Duelist League 18", "DL18"},
		{"Champion Pack: Game Three", "CP03"},
		{"Astral Pack Eight", "AP08"},
		{"Collector's Tins 2010", "2010 Collectors Tin"},
		{"2017 Mega-Tin Mega Pack", "2017 Mega-Tins Mega Pack"},
		{"Legend of Blue Eyes White Dragon", "LOB"},
		{"Spell Ruler (SDM)", "Spell Ruler (SDM)"},
	} {
		if got := yugiohEditions(tt.expansion)[0]; got != tt.want {
			t.Errorf("%q -> %q, want %q", tt.expansion, got, tt.want)
		}
	}
}
