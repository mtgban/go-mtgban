package bazaarofmagic

import "testing"

func TestSealedIndexFind(t *testing.T) {
	b := realDatastore(t)
	index := newSealedIndex(b)

	tests := []struct {
		listing string
		want    string // the datastore's name, or "" for no match
	}{
		{"Magic: The Gathering - Star Trek Play Booster Box (30 boosters)", "Star Trek Play Booster Box"},
		{"Boosterbox Theros (36 boosters)", "Theros Booster Box"},
		{"Fat Pack Journey into Nyx", "Journey into Nyx Fat Pack"},
		{"Commander 2014: Peer Through Time", "Commander 2014 Commander Deck Peer Through Time"},
		{"Duel Decks: Mind vs. Might", "Duel Decks: Mind vs. Might"},
		// A booster of no kind on a set with several is its draft booster.
		{"Booster Throne of Eldraine", "Throne of Eldraine Draft Booster Pack"},
		// The deck, not the kit of the same name.
		{"Magic: The Gathering - Marvel Super Heroes Commander Deck: Avengers Assemble", "Marvel Super Heroes Commander Deck Avengers Assemble"},
		{"Magic: The Gathering - Tarkir: Dragonstorm Prerelease Pack (set of 5)", "Tarkir: Dragonstorm Prerelease Kit Set of 5"},
		{"Magic: The Gathering - Secret Lair Drop Series: Secret Lair x Fallout: Beyond Vault 33 (foil)", "Secret Lair Drop Fallout Beyond Vault 33 Foil"},
		// Two drops whose names are the same words in another order.
		{"Magic: The Gathering - Secret Lair Drop Series: Cats Are Better Than Dogs", "Secret Lair Drop Cats Are Better Than Dogs"},
		{"Magic: The Gathering - Secret Lair Drop Series: Dogs Are Better Than Cats (foil)", "Secret Lair Drop Dogs Are Better Than Cats Rainbow Foil"},
		// The datastore holds neither a Japanese box nor a land pack.
		{"Magic: The Gathering - Final Fantasy Collector Booster Box (12 boosters - Japanese)", ""},
		{"Foil Basic Land Pack Kaldheim (40 cards)", ""},
	}
	for _, test := range tests {
		uuid, found := index.find(test.listing)
		var got string
		if found {
			co, err := b.GetUUID(uuid)
			if err != nil {
				t.Fatal(err)
			}
			got = co.Name
		}
		if got != test.want {
			t.Errorf("%q: got %q, want %q", test.listing, got, test.want)
		}
	}
}
