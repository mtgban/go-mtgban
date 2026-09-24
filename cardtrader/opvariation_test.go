package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestOpPlaceholderNumber pins the One Piece placeholder-number rule: a
// leader or DON!! blueprint's number ("P-L", "P", empty, or a DON!! set
// code) never names a printing, so the Version answers in its place. A
// number that carries a digit and an index tail the matcher cannot read
// ("OP07-047P2") is not a placeholder and keeps the old behavior of
// dropping the Version rather than guessing which digit-shape it is.
func TestOpPlaceholderNumber(t *testing.T) {
	for _, tt := range []struct {
		desc, name, version, number, want string
	}{
		{"a leader's 'P-L' number gives the version", "Monkey.D.Luffy", "Non-Foil | Leader Pack", "P-L", "Non-Foil | Leader Pack"},
		{"a leader's bare 'P' number gives the version", "Monkey.D.Luffy", "4th Anniversary Leader Card", "P", "4th Anniversary Leader Card"},
		{"a DON!! set-code number gives the version despite its digit", "DON!!", "Gol.D.Roger | Gold Foil", "OP13g", "Gol.D.Roger | Gold Foil"},
		{"a DON!! blueprint with no number at all gives the version", "DON!!", "Zoro | World United", "", "Zoro | World United"},
		{"a non-DON!! number with a digit is not a placeholder", "Some Card", "Winner Pack 2026 Vol.3", "OP07-047P2", "OP07-047P2"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			bp := Blueprint{Name: tt.name, Version: tt.version}
			if got := gameVariation(GameOnePiece, &bp, tt.number); got != tt.want {
				t.Errorf("gameVariation(%q, %q, %q) = %q, want %q", tt.name, tt.version, tt.number, got, tt.want)
			}
		})
	}
}

// TestOpDonPromoShelfEdition pins the DON!!-only edition rule: "One Piece
// Promos" itself names no set, so a DON!! sold there is pointed at the
// catalog's own promo shelf name. A non-DON!! card on the same shelf, and a
// DON!! on any other shelf, are both left alone.
func TestOpDonPromoShelfEdition(t *testing.T) {
	b := &mtgmatcher.Backend{}
	don := Blueprint{Name: "DON!!"}
	don.Expansion.Name = "One Piece Promos"
	if got := gameEdition(b, GameOnePiece, &don); got != "One Piece Promotion Cards" {
		t.Errorf("gameEdition(DON!!, One Piece Promos) = %q", got)
	}
	other := Blueprint{Name: "Monkey.D.Luffy"}
	other.Expansion.Name = "One Piece Promos"
	if got := gameEdition(b, GameOnePiece, &other); got != "One Piece Promos" {
		t.Errorf("gameEdition(non-DON!!, One Piece Promos) = %q, want the shelf kept", got)
	}
	elsewhere := Blueprint{Name: "DON!!"}
	elsewhere.Expansion.Name = "World United"
	if got := gameEdition(b, GameOnePiece, &elsewhere); got != "World United" {
		t.Errorf("gameEdition(DON!!, World United) = %q, want the shelf kept", got)
	}
}
