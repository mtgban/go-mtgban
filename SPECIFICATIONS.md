# go-mtgban — Architecture & Development Specification

**Module**: `github.com/mtgban/go-mtgban` (Go 1.25)
**License**: dual AGPLv3 + commercial (see [COPYING.md](COPYING.md))

go-mtgban is a Magic: The Gathering market-data platform: it scrapes retail
inventories and buylists from ~27 card stores and marketplaces, normalizes
every listing to a canonical card identity, and computes arbitrage
opportunities between them. Disney Lorcana is supported as a secondary game.

The system has three layers. `cmd/bantool` runs the scrapers under hourly
GitHub Actions and uploads per-store JSON snapshots; a long-running consumer
(the reference one is `mtgban-website`, see §7) loads those snapshots plus the
matcher datastore and serves search / pricing / arbitrage.

```
cmd/* tools, GitHub Actions                          orchestration & ops
─────────────────────────────────────────────
scraper packages (tcgplayer/, cardkingdom/, ...)     one package per store
─────────────────────────────────────────────
mtgmatcher/                                          canonical card identity
─────────────────────────────────────────────
mtgban/                                              interfaces, records, analysis
```

---

## 1. `mtgban/` — core framework

### 1.1 Interface hierarchy (`mtgban/mtgban.go`)

Two disjoint half-hierarchies — sell-side and buy-side — over a shared
`Scraper` base, plus a `MultiScraper` mixin for aggregate platforms:

```
Scraper        Load(ctx) error; Info() ScraperInfo
├── Seller     + Inventory() InventoryRecord
│   └── Market + MarketNames() []string   (also embeds MultiScraper)
└── Vendor     + Buylist() BuylistRecord
    └── Trader + TraderNames() []string    (also embeds MultiScraper)

MultiScraper   InfoForScraper(name) ScraperInfo + Scraper
Carter         Activate(ctx, user, pass) error; Add(ctx, InventoryEntry) error
GenericEntry   Pricing() float64; Condition() string; Qty() int
ScraperConfig  SetConfig(ScraperOptions)   // DisableRetail / DisableBuylist
```

A `Market` is **purely sell-side** and a `Trader` **purely buy-side** — there
is no cross-half method. A platform that both sells and buys (TCGplayer,
Cardtrader, Cardmarket) implements *both* interfaces on one type; because both
embed `MultiScraper`, the shared `InfoForScraper` is unambiguous. Its
`Inventory()` returns one merged record where each entry carries `SellerName`,
and `InventoryForSeller(market, name)` (`mtgban/base.go`) projects out a single
seller by filtering on that field; `BuylistForVendor` is the buy-side twin.

`GenericEntry` abstracts over the differently-named price fields — both
`InventoryEntry` and `BuylistEntry` implement it, with `Pricing()` returning
`Price` and `BuyPrice` respectively (do **not** assume a single field name).

`Carter` is the optional cart-automation hook for sellers that can push to an
online shopping cart; it does *not* embed `Scraper` and is discovered by
type-assertion. (Note its `Add` is unrelated to `InventoryRecord.Add`.)
`ScraperConfig` is likewise an optional mixin applied post-construction by
type-assertion — its in-source doc comment misnames it "ConfigOptions"; the
real interface name is `ScraperConfig`.

### 1.2 Records and entries

`InventoryRecord = map[string][]InventoryEntry` and
`BuylistRecord = map[string][]BuylistEntry` — both keyed by the mtgmatcher
UUID. The key is a string by convention; the `Add*` path does **not**
type-check it against the backend (sealed scrapers, for instance, insert the
product UUID directly without calling `Match()` — see §3).

- `InventoryEntry`: `Quantity`, `Conditions`, `Price` (USD), `URL`,
  `SellerName`, `Bundle` (part of a direct-shipping hub), `OriginalId`
  (store product id), `InstanceId` (SKU), `CustomFields map[string]string`,
  `ExtraValues map[string]float64`.
- `BuylistEntry`: swaps `Price` for `BuyPrice` + `PriceRatio` (buy/sell
  ratio, a desirability signal) and `SellerName` for `VendorName`.

**Insertion semantics** (`add()` in `mtgban/base.go`) — the de-dup engine
every scraper relies on. Defaults applied first: empty condition → `"NM"`,
zero quantity → `1`; conditions outside `FullGradeTags = [NM SP MP HP PO]`
are rejected with `ErrInvalidCondition`. Then a strictness cascade against
existing entries for the same card:

| Method       | strict | Behavior on duplicate                                          |
|--------------|--------|----------------------------------------------------------------|
| `AddRelaxed` | 0      | same condition+price+seller → merge quantities                 |
| `Add`        | 1      | merge, but error if URL, quantity and Bundle are also identical |
| `AddStrict`  | 2      | error if condition+price+seller already present                |
| `AddUnique`  | 3      | error if condition+seller already present — **ignores price** (one per condition) |

`AddUnique`'s gate is the strictest and deliberately *ignores price* — its
check is separate from `AddStrict`'s, not a nested superset. Buylist has only
the relaxed/strict pair (merge vs error on identical qty+condition+price+
vendor). After every insert the slice is re-sorted: condition index in
`FullGradeTags` first, then price **ascending** for inventory /
**descending** for buylist, then quantity descending.

**This sort produces a load-bearing invariant: `entries[0]` is the NM entry**
(more precisely, the lowest-index *present* grade). `Arbit` reads
`blEntries[0]` as the NM offer (`arbit.go`), and `CombineBuylists` reads
`entries[0] // aka NM` (`combine.go`). Any change that reorders entries — or
the sort direction in `add()` — silently corrupts both. Pin this in
`base_test.go` before touching the add path.

