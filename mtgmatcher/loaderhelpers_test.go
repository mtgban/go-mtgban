package mtgmatcher

import (
	"slices"
	"testing"
)

// TestPlainOrdinal pins the reduction the games numbering their cards
// behind a set code share: the ordinal, unpadded, and nothing for a number
// that carries none.
func TestPlainOrdinal(t *testing.T) {
	for _, tt := range []struct{ number, want string }{
		{"OP01-007", "7"},
		{"OP01-001a", "1"},
		{"WTR007", "7"},
		{"YS13-ENV07", "7"},
		{"GD01-001", "1"},
		{"P-041", "41"},
		{"000", "0"},
		{"LEADER", ""},
		{"DON", ""},
		{"", ""},
	} {
		if got := PlainOrdinal(tt.number); got != tt.want {
			t.Errorf("PlainOrdinal(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
	for _, tt := range []struct{ number, want string }{
		{"007", "7"}, {"0", "0"}, {"00", "0"}, {"", ""}, {"12", "12"},
	} {
		if got := CanonicalTail(tt.number); got != tt.want {
			t.Errorf("CanonicalTail(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
}

// TestProductKeyOf pins that a printing folds onto the product id it
// publishes and stands for itself where it publishes none.
func TestProductKeyOf(t *testing.T) {
	if got := ProductKeyOf(map[string]string{"tcgplayerProductId": "531486"}, "p-041_531486"); got != "531486" {
		t.Errorf("ProductKeyOf with an id = %q, want the id", got)
	}
	if got := ProductKeyOf(nil, "op01-003_ct277276"); got != "op01-003_ct277276" {
		t.Errorf("ProductKeyOf without an id = %q, want the uuid", got)
	}
	for _, tt := range []struct {
		values []string
		want   []string
	}{
		{nil, nil},
		{[]string{"Red"}, []string{"red"}},
		{[]string{"Grass", "Darkness"}, []string{"grass", "darkness"}},
		{[]string{"", "DARK "}, []string{"dark"}},
	} {
		got := ColorNames(tt.values)
		if !slices.Equal(got, tt.want) {
			t.Errorf("ColorNames(%q) = %v, want %v", tt.values, got, tt.want)
		}
	}
}

// TestColorsOfListsTheTermsLast pins a set's colours: in the game's order, or
// alphabetically without one, then colorless and multicolor the way Magic's
// sets list them.
func TestColorsOfListsTheTermsLast(t *testing.T) {
	cards := []Card{
		{Colors: []string{"red"}},
		{Colors: []string{"green", "blue"}},
		{Colors: nil},
		{Colors: []string{"colorless"}},
		{Colors: []string{"purple"}},
	}
	for _, tt := range []struct {
		order []string
		want  []string
	}{
		{[]string{"red", "green", "blue"}, []string{"red", "green", "blue", "purple", "colorless", "multicolor"}},
		{nil, []string{"blue", "green", "purple", "red", "colorless", "multicolor"}},
	} {
		got := ColorsOf(cards, tt.order)
		if !slices.Equal(got, tt.want) {
			t.Errorf("ColorsOf with %v = %v, want %v", tt.order, got, tt.want)
		}
	}
}

// TestRaritiesRankAsSetsListThem pins the one spelling a set lists and a
// search ranks a rarity by, whatever spelling the card carries.
func TestRaritiesRankAsSetsListThem(t *testing.T) {
	b := &Backend{
		UUIDs: map[string]*CardObject{
			"a": {Card: Card{Rarity: "Super Rare"}},
			"b": {Card: Card{Rarity: "Common"}},
			"c": {Card: Card{Rarity: "Code Card"}},
		},
		Rarities: RarityNames([]string{"Super Rare", " Rare", "Common", ""}),
	}
	b.IndexRarities()

	cards := []Card{{Rarity: "Common"}, {Rarity: "Super Rare"}, {Rarity: "Code Card"}}
	got := RaritiesOf(cards, b.Rarities)
	want := []string{"superrare", "common", "codecard"}
	if !slices.Equal(got, want) {
		t.Errorf("RaritiesOf = %v, want %v", got, want)
	}

	for _, tt := range []struct {
		rarity string
		rank   int
		ranked bool
	}{
		{"Super Rare", 0, true},
		{"superrare", 0, true},
		{"SUPER RARE", 0, true},
		{"rare", 1, true},
		{"Common", 2, true},
		{"Code Card", 0, false},
		{"mythic", 0, false},
	} {
		rank, ranked := b.RarityRank(tt.rarity)
		if rank != tt.rank || ranked != tt.ranked {
			t.Errorf("RarityRank(%q) = %d, %v, want %d, %v", tt.rarity, rank, ranked, tt.rank, tt.ranked)
		}
	}
}

// TestRarityQueryReadsLetters pins what a search's rarity words read as: a
// letter names the game's rarities it begins, and a word one rarity.
func TestRarityQueryReadsLetters(t *testing.T) {
	mythic := &Backend{Rarities: []string{"oversize", "special", "mythic", "rare", "uncommon", "common", "token"}}
	lorcana := &Backend{Rarities: RarityNames([]string{"Special", "Enchanted", "Super Rare", "Rare", "Common"})}
	for _, tt := range []struct {
		b     *Backend
		query string
		want  []string
	}{
		{mythic, "m,C", []string{"mythic", "common"}},
		{mythic, "o", []string{"oversize"}},
		{mythic, "Super Rare", []string{"superrare"}},
		{lorcana, "s", []string{"special", "superrare"}},
		{lorcana, "r,e", []string{"rare", "enchanted"}},
		{lorcana, "x", []string{"x"}},
	} {
		got := tt.b.RarityQuery(tt.query)
		if !slices.Equal(got, tt.want) {
			t.Errorf("RarityQuery(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}
