# starcitygames

Prices Star City Games' singles and sealed product, buy and sell, across
every game they carry: Magic, Flesh and Blood, Lorcana, Riftbound. Wired into
`cmd/bantool` as eight scrapers — one singles and one sealed scraper per
game — via `NewScraper(game, apiKey)` / `NewScraperSealed(game, apiKey)`.

This file is the accumulated knowledge from fixing a long run of mispriced
listings in this scraper — `mtgban/go-mtgban` PRs #493, #572, #573, #574,
#578, #582. The short version: **SCG hands two different printings the same
identifier often enough that this is the normal case to defend against, not
an edge case.** Read on for why, how it's caught, and how to catch the next
one.

## The data source

Everything comes from one export: `scgCatalogURL`, SCG's HawkSearch catalog
download, authenticated with an `x-api-key` header. It streams as one long
JSON array — retail (`price`, `qty`) and buylist (`sell_list_price`) for
every variant of every product, across every game, in one response. A single
`StreamCatalog` pass fills both records.

**The export is exhaustive — there is nothing else to find.** Dumped raw and
diffed against `CatalogProduct`/`CatalogVariant`: 14 top-level keys, 8 per
variant, nothing else. No artist, no promo-series name, no language beyond
the one `language` string, no second identifier space. Before writing a rule
that reads a vendor field, confirm the field is actually there (`python3 -c
"import json; print(json.load(open('catalog.json'))[0])"` on a fresh pull) —
several fixes in this history were held up by assuming a field existed, or
assuming one didn't, without checking.

A second, older data source is still live in two places:

- **`sell_list_products_v2`**, a public MeiliSearch index at
  `search.starcitygames.com`, same instance and same embedded search key as
  `sets_v2` (which `buylist.go`'s `SetIDs` already queries, for the
  sell-your-cards bookmark links). It's what `Hit`/`Variant` in `api.go`
  model — the scraper used to read this index directly, and `catalogHit`
  still synthesizes a compatible `Hit` from a `CatalogProduct` so
  `preprocess.go`'s wording path didn't need rewriting. It carries fields
  the catalog export doesn't: `primary_status` / `bulk_category_name` (see
  Bulk tiers, below), and a `subtitle` field close to a display name
  (`"Pleiades, Superstar"` / `"(Maori)"`) that turned out to be the only
  place SCG writes some distinguishing detail down at all.
- **`sets_v2`**, queried live by `SetIDs` to build sell-your-cards bookmark
  links (`SCGBuylistURL`).

Neither index is consulted during normal resolution — only the catalog
export is. When a fix needs data the catalog doesn't carry, check this index
before assuming the data doesn't exist anywhere (see Aurora, below, where it
did — `datastore-gen` grew the field, not this scraper).

## The core problem

SCG's own `scryfall_id` / `tcgplayer_id` are the fast path: `resolveProductID`
tries them first (after a handful of sku-driven steers that have to run
*before* the identifiers, described below), and when the id is present and
un-contradicted, it resolves directly with no wording involved.

The catch: **those ids are sometimes wrong, and sometimes shared between two
genuinely different products.** A League Tokens promo carries the *set
token's* Scryfall id, not its own. A Timeshifts retro-frame carries the
original printing's id. An Ampersand embossed card and its plain promo-pack
twin carry each other's. When that happens, both catalog products resolve to
the same uuid, and `mtgban/base.go`'s `Add` used to keep whichever arrived
with the higher price — silently, no error, no log line. A $0.29 token
listing would carry a $34.99 promo's price, or vice versa, and nothing said
so.

**The fix is never "trust the vendor id less in general."** A vendor's id is
theirs, and overriding one is a per-vendor, per-shape call, made after
proving the specific shape wrong, not a blanket discount. Every fix in this
history is a **named, provable shape**: replay the product through
`resolveProduct`, dump what the datastore actually holds at that
(set, number), and either write a rule keyed on a real field the product
carries, or — when nothing distinguishes the two — a closed table.

## `resolveProduct` — the pipeline

`resolveProduct` wraps `resolveProductID` with two checks that apply after
resolution, regardless of game or path taken:

