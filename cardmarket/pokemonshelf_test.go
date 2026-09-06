package cardmarket

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
)

// pokemonShelfDatastore is the published Pokemon datastore cut down to the
// printings these tests turn on, every row copied from it: the SM04 Pikachu
// of SM Promos, whose number the storefront writes without its programme,
// and the Rowlet of SM Base Set, a set the storefront names "Sun & Moon".
const pokemonShelfDatastore = `{
 "game": "pokemon",
 "sets": {"SMP": {"abbreviation": "SMP", "name": "SM Promos", "releaseDate": "2016-11-18", "type": "promo"}, "SM01": {"abbreviation": "SM01", "name": "SM Base Set", "releaseDate": "2017-02-03"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 127160}, "finish": "Holofoil", "id": "sm04_127160_holo", "name": "Pikachu", "number": "SM04", "rarity": "Promo", "setCode": "SMP"},
  {"externalLinks": {"tcgPlayerId": 126880}, "finish": "Normal", "id": "9-149_126880", "name": "Rowlet", "number": "9", "rarity": "Common", "setCode": "SM01"},
  {"externalLinks": {"tcgPlayerId": 126880}, "finish": "Reverse Holofoil", "id": "9-149_126880_reverse", "name": "Rowlet", "number": "9", "rarity": "Common", "setCode": "SM01"}
 ]
}`

// TestMatchPokemonShelves pins that the name path reaches the sets the
// storefront names its own way, with the programme put back on a promo's
// number, and says which kind of miss a miss is: a set we carry that holds
// no such card, or a catalog we do not carry at all.
func TestMatchPokemonShelves(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(pokemonShelfDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Index{gameID: GamePokemon}
	for _, tt := range []struct {
		expansion, name, number, want string
		err                           error
	}{
		{"SM Black Star Promos", "Pikachu ", "04", "sm04_127160_holo", nil},
		{"Sun & Moon", "Rowlet ", "9", "9-149_126880", nil},
		{"Sun & Moon", "Rowlet [Tackle | Leafage]", "9", "9-149_126880", nil},
		{"Sun & Moon", "Litten", "24", "", errNoPrinting},
		{"Shiny Treasure ex", "Pikachu", "04", "", errForeign},
		{"XY Promos", "Pikachu", "", "", errForeign},
	} {
		product := MKMProduct{Name: tt.name, Number: tt.number, ExpansionName: tt.expansion}
		got, err := mkm.matchPokemon(&product)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("%q in %q (%s) = %q, %v; want %q, %v", tt.name, tt.expansion, tt.number, got, err, tt.want, tt.err)
		}
	}
}

// TestPokemonTwins pins that a name reaching a printing a product of the
// same name already holds gives way, that the first of two named siblings
// holds the printing for both, and that a refusal beside such a sibling is
// counted as a twin rather than reported.
func TestPokemonTwins(t *testing.T) {
	product := func(id int, name, number string) *MKMProduct {
		return &MKMProduct{IDProduct: id, Name: name, Number: number, ExpansionName: "Sun & Moon"}
	}
	results := []resolved{
		{product: product(1, "Rowlet ", "9"), cardID: "rowlet"},
		{product: product(2, "Rowlet ", "9"), cardID: "rowlet", byName: true},
		{product: product(3, "Rowlet (Theme Deck)", "9P"), cardID: "rowlet", byName: true},
		{product: product(4, "Rowlet", ""), err: errNoPrinting},
		{product: product(5, "Litten ", "24"), cardID: "litten", byName: true},
		{product: product(6, "Litten ", "24"), cardID: "litten", byName: true},
		{product: product(7, "Torracat ", "25"), cardID: "torracat", byName: true},
		{product: product(8, "Popplio ", "39"), err: errNoPrinting},
	}
	twinsAmong(results, pokemonSameProduct, nil)
	want := map[int]error{2: errTwin, 3: errTwin, 4: errTwin, 6: errTwin, 8: errNoPrinting}
	for _, r := range results {
		if !errors.Is(r.err, want[r.product.IDProduct]) {
			t.Errorf("product %d: err = %v, want %v", r.product.IDProduct, r.err, want[r.product.IDProduct])
		}
		if want[r.product.IDProduct] != nil && r.cardID != "" {
			t.Errorf("product %d: still holds %q", r.product.IDProduct, r.cardID)
		}
	}
}

func TestPokemonSameProduct(t *testing.T) {
	for _, tt := range []struct {
		a, b MKMProduct
		want bool
	}{
		{MKMProduct{Name: "Galvantula ", Number: "27"}, MKMProduct{Name: "Galvantula (Theme Deck)", Number: "27P"}, true},
		{MKMProduct{Name: "Toxtricity V ", Number: "70"}, MKMProduct{Name: "Toxtricity V ", Number: "070"}, true},
		{MKMProduct{Name: "Darkness Energy", Number: ""}, MKMProduct{Name: "Basic Darkness Energy", Number: ""}, true},
		{MKMProduct{Name: "Flareon V ", Number: "149"}, MKMProduct{Name: "Flareon V ", Number: "SWSH 149"}, true},
		{MKMProduct{Name: "Magnetic [M] Energy", Number: "085"}, MKMProduct{Name: "Magnetic Metal Energy", Number: "85"}, true},
		{MKMProduct{Name: "Pikachu ", Number: "101"}, MKMProduct{Name: "Pikachu ", Number: "190"}, false},
		{MKMProduct{Name: "Judge", Number: "SVI 176"}, MKMProduct{Name: "Judge", Number: "DRI 167"}, false},
		{MKMProduct{Name: "Latios ", Number: "MEG 101"}, MKMProduct{Name: "Lunatone ", Number: "MEG 074"}, false},
	} {
		if got := pokemonSameProduct(&tt.a, &tt.b); got != tt.want {
			t.Errorf("%q/%q vs %q/%q = %v, want %v", tt.a.Name, tt.a.Number, tt.b.Name, tt.b.Number, got, tt.want)
		}
	}
}
