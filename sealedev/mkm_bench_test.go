package sealedev

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// mkmSnapshot builds a catalog the shape of a real one: a published guide
// over every card, and a market price on only the slice Cardmarket's scraper
// would have spent a call on. Trend is log-uniform over the range a catalog
// actually spans, and the market price is drawn around the same rising
// multiplier the published dumps show, so a fit against this lands where a
// fit against them does and the benchmark doubles as a sanity check on it.
func mkmSnapshot(cards, polled int) (*mtgmatcher.Backend, *BANPriceResponse) {
	rng := rand.New(rand.NewSource(1))
	b := &mtgmatcher.Backend{UUIDs: make(map[string]*mtgmatcher.CardObject, cards)}
	r := &BANPriceResponse{
		Retail:  make(map[string]map[string]*BanPrice, cards),
		Buylist: map[string]map[string]*BanPrice{},
	}
	for i := range cards {
		uuid := fmt.Sprintf("card%d", i)
		b.UUIDs[uuid] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: uuid, SetCode: "AAA"}}
		b.AllUUIDs = append(b.AllUUIDs, uuid)

		// $0.05 to $500, log-uniform.
		trend := math.Exp(math.Log(0.05) + rng.Float64()*math.Log(500/0.05))
		r.setRetail(b, uuid, mkmTrendStore, trend)
		r.setRetail(b, uuid, mkmLowStore, trend*(0.1+0.7*rng.Float64()))

		// The scraper polls the top of the catalog, so the polled slice is
		// the dearest cards rather than a random sample of them - which is
		// the whole reason the cheap bands go unmeasured in a real run.
		if trend > 7 && polled > 0 {
			polled--
			multiplier := 0.64 + 0.154*math.Log(trend)
			r.setRetail(b, uuid, mkmStore, trend*multiplier*(0.7+0.6*rng.Float64()))
		}
	}
	return b, r
}

// BenchmarkFitMKMCalibration times the pass the estimate adds to a run, and
// reports what it fitted. Nothing in a run prints those numbers, and they
// move with the market, so this is where to read them off: an EV that shifted
// between two runs is otherwise hard to tell from one whose prices shifted.
func BenchmarkFitMKMCalibration(bench *testing.B) {
	b, r := mkmSnapshot(150000, 22000)

	var c *mkmCalibration
	for range bench.N {
		c = fitMKMCalibration(b, r)
	}

	bench.StopTimer()
	if c == nil {
		bench.Fatal("no calibration over a catalog that should have carried one")
	}
	bench.ReportMetric(c.intercept, "intercept")
	bench.ReportMetric(c.slope, "slope")
	bench.ReportMetric(float64(c.measured), "bands")
	bench.Logf("multiplier = %.4f %+.4f*ln(trend), measured over %d polled cards", c.intercept, c.slope, c.samples)
	for i, edge := range mkmTrendBands {
		if c.bands[i] != 0 {
			bench.Logf("  band $%-5g %.3f", edge, c.bands[i])
		} else {
			bench.Logf("  band $%-5g curve (too few cards)", edge)
		}
	}
	var filled int
	for _, uuid := range b.AllUUIDs {
		if c.fill(b, r, uuid) {
			filled++
		}
	}
	bench.Logf("filled %d of %d cards", filled, len(b.AllUUIDs))
}

// BenchmarkMKMFill times the writing alone, the half that rides along in the
// caller's own pass, over a catalog rebuilt each round since filling one
// consumes it.
func BenchmarkMKMFill(bench *testing.B) {
	for range bench.N {
		bench.StopTimer()
		b, r := mkmSnapshot(150000, 22000)
		c := fitMKMCalibration(b, r)
		bench.StartTimer()

		for _, uuid := range b.AllUUIDs {
			c.fill(b, r, uuid)
		}
	}
}
