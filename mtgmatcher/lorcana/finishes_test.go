package lorcana

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// loadDatastore opens the datastore the finish suites run against, or skips.
// datastoreOnce loads the datastore the first time a test asks for it. The
// suite used to read and parse the file again on every call.
var datastoreOnce = sync.OnceValues(func() (*mtgmatcher.Backend, error) {
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		return nil, nil
	}
	f, err := datastore.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
})

func loadDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	b, err := datastoreOnce()
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Skip("LORCANA_PATH not set; skipping the Lorcana suite")
	}
	return b
}

// TestFinishPromotion walks every printing in the datastore and asks every
// one of its uuids for every finish it is sold in: plain to holofoil,
// holofoil to plain, and between two foils. Each answer has to be the uuid
// carrying that finish, which is also the check that no two finishes of a
// printing answer with one uuid.
func TestFinishPromotion(t *testing.T) {
	b := loadDatastore(t)

	var entries, promotions int
	for uuid, co := range b.UUIDs {
		if co.Sealed {
			continue
		}
		entries++

		seen := map[string]string{}
		for key, target := range co.FoilUUIDs {
			if other, found := seen[target]; found {
				t.Errorf("%s: finishes %q and %q share uuid %s", uuid, other, key, target)
			}
			seen[target] = key

			targetCo, err := b.GetUUID(target)
			if err != nil {
				t.Errorf("%s: finish %q names unknown uuid %s", uuid, key, target)
				continue
			}
			got, err := b.MatchIDFinish(uuid, targetCo.Finish)
			if err != nil || got != target {
				t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want %q",
					uuid, targetCo.Finish, got, err, target)
			}
			promotions++
		}
	}
	t.Logf("%d entries, %d promotions", entries, promotions)
}

// TestVendorFinishNames pins the alias table the loader registers, which is
// what lets a TCGplayer sku name the finish it prices: Normal and Cold Foil
// are the plain printing and the standard foil, and Holofoil is the treatment
// past the plain foil where the printing has one and its only foil where it
// does not. The catalog is what says so: of the 418 products whose single sku
// TCGplayer names Holofoil, 15 are foiled in plain silver, so refusing the
// name on those refuses the only sku they have. A printing sold in no foil at
// all still refuses it, which is the merge worth preventing.
func TestVendorFinishNames(t *testing.T) {
	b := loadDatastore(t)

	var withSubType, specialOnly, plainFoil int
	counted := map[string]bool{}
	for uuid, co := range b.UUIDs {
		if co.Sealed {
			continue
		}

		nonfoil := co.FoilUUIDs[mtgmatcher.FinishNonfoil]
		foil := co.FoilUUIDs[mtgmatcher.FinishFoil]
		var subType, foilFinish string
		for key := range co.FoilUUIDs {
			if key != mtgmatcher.FinishNonfoil && key != mtgmatcher.FinishFoil {
				subType = key
			}
		}
		if foil != "" {
			foilCo, err := b.GetUUID(foil)
			if err != nil {
				t.Fatalf("%s: foil sibling %s is not in the datastore", uuid, foil)
			}
			foilFinish = foilCo.Finish
		}

		// Every sibling answers the same, whichever one the caller sends
		for _, name := range []string{"Normal", "Cold Foil", "Foil"} {
			want := foil
			if name == "Normal" {
				want = nonfoil
			}
			got, err := b.MatchIDFinish(uuid, name)
			if want == "" {
				if !errors.Is(err, mtgmatcher.ErrCardWrongFinish) {
					t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want the finish refused", uuid, name, got, err)
				}
				continue
			}
			if err != nil || got != want {
				t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want %q", uuid, name, got, err, want)
			}
		}

		got, err := b.MatchIDFinish(uuid, "Holofoil")
		kind := ""
		switch {
		case subType != "":
			kind = "subtype"
			if err != nil || got != co.FoilUUIDs[subType] {
				t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want %q",
					uuid, "Holofoil", got, err, co.FoilUUIDs[subType])
			}
		case foil != "":
			kind = "special"
			if foilFinish == standardFoil {
				kind = "plain"
			}
			if err != nil || got != foil {
				t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want %q", uuid, "Holofoil", got, err, foil)
			}
		default:
			kind = "nofoil"
			if !errors.Is(err, mtgmatcher.ErrCardWrongFinish) {
				t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want the finish refused", uuid, "Holofoil", got, err)
			}
		}
		if counted[nonfoil+"|"+foil] {
			continue
		}
		counted[nonfoil+"|"+foil] = true
		switch kind {
		case "subtype":
			withSubType++
		case "special":
			specialOnly++
		default:
			plainFoil++
		}
	}
	if withSubType == 0 || specialOnly == 0 || plainFoil == 0 {
		t.Fatalf("datastore covers only part of the table: %d sub-typed, %d special-only, %d plain",
			withSubType, specialOnly, plainFoil)
	}
	t.Logf("%d sub-typed, %d special-foil-only, %d plain-foil printings", withSubType, specialOnly, plainFoil)
}
