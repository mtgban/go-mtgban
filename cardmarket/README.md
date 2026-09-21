# cardmarket

Prices Cardmarket across the seven games it and `mtgban` both carry — Magic,
Lorcana, Riftbound, One Piece, Yu-Gi-Oh, Flesh and Blood, Pokemon — via three
scrapers built on `github.com/mtgban/go-cardmarket`, the signed API client.
Gundam and Palworld are absent on purpose: Cardmarket itself does not sell
either (`mkmGames` names the seven it does, and `go-cardmarket`'s own `Game*`
constants list nineteen games total, twelve of which `mtgban` has never
modeled at all — see "Known gaps").

- **`Index`** (`cardmarket.go`, `idmap.go`) reads Cardmarket's published price
  guide: one bulk download per game, no credential, the low and trend columns
  as a lagging aggregate.
- **`Sealed`** (`sealed.go`) reads live listings for sealed product through
  the authenticated `Articles` endpoint, one request per product.
- **`Market`** (`market.go`, `market_filter.go`) reads the same live listings
  for singles, per printing rather than per product — the newest of the
  three, and the one most of this document is about.

All three need `MKM_APP_TOKEN`/`MKM_APP_SECRET` except `Index`, which is
public. `Market` additionally needs `BAN_API_KEY` (see "The offline
pre-filter").

## The shared resolver

`resolver.go` holds what `Index` and `Market` both need and neither owns
alone: turning a Cardmarket product — its name, number, expansion — into an
`mtgmatcher` uuid. Id-map lookup first, then name/number matching per game,
twins, foreign shelves, print runs. It used to live entirely on `Index`;
`Market` needed the identical logic and inherited it by embedding `resolver`
rather than reimplementing a thinner version of it, once for the eleven
original methods and twice more (`disownBridged`, `reportRefused`) for two
found only once `Market` actually needed them and `Index`'s own tests turned
out not to exercise directly enough to have caught the gap on the first
pass. Pure extraction — no logic changed, only where it's defined — verified
by running `Index`'s full pre-existing test suite unchanged before and after.

**A shared mcmId can name the wrong printing, and `Preprocess`'s own default
could throw away the one thing that would have caught it.** Found from a
live run: id 272488, "Urza's Mine (V.2)" under Chronicles, kept a Chronicles
Foreign Black Border (Japanese) printing instead of one of its own four
plain English arts. Two bugs, both real and both in code `Index` shares.
`Fallback`'s `checkLoadedID` trusts whatever mcmId-matching id it finds last
when none of them agrees with the product's own number - fine for genuine
cosmetic siblings (its own doc comment's "30A frame pairs"), wrong here,
because mtgjson stamps all four BCHR arts with 272488, the id of one of the
four *different* CHR arts instead; the four never agree with each other on
number, let alone the product's. Fixed by deferring to Preprocess/Match
instead of guessing when that shape shows up. Second, separately: Chronicles
has no dedicated case in `Preprocess`'s edition switch, so it fell to the
generic default ("old editions keep the V.1/V.2 style for variants.go to
resolve") - except that default overwrote `Variation` with Cardmarket's own
Number unconditionally, discarding the "(V.2)" `magic.VariantsTable`'s
`chrVariants` already carries a `"v.2"` key for, whenever Cardmarket's
Number field was non-empty - which for these same-numbered siblings, it
always is. Fixed by only taking the number when `VariantsTable` (resolved
through `magic.EditionTable`'s alias first, the same one `AdjustEdition`
consults later - Cardmarket's own edition spelling rarely matches the
matcher's) has no entry for this exact edition/card/variant to consult
instead. The same gap, found the same way, turned up one edition over:
"Fourth Edition: Alternate"'s three-art basic lands were failing to resolve
with "unknown variant" in the same run, for the same reason - fixed by the
same guard, plus a real, separate bug the fix surfaced: `ed4Variants`
itself named the bare 4ED number for a card mtgban's own loader only ever
synthesizes with "alt" appended (`Alternate Fourth Edition` is not a real
mtgjson set - `duplicate()` builds it from 4ED at load time), so none of
its values ever matched the card FilterCards was checking them against.
Pinned with a test that would have caught it directly:
`TestEd4VariantsNumbersAreReal` checks every `ed4Variants` value against the
loaded set's own numbers, not just that the card name exists there the way
the existing `TestVariants` did.

## Market's design

**The List's foil printings are never priced.** It reprints a card under
whatever treatment its original printing had, and sellers list its foil
copies under the wrong finish often enough that pricing them is noise
rather than a real signal (`isPLSTFoil`, checked in `queryPrintings` before
the foil-slot query ever fires).

**Strictly sequential, not pooled.** Measured directly: a concurrency ladder
(1→24 workers) plateaus at ~1.4 successful calls/sec regardless of worker
count, with the excess coming back as 429s, while 20 fully sequential
requests at zero delay cleared 20/20 with no 429s at all, at essentially the
same throughput. This is a per-token *concurrency* limiter, not a rate
limiter — parallelism buys nothing here and only manufactures wasted 429s —
so `Load` walks the catalog one product at a time. `Sealed` still pools with
`WorkerPool`; do not copy that shape into `Market`.

**Every edition ends its walk with one summary line, always.** A CI run
showed why: `"Processing Guru Lands (88) [86/760]"` followed straight by the
next edition's own `"Processing ..."` line reads as a hang or a silent
no-op, not as "nothing here needed a mention" - the skip/refusal lines
`walkExpansion` already prints only appear when there is something to
explain, so an edition with nothing skipped or refused ended in silence.
`"<edition>: priced %d/%d products"` prints unconditionally now, counting
every product that reached a live query and came back without error -
unlike the other lines, this one is never gated on being non-zero.

**`Content-Range`-driven early stop**, shared with `Sealed` as
`contentRangeCovered` in `utils.go`: once the pages already fetched have
covered everything a listing's `Content-Range` total promises, another page
can only come back empty. A `capped` response reports Cardmarket's 1000-result
ceiling rather than the real total; the API rejects a request starting at 1000,
so the ceiling is still the stopping point and listings beyond it cannot be
recovered by pagination.

**Server-side filters are never trusted alone.** `isFoil`, `isReverseHolo`,
`isFirstEd` are documented to fail open on a game or value they don't apply
to — silently answering the unfiltered list, not erroring — so every listing
is also checked client-side against its own article flags (`acceptArticle`)
before being priced onto a printing. `marketFinishParam` names the games this
is actually verified for: Magic/`isFoil`, Pokemon/`isReverseHolo`,
YuGiOh/`isFirstEd`, and — confirmed directly, not assumed from `isFoil`'s own
documentation, which names only Magic and Pokemon (deprecated) — Lorcana and
Riftbound both read `article.IsFoil` too. That documentation is about the
request-side filter, a separate question from whether the response's own
flag is populated for other games. A spread sample of thirty live products
(fifteen per game) confirmed it is: products selling nothing but foil copies
at a clear premium over their game's bulk floor (a Riftbound rare at
$15–$1000, a Lorcana one at $35–$1174, against $0.02–$0.03 commons), and
products mixing a handful of foil listings into an otherwise-plain one with
those listings priced distinctly above the plain floor — exactly the shape
`cardID`/`cardIDFoil` being the nonfoil/foil match of the same product
predicts, not noise. The request-side `isFoil` param's own narrowing effect
for these two games was not separately checked, and does not need to be —
correctness rests on the article-level check either way.

**Listings are not strictly price-ascending.** Replayed a real 100-listing
page (`testdata/articles_riftbound.json`, a heavily bulk-priced Riftbound
common) through `acceptArticle` rather than trusting a hand-built fixture,
and it proved a documented assumption wrong: the raw JSON prices ran
`0.02, 0.02, 0.03, 0.03, 0.02, 0.03, 0.02, ...` — the same handful of
cent-level prices repeating out of order, not climbing, at full wire
precision (not a rounding artifact on a longer float). "Hold the first
acceptable listing per condition" would occasionally hold one that wasn't
actually the cheapest. Fixed by holding the *cheapest* seen per condition
instead of the first, deferring emission to the end of the scan rather than
streaming immediately — both changes cost nothing extra, since the whole
page is scanned either way. `TestReplayListingsAreNotStrictlyPriceAscending`
pins the finding directly so a future tightening on Cardmarket's side would
surface as a test starting to skip, not as a silently stale assumption.

`CARDMARKET_MARKET_PATH` overrides the checked-in fixture with a fresher
export from `Articles()` itself, in the same `[]cm.Article` JSON shape — the
same `<GAME>_PATH` convention every datastore variable already uses.

**A 20-consecutive-failure circuit breaker.** `bounce`/`maxBounced` track
requests since the last usable answer (a price, or a clean empty result);
crossing the threshold aborts the whole `Load` run rather than grinding
through the rest of the catalog against a token Cardmarket is rejecting
outright.

**A second, Powerseller-only view.** Market implements `mtgban.Market`
(`MarketNames`/`InfoForScraper`), the same interface `cardtrader.Market`
uses to split into "Card Trader"/"Card Trader Zero"/"Card Trader 1DR" -
every entry gets one of two fixed `SellerName`s, `"Cardmarket"` or
`"Cardmarket Powersellers"`, with the article's own username moved to
`CustomFields["SubSellerName"]` instead. The Powerseller bucket holds its
own independent cheapest-per-condition view, live-restricted to German and
Dutch sellers whose `isCommercial` reads `2` - confirmed directly against
two real accounts on Cardmarket's own seller pages, `1` shows a
"Professional" badge and `2` shows "Powerseller"; nothing else in the
documented `0`/`1`/`2` scale reads as a power seller. Once the *main*
bucket's NM/SP/MP are all held, `queryOnePrinting`'s page loop spends up to
`marketPowersellerExtraPages` (4) more pages specifically chasing at least
one Powerseller listing before giving up on this product - a bounded,
paid-once fail-safe (`shouldStopPaging`), not a guarantee: gating the stop
on full Powerseller completeness (all three conditions, not just one)
would mean every product with no qualifying seller at all - most of them -
pages all the way to `Content-Range` coverage instead of stopping early,
which is exactly the budget this scraper is built around not spending. A
product with no qualifying seller anywhere in its listings still
contributes nothing to this bucket, the same as a product neither Zero nor
1DR carries for Card Trader's own split. `bantool` needs no new
registration for this:
`UnfoldScrapers` (already called for every scraper) dumps one output file
per `MarketNames` entry automatically, keyed by `InfoForScraper`'s own
Shorthand (`MKM`, `MKMPS`).

## The offline pre-filter

All seven games are restricted to a candidate set (`marketCandidates`)
computed from a snapshot
of `mtgban`'s own published prices, before `Market` ever queries Cardmarket
live: a card priced over $7, or an Arbit-style spread against a buylist
vendor (CK → SCG → CSI, first available) over 20%, or a Mismatch-style
spread against another retail seller over 80% — each spread leg guarded by a
price floor and minimum absolute difference (`marketFilterParams`) so a raw
percentage spread isn't dominated by cent-level noise on bulk cards. Magic,
Pokemon and YuGiOh need this to fit a nightly scrape budget at all; Lorcana,
Riftbound, Flesh and Blood and One Piece fit their budget unfiltered too,
but only clear the filter on 9–26% of their own priced uuids (measured
directly against each game's own snapshot), so filtering them still saves
real call volume against the one daily allowance this app token shares
across every game.

### The wrong-host saga

The price snapshot is published **one per game subdomain**
(`pokemon.mtgban.com`, `yugioh.mtgban.com`, ...), not shared. `loadBanSnapshot`
originally hardcoded `www.mtgban.com` — copied from `sealedev`'s own loader,
which is correct there only because `sealedev` is wired for Magic alone.
Querying the wrong host for another game answers with a real `200` and an
empty result: no error, just none of that game's own uuids in it. That is
exactly how Pokemon and YuGiOh's candidate sets went silently empty during
development — `marketCandidates` built a real, non-nil, *empty* map, and
`walkCatalog` read that as "skip everything," so `Market` would have priced
**zero** cards for either game had it actually run. Fixed by building the
snapshot URL per game (`banHost`), and it also corrected the candidate counts
measured earlier against the wrong host: Pokemon dropped from an estimated
14,012 candidates to a measured 11,247; YuGiOh from 8,170 to 6,085.

Two more ways the same fetch can go wrong without ever returning a non-200
status, both confirmed live and both guarded in `parseBanSnapshot`:

- **A rejected signature** answers `200` with `{"error": "..."}` instead of
  the `retail`/`buylist` fields — confirmed against One Piece's own host
  during development, which briefly rejected this app token's signature
  outright: a misconfiguration on a separate backend deployment (its
  `x-do-app-origin` header named a different DigitalOcean app instance than
  every other game subdomain), fixed server-side and re-measured clean
  afterward. Decoding that body into `banSnapshot` without checking `Error`
  first would silently produce the same empty-but-real-looking result as the
  wrong-host bug did.
- **A snapshot with no prices at all** — belt-and-braces past the two cases
  above, since nothing rules out some other reason a game's snapshot could
  come back genuinely empty.

Both read as "the fetch itself failed," not "nothing passed the filter" —
the same silent-failure shape the empty-body-on-error bug in
`go-cardmarket`'s own `get()` was, and the same principle: a non-200 or an
explicit error field is a real failure regardless of what status line
carries it.

## Known gaps

- **Magic, Pokemon and YuGiOh's workflows carry no schedule yet.** Even
  filtered, all three can run up to 12h — past what a GitHub-hosted
  `ubuntu-latest` job tolerates (a 5-6h practical ceiling). They're wired as
  `workflow_dispatch`-only in `.github/workflows/bantool-cardmarket_market*`
  until routed to a self-hosted runner.
- **Twelve of `go-cardmarket`'s nineteen `Game*` constants have no `mtgban`
  equivalent** — World of Warcraft, The Spoils, Force of Will, Cardfight!!
  Vanguard, Final Fantasy, Weiss Schwarz, Dragoborne, My Little Pony, Dragon
  Ball Super, Star Wars Destiny, Digimon, Battle Spirits Saga, Star Wars
  Unlimited. `mkmGames` covers exactly the games `mtgmatcher/games` blank-
  imports; adding any of these needs the full "Adding a game" checklist in
  `AGENTS.md` first; this package is the last, not the first, step.

## Environment

- `MKM_APP_TOKEN`, `MKM_APP_SECRET` — Cardmarket API credentials, needed by
  `Sealed` and `Market`, not by `Index`.
- `BAN_API_KEY` — authenticates the offline pre-filter's snapshot fetch; the
  same key `sealedev`'s own loader uses. Wired unconditionally for every
  `Market` scraper in `cmd/bantool/scrapers.go`. An unset key never surfaces
  at construction, only at `Load` time, and only for Magic, Pokemon and
  YuGiOh (`marketFilterRequired`) — the three whose catalogs don't fit a
  nightly budget unfiltered, so `Load` refuses to run them without it. The
  other four games in `marketFilterParams` fall back to running unfiltered
  instead: a missing key there loses call-volume savings, not the ability
  to run.
- `MTGJSON_MKMID_PATH` — the id-map catalog `Index` and `Market` both resolve
  products from; MTGJSON's own for Magic, `go-cardmarket`'s `mkmcatalog`
  builds the rest.
- `CARDTRADER_TOKEN_BEARER` — the bridge Flesh and Blood, Pokemon and Yu-Gi-Oh
  require and One Piece merely improves with, for all three scrapers alike
  (`TestCardmarketNeedsItsBridge` pins the refusal).
- `CARDMARKET_MARKET_PATH` — overrides the checked-in live-listing fixture
  `market_replay_test.go` reads by default.
