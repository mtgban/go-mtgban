# sealedev

Prices sealed product by what opening it is worth, rather than by reading a
storefront. There is no site behind this scraper: it asks `mtgmatcher` what a
product can contain and with what probability, prices those contents against
the MTGBAN price API, and publishes the total. Registered as `sealed_ev`, for
Magic alone — `banAPIURL` hardcodes `www.mtgban.com`, which serves Magic's
prices and nothing else, and `register.go` names `mtgban.GameMagic` as the
only game it answers for.

It needs `BAN_API_KEY` (`SecretBanKey`), and nothing else. One API call
fetches the whole price snapshot; everything after that is arithmetic.

## Twelve entries, two shapes

`evParameters` is the whole configuration surface. Each entry names a price
source and how to reduce a product's openings to one number, and each becomes
its own sub-seller:

| Name | Shorthand | Source | Shape |
| --- | --- | --- | --- |
| Singles Buylist (est.) | `SS` | `CK`, `SCG` | buylist |
| TCG Low EV / Sim | `TCGLowEV` / `TCGLowSim` | `TCGLow` | retail |
| TCG Direct (net) EV / Sim | `TCGDirectNetEV` / `TCGDirectNetSim` | `TCGDirectNet` | retail, priced off the buylist side |
| CT Zero EV / Sim | `CTZeroEV` / `CTZeroSim` | `CT0` | retail |
| Mana Pool EV / Sim | `MPEV` / `MPSim` | `MP` | retail |
| Cardmarket EV / Sim | `MKMEV` / `MKMSim` | `MKM` | retail |
| TCG Direct SYP (net) EV | `TCGDirectSYPNetEV` | `TCGDirectSYPNet` | retail, priced off the buylist side |

`FoundInBuylist` says which side of the snapshot a source is read from;
`TargetsBuylist` says which side the result is published to. Only
`Singles Buylist (est.)` sets the second, which is why `MarketNames` returns
eleven sub-sellers rather than twelve, and why `InfoForScraper` marks every
one of those eleven `MetadataOnly` — they are derived numbers, not an offer
anyone can take.

**EV and Sim are two answers to the same question, not two sources.** An EV
entry sums every possible card weighted by its probability: one deterministic
number, the mean of infinitely many openings. A Sim entry opens the product
`repetitions` times (5,000 by default) through
`backend.GetPicksForSealed` and takes the median, which is what a person
opening one box is more likely to actually see. The gap between them is the
skew, and it is large for anything with a chase card. Sim entries also carry
`stdDev` and `iqr` in `ExtraValues`; EV entries carry neither, having no
distribution to describe.

A product whose contents are fixed — `backend.SealedIsRandom` says so —
skips the Monte Carlo entirely and copies the deterministic value, because
simulating a deck that always contains the same cards 5,000 times answers the
same number 5,000 times.

## The price pass

`loadPrices` fetches `all[/SET].json?tag=tags&conds=true` once and then makes
one pass over the catalog, and that pass does more than it looks like:

- **`getPrice` reads near mint, then lightly played.** A card with no NM
  quote is priced from its SP one rather than dropped, per finish — a foil
  reads `NM_foil`, an etched card `NM_etched`. Reading the plain key for a
  foil would price it at its nonfoil sibling.
- **A price over `MaxSinglePrice` ($10,000) is discarded as broken**, except
  in LEA, LEB, 3ED, ARN and LEG, where five figures is a real price.
- **Anything under `BulkThreshold` ($0.50) is pruned**, from every store, and
  pruned *after* the adjustments below so that an estimate is held to the
  same floor as a quote.

Pruning is what stops a booster's worth of $0.03 commons from accumulating
into a meaningful-looking EV, at the cost of understating a product genuinely
made of bulk. That trade is uniform across stores on purpose: the EV columns
are meant to be compared to each other, and a source with its own threshold
would read as cheaper for a reason that has nothing to do with its prices.

## Two of the sources are estimated, not read

This is the part most likely to surprise someone reading an EV number.

