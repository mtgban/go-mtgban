package coolstuffinc

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

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

// TestOnePieceSpelling pins the names this storefront writes its own way.
func TestOnePieceSpelling(t *testing.T) {
	got := onePieceSpelling("But If We See Each Other Again...Will You Call Me Your Shipmate?!!")
	if got != "But If We Ever See Each Other Again... Will You Call Me Your Shipmate?!!" {
		t.Errorf("onePieceSpelling = %q", got)
	}
	if got := onePieceSpelling("Marco"); got != "Marco" {
		t.Errorf("onePieceSpelling(Marco) = %q", got)
	}
}

// TestOfferSeen pins that an offer arriving twice off two shelves is
// collected once, and that another condition of the same product is not
// the same offer.
func TestOfferSeen(t *testing.T) {
	seen := map[string]bool{}
	const url = "https://www.coolstuffinc.com/p/433402"
	offer := func(cardID, conditions, seller string) responseChan {
		return responseChan{cardID: cardID, invEntry: &mtgban.InventoryEntry{URL: url, Conditions: conditions, SellerName: seller, Quantity: 20}}
	}
	nm := offer("plain", "NM", availableMarketNames[0])
	if offerSeen(seen, nm) {
		t.Error("the first offer is new")
	}
	if !offerSeen(seen, nm) {
		t.Error("the same offer off another shelf is seen")
	}
	if offerSeen(seen, offer("plain", "SP", availableMarketNames[0])) {
		t.Error("another condition is another offer")
	}
	// The row's foil is filed at NM too, on the foil printing.
	if offerSeen(seen, offer("plain_foil", "NM", availableMarketNames[0])) {
		t.Error("the foil beside the plain copy is another offer")
	}
	// So is a graded copy, on the same printing, from the other seller.
	if offerSeen(seen, offer("plain", "NM", availableMarketNames[1])) {
		t.Error("a graded copy beside the plain one is another offer")
	}
}
