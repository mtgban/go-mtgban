package lorcana

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// testBackend is the datastore loadDatastore read, for the tests to match
// against directly.
var testBackend *mtgmatcher.Backend

// loadDatastore hands a test the datastore it read, or skips it where
// the run carries none: each suite runs under the job holding its own
// game's file, and not the others'.
var (
	datastoreOnce sync.Once
	datastoreErr  error
)

// loadDatastore installs the datastore the first time a test asks for it, and
// skips where the run carries none.
func loadDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("LORCANA_PATH")
		if path == "" {
			return
		}
		b, err := datastore.Read("lorcana", path)
		if err != nil {
			datastoreErr = err
			return
		}
		testBackend = b
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if testBackend == nil {
		t.Skip("Need LORCANA_PATH set to run this test")
	}
	return testBackend
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
			if other, found := seen[target]; found && !coarseFoilPair(key, other) {
				t.Errorf("%s: finishes %q and %q share uuid %s", uuid, other, key, target)
			}
			if _, found := seen[target]; !found {
				seen[target] = key
			}

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

// TestVendorFinishNames pins what the name of a TCGplayer sku answers with:
// each of Normal, Cold Foil and Holofoil reaches the printing sold under that
// name and is refused on a card that has none, and the bare word Foil
// reaches the standard foil, or the treatment on a card sold only in one.
// Holofoil used to answer with the standard foil on the 2,715 cards sold in
// no Holofoil; that was a finish the card is not sold in.
func TestVendorFinishNames(t *testing.T) {
	b := loadDatastore(t)

	var holofoil, noHolofoil int
	counted := map[string]bool{}
	for uuid, co := range b.UUIDs {
		if co.Sealed {
			continue
		}
		for _, name := range []string{"Normal", "Cold Foil", "Holofoil"} {
			want := co.FoilUUIDs[mtgmatcher.FinishSlug(name)]
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

		standard := co.FoilUUIDs["coldfoil"]
		if standard == "" {
			standard = co.FoilUUIDs["holofoil"]
		}
		if got := co.FoilUUIDs[mtgmatcher.FinishFoil]; got != standard {
			t.Errorf("%s: the bare foil flag reaches %q, want %q", uuid, got, standard)
		}
		got, err := b.MatchIDFinish(uuid, "Foil")
		if standard == "" {
			if !errors.Is(err, mtgmatcher.ErrCardWrongFinish) {
				t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want the finish refused", uuid, "Foil", got, err)
			}
		} else if err != nil || got != standard {
			t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want %q", uuid, "Foil", got, err, standard)
		}

		key := co.FoilUUIDs[mtgmatcher.FinishNonfoil] + "|" + standard
		if counted[key] {
			continue
		}
		counted[key] = true
		if co.FoilUUIDs["holofoil"] != "" {
			holofoil++
		} else {
			noHolofoil++
		}
	}
	if holofoil == 0 || noHolofoil == 0 {
		t.Fatalf("datastore covers only part of the table: %d printings with a Holofoil, %d without", holofoil, noHolofoil)
	}
	t.Logf("%d printings with a Holofoil, %d without", holofoil, noHolofoil)
}

// coarseFoilPair reports whether two keys sharing a uuid are the bare foil
// flag and the precise finish that answers it. A card sold only in a
// treatment has no standard foil printing, so the flag has to land on the
// treatment - Pokemon files the same key the same way, on 36,497 printings.
// The named form is not answered by it: FinishUUID refuses a key whose
// printing is sold in another finish. Any other pair sharing a uuid is two
// sku prices under one printing, which is what this guards.
func coarseFoilPair(a, b string) bool {
	if a == b {
		return false
	}
	for _, pair := range [2][2]string{{a, b}, {b, a}} {
		if pair[0] == mtgmatcher.FinishFoil && pair[1] != mtgmatcher.FinishNonfoil {
			return true
		}
	}
	return false
}