`ScraperInfo` carries identity (`Name`, `Shorthand`, `CountryFlag`, and
`Game` — **`""` means Magic**, `"Lorcana"` means Lorcana; an empty `Game` is
Magic, not "unknown") and behavior flags consumed by the analysis layer:
`MetadataOnly` (index prices only, no conditions/quantities),
`NoQuantityInventory`, `SealedMode`, `CreditMultiplier` (store-credit ratio),
`Family` (price-coalescing group), plus `InventoryTimestamp`/
`BuylistTimestamp` (`*time.Time`; **nil = never loaded** — used as the
load-completion sentinel).

`BaseSeller`/`BaseVendor` with `NewSellerFromInventory`/`NewVendorFromBuylist`
wrap pre-built records (used when deserializing and when unfolding markets, or
when a consumer already holds an `InventoryRecord`/`BuylistRecord`).
`UnfoldScrapers` decomposes a mixed `[]Scraper` into flat
`[]Seller, []Vendor` — it must run **after** `Load()` and skips any scraper
whose timestamp is nil. `CountScrapers` is the pre-Load-safe counterpart.

### 1.3 Arbitrage engine (`mtgban/arbit.go`)

`ArbitOpts` (~25 knobs) is resolved into an internal `resolvedOpts` with
`Rate` defaulting to 1.0 and `ProfitabilityConstant` defaulting to **0** (it
is only applied when the caller sets it `> 0`). Card-level filtering
(`filterCard`) runs once per UUID in this order: rarity denylist → foil/etched
(`NoFoil`/`OnlyFoil` — etched counts as foil) → sealed-without-decklist skip
(`SealedDecklist`) → reserved-list-only → edition deny/allow lists (matching
either edition name or set code) → per-edition collector-number range →
`CustomCardFilter(co)`, which can both skip the card and return a price
multiplier.

`Arbit(opts, vendor, seller)` then, for each buylist card present in the
inventory:

1. Take `blEntries[0]` (NM by sort invariant); skip if
   `PriceRatio > MaxPriceRatio` or `BuyPrice < MinBuyPrice`.
2. For each inventory entry: condition denylist, seller allowlist (also
   matched against `CustomFields["SubSellerName"]`, a Cardtrader detail),
   `OnlyBundles`, `MinQuantity` (skipped when the seller reports
   `NoQuantityInventory`), `MinPrice`, then `CustomPriceFilter` (its factor
   composes multiplicatively with the card factor).
3. Effective sell price = `Price × customFactor × Rate`. If the entry is not
   NM, linear-scan `blEntries` for the same condition; no matching condition
   → skip (no cross-grade arbitrage is fabricated).
4. Buy price ×= `vendor.Info().CreditMultiplier` when `UseTrades`. Compute
   `difference = buy − sell`, `spread = 100·difference/sell`; enforce
   `MinDiff`, `MinSpread`, `MaxSpread`.
5. Tradable `qty = min(invQty, blQty)` (buylist quantity 0 = unlimited).
   **Profitability** = `(difference / (sell + k)) · log10(1 + spread) · √qty`,
   where `√qty` is applied only when qty > 1 and `k` is
   `ProfitabilityConstant`, a denominator-stabilizing constant that keeps
   cheap cards from dominating (spec reference value 10, configurable via
   `ArbitOpts`, library default 0; the website passes 2, or 10 in global
   mode). Changing the log base only rescales PI by a constant — it is
   order-preserving for ranking but shifts the absolute `MinProfitability`
   gate. Enforce `MinProfitability`.

Results are `[]ArbitEntry{CardId, BuylistEntry, InventoryEntry, Difference,
Spread, AbsoluteDifference (= diff·qty), Quantity, Profitability}`.

`Mismatch(opts, reference, probe)` is the seller-vs-seller analogue:
identical filter scaffolding, but the comparison price is
`refPrice × defaultGradeMap[probeCondition]` with the ladder
`NM=1, SP=0.8, MP=0.6, HP=0.4`. **Any condition not in that map (notably `PO`,
which is a valid `FullGradeTags` grade) yields multiplier 0 and zeroes the
reference price** — a real trap to pin in tests. The result carries
`ReferenceEntry` instead of `BuylistEntry`. `NoQuantityInventory` bypasses the
qty gate here too.

`Pennystock(seller, full, thresholds...)` flags cheap mythics (≤ $0.12 by
default) and, in `full` mode, rares / full-art-or-foil basics / foils /
promos under per-category thresholds, excluding gold/silver/white borders,
funny sets, thick-display promos, and HP/PO copies.

`CombineInventories(sellers)` / `CombineBuylists(vendors, useCredit)`
(`mtgban/combine.go`) build a `CombineRoot{Names, Entries[cardId][scraperName]}`
price matrix — the input to comparison CSVs. The two functions select NM by
*different mechanisms*: inventory hard-filters `Conditions != "NM"`, buylist
takes `entries[0]` (relying on the §1.2 sort invariant) and prices it as
`BuyPrice × CreditMultiplier` when `useCredit`, also carrying `Ratio`.

### 1.4 Concurrency, serialization, utilities