1. **Language.** A non-English `language` field has to match the resolved
   printing's language, or refuse — this is what keeps the inherently
   foreign sets (Foreign Black Border, Rinascimento, …) from collapsing onto
   their English twin. English itself asserts nothing (it's the sku
   grammar's default, not a claim).
2. **Store Championship foil-only refusal.** A `SCHP_`-numbered sku sold
   unfoiled, landing on a foil-only printing, is refused — there is no
   unfoiled version of that card to be. Scoped to `SCHP_` specifically after
   checking the general shape (any unfoiled product landing on a foil-only
   printing) also caught three unrelated, real, differently-broken products
   that should have been *fixed*, not silenced — see Method, below.

`resolveProductID` then branches by game. In every branch, **sku-driven
steers run before the identifiers get a say** wherever the identifier is
known to name the wrong thing outright (Portal's `b`-suffixed second
printing, a lettered Magic prerelease, `promoShelfPrintings`) — for those,
trying the id first and discarding it would still cost the wording path a
chance to run cleanly. Everywhere else, the id is tried first and
**vetted** by `idContradictsProduct` before being trusted.

### Magic

Order: Portal `b`-suffix override → lettered prerelease (`PRE_..b`) → Duel
Decks: Anthology → `promoShelfPrintings` (sku-keyed, closed table) →
`ScryfallID` then `TCGPlayerID` through `idContradictsProduct` (with
`resalePrinting` applied to a surviving id) → WAR Japanese planeswalker /
`PP_` promo-pack fixups → generic `preprocess` + `mtgmatcher.Match` wording
fallback.

**`idContradictsProduct`** is the general defense — a resolved id is
rejected (falling through to the wording path) when the *product itself*
says something the id doesn't back up:

| guard | catches |
|---|---|
| `Double Rainbow` finish | id isn't tagged `doublerainbow`/`rainbowfoil`/`serialized` |
| `-AMP_` sku | id isn't tagged `embossed` |
| tagged `embossed` | sku isn't `-AMP_` and finish doesn't say `Embossed` (the dual of the row above — an Ampersand and its plain promo-pack twin share an id both ways) |
| `MH12` shelf | id isn't tagged `boosterfun` (Modern Horizons Timeshifts vs. the original printing) |
| `WCHP` shelf | id's set isn't the year's own deck set (`worldsSetFromSKU`) |
| `SSD_` shelf | id isn't tagged `standardshowdown` (Standard Showdown is an *event*, not a set — the sku's year has to pick among several) |

Each guard exists because a *survey* (replay the whole catalog, group by
resolved uuid, look for two skus on one id) found the shape live and
measured its blast radius before shipping — never guessed from the error
text alone. See Method.

**`resalePrinting`** handles WotC resale (`RESL_`) skus the same way: the id
names the right card in the right set but the wrong treatment (a promo pack
instead of the resale reprint), and the sibling search is bounded to the
same set + name so it can't wander.

### Flesh and Blood

FaB carries **no identifiers at all** (`scryfall_id`/`tcgplayer_id` are
always empty) — resolution is 100% sku- and wording-driven, via `fabMatch`
trying the sku's own numbers (most specific first) against
`mtgmatcher.Match`. Four steers run on a successful match, each closing a
*different* shape of "two products, one number, nothing else says which is
which":

| steer | what it fixes | keyed on |
|---|---|---|
| `fabPlainSibling` | a non-`"Marvel"` rarity landing on the Marvel-tagged twin (nothing excluded it) | a **general rule** — `rarity != "Marvel"` excludes a `marvel`-promoType candidate. Scoped: surveyed the whole datastore, 36 `(set, number, foil)` keys hold both a Marvel and non-Marvel row, 32 clean |
| `fabRenamedTwin` | SCG sends the same name for two *actually different cards* (English "Pleiades, Superstar" vs. the Maori printing, filed as "Hinewhitu, Hautipua" with the English name folded into a promo type) | closed table, one entry — surveyed for any other renamed pair, found none |
| `fabCreditedTwin` | two artist-commissioned prints share a name once "(Marvel)" is folded in (Aurora ROS008) | closed table, keyed on `Artist` — a field `datastore-gen` only started publishing (see below); **the go-to move once a field exists that the catalog itself never will** |
| `fabMarkedSibling` | a `fabVariantMarker`-suffixed set code (`SUP2`/`ROS2`/…, distinct from a lettered *number*) names a second printing at the same number, told apart only by a treatment the datastore carries in `PromoTypes` | reads the sibling directly, no table |

