package sealedev

import (
	"fmt"
	"math"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// mkmCards builds a backend and an empty price response over n plain cards.
func mkmCards(n int) (*mtgmatcher.Backend, *BANPriceResponse) {
	b := &mtgmatcher.Backend{UUIDs: map[string]*mtgmatcher.CardObject{}}
	for i := range n {
		uuid := fmt.Sprintf("card%d", i)
		b.UUIDs[uuid] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: uuid, SetCode: "AAA"}}
		b.AllUUIDs = append(b.AllUUIDs, uuid)
	}
	return b, &BANPriceResponse{
		Retail:  map[string]map[string]*BanPrice{},
		Buylist: map[string]map[string]*BanPrice{},
	}
}

func TestMKMEstimateBoundsWhatTheGuideSays(t *testing.T) {
	// One band measured at half of trend, so the arithmetic is checkable.
	c := &mkmCalibration{bands: make([]float64, len(mkmTrendBands))}
	c.bands[mkmBandOf(10)] = 0.5

	for _, tt := range []struct {
		desc      string
		low, want float64
		trend     float64
	}{
		{desc: "the scaled trend is the estimate", low: 1, trend: 10, want: 5},
		{desc: "low is a floor under it", low: 8, trend: 10, want: 8},
		{
			desc: "but a low far over trend is an artefact, not a floor",
			low:  5000, trend: 10, want: 20,
		},
		{
			desc: "a product trend says nothing about prices off its one listing",
			low:  42, trend: 0, want: 42,
		},
		{desc: "and nothing at all is still nothing", low: 0, trend: 0, want: 0},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := c.estimate(tt.low, tt.trend); got != tt.want {
				t.Errorf("estimate(%v, %v) = %v, want %v", tt.low, tt.trend, got, tt.want)
			}
		})
	}
}

// TestFitMKMCalibrationMeasuresTheScale feeds a known relationship back in:
// a band with enough cards answers with its own measured number, and a band
// without them falls through to the curve rather than to 1.
func TestFitMKMCalibrationMeasuresTheScale(t *testing.T) {
	b, r := mkmCards(mkmMinBandSamples + 10)
	for _, uuid := range b.GetUUIDs() {
		r.setRetail(b, uuid, mkmTrendStore, 10)
		r.setRetail(b, uuid, mkmStore, 15)
	}

	c := fitMKMCalibration(b, r)
	if c == nil {
		t.Fatal("fitMKMCalibration refused a sample it had enough of")
	}
	if got := c.multiplier(10); math.Abs(got-1.5) > 1e-9 {
		t.Errorf("measured band multiplier = %v, want 1.5", got)
	}
	if c.measured != 1 {
		t.Errorf("measured %d bands, want 1", c.measured)
	}
	// $100 lands in a band nothing was seen in, so the curve answers. With a
	// single trend observed the curve is flat at the mean ratio.
	if got := c.multiplier(100); math.Abs(got-1.5) > 1e-9 {
		t.Errorf("unmeasured band multiplier = %v, want the curve's 1.5", got)
	}
}

// TestFitMKMCalibrationRefusesTooFewCards pins the guard that keeps a game
// with almost no polled cards - Riftbound has 21 - unpriced instead of
// priced off noise.
func TestFitMKMCalibrationRefusesTooFewCards(t *testing.T) {
	b, r := mkmCards(mkmMinFitSamples - 1)
	for _, uuid := range b.GetUUIDs() {
		r.setRetail(b, uuid, mkmTrendStore, 10)
		r.setRetail(b, uuid, mkmStore, 15)
	}
	if c := fitMKMCalibration(b, r); c != nil {
		t.Errorf("fitMKMCalibration answered on %d cards, below its own floor", len(b.AllUUIDs))
	}
}

// TestFillKeepsPolledPrices is the ordering this depends on: a card the
// scraper really polled keeps its price, and an estimate written under the
// same name never becomes what the next fit is measured against.
func TestFillKeepsPolledPrices(t *testing.T) {
	b, r := mkmCards(mkmMinBandSamples + 10)
	uuids := b.GetUUIDs()
	for _, uuid := range uuids {
		r.setRetail(b, uuid, mkmTrendStore, 10)
		r.setRetail(b, uuid, mkmLowStore, 4)
		r.setRetail(b, uuid, mkmStore, 15)
	}
	// One card the market never answered for.
	gap := uuids[0]
	delete(r.Retail[gap], mkmStore)

	c := fitMKMCalibration(b, r)
	var filled int
	for _, uuid := range uuids {
		if c.fill(b, r, uuid) {
			filled++
		}
	}
	if filled != 1 {
		t.Fatalf("filled %d cards, want the 1 gap", filled)
	}

	if got := r.getRetail(b, uuids[1], mkmStore); got != 15 {
		t.Errorf("a polled card came back at %v, want its own 15", got)
	}
	// The other cards measured 1.5x trend, so the gap is priced at that too.
	if got := r.getRetail(b, gap, mkmStore); math.Abs(got-15) > 1e-9 {
		t.Errorf("the unpolled card came back at %v, want the measured 15", got)
	}

	// Refitting now sees the estimates just written. They must not have moved
	// the answer, which is what measuring before writing buys.
	if c := fitMKMCalibration(b, r); c == nil || math.Abs(c.multiplier(10)-1.5) > 1e-9 {
		t.Error("the written estimate moved a later fit")
	}
}
