package cardgamecorner

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
)

var (
	onePieceOnce    sync.Once
	onePieceErr     error
	onePieceBackend *mtgmatcher.Backend
)

// onePieceDatastore reads the One Piece datastore the first time a test asks
// for it, and skips where the run carries none.
func onePieceDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	onePieceOnce.Do(func() {
		path := os.Getenv("ONEPIECE_PATH")
		if path == "" {
			return
		}
		onePieceBackend, onePieceErr = datastore.Read(mtgmatcher.GameOnePiece, path)
	})
	if onePieceErr != nil {
		t.Fatal(onePieceErr)
	}
	if onePieceBackend == nil {
		t.Skip("Need ONEPIECE_PATH set to run this test")
	}
	return onePieceBackend
}

// A listing is placed by the Cardmarket product id its image is named after,
// through the id the datastore publishes or the bridge, and only where the
// printing carries the code the listing's own name states.
func TestMatchByProductImage(t *testing.T) {
	b := onePieceDatastore(t)
	cgc := &Cardgamecorner{backend: b, bridge: map[int]int{747537: 527020}}

	tests := []struct {
		desc  string
		name  string
		image string
		want  string
	}{
		{"the bridge names the printing", "Eustass \"Captain\" Kid (OP05-074) (V.3)", "/prodotti/singles/onepiece/op05/kid-op05074-v3_747537.jpg", "op05-074_527020_foil"},
		{"a doubled extension", "Eustass \"Captain\" Kid (OP05-074) (V.3)", "/prodotti/11/747537.jpg.png", "op05-074_527020_foil"},
		{"the published id needs no bridge", "Card (OP05-060)", "/prodotti/one_piece/814902.png", "op05-060_594591_foil"},
		{"a code the printing does not carry", "Eustass \"Captain\" Kid (OP05-075) (V.3)", "/prodotti/one_piece/747537.png", ""},
		{"a code written without its dash", "Eustass \"Captain\" Kid (OP05074) (V.3)", "/prodotti/one_piece/747537.png", "op05-074_527020_foil"},
		{"a name stating no code", "Eustass \"Captain\" Kid", "/prodotti/one_piece/747537.png", ""},
		{"an id no source links", "Eustass \"Captain\" Kid (OP05-074)", "/prodotti/one_piece/999999.png", ""},
		{"an image naming no id", "Eustass \"Captain\" Kid (OP05-074)", "/noimage.png", ""},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			id, found := cgc.matchByProductImage(&Card{Name: tt.name, Extra: tt.image}, false)
			if found != (tt.want != "") || id != tt.want {
				t.Errorf("matchByProductImage() = %q, %v, want %q", id, found, tt.want)
			}
		})
	}
}
