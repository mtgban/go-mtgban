package coolstuffinc

import "testing"

// TestOnePieceShelf pins which listings the bracket is allowed to move. The
// guard is the whole of it: a deck named beside a real set is the deck the
// card was reprinted from, and reading it there would move nine listings that
// are already right.
func TestOnePieceShelf(t *testing.T) {
	for _, tt := range []struct {
		desc, shelf, name, want string
	}{
		{"a promo shelf lets the bracket name the deck",
			"Promo", "Monkey.D.Luffy - P-041 (Starter Deck 18)", "Starter Deck 18"},
		{"a real set keeps its own name",
			"ST11 - Starter Deck 11: Uta", "Backlight - 003 (Starter Deck 16)", "ST11 - Starter Deck 11: Uta"},
		{"and so does a set that is not a deck at all",
			"OP12 - Legacy of the Master", "Kuzan - 040 (Starter Deck 33)", "OP12 - Legacy of the Master"},
		{"a promo naming no deck stays a promo",
			"Promo", "Monkey.D.Luffy - P-041 (Offline Regional Participation Pack 2024 Vol.2)", "Promo"},
		{"an unnumbered deck names nothing",
			"Promo", "Yamato (022) (Starter Deck)", "Promo"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := onePieceShelf(tt.shelf, tt.name); got != tt.want {
				t.Errorf("onePieceShelf(%q, %q) = %q, want %q", tt.shelf, tt.name, got, tt.want)
			}
		})
	}
}