And one **refusal** table: `fabDuplicateStock` — Sanctuary of Aria's `027a`
(titled "Assorted Back" on the storefront) and `027b` ("Feet") both sell the
*one* row the datastore has for ROS027, at their own price, and nothing
says which is the card's price. Resolving either was letting whichever
streamed first win the id, silently, forever — refusing both is
deterministic. This is the shape to reach for when a survey finds *no*
distinguishing field anywhere, not even in the sell-list index — see the
next section for what happens when one *does* turn up.

**These four mechanisms are easy to blur together; keep them apart by what
each is keyed on:**
`fabPlainSibling` — a *rule*, general, no table. `fabRenamedTwin` /
`fabCreditedTwin` — *closed tables*, because a survey proved the shape is a
genuine singleton with nothing to generalize. `fabDuplicateStock` — a
*refusal* table, because nothing distinguishes the pair at all, in any
source checked. Reach for the rule first; drop to a table only once a
survey shows the rule can't be derived; refuse only once a survey shows
even a table can't be built honestly.

### Lorcana

Reads its number off the sku (`lorcanaNumber`), which is more specific than
the product's own `collector_number` field (it keeps a printing letter the
product field drops). A lettered number is resolved one of two ways the
datastore uses interchangeably: the sibling carries the same letter as its
own number (`resolveLorcana` finds it directly), or the datastore instead
numbers both printings alike and tells them apart by name — a longer name
extending the base card's own (`"Bucky - Squirrel Squeak Tutor (Errata
Version)"` beside plain `"Bucky - Squirrel Squeak Tutor"`) — in which case
`lorcanaSibling` searches by name and picks the sole agreeing candidate,
refusing outright if it doesn't resolve to exactly one card. `lorcanaFinish`
separately handles the one treatment (rainbow foil / `RainbowPillars`) two
products can share a bare foil flag but not a uuid.

### Riftbound

No known collision shapes yet. Straight `mtgmatcher.Match` on name +
set + number + foil.

## Bulk tiers (`bulk.go`)

A second, orthogonal defect, on the buylist side only: SCG's
`sell_list_price` is two different things wearing one field. A card SCG
will buy *individually* gets a real bid. A card it will only take at a
**flat rate for its whole rarity/finish tier** — the shop won't look at the
specific printing — gets the tier's number instead: every Magic Rare
$0.08, every Common $0.008, no matter which Rare or Common it is. That's
96,975 of 159,874 Magic products, and publishing the flat rate as if it
were a bid prices the card at whatever the tier pays.

The catalog export has no field for this either. The MeiliSearch index does
(`primary_status`: `buying_at_cost` vs `buying_in_bulk`), but joining it live
is a dependency this scraper doesn't take on. Instead, `bulkBuyRates(game,
rarity, finish)` is a **table of the rates each tier actually quotes**,
derived once by crawling the index and comparing its `sell_list_price`
against `primary_status`, and it's small: each tier quotes only a handful of
distinct numbers.

**This table goes stale the moment SCG reprices a tier**, silently — nothing
in the catalog export flags it. The mitigation is a per-run counter,
`scg.bulkRated`, logged as `"Processed N products total, M buylist prices
were a bulk rate"`. Watch for `M` falling off a cliff between runs; that's
the signal to re-derive.

**Re-deriving:** `~/src/claude-scratchpad/ci-sweep/scg-bulk/crawl.py` (one
per game id: 1 Magic, 2 FaB, 3 Lorcana, 5 Riftbound — Magic alone needs
per-rarity slicing, the index paginates at 45,000) then `derive.py <dump-
dir>` scores a mechanically-derived table against the real one. It
deliberately can't reproduce two rates: Magic's Unstable cards are ordinary
Rares/Mythics to the catalog and their own $0.001 tier to the buylist (a
straggler at 0.4% of the combined tier, correctly *not* narrowed away), and
FaB's cold-foil promos are excluded from the cold-foil tier entirely (a
tier with zero bulk rows produces no rate for the deriver to see). Read
`derive.py`'s own docstring before trusting a re-derivation — it explains
both.

