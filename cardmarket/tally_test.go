package cardmarket

import (
	"context"
	"testing"
)

// TestUnreadExpansionIsCounted covers the census a run prints at the end.
// An expansion whose catalog never arrived is logged by the pool and the run
// carries on, and its products used to leave the tally without a trace: the
// "Walked %d products" line read as the whole catalog when it was only the
// part that answered, so a run several expansions timed out of looked like a
// storefront with less to sell rather than a run with holes in it.
func TestUnreadExpansionIsCounted(t *testing.T) {
	mkm := &Index{client: NewMKMClient("", "")}

	// A cancelled context is the cheapest fetch failure there is, and the
	// tally does not care which failure it was.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	channel := make(chan responseChan, 4)
	if err := mkm.processEdition(ctx, channel, MKMExpansion{Name: "Monarch - First"}); err == nil {
		t.Fatal("an expansion that never answered returned no error")
	}
	close(channel)

	var records []responseChan
	for result := range channel {
		records = append(records, result)
	}
	if len(records) != 1 {
		t.Fatalf("the expansion filed %d records, want the one saying it is missing", len(records))
	}
	if !records[0].tally || !records[0].unread {
		t.Errorf("filed %+v, want an unread tally", records[0])
	}
	if records[0].walked != 0 || records[0].refused != 0 {
		t.Errorf("an expansion nobody read counted %d walked and %d refused",
			records[0].walked, records[0].refused)
	}
}

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
