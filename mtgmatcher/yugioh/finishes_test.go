package yugioh

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPrintRunUUIDs walks every printing and asks each print run it carries
// for the uuid pricing it. Each has to answer with the entry stamped with
// that run, no two runs of a printing and no two printings may answer with
// one uuid, and every stored entry has to be some printing's run - a run
// silently overwritten in FoilUUIDs would leave a uuid nothing can reach.
// The shared names stay refused throughout: they are the flag slots aimed
// at the default run, and answering one with a run would hand a bare foil
// flag a printing nobody asked for.
func TestPrintRunUUIDs(t *testing.T) {
	b := loadBackend(t)

	owner := map[string]string{}
	var printings, runs int
	for _, code := range b.AllSets {
		for _, card := range b.Sets[code].Cards {
			printings++

			runOf := map[string]string{}
			for key, target := range card.FoilUUIDs {
				if key == mtgmatcher.FinishNonfoil || key == mtgmatcher.FinishFoil {
					continue
				}
				if other, found := runOf[target]; found {
					t.Errorf("%s: runs %q and %q share uuid %s", card.UUID, other, key, target)
				}
				runOf[target] = key
				if prev, found := owner[target]; found && prev != card.UUID {
					t.Errorf("uuid %s answers for printings %s and %s", target, prev, card.UUID)
				}
				owner[target] = card.UUID

				co, err := b.GetUUID(target)
				if err != nil {
					t.Errorf("%s: run %q names unknown uuid %s", card.UUID, key, target)
					continue
				}
				if co.Finish != key {
					t.Errorf("%s: run %q names uuid %s carrying finish %q",
						card.UUID, key, target, co.Finish)
				}
				got, err := b.MatchIDFinish(card.UUID, key)
				if err != nil || got != target {
					t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want %q",
						card.UUID, key, got, err, target)
				}
				runs++
			}

			// The flag slots are a compatibility layer, never a run of
			// their own: they have to land on a run the printing carries.
			for _, slot := range []string{mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil} {
				target := card.FoilUUIDs[slot]
				if target == "" {
					t.Errorf("%s: flag slot %q names no run", card.UUID, slot)
					continue
				}
				if _, found := runOf[target]; !found {
					t.Errorf("%s: flag slot %q names uuid %s that no run carries",
						card.UUID, slot, target)
				}
			}

			if len(card.FinishAliases) > 0 {
				t.Errorf("%s: the runs are spelled as the datastore carries them, got aliases %v",
					card.UUID, card.FinishAliases)
			}
			for _, name := range []string{"Nonfoil", "Foil", "Normal"} {
				got, err := b.MatchIDFinish(card.UUID, name)
				if !errors.Is(err, mtgmatcher.ErrCardUnnamedFinish) {
					t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want the shared name refused",
						card.UUID, name, got, err)
				}
			}
		}
	}

	var orphans int
	for uuid, co := range b.UUIDs {
		if co.Sealed {
			continue
		}
		if _, found := owner[uuid]; !found {
			orphans++
			t.Errorf("uuid %s carries finish %q but no printing names it", uuid, co.Finish)
		}
	}
	if printings == 0 || runs == 0 {
		t.Fatalf("datastore carries no print runs: %d printings, %d runs", printings, runs)
	}
	t.Logf("%d printings, %d print runs, %d orphaned entries", printings, runs, orphans)
}

// TestCanonicalFinish pins the vocabulary the loader keys FoilUUIDs with:
// the print runs pass through normalized, and the names every game shares
// are refused rather than placed.
func TestCanonicalFinish(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"1st Edition", finish1stEdition},
		{"1stEdition", finish1stEdition},
		{"Unlimited", finishUnlimited},
		{"Limited", finishLimited},
		{"Nonfoil", ""},
		{"Foil", ""},
		{"foil", ""},
		{"Normal", ""},
		{"", ""},
	}
	for _, test := range tests {
		if got := (Rules{}).CanonicalFinish(test.name); got != test.want {
			t.Errorf("CanonicalFinish(%q) = %q, want %q", test.name, got, test.want)
		}
	}
}