`scgMinBuyPrice` is a $0.01 floor applied *after* the tier check, on
purpose: most of what a tier quotes is itself under a cent, and letting the
tier check run first is what keeps the per-run counter meaningful as the
staleness signal.

## `AddUnique` — the backstop

`mtgban.InventoryRecord`/`BuylistRecord.Add*` come in four strictness
levels (`AddRelaxed` < `Add` < `AddStrict` < `AddUnique`). This scraper's
**primary** records call `AddUnique`: a second entry at one card, one grade
is this whole defect class's own signature, and it's now rare enough (post
`#572`, `#573`, `#574`, `#578`, `#582`) that refusing it loudly — one log
line per run — beats letting the dearer one silently win, which is what
`Add`/`AddStrict` used to do.

The **`_CC`-marked second-stock bucket** (`secondStock`, `bucketKey`,
`secondBucketMarker`) is a genuinely different, already-correct mechanism —
SCG occasionally splits one listing's stock across two product records
(`"SGL-FAB-AGB-014-ENN"` beside `"SGL-FAB-AGB-014_CC-ENN"`), agreeing on
everything but quantity, meant to be **summed**, not refused. It still calls
`Add`/`AddRelaxed`. Don't conflate the two: a `_CC` pair is one listing
written twice; an `AddUnique` refusal is two listings that shouldn't be one
price.

Switching to `AddUnique` was itself measured, not assumed safe — replayed
the full catalog through the real production `Add` calls before and after,
across all four games, checking the delta was small, expected, and
attributable (mostly Magic's known-deprioritized double-faced-token
collisions, already muted; a handful of genuine catches). See PR #578.

## Method — how to find and fix the next one

This is the actual repeatable process, extracted from doing it five times
(`#572`, `#573`, `#574`, `#578`, `#582` — `#493` was the bulk-tier problem,
a different method, its own section above):

1. **Pull a fresh catalog.** `curl -H "x-api-key: $SCG_API_KEY"
   https://api.starcitygames.com/hawksearch/catalog/download/json -o
   catalog.json` (~108MB). A catalog from even a few days ago can miss a
   product SCG has since added or repriced — don't reuse an old dump for a
   fresh finding.
2. **Replay, don't guess.** Decode into `[]CatalogProduct`, call
   `resolveProduct(b, game, p)` for real products against the backend the
   test loaded, group by returned uuid. Anything with more than one sku is a
   live collision. This is the same replay used throughout this history —
   see any of `#572`, `#573`, `#574`, `#578`, `#582` for the harness shape (a
   throwaway `zz_*_test.go`, never committed).
3. **Dump the raw record, every field, before writing anything.** `python3
   -c "print(json.dumps(next(p for p in catalog if p['sku']=='...'),
   indent=1))"`. This is what caught the Aurora case having genuinely
   nothing to key on in the catalog, and separately caught that the
   MeiliSearch index *did* carry the distinguishing detail (a `subtitle`)
   the catalog omitted.
4. **Check the datastore, not just the catalog.** `b.GetUUID(id)` and
   `b.MatchWithNumber("", setCode, number)` (name may be empty) to see every
   candidate row at that number, and what actually distinguishes them —
   `PromoTypes`, `Rarity`, `Artist`, `Finishes`. A field the struct exposes
   and the vendor's raw record also carries is worth trusting; a field
   neither carries is not there to key on, however plausible a rule sounds.
