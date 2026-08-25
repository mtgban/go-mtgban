package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestIsToken pins the half of the answer the datastore's own token list
// cannot give: the sets dropped when it is built, and the storefront
// spellings that decorate a carried token. It also pins the names the
// heuristic used to claim wrongly - real cards whose spelling brushes past a
// token word, and sealed products naming a card count - which now belong to
// nobody.
func TestIsToken(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		// Only the dropped sets know these
		{"Rules Tip Card", true},
		{"Tip: Something", true},
		{"Build a Deck: Something", true},
		{"Blank", true},
		{"On An Adventure", true},
		{"Magic Minigame: Cats vs Dogs", true},
		{"Arena Code Card", true},
		{"Decklist Card", true},
		{"Innistrad Checklist Card", true},
		// The face of a token the datastore only files whole
		{"Bounty: The Outsider", true},
		{"The Initiative", true},
		// Storefront spellings of a carried token
		{"Angel Token", true},
		{"Angel Token Token", true},
		{"Ajani Steadfast Emblem Token", true},
		// Real cards the dropped arms used to claim
		{"Copy Artifact", false},
		{"Copy Enchantment", false},
		{"Copycrook", false},
		{"Pirated Copy", false},
		{"Bounty Agent", false},
		{"Bounty of Might", false},
		{"Bounty Board", false},
		{"Earth's Mightiest Emblem", false},
		{"Tremblement de terre", false},
		{"Stella Lee, Wild Card", false},
		{"Deadpool, Trading Card", false},
		{"Sidequest: Card Collection", false},
		{"Emblem of the Warmind", false},
		{"Leering Emblem", false},
		{"Kavu Monarch", false},
		{"Our Market Research", false},
		{"Lightning Bolt", false},
		// Sealed products naming a card count
		{"Zendikar Six Card Booster Pack", false},
		{"Prophecy Theme Deck Slither", false},
		{"Mirrodin Theme Deck Little Bashers", false},
		{"Secrets of Strixhaven 60-Card Theme Deck Eerie", false},
	} {
		if got := (Rules{}).IsToken(nil, tt.name); got != tt.want {
			t.Errorf("IsToken(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
	// And the same answers through the loaded backend, which is how Match
	// asks: the hook is only reached once the rules are attached.
	if !mtgmatcher.IsToken("Rules Tip Card") {
		t.Error("the backend does not reach the game's own token names")
	}
	if mtgmatcher.IsToken("Lightning Bolt") {
		t.Error("a real card reads as a token")
	}
	var _ mtgmatcher.GameRules = Rules{}
}

// TestIsTokenDatastoreAnswersTheRest pins why the dropped arms could be
// dropped: the datastore carries those names itself, so the backend still
// reads them as tokens even though the heuristic no longer claims them.
func TestIsTokenDatastoreAnswersTheRest(t *testing.T) {
	for _, name := range []string{
		"Copy",
		"The Monarch",
		"Dungeon of the Mad Mage",
		"Giant Teddy Bear",
		"Ajani Steadfast Emblem",
		"Teferi, Temporal Archmage Emblem",
		"Elspeth, Sun's Champion Emblem",
		"Wrenn and Six Emblem",
	} {
		if (Rules{}).IsToken(nil, name) {
			t.Errorf("IsToken(%q) still answers from the heuristic", name)
		}
		if !mtgmatcher.IsToken(name) {
			t.Errorf("the datastore does not carry %q as a token", name)
		}
	}
}

// TestNarrowedIsTokenReachesRealCards pins the cards the wide heuristic threw
// away before they could be looked up: a face name, a printed foreign name and
// a Secret Lair flavor name, each of which merely spelled a token word. The
// matcher only asks IsToken once a name misses the canonical index, which is
// exactly where these three land, so claiming them meant answering
// ErrUnsupported instead of resolving them.
func TestNarrowedIsTokenReachesRealCards(t *testing.T) {
	for _, tt := range []struct {
		name, edition, want string
	}{
		// Arcane Signet's Marvel flavor name, tripped Contains("Emblem")
		{"Earth's Mightiest Emblem", "Secret Lair Drop", "b097a7b6-0700-5073-8fca-9572cddac774"},
		// Earthquake's French printed name, also inside it: trEMBLEMent
		{"Tremblement de terre", "Foreign Black Border", "f569bd42-1555-5534-a39d-4532871e424e"},
		// The front face of a Final Fantasy card, tripped Contains("Card ")
		{"Sidequest: Card Collection", "Final Fantasy", "26f46851-383f-5188-949c-9b4fd8c5d05d"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			in := mtgmatcher.InputCard{Name: tt.name, Edition: tt.edition}
			id, err := testBackend.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", in, err)
			}
			if id != tt.want {
				co, _ := testBackend.GetUUID(id)
				t.Errorf("Match(%v) = %s (%v), want %s", in, id, co, tt.want)
			}
		})
	}
}
