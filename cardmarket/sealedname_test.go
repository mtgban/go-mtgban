package cardmarket

import (
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// TestEditionRunSpellings pins how a name is offered to the resolver: one
// spelling when the name says which run it is, one per run when it does not.
func TestEditionRunSpellings(t *testing.T) {
	for _, tt := range []struct {
		name string
		want []string
	}{
		// Cardmarket files Flesh and Blood's runs as separate expansions
		// and writes the expansion's tail into every product name.
		{"Crucible of War - First Booster", []string{"Crucible of War Booster [1st Edition]"}},
		{"Crucible of War - Unlimited Booster Box", []string{"Crucible of War Booster Box [Unlimited Edition]"}},
		// Welcome to Rathe's first run is the only one called Alpha.
		{"Welcome to Rathe - Alpha Booster", []string{"Welcome to Rathe Booster [1st Edition]"}},
		// The older spelling puts the run before the word Edition and
		// drops the dash.
		{"Everfest First Edition Case (4 Booster Boxes)", []string{"Everfest Case (4 Booster Boxes) [1st Edition]"}},
		// The colon Cardmarket sometimes leaves behind goes with the run.
		{"Tales of Aria - First: Blitz Deck Set", []string{"Tales of Aria Blitz Deck Set [1st Edition]"}},
		// A name saying nothing about the run is offered both.
		{"Rise of the Duelist Booster", []string{
			"Rise of the Duelist Booster [1st Edition]",
			"Rise of the Duelist Booster [Unlimited Edition]",
		}},
		// An ordinal is not a run word, nor is a set that happens to be
		// named for one.
		{"First Strike: Aurora Deck", []string{
			"First Strike: Aurora Deck [1st Edition]",
			"First Strike: Aurora Deck [Unlimited Edition]",
		}},
	} {
		got := editionRunSpellings(tt.name)
		if !slices.Equal(got, tt.want) {
			t.Errorf("editionRunSpellings(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestSealedSaysSameWords pins the guard that keeps a manufactured bracket
// from buying a match. The words a marketplace and a datastore differ on for
// no reason fold together; anything left over is the product's own identity.
func TestSealedSaysSameWords(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		want bool
	}{
		// A box is a display and a booster is a pack, in either catalog.
		{"Rise of the Duelist Booster [1st Edition]", "Rise of the Duelist Booster Pack [1st Edition]", true},
		{"Maximum Gold: El Dorado Box [1st Edition]", "Maximum Gold: El Dorado Display [1st Edition]", true},
		{"Dark Revelation 1 Booster [Unlimited Edition]", "Dark Revelation 1 - Booster Pack [Unlimited Edition]", true},
		// A set number is not a count, whatever it looks like: Hidden
		// Arsenal 5 is not Hidden Arsenal, and the resolver forgives the
		// difference because a bare digit reads to it as a quantity.
		{"Hidden Arsenal 5 Booster [Unlimited Edition]", "Hidden Arsenal - Booster Pack [Unlimited Edition]", false},
		// Nor is a name the vendor spells more fully than the datastore.
		{"Duelist Pack: Yusei Fudo Booster [1st Edition]", "Duelist Pack: Yusei Booster Pack [1st Edition]", false},
		// How many the case holds is the vendor's own word too.
		{"Millennium Pack Booster Box (18 Booster) [1st Edition]", "Millennium Pack - Booster Box [1st Edition]", false},
		// A catalog pluralising the blister a storefront writes singular
		// is one more spelling neither of them means anything by, and the
		// resolver folds it too.
		{"Stellar Crown 3 Pack Blister [Latias]", "Stellar Crown 3 Pack Blisters [Latias]", true},
	} {
		if got := sealedSaysSameWords(tt.a, tt.b); got != tt.want {
			t.Errorf("sealedSaysSameWords(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestSealedRenamed pins the marketplace's own name for a product, and that
// it only answers for the game it belongs to.
func TestSealedRenamed(t *testing.T) {
	for _, tt := range []struct {
		gameID int
		name   string
		want   string
	}{
		{GameOnePiece, "The Best Booster", "Premium Booster Booster"},
		{GameOnePiece, "The Best Vol.2 Sleeved Booster", "Premium Booster Vol.2 Sleeved Booster"},
		// The word has to be the product's, not any word starting the same.
		{GameOnePiece, "The Bestiary Booster", ""},
		// Another game's catalog is another vocabulary.
		{GameYuGiOh, "The Best Booster", ""},
	} {
		got, found := sealedRenamed(tt.gameID, tt.name)
		if tt.want == "" {
			if found {
				t.Errorf("sealedRenamed(%d, %q) = %q, want no rename", tt.gameID, tt.name, got)
			}
			continue
		}
		if !found || got != tt.want {
			t.Errorf("sealedRenamed(%d, %q) = %q (%v), want %q", tt.gameID, tt.name, got, found, tt.want)
		}
	}
}

// TestSealedIsForeignPrinting pins the marks a product Cardmarket sells and
// an English-only datastore does not hold carries, the market's misspelt as
// Cardmarket misspells it.
func TestSealedIsForeignPrinting(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{"The Time of Battle Booster Box (Asia Region Legal)", true},
		{"Egghead Crisis Booster Box (Asia Region Lega)", true},
		{"One Piece Card Game 3rd Japanese Anniversary Set (Asia Region Legal Version)", true},
		{"The Best Booster Box (Non-English)", true},
		{"Origins Booster Box (Chinese, Slim)", true},
		{"The Time of Battle Booster Box", false},
	} {
		if got := sealedIsForeignPrinting(tt.name); got != tt.want {
			t.Errorf("sealedIsForeignPrinting(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// ygohDatastore is the published Yu-Gi-Oh! datastore cut down to the sealed
// rows these tests turn on, each copied verbatim from it. Hidden Arsenal is
// held in one print run, Hidden Arsenal 2's booster pack in both and its
// booster box in one - which is every case a run-silent name can be in.
const ygohDatastore = `{
 "game": "yugioh",
 "sets": {
  "HA01": {"name": "Hidden Arsenal", "releaseDate": "2009-11-10"},
  "HA02": {"name": "Hidden Arsenal 2", "releaseDate": "2010-07-20"}
 },
 "cards": [
  {"attribute": "WATER", "externalLinks": {"tcgPlayerId": 33740}, "finish": "Unlimited", "id": "ha01-en001_33740_unl", "name": "Blizzed, Defender of the Ice Barrier", "number": "HA01-EN001", "rarity": "Secret Rare", "setCode": "HA01", "type": "Effect Monster"}
 ],
 "sealed": [
  {"externalLinks": {"tcgPlayerId": 33771}, "id": "ha01-33771", "name": "Hidden Arsenal - Booster Pack [Unlimited Edition]", "releaseDate": "2009-11-10", "setCode": "HA01"},
  {"externalLinks": {"tcgPlayerId": 35744}, "id": "ha02-35744", "name": "Hidden Arsenal 2 - Booster Pack [1st Edition]", "releaseDate": "2010-07-20", "setCode": "HA02"},
  {"externalLinks": {"tcgPlayerId": 216871}, "id": "ha02-216871", "name": "Hidden Arsenal 2 - Booster Pack [Unlimited Edition]", "releaseDate": "2010-07-20", "setCode": "HA02"},
  {"externalLinks": {"tcgPlayerId": 57233}, "id": "ha02-57233", "name": "Hidden Arsenal 2 - Booster Box [1st Edition]", "releaseDate": "2010-07-20", "setCode": "HA02"}
 ]
}`

// TestResolveSealedNameRunSilent pins what a run-silent Cardmarket name
// answers with. Cardmarket never writes a print run into a Yu-Gi-Oh! name
// and the datastore writes one into every reprinted product's, so the
// catalogue has to decide: a product the datastore holds in one run is that
// run, one it holds in two is not said by the name at all.
func TestResolveSealedNameRunSilent(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(ygohDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Sealed{gameID: GameYuGiOh}
	for _, tt := range []struct {
		name, want string
	}{
		{"Hidden Arsenal Booster", "ha01-33771"},
		{"Hidden Arsenal Booster Box", ""},
		// Both runs answer as well as each other, so the name says neither.
		{"Hidden Arsenal 2 Booster", ""},
		{"Hidden Arsenal 2 Booster Box", "ha02-57233"},
		// The number is the set's, not a count of anything, and Hidden
		// Arsenal 5 is a product this datastore does not hold.
		{"Hidden Arsenal 5 Booster", ""},
		{"Hidden Arsenal 5 Booster Box", ""},
		// The runs Cardmarket sells for another market are not these.
		{"Hidden Arsenal (OCG) Booster", ""},
		{"Hidden Arsenal (Korean) Booster", ""},
	} {
		got, err := mkm.resolveSealedName(tt.name)
		if tt.want == "" {
			if err == nil {
				t.Errorf("resolveSealedName(%q) = %q, want a refusal", tt.name, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("resolveSealedName(%q) = %q (%v), want %q", tt.name, got, err, tt.want)
		}
	}
}

// fabSealedDatastore is the published Flesh and Blood datastore cut down to
// the booster packs of two sets, verbatim: Crucible of War, which the
// datastore holds in both print runs, and Everfest, which it holds in the
// first alone.
const fabSealedDatastore = `{
 "game": "fleshandblood",
 "sets": {
  "CRU": {"name": "Crucible of War", "releaseDate": "2020-08-28"},
  "EVR": {"name": "Everfest", "releaseDate": "2022-02-04"},
  "WTR": {"name": "Welcome to Rathe", "releaseDate": "2019-10-11"}
 },
 "cards": [
  {"externalLinks": {"tcgPlayerId": 261310}, "fabId": "EVR000", "finish": "1st Edition Cold Foil", "id": "evr000_261310_1ecold", "name": "Grandeur of Valahai", "number": "EVR000", "rarity": "Fabled", "setCode": "EVR"}
 ],
 "sealed": [
  {"externalLinks": {"tcgPlayerId": 224725}, "id": "cru-224725", "name": "Crucible of War Booster Pack [1st Edition]", "releaseDate": "2020-08-28", "setCode": "CRU"},
  {"externalLinks": {"tcgPlayerId": 245617}, "id": "cru-245617", "name": "Crucible of War Booster Pack [Unlimited Edition]", "releaseDate": "2020-08-28", "setCode": "CRU"},
  {"externalLinks": {"tcgPlayerId": 255920}, "id": "evr-255920", "name": "Everfest Booster Pack [1st Edition]", "releaseDate": "2022-02-04", "setCode": "EVR"},
  {"externalLinks": {"tcgPlayerId": 224713}, "id": "wtr-224713", "name": "Welcome to Rathe Booster Pack [1st Edition]", "releaseDate": "2019-10-11", "setCode": "WTR"},
  {"externalLinks": {"tcgPlayerId": 224728}, "id": "wtr-224728", "name": "Welcome to Rathe Booster Pack [Unlimited Edition]", "releaseDate": "2019-10-11", "setCode": "WTR"}
 ]
}`

// TestResolveSealedNameNamedRun pins the run a Cardmarket name spells out
// reaching the run the datastore brackets. The suffix has to come off before
// the product can be read - no product of ours is called "Crucible of War -
// First Booster" - and the run it named has to come back on, or the two runs
// answer each other's names.
func TestResolveSealedNameNamedRun(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(fabSealedDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Sealed{gameID: GameFleshAndBlood}
	for _, tt := range []struct {
		name, want string
	}{
		{"Crucible of War - First Booster", "cru-224725"},
		{"Crucible of War - Unlimited Booster", "cru-245617"},
		{"Everfest - First Booster", "evr-255920"},
		{"Welcome to Rathe - Alpha Booster", "wtr-224713"},
		{"Welcome to Rathe - Unlimited Booster", "wtr-224728"},
		// Everfest was never reprinted, so Cardmarket sells the only run
		// there is without naming it.
		{"Everfest Booster", "evr-255920"},
		// Crucible of War was, so the same silence names neither run.
		{"Crucible of War Booster", ""},
	} {
		got, err := mkm.resolveSealedName(tt.name)
		if tt.want == "" {
			if err == nil {
				t.Errorf("resolveSealedName(%q) = %q, want a refusal", tt.name, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("resolveSealedName(%q) = %q (%v), want %q", tt.name, got, err, tt.want)
		}
	}
}

// opDatastore is the published One Piece datastore cut down to the Premium
// Booster rows, verbatim. The set's own name carries the words Cardmarket
// sells it under, which is why the vendor's name is forgiven rather than
// understood: "best" is a word some set says, so the resolver holds nothing
// against a candidate for it - and still reaches none, because no candidate
// says "premium" the way the vendor would have to.
const opDatastore = `{
 "game": "onepiece",
 "sets": {
  "PRB-01": {"name": "Premium Booster -The Best-", "releaseDate": "2024-11-08"},
  "PRB-02": {"name": "Premium Booster -The Best- Vol. 2", "releaseDate": "2025-10-03"}
 },
 "cards": [
  {"bandaiId": "OP01-024_p3", "color": "Red", "externalLinks": {"tcgPlayerId": 586178}, "finish": "Foil", "id": "op01-024_586178_foil", "name": "Monkey.D.Luffy", "number": "OP01-024", "rarity": "SR", "setCode": "PRB-01", "type": "Character", "variant": "Alternate Art"}
 ],
 "sealed": [
  {"externalLinks": {"tcgPlayerId": 545398}, "id": "prb-01-545398", "name": "Premium Booster - Booster Pack", "releaseDate": "2024-11-08", "setCode": "PRB-01"},
  {"externalLinks": {"tcgPlayerId": 545399}, "id": "prb-01-545399", "name": "Premium Booster - Booster Box", "releaseDate": "2024-11-08", "setCode": "PRB-01"},
  {"externalLinks": {"tcgPlayerId": 622980}, "id": "prb-01-622980", "name": "Premium Booster - Sleeved Booster Pack", "releaseDate": "2024-11-08", "setCode": "PRB-01"},
  {"externalLinks": {"tcgPlayerId": 628451}, "id": "prb-02-628451", "name": "Premium Booster Vol. 2 - Booster Pack", "releaseDate": "2025-10-03", "setCode": "PRB-02"}
 ]
}`

// TestResolveSealedNameRenamed pins the marketplace's name reaching the
// datastore's product.
func TestResolveSealedNameRenamed(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(opDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Sealed{gameID: GameOnePiece}
	for _, tt := range []struct {
		name, want string
	}{
		{"The Best Booster", "prb-01-545398"},
		{"The Best Booster Box", "prb-01-545399"},
		{"The Best Sleeved Booster", "prb-01-622980"},
		{"The Best Vol.2 Booster", "prb-02-628451"},
	} {
		got, err := mkm.resolveSealedName(tt.name)
		if err != nil || got != tt.want {
			t.Errorf("resolveSealedName(%q) = %q (%v), want %q", tt.name, got, err, tt.want)
		}
	}
}
