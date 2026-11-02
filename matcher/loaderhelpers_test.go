package matcher

import (
	"slices"
	"strings"
	"testing"
	"time"
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

// TestRaritiesRankAsSetsListThem pins the one spelling a card carries, a set
// lists and a search ranks a rarity by, and the words it reads back as.
func TestRaritiesRankAsSetsListThem(t *testing.T) {
	b := NewBackend()
	var cards []Card
	for _, published := range []string{"Common", "Super Rare", "Code Card", "super rare", "rare"} {
		cards = append(cards, Card{Rarity: b.AddRarity(published)})
	}
	b.Rarities = RarityNames([]string{"Super Rare", " Rare", "Common", ""})
	b.IndexRarities()

	got := RaritiesOf(cards, b.Rarities)
	want := []string{"superrare", "rare", "common", "codecard"}
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
		{"codecard", 0, false},
		{"mythic", 0, false},
	} {
		rank, ranked := b.RarityRank(tt.rarity)
		if rank != tt.rank || ranked != tt.ranked {
			t.Errorf("RarityRank(%q) = %d, %v, want %d, %v", tt.rarity, rank, ranked, tt.rank, tt.ranked)
		}
	}

	for rarity, label := range map[string]string{
		"superrare": "Super Rare",
		"codecard":  "Code Card",
		"common":    "Common",
		"rare":      "Rare",
		"mythic":    "Mythic",
	} {
		got := b.RarityLabel(rarity)
		if got != label {
			t.Errorf("RarityLabel(%q) = %q, want %q", rarity, got, label)
		}
	}
}

// TestDecodeEnvelope pins that the payload is read from "data", and that a
// bare document decodes to the zero payload a loader then refuses.
func TestDecodeEnvelope(t *testing.T) {
	type payload struct {
		Game string `json:"game"`
	}
	for _, tt := range []struct{ doc, want string }{
		{`{"meta":{},"data":{"game":"gundam"}}`, "gundam"},
		{`{"game":"gundam"}`, ""},
	} {
		got, err := DecodeEnvelope[payload](strings.NewReader(tt.doc))
		if err != nil {
			t.Fatalf("DecodeEnvelope(%s): %v", tt.doc, err)
		}
		if got.Game != tt.want {
			t.Errorf("DecodeEnvelope(%s).Game = %q, want %q", tt.doc, got.Game, tt.want)
		}
	}
	_, err := DecodeEnvelope[payload](strings.NewReader(`{"data":`))
	if err == nil {
		t.Error("DecodeEnvelope of a truncated document returned no error")
	}
}

// TestAddSet pins that a set is filed under its code, in Sets and AllSets,
// with its release date parsed.
func TestAddSet(t *testing.T) {
	b := NewBackend()
	b.AddSet(&Set{Code: "GD01", Name: "Newtype Rising", ReleaseDate: "2025-07-11"})
	set := b.Sets["GD01"]
	if set == nil || !slices.Equal(b.AllSets, []string{"GD01"}) {
		t.Fatalf("AddSet filed Sets %v, AllSets %v", b.Sets, b.AllSets)
	}
	want := time.Date(2025, 7, 11, 0, 0, 0, 0, time.UTC)
	if !set.ReleaseDateTime.Equal(want) {
		t.Errorf("ReleaseDateTime = %v, want %v", set.ReleaseDateTime, want)
	}
}

// TestPrintingsByName pins the Printings a name's cards share: keyed by the
// name normalized, its sets in the order the entries first name them, each
// once.
func TestPrintingsByName(t *testing.T) {
	entries := [][2]string{{"Zaku II", "ST01"}, {"Gundam", "GD01"}, {"zaku ii", "GD01"}, {"Zaku II", "ST01"}}
	got := PrintingsByName(entries, func(entry *[2]string) (string, string) {
		return entry[0], entry[1]
	})
	if len(got) != 2 {
		t.Errorf("PrintingsByName indexed %d names, want 2: %v", len(got), got)
	}
	want := []string{"ST01", "GD01"}
	zaku := got[Normalize("Zaku II")]
	if !slices.Equal(zaku, want) {
		t.Errorf("PrintingsByName for Zaku II = %v, want %v", zaku, want)
	}
}

// TestCompleteFilesSealed pins what Complete adds to a filled Backend: each
// set's rarities from its cards, and the sealed products whose set is known,
// the rest dropped.
func TestCompleteFilesSealed(t *testing.T) {
	b := NewBackend()
	b.AddSet(&Set{Code: "GD01", Name: "Newtype Rising"})
	b.Sets["GD01"].Cards = []Card{{UUID: "gd01-001", Rarity: "rare"}}
	b.Rarities = []string{"rare"}
	b.Complete(candidateTestRules{}, []DatastoreSealed{
		{ID: "gd01-box", Name: "Newtype Rising Booster Box", SetCode: "GD01"},
		{ID: "xx-box", Name: "Unknown Booster Box", SetCode: "XX"},
	})
	rarities := b.Sets["GD01"].Rarities
	if !slices.Equal(rarities, []string{"rare"}) {
		t.Errorf("GD01 Rarities = %v, want [rare]", rarities)
	}
	if len(b.SetSealedUUIDs) != 1 || !slices.Equal(b.SetSealedUUIDs["GD01"], []string{"gd01-box"}) {
		t.Errorf("SetSealedUUIDs = %v, want GD01's box alone", b.SetSealedUUIDs)
	}
}

// TestPromoTypeSlugs pins a printing's promo types as the tokens it stores,
// and nil for one that publishes none.
func TestPromoTypeSlugs(t *testing.T) {
	none := PromoTypeSlugs(nil)
	if none != nil {
		t.Errorf("PromoTypeSlugs(nil) = %#v, want nil", none)
	}
	got := PromoTypeSlugs([]string{"Link Rare", "1st Place"})
	want := []string{"linkrare", "1stplace"}
	if !slices.Equal(got, want) {
		t.Errorf("PromoTypeSlugs = %v, want %v", got, want)
	}
}

// TestLabelPromoType pins a token's words: the game's table where it has an
// entry, and the title-cased token where it has none.
func TestLabelPromoType(t *testing.T) {
	labels := map[string]string{"linkrare": "Link Rare"}
	for _, tt := range []struct{ slug, want string }{
		{"linkrare", "Link Rare"},
		{"finalist", "Finalist"},
	} {
		got := LabelPromoType(labels, tt.slug)
		if got != tt.want {
			t.Errorf("LabelPromoType(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}
