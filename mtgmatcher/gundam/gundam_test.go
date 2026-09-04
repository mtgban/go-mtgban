package gundam

import (
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// datastoreOnce loads the datastore the first time a test asks for it.
var datastoreOnce = sync.OnceValues(func() (*mtgmatcher.Backend, error) {
	path := os.Getenv("GUNDAM_PATH")
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

func loadBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	b, err := datastoreOnce()
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Skip("GUNDAM_PATH not set; skipping the Gundam suite")
	}
	return b
}

// TestFinishIdentity pins the invariant the per-printing entries exist for:
// one uuid per finish, and no uuid answering for two. A product sold both
// plain and holofoil is two price points, and folding them together would
// file one price under the other's uuid.
func TestFinishIdentity(t *testing.T) {
	b := loadBackend(t)

	for uuid, co := range b.UUIDs {
		if co.Sealed {
			continue
		}
		if co.Finish == "" {
			t.Errorf("%s carries no finish", uuid)
			continue
		}
		if got := co.FoilUUIDs[co.Finish]; got != uuid {
			t.Errorf("%s is stored under finish %q, which resolves to %q", uuid, co.Finish, got)
		}
		for finish, other := range co.FoilUUIDs {
			if finish != co.Finish && other == uuid {
				t.Errorf("%s answers for both %q and %q", uuid, co.Finish, finish)
			}
		}
	}
}

// TestPlainNumberIsPlain pins the contract every game keeps: OriginalNumber
// is Number with the game's decorations stripped, never anything wider. The
// website's "cn:" search filters on it and "cns:" on Number.
func TestPlainNumberIsPlain(t *testing.T) {
	b := loadBackend(t)

	for uuid, co := range b.UUIDs {
		if co.Sealed || co.Number == "" {
			continue
		}
		if len(co.OriginalNumber) > len(co.Number) {
			t.Errorf("%s: OriginalNumber %q is wider than Number %q", uuid, co.OriginalNumber, co.Number)
		}
	}
}

// TestHolofoilIsTheFoil pins the game's own finish name. The catalog calls a
// stamped printing "Holofoil", which the shared vocabulary does not place:
// left to it, every holofoil entry would load with no finish at all.
func TestHolofoilIsTheFoil(t *testing.T) {
	for _, tt := range []struct {
		in, want string
	}{
		{"Holofoil", mtgmatcher.FinishFoil},
		{"holofoil", mtgmatcher.FinishFoil},
		{"Holo", mtgmatcher.FinishFoil},
		{"Foil", mtgmatcher.FinishFoil},
		{"Normal", mtgmatcher.FinishNonfoil},
		{"Cold Foil", ""},
	} {
		if got := (Rules{}).CanonicalFinish(tt.in); got != tt.want {
			t.Errorf("CanonicalFinish(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestRarityTellsParallelsApart pins what identifies a printing in this
// game. 382 of its collector numbers are carried by two products apiece,
// under the same name and with no variant label between them, and the only
// thing that differs is the rarity: the parallel run suffixes it with "+".
// A wording that names neither has to reach the plain run, since that is
// what a listing without a qualifier means.
func TestRarityTellsParallelsApart(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc       string
		in         mtgmatcher.InputCard
		wantRarity string
	}{
		{
			desc:       "a bare listing reaches the plain run",
			in:         mtgmatcher.InputCard{Name: "Gundam", Edition: "Edition Beta", Variation: "ST01-001"},
			wantRarity: "Legend Rare",
		},
		{
			desc:       "the parallel is reached by its own rarity code",
			in:         mtgmatcher.InputCard{Name: "Gundam", Edition: "Edition Beta", Variation: "ST01-001 LR+"},
			wantRarity: "LR+",
		},
		{
			desc:       "the plain run is not reached by the parallel's code",
			in:         mtgmatcher.InputCard{Name: "Gundam", Edition: "Edition Beta", Variation: "ST01-001 Legend Rare"},
			wantRarity: "Legend Rare",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.Rarity != tt.wantRarity {
				t.Errorf("Match(%v) = %s (%s), want rarity %s", tt.in, co.UUID, co.Rarity, tt.wantRarity)
			}
		})
	}
}

// TestNumberReachesItsPrinting replays every printing under its own name,
// edition and collector number. Where the number is shared the rarity is
// added, which is the whole of what tells those printings apart; anything
// that fails to come back is a printing no storefront writing the catalog's
// own words could reach.
func TestNumberReachesItsPrinting(t *testing.T) {
	b := loadBackend(t)

	var probes, hits int
	misses := map[string]int{}
	for _, co := range b.UUIDs {
		if co.Sealed || co.Number == "" {
			continue
		}
		set := b.Sets[co.SetCode]
		if set == nil {
			continue
		}
		variation := co.Number
		if co.Rarity != "" {
			variation += " " + co.Rarity
		}
		// The promo set reprints a card once per event it was handed out
		// at, every one of them at the main set's number and rarity, so the
		// event is part of what names the printing.
		for _, promoType := range co.PromoTypes {
			variation += " " + b.PromoTypeLabels[promoType]
		}
		in := mtgmatcher.InputCard{Name: co.Name, Edition: set.Name, Variation: variation}
		probes++
		id, err := b.Match(&in)
		if err != nil {
			misses[err.Error()]++
			continue
		}
		got, err := b.GetUUID(id)
		if err != nil {
			misses["GetUUID: "+err.Error()]++
			continue
		}
		// A product sold in two finishes answers with whichever the flags
		// resolve to, so the printing is what has to match, not the uuid.
		if got.SetCode == co.SetCode && got.Number == co.Number && got.Rarity == co.Rarity &&
			slices.Equal(got.PromoTypes, co.PromoTypes) {
			hits++
			continue
		}
		misses["wrong printing"]++
	}
	if probes == 0 {
		t.Fatal("no printing to probe")
	}
	t.Logf("%d of %d printings answered (%.1f%%)", hits, probes, 100*float64(hits)/float64(probes))
	for reason, n := range misses {
		t.Logf("   %5d %s", n, reason)
	}
	if hits != probes {
		t.Errorf("%d printings did not answer to their own number and rarity", probes-hits)
	}
}

// TestSealedIsNotACard pins that the sealed products stay in the sealed
// namespace: they share the name buckets, and a sealed product answering a
// card query would price a box as a single.
func TestSealedIsNotACard(t *testing.T) {
	b := loadBackend(t)

	var sealed int
	for _, co := range b.UUIDs {
		if !co.Sealed {
			continue
		}
		sealed++
		if !strings.Contains(strings.ToLower(co.Name), "gundam") && co.Name == "" {
			t.Errorf("%s is sealed but unnamed", co.UUID)
		}
	}
	if sealed == 0 {
		t.Error("the datastore carries no sealed product")
	}
}
