package pokemon

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// formerIDsData is a cut of the datastore shape: Jolteon is sold under
// 653353 and was sold under 611828 until TCGplayer merged the two; Red's
// Pikachu was moved out of the category, so it is minted naming only the
// product it was; Eevee claims the live Jolteon product as a former one,
// and Vaporeon claims Jolteon's former product after it.
const formerIDsData = `{"meta": {"version": "1"}, "data": {
  "game": "pokemon",
  "sets": {
    "SVP": {"name": "SV: Scarlet & Violet Promo Cards", "type": "promo"},
    "SMP": {"name": "SM Promos", "type": "promo"}
  },
  "cards": [
    {"id": "169_653353_holofoil", "name": "Jolteon", "setCode": "SVP", "number": "169", "rarity": "Promo", "finish": "Holofoil",
     "externalLinks": {"tcgPlayerId": 653353, "formerTcgPlayerIds": [611828]}},
    {"id": "270-sm-p_489919_holofoil", "name": "Red's Pikachu", "setCode": "SMP", "number": "270", "total": "SM-P", "rarity": "Promo", "finish": "Holofoil",
     "externalLinks": {"formerTcgPlayerIds": [489919]}},
    {"id": "sm184_609698_holofoil", "name": "Eevee", "setCode": "SMP", "number": "SM184", "rarity": "Promo", "finish": "Holofoil",
     "externalLinks": {"tcgPlayerId": 609698, "formerTcgPlayerIds": [653353]}},
    {"id": "sm185_609699_holofoil", "name": "Vaporeon", "setCode": "SMP", "number": "SM185", "rarity": "Promo", "finish": "Holofoil",
     "externalLinks": {"tcgPlayerId": 609699, "formerTcgPlayerIds": [611828]}}
  ]
}}`

func TestFormerProductIDs(t *testing.T) {
	b, err := Load(strings.NewReader(formerIDsData))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id, finish, want string
	}{
		{"611828", "Holofoil", "169_653353_holofoil"},
		{"489919", "Holofoil", "270-sm-p_489919_holofoil"},
		// A product TCGplayer sells keeps its own card.
		{"653353", "Holofoil", "169_653353_holofoil"},
		// A former product claimed twice keeps its first card.
		{"611828", "Holofoil", "169_653353_holofoil"},
	} {
		got, err := b.Match(&matcher.InputCard{ID: tc.id, Finish: tc.finish, Foil: true})
		if err != nil || got != tc.want {
			t.Errorf("Match(%s, %s) = (%q, %v), want %q", tc.id, tc.finish, got, err, tc.want)
		}
	}
	// A former id is not the card's product: links and prices read that.
	co, err := b.GetUUID("270-sm-p_489919_holofoil")
	if err != nil {
		t.Fatal(err)
	}
	if pid := co.Identifiers["tcgplayerProductId"]; pid != "" {
		t.Errorf("minted card carries product %q", pid)
	}
}
