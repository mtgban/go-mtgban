package cardmarket

import (
	"context"
	"testing"
)

// TestCollectPricesCountsUnreadExpansions pins what the collector makes of
// those records: they add nothing to what was walked, and they are counted, so
// the run can say how much of the catalog its total is a total of.
func TestCollectPricesCountsUnreadExpansions(t *testing.T) {
	mkm := &Index{gameID: GameYuGiOh}
	items := []MKMExpansion{{Name: "read"}, {Name: "unread"}, {Name: "read too"}}

	walked, refused, unread := mkm.collectPrices(context.Background(), items,
		func(_ context.Context, exp MKMExpansion, channel chan<- responseChan) error {
			if exp.Name == "unread" {
				channel <- responseChan{tally: true, unread: true}
				return context.DeadlineExceeded
			}
			channel <- responseChan{tally: true, walked: 10, refused: 3}
			return nil
		})

	if walked != 20 || refused != 6 {
		t.Errorf("walked %d products and refused %d, want 20 and 6", walked, refused)
	}
	if unread != 1 {
		t.Errorf("counted %d expansions as missing from that, want 1", unread)
	}
}
