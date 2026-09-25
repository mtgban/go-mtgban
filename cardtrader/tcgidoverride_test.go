package cardtrader

import "testing"

// TestTCGplayerIDOverrides pins tcgIDOverrides' corrections against a few of
// the blueprints measured on the live catalog: three whose own id names a
// sibling card, one One Piece DON!! promo Card Trader sends no id for at
// all, and one ordinary blueprint outside the table, which must pass its
// own id through unchanged.
func TestTCGplayerIDOverrides(t *testing.T) {
	tests := []struct {
		desc string
		bp   *Blueprint
		want int
	}{
		{"Saltwater Swell (Yellow) SEA142, id named Blue", &Blueprint{ID: 334431, TCGplayerID: 633284}, 633285},
		{"Saltwater Swell (Blue) SEA143, id named Yellow", &Blueprint{ID: 334432, TCGplayerID: 633285}, 633284},
		{"Judge Professor Program Stamp 167/182, id named a different Judge", &Blueprint{ID: 357942, TCGplayerID: 658743}, 685999},
		{"One Piece DON!! Pop Art, no id sent at all", &Blueprint{ID: 290971, TCGplayerID: 0}, 544805},
		{"O-Nami Dash Pack 2025, id named the Illustration Box", &Blueprint{ID: 326797, TCGplayerID: 623070}, 712033},
		{"an ordinary blueprint outside the table", &Blueprint{ID: 999999, TCGplayerID: 12345}, 12345},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := tt.bp.TCGplayerProductID(); got != tt.want {
				t.Errorf("TCGplayerProductID(%d) = %d, want %d", tt.bp.ID, got, tt.want)
			}
		})
	}
}
