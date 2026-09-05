package cardtrader

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher/gundam"
)

// shelf names a blueprint's expansion, which is the only field the guard
// reads besides the game and the id.
func shelf(name string) Blueprint {
	var bp Blueprint
	bp.Expansion.Name = name
	return bp
}

// TestPromoShelfNeedsLabel pins the two answers that need no datastore:
// no game but Gundam is ever asked for a label, and neither is a Gundam
// blueprint carrying a TCGplayer id, the id naming the printing outright.
func TestPromoShelfNeedsLabel(t *testing.T) {
	for _, tt := range []struct {
		desc   string
		gameID int
		bp     Blueprint
		want   bool
	}{
		{
			desc:   "another game is left alone",
			gameID: GameOnePiece,
			bp:     Blueprint{},
			want:   false,
		},
		{
			desc:   "a Gundam blueprint with an id never reaches the name path",
			gameID: GameGundam,
			bp:     Blueprint{TCGplayerID: 616528},
			want:   false,
		},
		{
			// The set shelves the catalog cannot be asked for by name. It
			// files the starter decks as "Starter Deck 01: Heroic
			// Beginnings", so the shelf is only recognisable by its code.
			desc:   "a numbered set shelf is not a promotional one",
			gameID: GameGundam,
			bp:     shelf("ST-01: Heroic Beginnings"),
			want:   false,
		},
		{
			desc:   "nor is it when Card Trader lowercases the code",
			gameID: GameGundam,
			bp:     shelf("St-14: Heavy Dominion"),
			want:   false,
		},
		{
			desc:   "nor a booster set's own shelf",
			gameID: GameGundam,
			bp:     shelf("GD-01: Newtype Rising"),
			want:   false,
		},
		{
			// What the guard is actually for: a shelf naming no set and
			// carrying no code, selling reprints under another set's number.
			desc:   "a promotional shelf still is",
			gameID: GameGundam,
			bp:     shelf("Premium Accessory and Card Set"),
			want:   true,
		},
		{
			// The reprint shelf sells one set of ours whole, so it is
			// mapped to that set's name and the number stops answering
			// alone; there is nothing left for the guard to refuse.
			desc:   "a mapped shelf names a set after all",
			gameID: GameGundam,
			bp:     shelf("Reprints"),
			want:   false,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			bp := tt.bp
			if got := promoShelfNeedsLabel(tt.gameID, &bp); got != tt.want {
				t.Errorf("promoShelfNeedsLabel = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGundamShelvesNameASet pins the split the guard rests on, which is a
// fact about Card Trader's shelf names rather than about this code: the
// shelves selling nothing but promotional reprints are named nothing the
// datastore knows, and the shelves whose id-less blueprints name their own
// set are named exactly as the datastore names those sets.
//
// A shelf renamed on either side moves every listing on it between those
// groups silently, which is what this is here to catch.
//
// promoShelfNeedsLabel asks the same question of the global datastore; the
// lookup is the whole of it, so this asks the loaded backend directly
// rather than swapping what every other test in this package runs against.
func TestGundamShelvesNameASet(t *testing.T) {
	path := os.Getenv("GUNDAM_PATH")
	if path == "" {
		t.Skip("GUNDAM_PATH not set; skipping the Gundam shelf split")
	}
	f, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := gundam.Load(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		shelf string
		code  string
	}{
		// Reprint shelves: no set of ours is named this, so an id-less
		// blueprint here has nothing but its number to go on and the number
		// belongs to the card being reprinted.
		{"Premium Accessory and Card Set", ""},
		{"Gundam Championships", ""},
		{"Reprints", ""},
		// Token shelves: named exactly as the datastore names the sets, so
		// the edition narrows and the number then names one printing.
		{"Promotional EX Base Tokens", "EXBP"},
		{"Promotional EX Resource Tokens", "EXRP"},
		{"Promotional Resource Tokens", "RP"},
	} {
		t.Run(tt.shelf, func(t *testing.T) {
			set, err := b.GetSetByName(tt.shelf)
			got := ""
			if err == nil && set != nil {
				got = set.Code
			}
			if got != tt.code {
				t.Errorf("GetSetByName(%q) = %q, want %q", tt.shelf, got, tt.code)
			}
		})
	}
}

// TestGundamShelfSets pins both halves of what gundamShelfSets asserts: the
// shelf names no set of ours, which is why the mapping is needed at all, and
// the code it maps to names one, which is what the mapping answers with.
//
// Either half moving silently reverts the shelf to aliasing on its numbers,
// so this fails rather than the listings quietly changing identity.
func TestGundamShelfSets(t *testing.T) {
	path := os.Getenv("GUNDAM_PATH")
	if path == "" {
		t.Skip("GUNDAM_PATH not set; skipping the Gundam shelf mapping")
	}
	f, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := gundam.Load(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}

	for name, code := range gundamShelfSets {
		t.Run(name, func(t *testing.T) {
			set, err := b.GetSetByName(name)
			if err == nil && set != nil {
				t.Errorf("shelf %q names set %q; the mapping is unnecessary", name, set.Code)
			}
			set, err = b.GetSet(code)
			if err != nil || set == nil {
				t.Fatalf("GetSet(%q) = %v; the mapping names no set", code, err)
			}
			if set.Name == "" {
				t.Errorf("set %q has no name to answer with", code)
			}
		})
	}
}
