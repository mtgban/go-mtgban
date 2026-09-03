package pokemon

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestNameRespelling pins the names the catalog misspells reaching the card
// the datastore carries under the corrected spelling. Both spellings are
// asserted, and both are asserted to reach the same uuid: a storefront
// copying the catalog goes on writing the misspelling long after the
// datastore stops, and the pair is what keeps it resolving.
//
// The uuid is what is pinned rather than the name, so these hold whether or
// not the datastore under test has taken the correction yet.
func TestNameRespelling(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"the catalog's spelling of the Pokemon reaches it",
			mtgmatcher.InputCard{Name: "Dark Exeggcutor", Edition: "Neo Destiny", Variation: "033 033/105"},
			"033-105_84593_unl"},
		{"and so does the Pokemon's own",
			mtgmatcher.InputCard{Name: "Dark Exeggutor", Edition: "Neo Destiny", Variation: "033 033/105"},
			"033-105_84593_unl"},
		{"the catalog's spelling of Drowzee reaches it",
			mtgmatcher.InputCard{Name: "Drowsee", Edition: "EX FireRed & LeafGreen", Variation: "32 32/112"},
			"32-112_84973"},
		{"and so does the Pokemon's own",
			mtgmatcher.InputCard{Name: "Drowzee", Edition: "EX FireRed & LeafGreen", Variation: "32 32/112"},
			"32-112_84973"},
		// Impostor Professor Oak is not a respelling and must not become
		// one: Wizards printed the Base Set card "Impostor" and the Base
		// Set 2 and Celebrations reprints "Imposter", so both spellings
		// name real cards and each has to reach its own printing.
		{"the Base Set printing keeps the spelling it was printed with",
			mtgmatcher.InputCard{Name: "Impostor Professor Oak", Edition: "Base Set", Variation: "073 073/102"},
			"073-102_86271"},
		{"and the Base Set 2 reprint keeps its own",
			mtgmatcher.InputCard{Name: "Imposter Professor Oak", Edition: "Base Set 2", Variation: "102 102/130"},
			"102-130_42552"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}
