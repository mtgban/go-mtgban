package magic

import (
	"slices"
	"testing"
	"time"
)

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
		return Card{Name: "Plains", Number: number, UUID: "plains-" + number, Printings: printings}
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
}
