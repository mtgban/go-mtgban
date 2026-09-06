package cardmarket

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// logSink collects what a scraper says, so a test can read it back.
type logSink struct{ lines []string }

func (s *logSink) callback(format string, a ...any) {
	s.lines = append(s.lines, fmt.Sprintf(format, a...))
}

// TestProcessProductRefusal pins that the route through the bridge and the
// catalog's own words says so when it names no printing. It used to answer a
// refusal with a plain nil, which is what a priced product answers with too,
// so a whole catalog going unpriced looked exactly like a run with nothing
// to report.
func TestProcessProductRefusal(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(ygoDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Index{
		gameID:       GameYuGiOh,
		exchangeRate: 1,
		priceGuide: map[int]PriceGuide{
			1: {IDProduct: 1, LowPrice: 1, TrendPrice: 2},
			2: {IDProduct: 2, LowPrice: 1, TrendPrice: 2},
		},
	}
	channel := make(chan responseChan, 8)

	err = mkm.processProduct(channel, &MKMProduct{
		IDProduct:     1,
		Name:          "Blue-Eyes White Dragon",
		Number:        "001",
		ExpansionName: "Beginner's Edition 1",
	})
	if !errors.Is(err, errNoPrinting) {
		t.Errorf("a product of a catalog we carry no set for returned %v, want %v", err, errNoPrinting)
	}

	err = mkm.processProduct(channel, &MKMProduct{
		IDProduct:     2,
		Name:          "Guardian Elma (V.1 - Common)",
		Number:        "005",
		ExpansionName: "Dark Crisis",
	})
	if err != nil {
		t.Errorf("a product we do carry the printing of returned %v, want nil", err)
	}
}

// TestReportRefused pins how loud a refusal is. Every one is named, so a gap
// in a set we carry can be read off the log the way the id route's misses
// already can - except in an expansion nothing resolved in, which is a
// catalog we carry no set for and says so once instead of once per product.
func TestReportRefused(t *testing.T) {
	for _, tt := range []struct {
		name    string
		total   int
		refused []string
		want    []string
	}{
		{
			name:    "an expansion that resolved everything says nothing",
			total:   3,
			refused: nil,
			want:    nil,
		},
		{
			name:    "a partly resolved expansion names each refusal",
			total:   3,
			refused: []string{"first", "second"},
			want: []string{
				"[MKMIndex] Terminal World: 2 of 3 products named no printing of ours",
				"[MKMIndex] no printing for first",
				"[MKMIndex] no printing for second",
			},
		},
		{
			name:    "an expansion nothing resolved in says so once",
			total:   2,
			refused: []string{"first", "second"},
			want:    []string{"[MKMIndex] Terminal World: 2 of 2 products named no printing of ours"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var sink logSink
			mkm := &Index{LogCallback: sink.callback}
			mkm.reportRefused("Terminal World", tt.total, tt.refused, 0, 0)

			if len(sink.lines) != len(tt.want) {
				t.Fatalf("said %d lines %q, want %d", len(sink.lines), sink.lines, len(tt.want))
			}
			for i := range tt.want {
				if sink.lines[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, sink.lines[i], tt.want[i])
				}
			}
		})
	}
}
