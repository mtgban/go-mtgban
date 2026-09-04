package palworld

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// datastoreOnce loads the datastore the first time a test asks for it.
var datastoreOnce = sync.OnceValues(func() (*mtgmatcher.Backend, error) {
	path := os.Getenv("PALWORLD_PATH")
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
		t.Skip("PALWORLD_PATH not set; skipping the Palworld suite")
	}
	return b
}

// TestFinishIdentity pins the invariant the per-printing entries exist for:
// one uuid per finish, and no uuid answering for two.
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
	}
}

// TestPlainNumberIsPlain pins the contract every game keeps: OriginalNumber
// is Number with the game's decorations stripped, never anything wider. The
// rarity code this game's numbers end in is part of the number rather than
// a decoration over it, so the two are equal throughout.
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

// TestNumbersAreUnique pins what identifies a printing in this game. Unlike
// the other Bandai-shaped games, a parallel is numbered apart from the card
// it parallels rather than sharing its number under a different rarity, so
// a number decides on its own and the rules never have to tier.
func TestNumbersAreUnique(t *testing.T) {
	b := loadBackend(t)

	seen := map[string]string{}
	for _, co := range b.UUIDs {
		if co.Sealed || co.Number == "" {
			continue
		}
		key := co.SetCode + "|" + co.Number
		if other, found := seen[key]; found && other != co.Name {
			t.Errorf("%s is carried by both %q and %q", key, other, co.Name)
		}
		seen[key] = co.Name
	}
}

// TestRarityTailSplit pins the reading of a number's tail, which is the
// rarity's code and the whole of what tells a parallel from the card it
// parallels.
func TestRarityTailSplit(t *testing.T) {
	for _, tt := range []struct{ number, run, tail string }{
		{"ETD01-001", "ETD01-001", ""},
		{"ETD01-001TSR", "ETD01-001", "TSR"},
		{"EBP01-002SP", "EBP01-002", "SP"},
		{"EBP01-001SSP", "EBP01-001", "SSP"},
		{"EPR-004", "EPR-004", ""},
		{"", "", ""},
	} {
		run, tail := splitNumber(tt.number)
		if run != tt.run || tail != tt.tail {
			t.Errorf("splitNumber(%q) = %q, %q, want %q, %q", tt.number, run, tail, tt.run, tt.tail)
		}
	}
}

// TestTailReachesItsParallel pins that a listing naming the rarity code
// reaches the parallel, and that one naming none reaches the plain card.
// TCGplayer names its own products this way - "Grizzbolt - Rumbling Tank
// (TSR)" is ETD01-001TSR - so the code is what usually arrives.
func TestTailReachesItsParallel(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc       string
		in         mtgmatcher.InputCard
		wantNumber string
	}{
		{
			desc:       "the number as written decides",
			in:         mtgmatcher.InputCard{Name: "Grizzbolt - Rumbling Tank", Variation: "ETD01-001TSR"},
			wantNumber: "ETD01-001TSR",
		},
		{
			desc:       "the run's number with the code beside it reaches the parallel",
			in:         mtgmatcher.InputCard{Name: "Grizzbolt - Rumbling Tank", Variation: "ETD01-001 TSR"},
			wantNumber: "ETD01-001TSR",
		},
		{
			desc:       "and the code spelled out reaches it too",
			in:         mtgmatcher.InputCard{Name: "Grizzbolt - Rumbling Tank", Variation: "ETD01-001 Trial Deck Super Deck Rare"},
			wantNumber: "ETD01-001TSR",
		},
		{
			desc:       "a bare number reaches the plain card",
			in:         mtgmatcher.InputCard{Name: "Grizzbolt - Rumbling Tank", Variation: "ETD01-001"},
			wantNumber: "ETD01-001",
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
			if co.Number != tt.wantNumber {
				t.Errorf("Match(%v) = %s (%s), want number %s", tt.in, co.UUID, co.Number, tt.wantNumber)
			}
		})
	}
}

// TestNumberReachesItsPrinting replays every printing under its own name,
// edition and collector number. Anything that fails to come back is a
// printing no storefront writing the catalog's own words could reach.
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
		in := mtgmatcher.InputCard{Name: co.Name, Edition: set.Name, Variation: co.Number}
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
		if got.SetCode == co.SetCode && got.Number == co.Number {
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
		t.Errorf("%d printings did not answer to their own number", probes-hits)
	}
}

// TestSetCodeIsNotTheNumberPrefix pins the mismatch this game carries: the
// card list numbers a card with an "E" the set code does not have, so BP01's
// cards are numbered EBP01-nnn. Nothing may read a set out of a number's
// prefix here.
func TestSetCodeIsNotTheNumberPrefix(t *testing.T) {
	b := loadBackend(t)

	var agree, disagree int
	for _, co := range b.UUIDs {
		if co.Sealed || co.Number == "" {
			continue
		}
		run, _ := splitNumber(co.Number)
		prefix, _, _ := cutPrefix(run)
		if prefix == co.SetCode {
			agree++
		} else {
			disagree++
		}
	}
	if agree > 0 {
		t.Errorf("%d printings have a number prefix matching their set code, which this game does not do", agree)
	}
	if disagree == 0 {
		t.Error("no printing to check")
	}
}

// cutPrefix takes the letters and digits before a collector number's dash.
func cutPrefix(number string) (string, string, bool) {
	for i := 0; i < len(number); i++ {
		if number[i] == '-' {
			return number[:i], number[i+1:], true
		}
	}
	return number, "", false
}