`WorkerPool[T,R]` (`mtgban/pool.go`) is the standard fetch primitive:
N worker goroutines consume an item channel and push results to a result
channel; a dispatcher goroutine feeds items until done or `ctx` is
cancelled — cancellation stops *dispatch* but lets in-flight workers finish,
so partial results are preserved. `consume(R)` runs on the caller's
goroutine, so consumers need no locking. Worker errors go to a `logErr`
callback. Logging across the codebase is via injected
`LogCallbackFunc = func(format string, a ...interface{})` fields, never
globals.

`mtgban/json.go` round-trips `{info, inventory, buylist}`
(`WriteSellerToJSON`/`ReadVendorFromJSON`, etc., reconstructing
`BaseSeller`/`BaseVendor`). `mtgban/csv.go` defines layered headers —
`CardHeader` (Key/Name/Edition/Finish/Number/Rarity) extended into
`InventoryHeader`, `MarketHeader` (+Seller/Bundle), `CartHeader` (+ids),
`BuylistHeader` (+Trade Price), `ArbitHeader`, `MismatchHeader` — with
writers auto-selecting the header from the data shape, and loaders accepting
a non-strict mode that logs and skips bad rows. **`WriteBuylistToCSV(buylist,
creditMuliplier float64, w)` is the one writer with a middle multiplier
argument** (all others are `(data, w)`); the param is spelled `creditMuliplier`
(missing "t") in source, and it computes the Trade Price column as
`BuyPrice × creditMuliplier`.

`mtgban/utils.go` supplies `GetExchangeRate(ctx, currency)` (fawazahmed0
currency CDN, `@latest`/unpinned) — which returns the **reciprocal**, i.e. a
*multiply-to-USD* factor, not the raw quoted rate — and `DateEqual`.

---

## 2. `mtgmatcher/` — the matching engine

The hard problem: store listings name cards inconsistently
("Lightning Bolt (Borderless) - MH2 *F*"), while prices must key on an exact
printing. mtgmatcher resolves free text to a unique UUID across ~900 sets
and ~40 promo classes.

### 2.1 Datastore loading (`mtgmatcher/backend.go`)

Loading is mediated by a small interface:

```go
type DataStore interface {
    Load() cardBackend   // builds the in-memory backend
}
```

`cardBackend` is **unexported by design** — the interface returns it by
value, so only package `mtgmatcher` can implement `DataStore`. There are two
implementers and two parser entry points returning the interface:

- `AllPrintings` (MTGJSON) — the heavyweight loader (`AllPrintings.Load()`,
  ~500 lines of repair + indexing); reached via
  `LoadAllPrintings(io.Reader) (DataStore, error)` (errors on empty `Data`).
- `LorcanaJSON` (Lorcana) — a simpler converter with no patch tables; reached
  via `LoadLorcana(io.Reader) (DataStore, error)` (errors on empty cards/sets).

The public entry point is `LoadDatastore(reader)`:

```go
func LoadDatastore(reader io.Reader) error {
    var buf bytes.Buffer
    tee := io.TeeReader(reader, &buf)
    datastore, err := LoadAllPrintings(tee)   // try MTGJSON first
    if err != nil {
        datastore, err = LoadLorcana(&buf)     // replay buffered bytes
        if err != nil {
            return err
        }
    }
    defaultBackend = datastore.Load()          // assigned DIRECTLY
    return nil
}
```

It picks the format by trial parse: an `io.TeeReader` mirrors the bytes
consumed by the AllPrintings attempt into `buf`, and on failure replays them
into the Lorcana parser. The chosen `DataStore.Load()` result is assigned
**directly** to the package-global `defaultBackend` — `LoadDatastore` does
**not** call `SetGlobalDatastore`. `LoadDatastoreFile(path)` opens the file
and delegates. A side effect of the trial parse: a malformed MTGJSON file
falls through to Lorcana and may surface as the less obvious "empty
LorcanaJSON file" error.

`SetGlobalDatastore(cardBackend)` is a **separate** entry point for swapping
in a pre-built backend (e.g. a cached one) without re-running `Load()`. The
package logger is `log.New(io.Discard, ...)` — silent unless an embedder
calls `SetGlobalLogger`.

**The global-backend concurrency contract (and its real-world violation).**
`defaultBackend` is a package-global *struct value* (`var defaultBackend
cardBackend`) with **no mutex/RWMutex/atomic guarding it**. All `api.go`
accessors (`GetUUID`, `GetSet`, `Search*`, ...) read its maps/slices directly
with no locking. The intended contract is "build once, read-only after" —
concurrency safety by immutability, not by locks. **But that contract is
violated in practice**: the reference consumer exposes an authenticated
`/api/load/datastore` endpoint that re-runs `LoadDatastore` on a live server,
reassigning the global while HTTP handlers concurrently read it. Reassigning a
multi-word struct value concurrent with readers is a data race under the Go
memory model. Tellingly, the same consumer *does* hot-swap its scraper sets
safely via `atomic.Pointer[[]Seller]` — but the matcher API offers no
equivalent, because `cardBackend` is an unexported value type. Treat
"immutable after load" as the *documented intent*; reloads must quiesce reads
(or the backend should be made atomically swappable — see `todo/refactor.md`
§4.4).

**What `Load()` does** (data *repair*, not just indexing):

- `skipSet()` drops online-only / art-series / token / empty sets; tokens
  whose names do not collide with real cards are promoted into the card list
  (colliding ones get a `" Token"` suffix).
