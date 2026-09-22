package sealedev

import (
	"math"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Cardmarket's market scraper prices only the cards worth a live call - a
// trend price over $7, or a spread wide enough to pay for one, see
// cardmarket's marketCandidates - so about 86% of the catalog carries the
// published guide's Low and Trend columns and no market price at all. An EV
// cannot have holes the way a search page can: a card with no price counts as
// zero rather than as unknown, and a booster is mostly cards the scraper
// never polled. So the gap is filled from the two guide columns.
//
// Trend, scaled, is the estimate. Low is a floor rather than half of an
// average: the market price sits above it on 98% of cards, and on bulk it
// sits at three times it, so averaging the two lands at a third of the answer.
// What Trend has to be scaled by rises with price - a bulk card has hundreds
// of listings, so the cheapest sits far under the trend, while a $50 card has
// a handful and the cheapest sits at or above it.
//
// The scale is fitted from the snapshot being priced rather than carried as
// constants, because it is not the same number in every game and a table would
// have to be maintained per game: measured against a day of published dumps,
// Magic runs 0.43 to 1.24 across the price range, One Piece sits near 0.95
// throughout, Flesh and Blood near 0.95, and Yu-Gi-Oh runs 2.2 to 3.7. Fitting
// also keeps the numbers honest as the market moves; they drift about 1% a day,
// which moves a priced catalog by 0.2%.
//
// # Accuracy, and where it is not good enough
//
// Scored against the cards that do carry a market price, holding out the day
// the constants were fitted on, the estimate lands within a few percent of
// unbiased in aggregate and around 24% median error per card for Magic. That
// is the number that matters for an EV, which is a sum. By game:
//
//   - Magic, One Piece, Flesh and Blood estimate cleanly (12-24% median error,
//     aggregate bias within a couple of percent).
//   - Lorcana carries too few polled cards to measure a single price band
//     (391), so only the fitted curve answers. It still beats a fixed table.
//   - Pokemon and Yu-Gi-Oh do not estimate. Their median error stays near 58%
//     and 69% whatever is fitted, because their guide columns and their
//     listings are not describing the same printing - Pokemon's guide has no
//     column for the print-run axis at all, so one blended Low and Trend is
//     filed under every uuid a product resolves to. That is an upstream
//     mapping problem and no scaling of Trend repairs it.
//   - Riftbound carries 21 polled cards in total, which is not enough to fit
//     anything; the guard below leaves the game unpriced rather than guessing.
//
// A sealed EV sourced from Cardmarket is worth publishing for the first group.
// For Pokemon and Yu-Gi-Oh the estimate would be confidently wrong, and the
// fix belongs in how those games' guide columns are mapped to uuids.
const (
	mkmStore      = "MKM"
	mkmLowStore   = "MKMLow"
	mkmTrendStore = "MKMTrend"

	// mkmMinFitSamples is how many polled cards the curve needs before it
	// says anything. The fit is stable from about a hundred - resampling
	// Magic down to that many moves the coefficients by under 0.01 - and
	// below it a game is better left unpriced than priced from noise.
	mkmMinFitSamples = 100

	// mkmMinBandSamples is how many cards a price band needs before its own
	// mean is trusted ahead of the curve. Nine bands at this floor want
	// around 1,800 polled cards, which Magic and Pokemon have and the
	// smaller games do not; they fall back to the curve, band by band.
	mkmMinBandSamples = 200

	// mkmRatioCap bounds one card's influence on the fit. The mean ratio is
	// the right statistic for an EV, but unbounded it is decided by a handful
	// of Reserved List cards whose only listing is orders of magnitude over
	// their trend.
	mkmRatioCap = 5.0

	// mkmMultiplierFloor and mkmMultiplierCeiling bound the fitted curve
	// outside the range it was measured on, so a game whose prices run past
	// the observed span cannot extrapolate into nonsense.
	mkmMultiplierFloor   = 0.20
	mkmMultiplierCeiling = 2.0

	// mkmTrendCeiling caps the estimate against Trend. Low is a floor, but a
	// Low far above Trend is a listing artefact rather than a price: 0.4% of
	// the unpriced cards carry 85% of all the Low value among them, one of
	// them a six-figure listing against a four-figure trend. The cap
	// deliberately wins over the floor.
	mkmTrendCeiling = 2.0
)

// mkmTrendBands are the Trend levels the multiplier is measured in. They are
// the lower edge of each band, so a band is [edge, next edge).
//
// These stay written down where the multipliers do not, because they are not
// the same kind of number: an edge says where to split, not what the answer
// is, so it does not go stale as a market moves. The curve alone would remove
// them, and costs more than they do - it flattens where the data keeps
// climbing, leaving cards over $50 biased about 10% high against the bands'
// 2%, on the bucket that carries a third of an EV. Splitting on quantiles of
// each game's own prices instead was measured too, and is worse.
var mkmTrendBands = []float64{0, 1, 2, 3, 5, 7, 10, 20, 50}

func mkmBandOf(trend float64) int {
	for i := len(mkmTrendBands) - 1; i > 0; i-- {
		if trend >= mkmTrendBands[i] {
			return i
		}
	}
	return 0
}

// mkmCalibration turns a Trend price into what a market listing sits at.
type mkmCalibration struct {
	// intercept and slope describe multiplier = intercept + slope*ln(trend),
	// fitted across every polled card. Two numbers answer at any price, which
	// is what lets a game with a few hundred cards be calibrated at all.
	intercept float64
	slope     float64

	// bands holds a band's own measured multiplier, and zero where the band
	// had too few cards to measure one - there the curve answers instead.
	bands []float64

	samples  int
	measured int
}

// multiplier is what Trend is worth at this price.
func (c *mkmCalibration) multiplier(trend float64) float64 {
	if band := mkmBandOf(trend); c.bands[band] != 0 {
		return c.bands[band]
	}
	fitted := c.intercept + c.slope*math.Log(trend)
	return math.Min(math.Max(fitted, mkmMultiplierFloor), mkmMultiplierCeiling)
}

// estimate prices one card from the guide's two columns.
func (c *mkmCalibration) estimate(low, trend float64) float64 {
	// A product Trend says nothing about is one with a single listing, and
	// that listing is the price: the measured median of market over Low
	// there is 1.001, with no spread at all.
	if trend <= 0 {
		return math.Max(low, 0)
	}
	return math.Min(math.Max(c.multiplier(trend)*trend, low), mkmTrendCeiling*trend)
}

// fitMKMCalibration measures the scale against every card carrying both a
// market price and a Trend, in one pass that completes before anything is
// written back - an estimate filed under the market's own name would
// otherwise become training data for the next run and drag the fit toward
// whatever it already said. It returns nil where there is not enough to
// measure, which leaves the game unpriced rather than guessed at.
//
// Both sides are read the way an EV reads them - near mint, with lightly
// played behind it - rather than near mint alone, deliberately: what this
// predicts has to be the number the caller will go on to ask for, and on the
// 0.7% of polled cards with no near mint listing at all, that number is the
// played one. The guide columns never fall back, carrying near mint and
// nothing else, so the mixing is one-sided and slight: fitting near mint
// alone moves the intercept by 0.002 and no band by more than 0.005, against
// a day-to-day drift of 0.006 to 0.008. It does not move the way it looks
// like it should, either - a card with no near mint copy is one with a thin
// market, which is the same thing that makes market-over-trend high, so those
// rows sit above the typical multiplier rather than below it.
func fitMKMCalibration(b *mtgmatcher.Backend, r *BANPriceResponse) *mkmCalibration {
	var n, sumX, sumY, sumXY, sumXX float64
	bandSum := make([]float64, len(mkmTrendBands))
	bandNum := make([]int, len(mkmTrendBands))

	for _, uuid := range b.GetUUIDs() {
		market := r.getRetail(b, uuid, mkmStore)
		trend := r.getRetail(b, uuid, mkmTrendStore)
		if market <= 0 || trend <= 0 {
			continue
		}
		ratio := math.Min(market/trend, mkmRatioCap)
		x := math.Log(trend)

		n++
		sumX += x
		sumY += ratio
		sumXY += x * ratio
		sumXX += x * x

		band := mkmBandOf(trend)
		bandSum[band] += ratio
		bandNum[band]++
	}

	if n < mkmMinFitSamples {
		return nil
	}

	c := mkmCalibration{samples: int(n), bands: make([]float64, len(mkmTrendBands))}
	// The two terms of the denominator cancel to the spread of the trends.
	// Where there is no spread - a catalog priced at one number - what is
	// left is rounding, and a slope divided by it is arbitrarily large, so
	// the curve stays flat at the mean instead.
	if denominator := n*sumXX - sumX*sumX; denominator > 1e-9*n*sumXX {
		c.slope = (n*sumXY - sumX*sumY) / denominator
	}
	c.intercept = (sumY - c.slope*sumX) / n
	for band := range c.bands {
		if bandNum[band] >= mkmMinBandSamples {
			c.bands[band] = bandSum[band] / float64(bandNum[band])
			c.measured++
		}
	}
	return &c
}

// fill prices one card the market scraper never answered for, so that an
// opening counts it as the card it is rather than as zero, and reports
// whether it did. It is per card rather than a pass of its own because the
// caller is already walking the catalog; only the measuring above has to see
// every card before any of them is written.
//
// A nil calibration fills nothing, which is how a game with too few polled
// cards to measure one stays unpriced instead of guessed at.
//
// The price lands under the near mint key because that is the one a reader
// looks in first, not as a claim about what condition is on the shelf.
func (c *mkmCalibration) fill(b *mtgmatcher.Backend, r *BANPriceResponse, uuid string) bool {
	if c == nil || r.getRetail(b, uuid, mkmStore) > 0 {
		return false
	}
	low := r.getRetail(b, uuid, mkmLowStore)
	trend := r.getRetail(b, uuid, mkmTrendStore)
	if low <= 0 && trend <= 0 {
		return false
	}
	price := c.estimate(low, trend)
	if price <= 0 {
		return false
	}
	r.setRetail(b, uuid, mkmStore, price)
	return true
}