5. **If nothing distinguishes the rows anywhere — catalog, datastore, index,
   storefront page — that's a refusal, not a guess.** (`fabDuplicateStock`.)
   If the storefront page or the MeiliSearch index has the answer but the
   scraper's own inputs don't, that's either (a) a case for the closed-table
   pattern keyed on sku, verified once by hand against the live page/index
   (`fabRenamedTwins`, `fabCreditedTwins`, `promoShelfPrintings`), or (b) if
   it's genuinely general and datastore-gen can be taught to carry the field
   (as happened for FaB's `Artist`), the better fix is upstream — see next.
6. **Survey the blast radius before shipping either a rule or a table.**
   Every guard and steer above was sized this way: group the whole datastore
   by the shape the rule would key on, count how many `(set, number, foil)`
   keys it touches, check how many are "clean" (exactly the two-way tie the
   rule is meant to break) versus genuinely ambiguous. A rule that fires on
   30 clean cases and silently mishandles 4 ambiguous ones needs the
   ambiguous ones excluded explicitly, not ignored.
7. **Verify against the real production path, before and after, with a line-
   level diff — not just a count.** Replay the full catalog through
   `resolveProduct` with and without the change; confirm the exact
   collisions predicted are the ones that close, and that nothing new
   opens. A count going from 40 to 38 could mean two closed and zero opened,
   or four closed and two opened — check the lines.
8. **Pin it with a test using fixtures copied from the export verbatim** —
   not hand-typed approximations. `catalog_shared_id_test.go`,
   `catalog_fab_twins_test.go`, `catalog_showdown_test.go` are the pattern.
   A wrong hand-typed fixture would test nothing; a copied one is the shape
   that actually broke.
9. **If the fix depends on a datastore field this checkout might not have
   yet** (a fresh `datastore-gen` field not yet republished — see Aurora),
   guard the test with a skip, not a hard failure: `requireCredit` mirrors
   `requireSibling` (already established for Lorcana). CI's `cache-*` jobs
   pull a **pre-built** datastore from `b2://mtgban-datastore/<game>/
   <game>.json.xz` — republishing that is a separate, deliberate step, not
   something a PR does as a side effect.

## Upstream, not scraper-side: the two things left alone on purpose

- **Magic double-faced tokens.** mtgjson models each face of a token as its
  own row; the catalog sells one physical product covering both. This is by
  far the largest collision class by count: a full replay of 151,337 Magic
  singles through `resolveProduct` found 931 uuids receiving more than one
  product, 803 of them these token pairs. It is **deliberately not fixed
  here**: this is a datastore matter, not a scraper one, and has been
  treated as low priority across this whole project. No scraper-side
  dedupe or DFC-pricing logic; it needs the datastore to carry a
  `tokenProducts`-style mapping from a token uuid to the individual faces
  it's sold as.
- **Aurora's artist field, until republished.** `fabCreditedTwin` reads
  `Card.Artist`, which `datastore-gen@698813b` now populates for FaB and
  `mtgmatcher/fleshandblood@4f21ab43` now reads — but CI's pinned datastore
  is whatever's currently published to B2, and republishing that is
  someone's deliberate call, not this scraper's. Until it happens, the
  steer is a safe no-op (`Artist == ""` on both candidates never matches
  `want`) and its test (`TestResolveFleshAndBloodCreditedTwins`) skips
  rather than fails.

## Environment

- `SCG_API_KEY` — the HawkSearch catalog download. Local: `go-mtgban/.env`.
- `FLESHANDBLOOD_PATH` / `ALLPRINTINGS5_PATH` / `LORCANA_PATH` /
  `RIFTBOUND_PATH` — the datastores tests load lazily (`withMagic`,
  `withGameDatastore`); each CI job sets exactly one, so a test needing a
  specific game skips cleanly everywhere else.
- Rebuilding a fresh Flesh and Blood datastore locally (needed to verify
  anything that reads `Card.Artist`, or any FaB fix generally, against
  current upstream data rather than whatever's cached): pull
  `b2://mtgban-datastore/fleshandblood/tcgplayer-catalog.json.xz`
  (credentials: `B2_APPLICATION_KEY_ID_DATASTORE` /
  `B2_APPLICATION_KEY_DATASTORE`), decompress, then in a `datastore-gen`
  worktree: `go run ./cmd/fleshandblood -tcg-catalog tcgplayer-catalog.json
  -o fleshandblood.json`.