- Per-set patch tables (a `switch set.Code` of FBB/4BB language overrides,
  STA/PLST frame strips, SLD per-number finish/tag/frame fixes, CMB1/CMB2
  playtest renames, PALP/PELP flavor tags, PMIC/PPC1 promo flags, DFT/SLC/
  SLX/TBTH/TMC tweaks) fix upstream MTGJSON gaps; the companion data tables
  live in `mtgmatcher/table.go`.
- `tcgplayerAlternativeFoilProductId` cards are split into a second foil card
  with a `_f` UUID and `★` number suffix; some sets are duplicated
  (LEGITA/DRKITA/4EDALT and SLD/PURL JPN language dupes).
- Same-name double-faced cards collapse to one name, flagged via
  `Identifiers["isDFCSameName"]`. Scryfall image URLs and `ReleaseDateTime`
  are precomputed. `SourceProducts[finish]` is filtered through
  `isBaseSealed`/`contentsContainCard` (direct containment per finish — it
  deliberately does *not* recurse into nested sealed sub-products).

Resulting indexes on `cardBackend`: `Sets` (code → `*Set`), `CanonicalNames`
(normalized → canonical), `UUIDs` (UUID → `CardObject`), `Hashes` (normalized
name → UUID list), `ExternalIdentifiers` (Scryfall/TCG/etched id → UUID),
`AlternateProps` (flavor names), sorted name/sealed-name arrays for
prefix/contains/regexp search, `AllPromoTypes`, `SLDDeckNames`,
`CommanderKeywordMap`, and partitioned `AllUUIDs`/`AllSealedUUIDs`.

**UUID scheme**: the MTGJSON UUID identifies a printing; finishes get the
load-generated suffixes `_f` (foil) and `_e` (etched); split foil printings
also get `★`/`†` number suffixes. The base UUID denotes the "most basic"
finish (nonfoil if present, else foil). These suffixed UUIDs are first-class —
resolve them only via `GetUUID`/`ExternalUUID`.

**Data model** (`mtgmatcher/mtgjson.go`): `CardObject` = MTGJSON `Card` +
resolved `Edition`/`Foil`/`Etched`/`Sealed`. `Card` carries the full MTGJSON
field set including a recently-added **`Legalities map[string]string`** (JSON
tag `legalities`, format → legality). It is populated only by the MTGJSON
loader and is **nil for Lorcana** cards — consumers must handle that. The
Lorcana loader (`mtgmatcher/lorcana.go`) has no patch tables; it converts and
computes set-level `Rarities`/`Colors`/`IsFoilOnly`/`BaseSetSize`.

### 2.2 Normalization (`mtgmatcher/replacer.go`, `mtgmatcher/utils.go`)

`Normalize()` lowercases and runs a single `strings.Replacer` that deletes
spaces, punctuation, quotes, dashes, accents, `" the "`, the plural/trailing
`s`, and separators — with explicit *protection* entries for names that
would otherwise alias (`"waste land"` stays distinct from "Wasteland",
`"lossom"` protects Blossom vs Lotus Bloom, `"vs"` is preserved as the Duel
Decks discriminator). `Equals`/`Contains`/`HasPrefix`/`HasSuffix` are
normalized comparisons used everywhere. Editing this table can silently
re-alias unrelated cards; **run the full matcher test suite after any change.**