**TCG Direct (net)** is derived in `loadPrices`. Where the buylist carries no
Direct quote, it is estimated from TCG Market (falling back to TCG Low) put
through `tcgplayer.DirectPriceAfterFees`. Where it carries one that looks
wrong — `directNet/2 > tcgMarket`, i.e. Direct claims more than twice what
the card sells for — it is either capped at twice Low or dropped outright.
`TCGDirectSYPNet` is computed unconditionally from `TCGDirect`, and `CT0` has
its flat per-band fee (`getCT0fees`) subtracted in place.

**Cardmarket** is the larger estimate, and `mkm.go` is entirely about it.
Cardmarket's market scraper only spends a live API call where one is worth
spending — a trend price over $7, or a wide enough spread, see
`cardmarket.marketCandidates` — so roughly **86% of the catalog has a
published guide price and no usable market price** (129,720 of the 151,466
Magic cards carrying a guide entry, measured 2026-09-22). A search page can
leave that blank. An EV cannot: a card with no price counts as zero, not as
unknown, and a booster is mostly cards that were never polled.

So the gap is filled from the guide's Low and Trend columns. Trend, scaled, is
the estimate; Low is a floor rather than half of an average. The scale is
fitted from the same snapshot being priced rather than written down, because
it differs per game and moves with the market. `mkm.go`'s own doc comment
carries the measured accuracy, the per-game breakdown, and the two games the
estimate is not honest enough for.

The consequence worth internalising: **a Cardmarket EV can move between two
runs because the calibration moved, not because any price did.** The drift is
about 1% a day, which moves a priced catalog by 0.2%, and
`BenchmarkFitMKMCalibration` is where to read the fitted numbers off — nothing
in a run prints them.

## What never counts toward an EV

`skipFromEV` drops a card from the contents entirely:

- **Serialized** printings, which have no usable market price at all.
- **Cosmic foil** printings, same.
- **Secret Lair bonus cards** whose published distribution is not fixed
  (`probability < 1`). A fixed one counts.

These are dropped before pricing, so they cost nothing and contribute
nothing — the EV is the value of the openable rest.

Whole products and sets are skipped earlier, in `Load`:

- Sets with no usable sealed or booster data: `FBB`, `4BB`, `DRKITA`,
  `LEGITA`, `RIN`, `4EDALT`, `BCHR`.
- Products of category `land_station`.
- Products with `Japanese` in the name, whose contents are not the English
  ones the prices describe.

## Running it

`WithTargetProduct` narrows to a single product by name or UUID;
`opts.TargetEdition` narrows to one set, and does double duty — it also
narrows the price fetch to `all/<SETCODE>.json`, which is the difference
between one small request and the whole catalog. Both are the way to iterate
on this without a multi-minute run.

`maxConcurrency` (8 by default) is the width of the simulation, not of any
network work — there is only ever one request. `repetitions` is settable in
tests to make a run that cannot finish, or to make one finish instantly.

Retail entries link to TCGplayer where the product carries a
`tcgplayerProductId`, with the Direct flag set for the TCG Direct (net)
entries only; the buylist entry links back to a `contents:` search on the
product's own name.

## Known gaps

- **Magic only**, and the constraint is `banAPIURL` rather than anything
  structural: each game's prices are published on its own subdomain, so
  another game needs that URL parameterised before anything else is worth
  doing. `mkm.go`'s calibration is already per-game and would follow on its
  own; the accuracy notes there say which games would survive the move.
- **`maxStorePrice` takes the highest price across a source's stores**, which
  matters only for `Singles Buylist (est.)` — the one entry naming more than
  one store (`CK`, `SCG`). Every other source names exactly one, so the max
  is over a single value.
- **An EV is only as good as the contents data.** `GetProbabilitiesForSealed`
  returning nothing is reported per product and the product is skipped; a
  product whose contents are *wrong* is not detectable here at all, and shows
  up as an EV that looks fine.
- **The estimated sources are not marked as estimated** in the published
  entry. A consumer cannot tell a read Cardmarket price from a fitted one.
