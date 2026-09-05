package cardtrader

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher/gundam"
)

// TestNameableByEdition pins the two answers that need no datastore: every
// game but Gundam is nameable, and so is a Gundam blueprint carrying a
// TCGplayer id, the id naming the printing outright.
func TestNameableByEdition(t *testing.T) {
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
			want:   true,
		},
		{
			desc:   "a Gundam blueprint with an id never reaches the name path",
			gameID: GameGundam,
			bp:     Blueprint{TCGplayerID: 616528},
			want:   true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			bp := tt.bp
			if got := nameableByEdition(tt.gameID, &bp); got != tt.want {
				t.Errorf("nameableByEdition = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGundamShelvesNameASet pins the split the guard rests on, which is a
// fact about Card Trader's shelf names rather than about this code: the
// shelves whose id-less blueprints resolve to the right printing are named
// exactly as the datastore names those sets, and the shelves that reprint
// another set's card under its own number are named nothing the datastore
// knows.
//
// Measured over every id-less Gundam blueprint that resolves by name: the
// three shelves below that name no set carry all 15 that resolved to the
// wrong printing, and the three that do carry all 8 that resolved to the
// right one. A shelf renamed on either side moves a listing between those
// groups silently, which is what this is here to catch.
//
// nameableByEdition asks the same question of the global datastore; the
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
