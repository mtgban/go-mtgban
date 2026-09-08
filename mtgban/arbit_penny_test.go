package mtgban

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

func pennyCard(co *mtgmatcher.CardObject) map[string]*mtgmatcher.CardObject {
	return map[string]*mtgmatcher.CardObject{"card": co}
}

func pennySeller(price float64, conditions string) Seller {
	return sellerOf(InventoryRecord{
		"card": {{Conditions: conditions, Price: price, Quantity: 1}},
	}, ScraperInfo{Name: "seller"})
}

// TestPennystockNamesWhatCanFall pins which cards the report is about. A card
// priced at the floor is only worth naming if it had somewhere to fall from,
// so the rarities and the treatments that are cheap for reasons which will
// not change are left out.
func TestPennystockNamesWhatCanFall(t *testing.T) {
	for _, tt := range []struct {
		desc  string
		card  *mtgmatcher.CardObject
		price float64
		full  bool
		want  bool
	}{
		{"a mythic at the floor is named without asking for the full report",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Big", Rarity: "mythic"}}, 0.10, false, true},
		{"the same mythic above the floor is not",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Big", Rarity: "mythic"}}, 0.50, false, false},
		{"a rare needs the full report",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Small", Rarity: "rare"}}, 0.01, false, false},
		{"and is named once it is asked for",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Small", Rarity: "rare"}}, 0.01, true, true},
		{"a common is never named, however cheap",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Filler", Rarity: "common"}}, 0.01, true, false},
		{"a basic land is named on its own account",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Island", Rarity: "common", IsFullArt: true}}, 0.01, true, true},
		{"a promo is named by the flag it carries",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Handout", Rarity: "common", IsPromo: true}}, 0.01, true, true},
		{"or by the edition it is filed in",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Handout", Rarity: "common"}, Edition: "Whatever Promos"}, 0.01, true, true},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, pennyCard(tt.card))
			got := len(Pennystock(pennySeller(tt.price, "NM"), tt.full)) > 0
			if got != tt.want {
				t.Errorf("Pennystock named it = %v, want %v", got, tt.want)
			}
		})
	}
}

// The borders and treatments that are cheap for reasons which will not change
// are skipped even when everything else about the card qualifies.
func TestPennystockSkipsWhatStaysCheap(t *testing.T) {
	for _, tt := range []struct {
		desc string
		card *mtgmatcher.CardObject
	}{
		{"a gold border", &mtgmatcher.CardObject{
			Card: mtgmatcher.Card{Name: "Big", Rarity: "mythic", BorderColor: "gold"}}},
		{"a silver border", &mtgmatcher.CardObject{
			Card: mtgmatcher.Card{Name: "Big", Rarity: "mythic", BorderColor: "silver"}}},
		{"a white border", &mtgmatcher.CardObject{
			Card: mtgmatcher.Card{Name: "Big", Rarity: "mythic", BorderColor: "white"}}},
		{"a funny card", &mtgmatcher.CardObject{
			Card: mtgmatcher.Card{Name: "Big", Rarity: "mythic", IsFunny: true}}},
		{"a thick display card", &mtgmatcher.CardObject{
			Card: mtgmatcher.Card{Name: "Big", Rarity: "mythic",
				PromoTypes: []string{magic.PromoTypeThickDisplay}}}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, pennyCard(tt.card))
			if got := Pennystock(pennySeller(0.10, "NM"), true); len(got) != 0 {
				t.Errorf("Pennystock named %d entries, want none", len(got))
			}
		})
	}
}

// A copy already worn is not a card priced at the floor, it is a worn copy.
func TestPennystockSkipsAWornCopy(t *testing.T) {
	for _, conditions := range []string{"HP", "PO"} {
		t.Run(conditions, func(t *testing.T) {
			installCards(t, pennyCard(&mtgmatcher.CardObject{
				Card: mtgmatcher.Card{Name: "Big", Rarity: "mythic"},
			}))
			if got := Pennystock(pennySeller(0.10, conditions), true); len(got) != 0 {
				t.Errorf("Pennystock named %d entries, want none", len(got))
			}
		})
	}
}

// TestPennystockThresholds pins the override: a caller's ceiling replaces the
// default in its position, a zero leaves that position alone, and more
// ceilings than there are positions are ignored rather than panicking.
func TestPennystockThresholds(t *testing.T) {
	mythic := func() map[string]*mtgmatcher.CardObject {
		return pennyCard(&mtgmatcher.CardObject{
			Card: mtgmatcher.Card{Name: "Big", Rarity: "mythic"},
		})
	}

	installCards(t, mythic())
	if got := Pennystock(pennySeller(0.50, "NM"), false); len(got) != 0 {
		t.Fatal("the default ceiling named a card above it")
	}

	installCards(t, mythic())
	if got := Pennystock(pennySeller(0.50, "NM"), false, 1.00); len(got) != 1 {
		t.Errorf("a raised ceiling named %d entries, want 1", len(got))
	}

	installCards(t, mythic())
	if got := Pennystock(pennySeller(0.10, "NM"), false, 0); len(got) != 1 {
		t.Errorf("a zero ceiling did not leave the default alone: %d entries", len(got))
	}

	// Six positions exist; a seventh has nowhere to go and must not reach
	// past the end of the slice.
	installCards(t, mythic())
	if got := Pennystock(pennySeller(0.10, "NM"), true, 1, 1, 1, 1, 1, 1, 1, 1); len(got) != 1 {
		t.Errorf("more ceilings than positions named %d entries, want 1", len(got))
	}
}

// A card the datastore does not hold is skipped rather than reported unnamed.
func TestPennystockSkipsAnUnknownCard(t *testing.T) {
	installCards(t, map[string]*mtgmatcher.CardObject{})
	if got := Pennystock(pennySeller(0.01, "NM"), true); len(got) != 0 {
		t.Errorf("Pennystock named %d entries, want none", len(got))
	}
}