`ExtractNumber()` pulls the first collector number `< 1993` from a string
(1993 separates numbers from years), refusing strings containing month
names (dates), ordinals (`30th`), and set-code lookalikes; it preserves
single-letter suffixes lowercased (`123s` prerelease, `123p` promo pack) and
understands PLST's `SET-123` format. `ExtractYear()` handles `'06`/`M15`
style abbreviations. Behavior-gating date constants live here too:
`NewPrereleaseDate` (2014-09), `BuyABoxInExpansionSetsDate` (2018-04),
`PromosForEverybodyYay` (2019-10), `BuyABoxNotUniqueDate` (2020-09),
`SeparateFinishCollectorNumberDate` (2022-02).

### 2.3 Input and ID matching

`InputCard` (`mtgmatcher/card.go`) =
`{Id, Name, Variation, Edition, Foil, Language}` + internal flags
(`beyondBaseSet`, `promoWildcard`, `originalName`), with ~30 normalized
predicates (`isPrerelease()`, `isPromoPack()`, `isBundle()`,
`isSecretLair()`, `isWorldChamp()`, `isSerialized()`, ...) that drive
filtering.

`MatchId(inputId string, finishes ...bool)`: `finishes[0]` = foil,
`finishes[1]` = etched. Strip any `_*` suffix, look up `UUIDs` directly then
`ExternalIdentifiers` (accepts MTGJSON/Scryfall UUIDs or numeric TCG product
ids — including the etched product id); if the stored finish already matches
the request, return it, else re-derive via `output()`. If the requested
finish lives on a *different printing* (post-2022 sets give etched cards
separate collector numbers), it scans the card's `Variations`, comparing
numeric collector-number values, and verifies the alternate genuinely differs
in finish before swapping.

`output(card, foil, etched)` is the finish reconciler: it clamps the
requested flags against the printing's actual `Finishes` (a foil request for
a nonfoil-only printing degrades gracefully; a foil-only printing upgrades
automatically), then appends `_e`/`_f` only when the printing has multiple
finishes. This tolerance for wrong foil flags from scrapers is a deliberate
design point — **trust the matcher's finish, not the scraper's input.**

### 2.4 The `Match()` pipeline (`mtgmatcher/mtgmatcher.go`)

1. **Language resolution** — map codes via `LanguageCode2LanguageTag`, then
   scan name/variation for embedded language tags.
2. **Id fast path** — if `Id` is set, `MatchId()`; the hit is *validated*:
   wrong language resets the input to the resolved card's fields and falls
   through to full matching; prerelease/promo-pack/serialized claims not yet
   reflected upstream return `ErrUnsupported` (these MTGJSON tags lag
   releases).
3. **Name surgery** — Binderpos `Name [Edition]` syntax (resolving the
   bracket as a set name, falling back to variation, with the TCG
   `PP`-prefix promo-pack quirk); parenthesized variants via
   `SplitVariants()` (which protects legitimately-parenthesized names like
   *Erase (Not the Urza's Legacy One)* and *B.F.M.*); ` - ` suffix variants.
   Plus a prefilter renaming playtest/token name collisions (Red Herring,
   Unquenchable Fury, Shapeshifter).
4. **Canonicalization** — `CanonicalNames[Normalize(name)]`; on miss,
   `adjustName()` (typo/token/number fixups, flavor-name resolution via
   `AlternateProps`) and retry. Final miss → `ErrUnsupported` for
   tokens/oversize, else `ErrCardDoesNotExist`.
5. **Edition adjustment** — `adjustEdition()`: a ~630-line ladder
   (`mtgmatcher.go:729–1365`) applying `EditionTable` aliases ("Alpha" →
   "Limited Edition Alpha", Universes Beyond names, ...), variation-implies-
   edition rules ("Invocation" → Amonkhet Invocations), Commander-product
   parsing, and a terminal `default: switch inCard.Name` (`:1099–1351`) of
   ~38 hardcoded per-card fixups. Followed by hard `ErrUnsupported` gates
   (custom token sets, most oversize). The per-card switch is the chief
   "tables before code" violation flagged for extraction in
   `todo/refactor.md` §4.1.
6. **Set selection** — `Printings4Card()` then, if multiple,
   `filterPrintings()` (see 2.5); then a three-pass loop builds
   `cardSet map[setCode][]Card` via `MatchInSet()`:
   (a) perfect normalized edition-name match — and for
   prerelease/promo-pack/bundle/BaB inputs it *also* enrolls the `P<code>`
   promo sibling set (or vice versa);
   (b) heuristic pass: edition substring containment, generic promos
   restricted to `*Promos` sets, bundle/BaB allowed into recent-enough base
   sets per the date constants;
   (c) YOLO pass: all printings, trusting downstream filtering.
   Passes (a)/(b) are skipped in `promoWildcard`/Secret Lair mode, which
   wants maximal candidates.
7. **Card-level disambiguation** — if more than one candidate survives,
   `filterCards()` (see 2.5). World Championship inputs keep only the first
   match (decks are per-player duplicates). A language filter then drops
   non-English prints unless a language was requested.
8. **Verdict** — 0 cards: `ErrCardWrongVariant` (or `ErrCardMissingVariant`
   if no variation was given, `ErrUnsupported` if a language was involved);
   1 card: `output()` + a final prerelease/promo-pack/serialized tag
   validation; 2+: `AliasingError`, whose `Probe()` returns all candidate
   UUIDs — consumers log these as data-quality alarms (and, like
   `mtgban-website`, may pick the newest printing from `Probe()`).

Error taxonomy (`mtgmatcher/utils.go`): `ErrDatastoreEmpty`,
`ErrCardUnknownId`, `ErrCardDoesNotExist`, `ErrCardNotInEdition`,
`ErrCardWrongVariant`, `ErrCardMissingVariant`, `ErrUnsupported`,
`AliasingError`. `ErrUnsupported` doubles as a silent-skip channel *and* a
found-but-invalid-promo-tag signal.

### 2.5 The heuristic layer (`filter.go`, `variants.go`, `callbacks.go`)

`filterPrintings()` (~770 lines, in a 1116-line `filter.go`) eliminates whole
sets using the input's promo predicates against set type, release dates, and
name patterns — dedicated, repetitive blocks for prerelease vs promo-pack,
release/launch promos, BaB, bundles, Secret Lair vs Mystery List, WCD,
MagicFest, Duel Decks, 30th Anniversary, judge promos, and a wildcard-promo
mode (the repeated FNM/Judge/Arena shape is the §4.2 split candidate).

`filterCards()` (~340 lines) disambiguates within sets, consulting in order:
the hand-curated `VariantsTable` (`variants.go`, ~4,651 lines of pure data:
set → card → variant tag → collector number) → `ExtractNumber` with suffix
semantics → promo-type validation through the `promoTypeElements` table (each
entry: tag strings, an optional `TagFunc`, an activation date, wildcard
eligibility) → per-set `simpleFilterCallbacks`/`complexFilterCallbacks`/
`numberFilterCallbacks` (`callbacks.go`, ~1,194 lines) for sets whose
disambiguation needs code → finish/frame separation (etched, borderless,
extended art, showcase) and prerelease/promo-pack dedup via `multiPromosTable`.

This three-tier design — **data tables first, generic number/promo logic
second, per-set code last** — is the package's core maintenance pattern: most
new-set support lands as table entries, not code.

### 2.6 Sealed products & search API (`mtgmatcher/api.go`)

Beyond lookups (`GetUUID`, `GetSet`, `GetSetByName`, `GetAllSets`,
`GetUUIDs`/`GetSealedUUIDs`, `Printings4Card`, `CardReleaseDate`,
`ExternalUUID`, `AllPromoTypes`) and the normalized search family
(`SearchEquals`/`SearchHasPrefix`/`SearchContains`/`SearchRegexp` over the
name arrays, `SearchSealedEquals`/`SearchSealedContains`), the package models
sealed products end-to-end:

- `BoosterGen(set, boosterType)` performs MTGJSON-rule weighted sheet draws
  (`weightedrand`), honoring `BalanceColors` (an approximation citing
  magic-search-engine) and per-sheet `AllowDuplicates`; its single hard-fail
  is `maxRerollThreshold = 50` ("reroll threshold reached"). The `slc` Secret
  Lair random-foil ~30% behavior is a hardcoded special case.
- `GetPicksForSealed` recursively expands product contents
  (card/pack/deck/sealed/variable). `GetDecklist`/`SealedHasDecklist`
  distinguish fixed-content products; `SealedIsRandom` flags random ones.
- `GetProbabilitiesForSealed`/`SealedBoosterProbabilities` compute exact
  per-card pull probabilities — the inputs to `sealedev`'s EV computation.
- `BuildSealedProductMap` and the load-time reverse index
  (`fillinSealedContents`) link single cards back to the products containing
  them.

### 2.7 Testing — strategy & coverage map

**What exists.** Data-driven against a real AllPrintings file
(`ALLPRINTINGS5_PATH` env var): `matcher_test.go` replays
`matcher_test_data.json` (input card → expected UUID or error), with a `-u`
flag that regenerates expectations after intentional changes — the regression
harness for the heuristic tables. Run `go test -run TestMatch -u
./mtgmatcher/` after a *deliberate* matching change and **review the diff** —
never blindly accept it. Unit tests cover normalization, number/year
extraction, variants-table integrity, the search surface, and (in
`mtgban/base_test.go`) the `Add*` family. Scraper packages are otherwise
validated operationally.

**The pyramid is inverted here.** Card identity is only meaningful against a
~600 MB MTGJSON dataset, so the *largest* test surface (the matcher) is a
data-backed integration replay, not a unit test — while the most
business-critical code, the money path, needs **no** external data yet has
zero direct tests. The strategy follows from that asymmetry:

| Layer | Targets | Test type | Needs dataset? | Today |
|-------|---------|-----------|----------------|-------|
| **Money path** (top risk) | `Arbit`, `Mismatch`, `Pennystock`, `Combine*`, `add()` invariants, profitability formula | unit / golden on synthetic records | **No** — runs in CI | none beyond `Add*` |
| **Matcher** (data integrity) | `Match`/`MatchId`, normalization, variants/editions, sealed API | data-backed regression replay | **Yes** — skips without it | replay + unit |
| **Scraper preprocess** (breadth) | per-store title → `InputCard` → `Match` | table tests on captured fixtures | partial | 2 of 23 |

Principles: (1) **the money path is unit-testable and unprotected — cover it
first**, with in-test records and no MTGJSON dependency; (2)
**characterization before refactor** — pin `Arbit`/`Mismatch`/`Combine*`/
`add()` outputs *before* the §2.2–§2.4 dedups and refactor under green;
(3) **assert invariants, not just functions** — the `entries[0] == NM`
ordering is a sort side effect that `Arbit`/`CombineBuylists` depend on, so pin
it directly; (4) **scrapers: breadth over depth** — a few fixture table tests
for the gnarliest preprocessors (cardmarket, cardtrader, tcgplayer) catch the
realistic break.

**CI caveat — data-dependent skips read as green.** CI runs only `go test
./... -v` with no `ALLPRINTINGS5_PATH`, so every matcher/preprocess test that
needs the dataset **silently skips**; the suite is green while a large slice
never runs. Hence (a) the data-free money-path tests are doubly valuable
because they actually execute in CI, and (b) the §0.5 CI gate should provision
a cached `AllPrintings5.json` (or at least report the skip count) so "green"
isn't mistaken for "covered" — tracked as `todo/refactor.md` §3.4.

---

## 3. Scraper packages

Idealized shape: `NewScraper(creds)` returning a struct with `LogCallback`
(exported, always first), `MaxConcurrency` (exported, default 8), optional
`Partner`/`Affiliate`, exported `DisableRetail`/`DisableBuylist`, and
unexported `inventory`/`buylist` + `inventoryDate`/`buylistDate`. `Load(ctx)`
fans out via `mtgban.WorkerPool` (2–8 workers) over `retryablehttp` clients
(LinearJitterBackoff); a `preprocess.go` translates store naming into
`InputCard` + `Match()`, skipping `ErrUnsupported`, logging `AliasingError`s;
results inserted via the `Add*` family. Every scraper has a tagged `printf`
helper (`x.LogCallback("[TAG] "+format, a...)`). File convention:
`<store>.go` / `api.go` / `preprocess.go` / optional `sealed.go` (a *separate*
scraper struct with its own `SealedMode` `Info`).

**The preprocess → Match() contract** (per listing): build an `InputCard` →
`Match()` (or `MatchId(scryfallID,...)` / `SimpleSearch(name,number,foil)`)
→ on `ErrUnsupported` silently `continue`; on `AliasingError` log + `Probe()`;
on other errors log with context (many scrapers suppress known-noisy editions
first) → insert with `Add*`. `PriceRatio` is computed by reading back
`inventory[cardId]` before inserting the buylist row.

### API-based

| Package | Service & auth | Notes |
|---|---|---|
| `tcgplayer` | OAuth via `go-tcgplayer` + cookie-authed marketplace APIs | Largest: Market/Index/Sealed/per-seller scrapers; SKU map keyed by UUID; TCG Direct modeled as a Vendor with net-after-fees pricing |
| `cardmarket` | OAuth 1.0 HMAC-SHA1 (gentle retry) | `CardMarketIndex` is a **Market** (`MarketNames → MKM Low/Trend`, `MetadataOnly`, `Family="MKM"`); EUR→USD; Lorcana via game id; `CardMarketSealed` separate |
| `cardtrader` | Bearer token | `CardtraderMarket` (**Market**, 3 seller tiers, `Family="CT"`, `CountryFlag="EU"`); `CardtraderSealed` mirror; bulk upload + cart APIs |
| `cardkingdom` | Public pricelist via `go-cardkingdom` (file/URL-fed, no own client) | Full 4-condition buylist with price ratios; `CreditMultiplier 1.3`; singles + `sealed.go` + `graded.go` are three scrapers |
| `manapool` | Public JSON API + SvelteKit shop pages | Three structs in one package: aggregate (`MatchId` by ScryfallID, `NoQuantityInventory`), `ManapoolSealed`, `ManapoolSeller(slug)`; per-seller skip preprocess |
| `cardsphere` | Session cookie (gentle 3s) | Buylist-only; `BuyPrice ×0.87` fee **and** `CreditMultiplier 1.1` |
| `mtgstocks` | Public API, **UA rotation** (`uarand`) | MetadataOnly index (average/market interests) |

### HTML / crawler

`starcitygames` (HawkSearch/Meilisearch APIs, Lorcana, serialized detection,
sealed), `coolstuffinc` (multi-game, `CreditMultiplier 1.25`), `hareruya`
(JPY, **bespoke 403 → 5-min backoff**), `magiccorner` (EUR, Italian),
`abugames` (Solr, MINT-aware grading, `InfoForScraper`), `mtgseattle`
(`CreditMultiplier 1.33`), `ninetyfive`, `mintcard` (rides TCG SKUs,
`CreditMultiplier 1.1`), `vegassingles`, `secretdeskorrigans` (CAD, French),
`toamagic` (Spanish), `miniaturemarket` (sealed-only).

**Legacy cohort — `gocolly` + hand-rolled goroutines, predate WorkerPool:**
`trollandtoad` (+ a `generic.go` Lorcana scraper + sealed), `wizardscupboard`,
`strikezone`. Slated for WorkerPool migration (`todo/refactor.md` §2.1).

`sealedev` builds sealed-EV "scrapers" from mtgmatcher probabilities or
5,000-run booster simulations priced against the MTGBAN API, emitting EV
entries with dispersion stats (std-dev/IQR), `Family="EV"`, `SealedMode`, and
`MetadataOnly` toggled per sub-scraper. `synthetic/` (untracked WIP) is a
*computed* buylist (no site) synthesizing prices from TCG/CK/SCG,
`MetadataOnly`.

**Not templates:** `amazon/` is a stub — `api.go` only, a complete Amazon
**Creators-API** OAuth2 client (`SearchItems`/`GetItems`) with **no scraper
struct yet**, blocked on Associates credentials (§7 of the Amazon plan).
`mvpsportsandgames/` (untracked) has a non-conforming `Inventory() (record,
error)` that does **not** satisfy `mtgban.Seller`. For new work, copy
`ninetyfive` (API) or `mtgseattle` (HTML) — never the legacy trio, the stub,
or the orphan.

---

## 4. Tooling — `cmd/` and CI

Committed tools: **bantool** (the production orchestrator), **boosterGen**,
**boosterList**, **manapoolOrders**, **manapoolSeller**, **mkmPriceGuide**,
**mkmhtml2csv**, **mp2ckbl**, **tcgid4scryfall** (TCG id → Scryfall id export),
**amazonsearch**. Several more tools (omnitool-3g, autocart, the `ck*`/`ct*`/
`mkm*` family) are **untracked WIP** — and some embed live credentials (see
`todo/refactor.md` §0.3).

- **bantool** — a registry of `scraperOption{constructor, flags}` for every
  store; `-scrapers`/`-sellers`/`-vendors` selection, `-format` json/csv/
  ndjson, output through `mtgban/simplecloud` to local/B2/GCS/S3/HTTP,
  optional HMAC signing (`BAN_SECRET`); all credentials via env vars
  (godotenv autoload). Init closures set `scraper.LogCallback =
  GlobalLogCallback` as a **direct field assignment on the concrete pointer**
  in ~40 places — the binding constraint on any `BaseScraper` refactor (the
  field must stay exported and embedding-reachable).
- **mp2ckbl** — Mana Pool → Card Kingdom buylist arbitrage with cart
  automation (Supabase publishable key, Firefox-cookie auth via `kooky`).
- **manapoolOrders / manapoolSeller** — Mana Pool buyer-order CSV dumps and
  single-seller inventory extraction.
- **mkmhtml2csv** — offline parser of saved Cardmarket HTML into card CSVs via
  an mcmId → UUID map (skips altered/signed/inked + non-English).
- **boosterGen / boosterList** — booster simulation and sealed introspection
  over the mtgmatcher sealed API.
- **amazonsearch** — test harness for the new `amazon/` Creators-API client.

**CI** (`.github/workflows/`): `ci.yml` caches AllPrintings5 (`cache-file.yml`
reusable workflow) and runs **`go test ./... -v` only — there is no
`gofmt`/`go vet` gate**, which is why unformatted files recur (the durable fix
is to add the gate; see `todo/refactor.md`). One `bantool-<store>.yml` per
store (cron + manual/`repository_dispatch`) chains `cache-datastore` →
`run-bantool.yml`. No Makefile/Docker — plain `go build` per `cmd/`
subdirectory; `go test ./...` (mtgmatcher requires `ALLPRINTINGS5_PATH`).

**Key dependencies**: goquery/colly (HTML), retryablehttp + cleanhttp
(HTTP), kooky (browser cookies), simplecloud (storage abstraction),
go-ndjson, weightedrand (boosters), montanaflynn/stats (EV),
golang.org/x/text (normalization), uarand (UA rotation).

---

## 5. Design through-lines

> The load-bearing decisions below are recorded as ADRs in
> [`docs/adr/`](docs/adr/) with full context and alternatives: UUID-as-key
> (ADR-0001), the immutable global backend (ADR-0002), the Profitability Index
> formula (ADR-0003), `WorkerPool` (ADR-0004), and the abugames Solr request
> shape (ADR-0005).

1. **Everything keys on the mtgmatcher UUID** — scrapers are thin
   translators; correctness lives in one place. (By convention, not
   type-enforced; sealed inserts the product UUID directly.)
2. **Sorted-record invariants instead of queries** — `entries[0] == NM` is
   produced by `add()`'s sort and consumed by `Arbit` and `CombineBuylists`.
   The CSV writers rely on the ordering too.
3. **Tables before code** — new-set support is data (VariantsTable, edition
   aliases, promo elements); per-set callbacks are the escape hatch.
4. **Graceful degradation on dirty input** — `output()` finish clamping, the
   Id-path validation/reset, non-strict CSV loading, `ErrUnsupported` as a
   silent-skip channel distinct from real errors.
5. **Injected logging + bounded worker pools** — uniform operational
   behavior; the WorkerPool migration of the legacy colly trio is the
   remaining standardization gap.
6. **Build-once, read-only state** — the matcher backend is an immutable
   global *by convention*. Note the contract is currently *unenforced* and
   *violated* by the consumer's runtime reload endpoint (§2.1); hardening it
   is open work.

## 6. Extending the system

Adding a store = create a package with the four-file layout, implement
`Seller` and/or `Vendor` (and `Market`/`Trader` if it has sub-sellers), fetch
with `WorkerPool` + retryablehttp, write a `preprocess.go` that builds
`InputCard`s and handles the store's naming quirks, register a `scraperOption`
in bantool, and add a GitHub Actions workflow. Set the right `ScraperInfo`
flags (`MetadataOnly`, `NoQuantityInventory`, `SealedMode`, `CreditMultiplier`,
`Family`, `Game`). The hard part is always preprocessing — which is why
mtgmatcher's typed errors, variant tables, and per-set callbacks exist.

---

## 7. Primary consumer — `mtgban-website` (usage reference)

The reference embedder demonstrates the intended production topology:
**scraping and serving are decoupled.** bantool scrapes and uploads per-store
`Seller`/`Vendor` JSON; the website loads those snapshots and the matcher
datastore, and never runs scrapers in-process. Canonical patterns:

- **Load the datastore once at startup** — `mtgmatcher.LoadDatastore(reader)`
  streamed from a `simplecloud` bucket, then fire async cache builds. A
  signature-verified `/api/load/datastore` endpoint can reload it at runtime
  (see the §2.1 race caveat).
- **Consume pre-scraped JSON** — `mtgban.ReadSellerFromJSON` /
  `ReadVendorFromJSON` per `game/name/kind/shorthand`. The live sets sit
  behind `atomic.Pointer[[]mtgban.Seller]` / `[[]mtgban.Vendor]` for lock-free
  reads with single-writer publish — the correct concurrency pattern for a
  long-running server over swappable snapshots (and the pattern the matcher
  backend lacks).
- **Build a store from records** — `mtgban.NewSellerFromInventory` /
  `NewVendorFromBuylist` when you already hold an `InventoryRecord` /
  `BuylistRecord`.
- **Match user input → UUID, handle aliasing** — `mtgmatcher.Match(&InputCard)`
  with `errors.As(err, &AliasingError)` → `Probe()` to pick the newest
  printing (the upload flow); `mtgmatcher.MatchId(scryfallID, foil, etched)`
  for the external-id fast path.
- **Search dispatcher** — switch over `SearchEquals`/`SearchContains`/
  `SearchHasPrefix`/`SearchRegexp`/`SearchSealedEquals`, falling back to
  `Match`.
- **Arbitrage pipeline** — construct one `mtgban.ArbitOpts`, tune it with
  `CustomCardFilter(co *mtgmatcher.CardObject)` and
  `CustomPriceFilter(cardId, mtgban.InventoryEntry)` closures, then dispatch
  `mtgban.Arbit`/`mtgban.Mismatch` by direction over `GetSellers()`/
  `GetVendors()`; sort the `[]ArbitEntry`.
- **Buylist pricing reducer** — range `GetVendors()`, filter by `SealedMode`/
  shorthand, fold `vendor.Buylist()` entries into a price map (`getVendorPrices`).
- **Sealed introspection** — `GetSealedUUIDs`/`GetDecklist`/`GetPicksForSealed`/
  `SealedIsRandom`/`SealedHasDecklist` (the website surfaces booster/deck flags
  but delegates generation to the matcher). It does *not* call `Combine*` or
  `BoosterGen` directly — for those, the embedded `cmd/` tools are the example.
- **CSV export** — `mtgban.WriteBuylistToCSV(records, creditMultiplier, w)`
  straight to an HTTP writer.
