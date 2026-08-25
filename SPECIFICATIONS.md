# go-mtgban — Architecture & Development Specification

**Module**: `github.com/mtgban/go-mtgban` (Go 1.25)
**License**: dual AGPLv3 + commercial (see [COPYING.md](COPYING.md))

go-mtgban is a trading-card market-data platform: it scrapes retail
inventories and buylists from a couple of dozen card stores and marketplaces,
normalizes every listing to a canonical card identity, and computes arbitrage
opportunities between them. Magic: The Gathering is the primary game; Disney
Lorcana and Riftbound (the League of Legends TCG) are supported alongside it,
each with its own datastore, its own matching rules, and its own set of
scraper targets.

The system has three layers. `cmd/bantool` runs the scrapers under scheduled
GitHub Actions and uploads per-store JSON snapshots; a long-running consumer
(the reference one is `mtgban-website`, see §7) loads those snapshots plus the
matcher datastore and serves search / pricing / arbitrage.

```
cmd/* tools, GitHub Actions                          orchestration & ops
─────────────────────────────────────────────
scraper packages (tcgplayer/, cardkingdom/, ...)     one package per store
─────────────────────────────────────────────
mtgmatcher/                                          game-agnostic core
  + mtgmatcher/{magic,lorcana,riftbound}             per-game loaders & rules
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
every scraper relies on. Defaults are applied first, and the two sides differ:
both default an empty condition to `"NM"`, but only the inventory side
defaults a zero quantity to `1`. A buylist entry with quantity `0` keeps that
value, which `Arbit` reads as "unlimited". Conditions outside
`FullGradeTags = [NM SP MP HP PO]` are rejected with `ErrInvalidCondition`.
Then a strictness cascade against existing entries for the same card:

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
`blEntries[0]` as the NM offer — the comment in `mtgban/arbit.go` states it
outright — and the CSV writers emit rows in the same order. Any change that
reorders entries, or flips the sort direction in `add()`, silently corrupts
both. Pin this in `base_test.go` before touching the add path.

`ScraperInfo` carries identity (`Name`, `Shorthand`, `CountryFlag`, and
`Game`) plus behavior flags consumed by the analysis layer. `Game` draws from
the named constants `mtgban.GameMagic`, `GameLorcana` and `GameRiftbound`,
where **`GameMagic` is the empty string**: an empty `Game` means Magic, not
"unknown". The scrapers that serve more than one game set it from their own
per-game construction: cardmarket, cardtrader, coolstuffinc, ninetyfive,
starcitygames, strikezone, tcgplayer's `TCGGame`/`TCGGameIndex`, and
trollandtoad's `generic.go`. The behavior flags are `MetadataOnly` (index
prices only, no conditions or quantities), `NoQuantityInventory`,
`SealedMode`, `CreditMultiplier`
(store-credit ratio), `Family` (price-coalescing group), plus
`InventoryTimestamp`/`BuylistTimestamp` (`*time.Time`; **nil = never
loaded** — used as the load-completion sentinel).

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
4. Compute `difference = buy − sell` and `spread = 100·difference/sell`
   straight from the buylist entry's `BuyPrice`; enforce `MinDiff`,
   `MinSpread`, `MaxSpread`. `Arbit` does **not** apply
   `CreditMultiplier` — that field is metadata a consumer (or
   `WriteBuylistToCSV`) applies for itself when it wants store-credit
   pricing.
5. Tradable `qty = min(invQty, blQty)` (buylist quantity 0 = unlimited).
   **Profitability** = `(difference / (sell + k)) · log10(1 + spread) · √qty`,
   where `√qty` is applied only when qty > 1 and `k` is
   `ProfitabilityConstant`, a denominator-stabilizing constant that keeps
   cheap cards from dominating (spec reference value 10, configurable via
   `ArbitOpts`, library default 0; the website passes 2, or 10 in global
   mode). Changing the log base only rescales the index by a constant — it is
   order-preserving for ranking but shifts the absolute `MinProfitability`
   gate. Enforce `MinProfitability`.

Results are `[]ArbitEntry{CardId, BuylistEntry, InventoryEntry, Difference,
Spread, AbsoluteDifference (= diff·qty), Quantity, Profitability}`.

`Mismatch(opts, reference, probe)` is the seller-vs-seller analogue with
identical filter scaffolding, but it compares across grades instead of
requiring an exact condition match. `defaultGradeMap` is the ladder
`NM=1, SP=0.8, MP=0.6, HP=0.4, PO=0`, and the reference price is scaled by
`invGrade / refGrade` — dividing out the reference entry's own grade before
applying the probe's, so a non-NM reference is not compared against a
rescaled copy of itself. A pair whose grade is missing from the map or is
`≤ 0` (which `PO` is) is skipped with `continue` rather than silently priced
at zero. The result carries `ReferenceEntry` instead of `BuylistEntry`, and
`NoQuantityInventory` bypasses the qty gate here too.

`Pennystock(seller, full, thresholds...)` flags cheap mythics (≤ $0.12 by
default) and, in `full` mode, rares / full-art-or-foil basics / foils /
promos under per-category thresholds, excluding gold/silver/white borders,
funny sets, thick-display promos, and HP/PO copies. This is the one place
`mtgban` reaches into a game-specific vocabulary: it imports
`mtgmatcher/magic` for `PromoTypeThickDisplay`.

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
`CardHeader` (UUID/Name/Edition/Finish/Number/Rarity) extended into
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
printing. mtgmatcher resolves free text to a unique UUID across hundreds of
sets and dozens of promo classes.

The package is **game-agnostic core plus one package per game**. Core owns the
`Backend` data model, the `Match()` skeleton, normalization, the search API and
the sealed-product API; `mtgmatcher/magic`, `mtgmatcher/lorcana` and
`mtgmatcher/riftbound` each own their datastore loader and the game-specific
identification logic that core dispatches through the `GameRules` interface.
The dependency runs one way only — the game packages import core, core never
imports a game package — which is what keeps the vocabularies from bleeding
into each other.

### 2.1 Games, loaders, and the `Backend`

**Registration.** Loaders register themselves in the style of `database/sql`
drivers (`mtgmatcher/datastore.go`):

```go
type GameLoader func(io.Reader) (*Backend, error)

func RegisterGame(name string, load GameLoader)  // panics on nil or duplicate
func RegisteredGames() []string                  // registration order
```

Each game package has a `register.go` whose `init()` calls `RegisterGame` —
`"magic"`, `"lorcana"`, `"riftbound"` — so a consumer activates a game with a
blank import and pays for nothing it does not use:

```go
import _ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
```

`mtgmatcher/games` is a meta-package that blank-imports all three, for
consumers that want every game with a single import; `cmd/bantool` does
exactly that. The trade-off is spelled out in its doc comment: it links every
game and its transitive dependencies into the binary and puts every loader
into auto-detection.

**Loading.** `LoadDatastore(reader)` auto-detects the game. It fails fast with
a pointed error if no game is registered ("blank-import a game package such
as …"), reads the input once into memory, and then hands a fresh
`bytes.Reader` to each registered loader in registration order. Loaders are
written to reject formats they do not recognize, so the first success wins and
is installed with `SetGlobalDatastore(b)`. If every loader fails, the returned
error is `"mtgmatcher: no registered game could load the datastore: %w"`
wrapping the *first* loader's error — so a malformed MTGJSON file may surface
as whichever game happened to be tried first, not as a Magic parse error. Keep
that in mind when reading a load failure. `LoadDatastoreFile(path)` opens the
file and delegates.

When the game is known, skip auto-detection entirely:

```go
func Open(name string, reader io.Reader) (*Backend, error)
```

`Open` loads exactly the named game and returns the `Backend` **without**
installing it as the global one — the escape hatch for consumers that want to
own their backend's lifetime (see the concurrency note below).

**The global-backend concurrency contract.** `defaultBackend` is a
package-global *struct value* (`var defaultBackend Backend`) with **no
mutex/RWMutex/atomic guarding it**, and `SetGlobalDatastore(b *Backend)`
simply copies the pointed-to struct into it. All the package-level accessors
(`GetUUID`, `GetSet`, `Match`, `Search*`, …) read its maps and slices directly
with no locking. The intended contract is "build once, read-only after" —
concurrency safety by immutability, not by locks. That contract is violated in
practice: the reference consumer exposes an authenticated
`/api/load/datastore` endpoint that re-runs `LoadDatastore` on a live server,
reassigning the global while HTTP handlers concurrently read it, and
reassigning a multi-word struct value concurrent with readers is a data race
under the Go memory model.

The escape hatch now exists, though. `Backend` is an exported type carrying
almost the whole accessor surface as instance methods, and `Open()` hands one
back without touching the global, so an embedder can publish it through an
`atomic.Pointer[mtgmatcher.Backend]` exactly as the same consumer already does
for its scraper sets. Treat "immutable after load" as the documented intent of
the *package-level* API; a consumer that needs hot swaps should manage its own
`*Backend`. One caveat applies to Magic specifically: `mtgmatcher/magic/doc.go`
records that a handful of the filter callbacks still resolve auxiliary lookups
through the package-level helpers, which read the global. Magic rules
therefore assume the `Backend` they serve is *also* the installed global one;
a side backend opened without being made global may answer those auxiliary
lookups from the wrong data.

**Backend as a type.** Nearly every operation exists twice: as a method on
`*Backend` and as a thin package-level function delegating to
`defaultBackend`. So `mtgmatcher.Match(inCard)` is
`defaultBackend.Match(inCard)`, and likewise for `MatchId`, `GetUUID`,
`GetSet`, `Search*`, `BoosterGen`, the sealed API and the rest. New code that
owns its backend should call the methods; the package-level wrappers exist for
the (large) body of existing callers and for convenience. The conversion is not
complete: `GetUUIDsInSet`, `GetSealedUUIDsInSet`, `AllNames`, `AllPromoTypes`
and `HasPrinting` still exist only as package-level functions reading
`defaultBackend`, so a consumer holding its own `*Backend` cannot reach them —
the same global coupling `magic/doc.go` warns about, seen from the caller's
side.

**What the Magic loader does** — data *repair*, not just indexing. This is the
heavyweight path, and it now lives entirely in `mtgmatcher/magic/mtgjson.go`
(`magic.Load`), with its companion tables in `mtgmatcher/magic/table.go`
(the missing PALP/PELP tag lists, `sldJPNLangDupes`, `productsWithOnlyFoils`,
the Magic color-name map). Core's `mtgmatcher/table.go`
retains only the language maps (`LanguageCode2LanguageTag` and its inverse).

- `skipSet()` drops online-only / art-series / empty sets. Tokens are filed
  under the set mtgjson names in `tokenSetCode`, called `"<set> Tokens"`, and
  promoted into its card list; a token whose name a real card answers to gets
  a `" Token"` suffix.
- Per-set patch tables (a `switch set.Code` of FBB/4BB language overrides,
  STA/PLST frame strips, SLD per-number finish/tag/frame fixes, CMB1/CMB2
  playtest renames, PALP/PELP flavor tags, PMIC/PPC1 promo flags, DFT/SLC/
  SLX/TBTH/TMC tweaks) fix upstream MTGJSON gaps.
- `tcgplayerAlternativeFoilProductId` cards are split into a second foil card
  with a `_f` UUID and `★` number suffix; some sets are duplicated
  (LEGITA/DRKITA/4EDALT and SLD/PURL JPN language dupes).
- Same-name double-faced cards collapse to one name, flagged via
  `Identifiers["isDFCSameName"]`. Scryfall image URLs and `ReleaseDateTime`
  are precomputed. `SourceProducts[finish]` is filtered through
  `isBaseSealed`/`contentsContainCard` (direct containment per finish — it
  deliberately does *not* recurse into nested sealed sub-products).

The Lorcana loader (`mtgmatcher/lorcana/lorcana.go`) and the Riftbound loader
(`mtgmatcher/riftbound/riftbound.go`) are far simpler converters with no patch
tables; both derive the set-level `Rarities`/`Colors` once the cards are in,
Lorcana additionally deriving `IsFoilOnly`/`IsNonFoilOnly` and inferring
`BaseSetSize` from the first enchanted-rarity number, while Riftbound takes
`BaseSetSize` straight from the gallery's `collectorNumberMax`. Both set every
card's `Language` to `"English"`
on purpose: core `Match`'s language filter drops candidates whose `Language`
differs from English whenever more than one survives, so leaving the field
empty would turn every legitimate aliasing result into a bogus wrong-variant
error.

**Riftbound in particular.** The datastore is the official card-gallery
payload served by the Riftbound site, enriched by
`github.com/mtgban/riftbound-datastore`, which stamps every printing with its
TCGplayer product id and appends the promotional printings the gallery does
not carry; that repository publishes a ready-made file daily to a private
bucket. The plain gallery payload loads too, only without external identifiers
or promo sets. Two consequences show up in the rules: the datastore builder
marks the sets it appends with `Type == "promo"`, and Riftbound's `FilterCards`
refuses those printings unless the input edition itself resolves to a promo set
(promos reuse the main sets' collector numbers, so they would otherwise alias
every base card). And because the gallery exports no finish information at all,
every card is registered as available in *both* finishes — matching how
TCGplayer lists Riftbound singles as Normal and Foil.

**Indexes on `Backend`** (`mtgmatcher/backend.go`): `AllSets` and `Sets`
(code → `*Set`); `NormalizedSets` (normalized set name → `*Set`); `UUIDs`
(UUID → `*CardObject`); `CanonicalNames` (normalized → canonical); `Tokens`;
`Hashes` (normalized name → UUID list); `ExternalIdentifiers` (Scryfall/TCG/
etched id → UUID); `AlternateProps` (flavor names); the sorted name and
sealed-name arrays backing prefix/contains/regexp search; `SetUUIDs` and
`SetSealedUUIDs` (per-set sorted UUID buckets); `AllPromoTypes`,
`SLDDeckNames`, `CommanderKeywordMap`; the partitioned `AllUUIDs`/
`AllSealedUUIDs`; and the unexported `rules GameRules` that a loader attaches
with `SetRules`.

Two of those deserve their own note:

- **`UUIDs` holds pointers.** `map[string]*CardObject`, and `GetUUID` hands
  the pointer straight back. The object is shared with every other caller and
  **must never be modified after the load completes** — both the field and the
  accessor say so in-source. Copy before mutating.
- **`NormalizedSets` is built by the exported `IndexSets()`**, which every
  loader must call once its `Sets` are populated (hence exported: the loaders
  live in other packages). It visits set codes in sorted order so that two
  sets normalizing to the same name resolve deterministically — lowest code
  wins — replacing a linear rescan that followed random map order.

**UUID scheme and finishes.** The source datastore's UUID identifies a
printing; a card that exists in several finishes registers each one
explicitly in `Card.FoilUUIDs`, a finish → UUID map the loaders populate.
Magic keeps the historical suffixes there (`_f` for foil, `_e` for etched, and
split foil printings also carry `★`/`†` number suffixes); Riftbound spells the
finish out on every uuid, so a card yields `<id>_nonfoil` and `<id>_foil` and
no printing owns the bare gallery id; Lorcana keeps the base UUID and
`_f` for the primary pair, then gives every additional foil sub-type its own
name-derived UUID (`_rainbowpillars`, …) so none is lost — derived from the
sub-type *name* rather than its position, so it stays stable across data
updates that reorder foil types. `Card.Finish` records the verbatim,
lowercased finish name of the specific stored UUID, which is what keeps two
entries apart when the `Foil` boolean alone cannot. The base UUID still denotes
the most basic finish. These suffixed UUIDs are first-class — resolve them only
via `GetUUID`/`ExternalUUID`.

**Data model.** `Card`, `CardObject`, `Set` and `SealedProduct` are core types
declared in `mtgmatcher/backend.go`; the MTGJSON-shaped `AllPrintings`
structures are the Magic loader's own, in `mtgmatcher/magic/mtgjson.go`.
`CardObject` is a `Card` plus the resolved `Edition`/`Foil`/`Etched`/`Sealed`.
`Card` carries the MTGJSON field set, including `Legalities map[string]string`
(JSON tag `legalities`, format → legality) — populated only by the Magic
loader and **nil for both Lorcana and Riftbound** cards, so consumers must
handle that. It also carries the cross-game additions described above:
`FoilUUIDs`, `Finish`, `Images` (at minimum a `"full"` and a `"thumbnail"`
URL) and `OriginalNumber` (the pre-canonicalization collector number).

**No compatibility shims.** The Magic promo-type constants live only in
`mtgmatcher/magic`; core keeps no re-declared copies of them. Downstream code
that used to resolve `PromoTypeBoosterfun` and friends from `mtgmatcher`
imports the `magic` package instead. Core cannot import `magic` (it would
cycle), so a shim would have had to duplicate the values rather than alias
them, and a duplicated constant that silently drifts is worse than a build
error that names the symbol.

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
style abbreviations. `SplitVariants()` splits the parenthesized fields stores
use to distinguish printings while protecting legitimately-parenthesized
names such as *Erase (Not the Urza's Legacy One)* and *B.F.M.*; all three
games' `Prefilter` hooks call it.

Two behavior-gating date constants stay in core `mtgmatcher/utils.go` —
`BuyABoxInExpansionSetsDate` (2018-04) and `PromosForEverybodyYay`
(2019-10) — because the core `Match` skeleton itself consults them when it
decides whether to enrol a promo sibling set. The rest of the Magic
vocabulary moved to `mtgmatcher/magic/mtgjson.go`: `NewPrereleaseDate`
(2014-09), `BuyABoxNotUniqueDate` (2020-09) and
`SeparateFinishCollectorNumberDate` (2022-02), alongside the
`PromoType*`/`FrameEffect*`/`BorderColor*` constant set and the `★`/`†`
number suffixes.

### 2.3 Input and ID matching

`InputCard` (`mtgmatcher/card.go`) is
`{Id, Name, Variation, Edition, Foil, Language}` plus three flags that are
exported because the per-game rules packages set and read them across the
package boundary: `BeyondBaseSet` and `OriginalName` are internal matcher
state (`json:"-"`), while `PromoWildcard` is part of the serialized input
(`json:"PromoWildcard,omitempty"`). It carries 49 exported `Is*()` predicates
(`IsPrerelease()`, `IsPromoPack()`, `IsBundle()`, `IsSecretLair()`,
`IsWorldChamp()`, `IsSerialized()`, …) built on normalized comparisons; these
are the vocabulary the rules packages filter with.

`MatchId(inputId string, finishes ...bool)`: `finishes[0]` = foil,
`finishes[1]` = etched. The id is split at the first `_` only to *validate*
its shape (a UUID or a plain number for a TCG product id); the map lookups
themselves use the full input id, suffix included, which is how an `_f`/`_e`
UUID hits `UUIDs` directly. On a miss it retries through
`ExternalIdentifiers` (MTGJSON/Scryfall UUIDs or numeric TCG product ids,
including the etched product id). If the stored finish already matches the
request it returns immediately, else it re-derives via `output()`. If the
requested finish lives on a *different printing* (post-2022 Magic sets give
etched cards separate collector numbers), it scans the card's `Variations`,
comparing `ExtractNumberValue` of the collector numbers, and verifies the
alternate genuinely differs in finish before swapping.

`output(card, foil, etched)` is the finish reconciler. It first clamps the
requested flags against the printing's actual `Finishes` — a foil request for
a nonfoil-only printing degrades gracefully, a foil-only printing upgrades
automatically — and then resolves the clamped finish through `card.FoilUUIDs`,
which is the common path now that every loader registers a UUID per finish.
Only a `Card` without a registered map falls back to the historical `_f`/`_e`
suffix rules. Lorcana adds a wrinkle here: when a variation names a foil
sub-type, its `FilterCards` re-keys a copy of `FoilUUIDs` so that the
flag-driven resolution lands on that sub-type's UUID rather than the primary
foil's — a direct mention of the exported sub-type name wins, and failing
that, TCGplayer's convention of calling every sub-type past the primary cold
foil "Holofoil" resolves when the card stores exactly one such sub-type. The
tolerance for wrong foil flags from scrapers is a deliberate design point —
**trust the matcher's finish, not the scraper's input.**

### 2.4 The `Match()` pipeline and `GameRules`

`Match` is a `Backend` method (`mtgmatcher/mtgmatcher.go`) with a package-level
wrapper. It owns the skeleton; every game-specific stage is dispatched through
the `GameRules` value the loader attached:

```go
type GameRules interface {
    Prefilter(b *Backend, inCard *InputCard)
    AdjustName(b *Backend, inCard *InputCard)
    AdjustEdition(b *Backend, inCard *InputCard)
    FilterPrintings(b *Backend, inCard *InputCard, editions []string) []string
    FilterCards(b *Backend, inCard *InputCard, cardSet map[string][]Card) []Card
    IsUnsupported(b *Backend, inCard *InputCard) bool
    IsSpecificUnsupported(b *Backend, inCard *InputCard) bool
    MissingPromoTag(b *Backend, inCard *InputCard, co *CardObject) bool
}
```

Two contracts matter when implementing it. **Hooks receive the `InputCard` by
pointer and may mutate it**; mutations persist for the rest of the pipeline
*and* are visible to the caller after `Match` returns — the Magic
`FilterPrintings` sets `PromoWildcard` and `BeyondBaseSet` that way, and later
stages read them. And **`FilterCards` owns determinism**: the `cardSet` map it
receives iterates in random order, so an implementation returning more than one
candidate must impose its own ordering, because that result feeds the
user-visible aliasing diagnostics. Lorcana and Riftbound sidestep the map by
iterating `Hashes` (stable load order) and using `cardSet`'s keys only as an
edition filter.

A `Backend` with no rules attached returns `ErrDatastoreEmpty` from the name
path — checked explicitly before the prefilter runs.

The pipeline:

1. **Language resolution** — map codes via `LanguageCode2LanguageTag`, then
   scan for an embedded language tag; `InputCard.Contains` looks at the
   edition and the variation, never the name. Core owns this.
2. **Id fast path** — if `Id` is set, `MatchId()`; the hit is *validated*.
   A wrong language resets the input to the resolved card's fields and falls
   through to full matching; a token id in a non-default language returns
   `ErrUnsupported`; and `rules.MissingPromoTag` rejects prerelease /
   promo-pack / serialized claims the resolved card does not carry (upstream
   tags lag releases). This runs before the rules are known non-nil, so the
   hook call is guarded.
3. **Name surgery** — `rules.Prefilter`. Magic's version handles the
   Binderpos `Name [Edition]` syntax (resolving the bracket as a set name,
   falling back to variation, with the TCG `PP`-prefix promo-pack quirk),
   parenthesized variants via `SplitVariants()`, ` - ` suffix variants, and a
   prefilter renaming playtest/token name collisions (Red Herring,
   Unquenchable Fury, Shapeshifter). Lorcana splits only the parenthetical,
   because its names are themselves "Character - Title". Riftbound splits the
   parenthetical too but first gates on promo-only names, since promotional
   printings keep their storefront names verbatim. Core re-checks the foil
   flag afterwards, in case the prefilter moved a finish hint into the
   variant.
4. **Unsupported gate** — `rules.IsUnsupported`, checked before name
   resolution.
5. **Canonicalization** — `CanonicalNames[Normalize(name)]`; on miss,
   `rules.AdjustName` (Magic: typo/token/number fixups and flavor-name
   resolution via `AlternateProps`; Lorcana and Riftbound: a prefix fallback
   for feeds that truncate "Character - Title" names, plus Riftbound's
   champion-first legend remapping) and retry. Final miss → `ErrUnsupported`
   for tokens/oversize, else `ErrCardDoesNotExist`.
6. **Edition adjustment** — `rules.AdjustEdition`. For Magic this is the
   ~630-line ladder at the head of `mtgmatcher/magic/rules.go`: `EditionTable`
   aliases ("Alpha" → "Limited Edition Alpha", Universes Beyond names, …),
   variation-implies-edition rules ("Invocation" → Amonkhet Invocations),
   Commander-product parsing, and a terminal `default: switch inCard.Name` of
   per-card fixups. Lorcana and Riftbound instead trim the storefront noise
   ("Disney Lorcana: …", "Riftbound: League of Legends …", a trailing
   "Singles"). Core then applies its own hard `ErrUnsupported` gates for
   custom token sets and most oversize cards, and consults
   `rules.IsSpecificUnsupported`.
7. **Set selection** — `Printings4Card()`, then `rules.FilterPrintings` when
   more than one printing survives *or* the original name ended in "Token"
   (single-printing token names still need filtering). An empty result is
   `ErrCardNotInEdition`, or `ErrUnsupported` for tokens/oversize. Then a
   three-pass loop builds `cardSet map[setCode][]Card` via `MatchInSet()`:
   (a) perfect normalized edition-name match — and for
   prerelease/promo-pack/bundle/BaB inputs it *also* enrols the `P<code>`
   promo sibling set (or strips the `P` for the reverse), skipping JPN cards
   and gating bundle/BaB on the core date constants;
   (b) heuristic pass: edition substring containment, generic promos
   restricted to `*Promos` sets, bundle/BaB allowed into recent-enough base
   sets — skipped wholesale for World Championship inputs, whose short set
   names would over-match;
   (c) YOLO pass: all printings, trusting downstream filtering.
   Passes (a)/(b) are skipped in `PromoWildcard`/Secret Lair mode, which
   wants maximal candidates.
8. **Card-level disambiguation** — `rules.FilterCards`, run
   **unconditionally**. This is a deliberate change from the pre-`GameRules`
   pipeline, which returned a lone candidate without validating it: Lorcana
   enforces the collector number in this hook, and the old shortcut let a
   wrong-numbered card through. Magic preserves the historical behavior
   *inside* its own hook — a single card in a single set is returned as-is,
   so a lone candidate still matches even when the variation carries junk —
   but that is now the game's choice rather than the skeleton's. World
   Championship inputs then keep only the first match (decks are per-player
   duplicates), and a core language filter drops non-English prints unless a
   language was requested.
9. **Verdict** — 0 cards: `ErrCardWrongVariant` (or `ErrCardMissingVariant`
   if no variation was given, `ErrUnsupported` if a language was involved);
   1 card: `output()` plus a final `rules.MissingPromoTag` validation;
   2+: `AliasingError`, whose `Probe()` returns all candidate UUIDs —
   consumers log these as data-quality alarms (and, like `mtgban-website`,
   may pick the newest printing from `Probe()`).

Error taxonomy (`mtgmatcher/utils.go`): `ErrDatastoreEmpty`,
`ErrCardUnknownId`, `ErrCardDoesNotExist`, `ErrCardNotInEdition`,
`ErrCardWrongVariant`, `ErrCardMissingVariant`, `ErrUnsupported`,
`AliasingError`. `ErrUnsupported` doubles as a silent-skip channel *and* a
found-but-invalid-promo-tag signal.

### 2.5 The per-game rules packages

Each game package has the same shape: a `Load()` datastore converter, a
`rules.go` holding the `GameRules` implementation, a `register.go` whose
`init()` calls `RegisterGame`, and a golden replay suite. Lorcana and Riftbound
keep that suite inside the package and need nothing else — three source files
apiece. Magic needs considerably more (`callbacks.go`, `table.go`,
`wrappers.go`, a `doc.go`), and its replay suite is the one piece that did not
move: it still lives in core's `mtgmatcher` test package (§2.7). What differs
between the games is how much identification logic each needs.

**Magic** (`mtgmatcher/magic/rules.go`, ~2,050 lines) carries essentially all
of it. `FilterPrintings` eliminates whole sets using the input's promo
predicates against set type, release dates and name patterns — dedicated,
repetitive blocks for prerelease vs promo-pack, release/launch promos, BaB,
bundles, Secret Lair vs Mystery List, WCD, MagicFest, Duel Decks, 30th
Anniversary, judge promos, and a wildcard-promo mode. `FilterCards`
disambiguates within sets, consulting in order:

1. the hand-curated `VariantsTable` — which moved into the game package with
   the rules that read it (`mtgmatcher/magic/variants.go`, ~4,650 lines of
   pure data: set → card → variant tag → collector number, alongside
   `MultiPromosTable`); the `EditionTable` aliases are still core-level
   (`mtgmatcher/editions.go`);
2. `ExtractNumber` with its suffix semantics;
3. promo-type validation through the `promoTypeElements` table (each entry:
   tag strings, an optional `TagFunc`, an activation date, wildcard
   eligibility);
4. per-set `simpleFilterCallbacks` / `complexFilterCallbacks` /
   `numberFilterCallbacks` (`mtgmatcher/magic/callbacks.go`, ~1,200 lines)
   for sets whose disambiguation needs real code;
5. per-set promo dedup via core's `MultiPromosTable`, then finish/frame
   separation — etched (gated on `SeparateFinishCollectorNumberDate`),
   borderless, extended art (gated on `PromosForEverybodyYay`) and showcase —
   each pass discarding its result rather than the candidates when it would
   filter everything away.

This three-tier design — **data tables first, generic number/promo logic
second, per-set code last** — is the Magic package's core maintenance pattern:
most new-set support lands as table entries, not code.

**Lorcana and Riftbound** need none of that. Both have no edition aliases, no
variant tables and no promo types, so `FilterPrintings`, `IsUnsupported`,
`IsSpecificUnsupported` and `MissingPromoTag` are no-ops, and the real work is
name + collector number + finish narrowing in `FilterCards`, with the edition
breaking ties when it resolves. The interesting details are the ones the data
forces: Lorcana honors the name hash rather than the edition-keyed `cardSet`
values so that case-variant spellings (three real pairs exist) stay reachable,
and it strips leading zeros from numbers while keeping a genuine `"0"`
reachable; Riftbound canonicalizes numbers out of the public code
("OGN-066a/298" → "66a") and refuses promo sets unless explicitly targeted.

### 2.6 Sealed products & search API (`mtgmatcher/api.go`)

Lookups: `GetUUID`, `GetSet`, `GetSetByName`, `GetAllSets`, `GetUUIDs`/
`GetSealedUUIDs`, the per-set `GetUUIDsInSet`/`GetSealedUUIDsInSet` (backed by
the `SetUUIDs`/`SetSealedUUIDs` buckets — the result aliases the index and must
not be modified), `Printings4Card`, `CardReleaseDate`, `ExternalUUID`,
`AllPromoTypes`, `AllNames(variant, sealed)` and `NameIsToken`.

Search: `SearchEquals`/`SearchHasPrefix`/`SearchContains`/`SearchRegexp` over
the sorted name arrays, with `SearchSealedEquals`/`SearchSealedContains` for
products.

`HasPrinting(name, field, value, editions...)` is the exported generic
"does any printing of this name carry X" query. The finish-based
`HasNonfoilPrinting`/`HasFoilPrinting`/`HasEtchedPrinting` stay in core
because every game has finishes; the Magic-vocabulary wrappers
(`HasBorderlessPrinting`, `HasExtendedArtPrinting`, `HasShowcasePrinting`,
`HasPromoPackPrinting`, `HasSerializedPrinting`, `HasRetroFramePrinting`) live
in `mtgmatcher/magic/wrappers.go`, where the treatments they name do.

**Name buckets are not one-card-per-bucket.** `Hashes` deliberately hashes a
card under its face, flavor and printed names as well as its full name, so a
query naming a single face still finds the card — which means one bucket can
hold several *distinct* cards ("Servo" hashes both the Servo token and
"Servo // Thopter"). `entry4Name` is the disambiguator behind
`Printings4Card` and `NameIsToken`: it prefers the entry whose name matches
verbatim, then any entry whose name normalizes the same (normalization folds
plurals, so "Cat Warrior" and "Cat Warriors" are distinct cards sharing a
bucket), and only then falls back to the first entry, which is the right
answer for alias-only buckets such as flavor names.

Sealed products are modeled end-to-end:

- `BoosterGen(set, boosterType)` performs MTGJSON-rule weighted sheet draws
  (`weightedrand`), honoring `BalanceColors` (an approximation citing
  magic-search-engine) and per-sheet `AllowDuplicates`; its single hard-fail
  is `maxRerollThreshold = 50` ("reroll threshold reached"). The `slc` Secret
  Lair random-foil ~30% behavior is a hardcoded special case.
- `GetPicksForSealed` recursively expands product contents
  (card/pack/deck/sealed/variable), and `GetPicksForDeck` does the same for a
  named deck. `GetDecklist`/`SealedHasDecklist` distinguish fixed-content
  products, `SealedIsRandom` flags random ones, and `SealedCardUnit` reports
  how many cards a product yields.
- `GetProbabilitiesForSealed`, `SealedBoosterProbabilities` and
  `SealedSheetProbabilities` compute exact per-card pull probabilities — the
  inputs to `sealedev`'s EV computation.
- `BuildSealedProductMap` and the load-time reverse index
  (`fillinSealedContents`, in the Magic loader) link single cards back to the
  products containing them.

### 2.7 Testing — strategy & coverage map

**What exists.** Each game has a data-driven golden replay suite that matches
a corpus of real inputs against expected UUIDs or errors, gated on an
environment variable pointing at a real datastore, with a flag that
regenerates expectations after intentional changes:

| Game | Suite | Corpus | Env var | Regenerate |
|---|---|---|---|---|
| Magic | `mtgmatcher/magic/matcher_test.go` | `mtgmatcher/magic/testdata/magic_test_data.json` | `ALLPRINTINGS5_PATH` | `-u` |
| Lorcana | `mtgmatcher/lorcana/matcher_test.go` | `mtgmatcher/lorcana/testdata/lorcana_test_data.json` | `LORCANA_PATH` | `-update-lorcana` |
| Riftbound | `mtgmatcher/riftbound/matcher_test.go` | `mtgmatcher/riftbound/testdata/riftbound_test_data.json` | `RIFTBOUND_PATH` | `-update-riftbound` |

These are the regression harness for the heuristic tables. Run the relevant
regeneration after a *deliberate* matching change and **review the diff** —
never blindly accept it. The Lorcana and Riftbound suites additionally carry
hand-authored seed cases whose expected verdicts are baked by the regeneration
rather than hard-coded, with a `"negative:"` description prefix declaring the
author's intent so the regeneration fails loudly when the outcome class
changes.

Note the three suites do not behave alike when their datastore is missing: the
Lorcana and Riftbound suites call `t.Skip`, while the Magic suite's `TestMain`
fails outright ("Need ALLPRINTINGS5_PATH variable set to run this suite").
Set `ALLPRINTINGS5_PATH` before running `go test ./mtgmatcher/...`.

Unit tests cover normalization, number/year extraction, variants-table
integrity, the set index, the search surface, `HasPrinting`, CSV write-error
propagation (`mtgban/csv_error_test.go`), and (in `mtgban/base_test.go`) the
`Add*` family. `mtgmatcher/rules_test.go` is the one that pins the new
pipeline contract: it builds a `Backend` by hand — so `SetRules` was never
called — and asserts that every rules-dependent entry point returns
`ErrDatastoreEmpty` rather than panicking on the nil hooks. Scraper packages
are otherwise validated operationally.

**The pyramid is inverted here.** Card identity is only meaningful against a
large real dataset (AllPrintings alone is a few hundred MB), so the *largest*
test surface — the matcher — is a data-backed integration replay rather than a
unit test, while the most business-critical code, the money path, needs **no**
external data yet has almost no direct tests. The strategy follows from that
asymmetry:

| Layer | Targets | Test type | Needs dataset? | Today |
|-------|---------|-----------|----------------|-------|
| **Money path** (top risk) | `Arbit`, `Mismatch`, `Pennystock`, `add()` invariants, profitability formula | unit / golden on synthetic records | **No** — runs in CI | none beyond `Add*` |
| **Matcher** (data integrity) | `Match`/`MatchId`, normalization, variants/editions, sealed API | data-backed regression replay | **Yes** — one per game | replay + unit |
| **Scraper preprocess** (breadth) | per-store title → `InputCard` → `Match` | table tests on captured fixtures | partial | 3 of 24 |

Principles: (1) **the money path is unit-testable and unprotected — cover it
first**, with in-test records and no datastore dependency; (2)
**characterization before refactor** — pin `Arbit`/`Mismatch`/`add()` outputs
*before* changing them, and refactor under green; (3) **assert invariants, not
just functions** — the `entries[0] == NM` ordering is a sort side effect that
`Arbit` and the CSV writers depend on, so pin it directly; (4) **scrapers:
breadth over depth** — a few fixture table tests for the gnarliest
preprocessors (cardmarket, cardtrader, tcgplayer) catch the realistic break.
`abugames`, `cardkingdom` and `starcitygames` already have `preprocess_test.go`
files to copy from.

**CI provisions all three datastores.** `.github/workflows/ci.yml` runs
`cache-datastore` (Magic, via the reusable `cache-file.yml`), `cache-lorcana`
(same, over a plain uncompressed LorcanaJSON file) and a bespoke
`cache-riftbound` job — Riftbound has no public URL, so the file built by
`github.com/mtgban/riftbound-datastore` is pulled from its private B2 bucket
and cached under the same key/filepath contract. The test step then exports
`ALLPRINTINGS5_PATH`, `LORCANA_PATH` and `RIFTBOUND_PATH` before running
`go test ./... -v`, so the data-backed suites actually execute in CI rather
than skipping into a falsely green run.

---

## 3. Scraper packages

Idealized shape: `NewScraper(creds)` returning a struct with `LogCallback`
(exported, always first), `MaxConcurrency` (exported, default 8), optional
`Partner`/`Affiliate`, exported `DisableRetail`/`DisableBuylist`, and
unexported `inventory`/`buylist` + `inventoryDate`/`buylistDate`. `Load(ctx)`
fans out via `mtgban.WorkerPool` (2–8 workers) over `retryablehttp` clients
(the politest of them, cardmarket / cardsphere / mtgstocks, additionally set
`LinearJitterBackoff`); a `preprocess.go` translates store naming into
`InputCard` + `Match()`, skipping `ErrUnsupported`, logging `AliasingError`s;
results inserted via the `Add*` family. Every scraper has a tagged `printf`
helper (`x.LogCallback("[TAG] "+format, a...)`). File convention:
`<store>.go` / `api.go` / `preprocess.go` / optional `sealed.go` (a *separate*
scraper struct with its own `SealedMode` `Info`).

**The preprocess → Match() contract** (per listing): build an `InputCard`,
call `Match()` — or `MatchId(externalID, foil, etched)` when the store exposes
a Scryfall or TCGplayer id. Those two are the only entry points: every
scraper's free-text path runs through the same `Match()` pipeline, so a
matching improvement lands for all of them at once and no scraper carries a
private shortcut. On `ErrUnsupported` silently `continue`; on `AliasingError`
log and `Probe()`; on other errors log with context (many scrapers suppress
known-noisy editions first); then insert with `Add*`. `PriceRatio` is computed
by reading back `inventory[cardId]` before inserting the buylist row.

### API-based

| Package | Service & auth | Notes |
|---|---|---|
| `tcgplayer` | OAuth via `go-tcgplayer` + cookie-authed marketplace APIs | Largest: Market/Index/Sealed/SYP-list/per-seller scrapers, plus the table-driven single-game pair (see below); SKU map keyed by UUID; TCG Direct modeled as a Vendor with net-after-fees pricing |
| `cardmarket` | OAuth 1.0 HMAC-SHA1 (gentle retry) | `CardMarketIndex` is a **Market** (`MarketNames → MKM Low/Trend`, `MetadataOnly`, `Family="MKM"`); EUR→USD; Lorcana and Riftbound via game id; `CardMarketSealed` separate |
| `cardtrader` | Bearer token | `CardtraderMarket` (**Market**, 3 seller tiers, `Family="CT"`, `CountryFlag="EU"`); Lorcana and Riftbound via game id; `CardtraderSealed` mirror; bulk upload + cart APIs |
| `cardkingdom` | Public pricelist via `go-cardkingdom` (file/URL-fed, no own client) | Full 4-condition buylist with price ratios; `CreditMultiplier 1.3`; singles + `sealed.go` + `graded.go` are three scrapers |
| `manapool` | Public JSON API | Exactly two scrapers: `Manapool` (aggregate, `MatchId` by Scryfall id, `NoQuantityInventory`) and `ManapoolSealed` |
| `arcanafrisia` | Public buylist endpoint | Buylist-only EU vendor, shorthand `AF`; matches by Scryfall id and maps the store's NM/EX/GD grades onto NM/SP/MP |
| `cardsphere` | Session cookie (gentle 3s) | Buylist-only; `BuyPrice ×0.87` fee **and** `CreditMultiplier 1.1` |
| `mtgstocks` | Public API, **UA rotation** (`uarand`) | MetadataOnly index (average/market interests) |

**The tcgplayer single-game scrapers.** `TCGGame` (retail) and `TCGGameIndex`
(index pricing) in `tcgplayer/game.go` and `gameindex.go` are built from one
table:

```go
var SupportedGames = map[string]int{
    mtgban.GameLorcana:   tcgplayer.CategoryLorcana,
    mtgban.GameRiftbound: tcgplayer.CategoryRiftbound,
}
```

Supporting one more game is one entry here, provided the matcher has a
datastore for it. Magic is deliberately absent: it is identified by SKU and
has its own scrapers. Both game scrapers pass the printing name through the
`InputCard.Variation` field alongside the collector number, so the game's
rules can tell foil sub-types apart (this is what makes Lorcana's
"Holofoil" convention resolvable — see §2.3).

### HTML / crawler

`starcitygames` (HawkSearch/Meilisearch APIs, serialized detection, sealed,
plus Lorcana and Riftbound behind its numeric game ids), `coolstuffinc`
(multi-game including Lorcana and Riftbound, `CreditMultiplier 1.25`),
`hareruya` (JPY, **bespoke 403 → 5-min backoff**), `magiccorner` (EUR,
Italian), `abugames` (Solr, MINT-aware grading, `InfoForScraper`), `mtgseattle`
(`CreditMultiplier 1.33`), `ninetyfive`, `mintcard` (rides TCG SKUs,
`CreditMultiplier 1.1`), `vegassingles`, `secretdeskorrigans` (CAD, French),
`toamagic` (Spanish), `miniaturemarket` (sealed-only).

**Legacy cohort — `gocolly` + hand-rolled goroutines, predate WorkerPool:**
`trollandtoad` (plus a `generic.go` Lorcana scraper and sealed),
`wizardscupboard`, `strikezone`. These are the remaining standardization gap;
migrating them to `WorkerPool` would make operational behavior uniform.

`sealedev` builds sealed-EV "scrapers" from mtgmatcher probabilities or
5,000-run booster simulations priced against the MTGBAN API, emitting EV
entries with dispersion stats (std-dev/IQR), `Family="EV"`, `SealedMode`, and
`MetadataOnly` toggled per sub-scraper.

**Not templates.** Some directories in a working tree are untracked WIP and
are not part of the committed module — `synthetic/` (a *computed* buylist with
no site behind it, synthesizing prices from TCG/CK/SCG, `MetadataOnly`) and
`mvpsportsandgames/`, whose non-conforming `Inventory() (record, error)` does
**not** satisfy `mtgban.Seller`. For new work, copy `ninetyfive` (API) or
`mtgseattle` (HTML) — never the legacy colly trio or an untracked orphan.

---

## 4. Tooling — `cmd/` and CI

Committed tools: **bantool** (the production orchestrator), **boosterGen**,
**boosterList**, **manapoolOrders**, **mkmPriceGuide**, and **tcgid4scryfall**
(TCG id → Scryfall id export). A long tail of further tools exists only as
untracked working-tree WIP (`manapoolSeller`, `mkmhtml2csv`, `mp2ckbl`,
`amazonsearch`, `omnitool-3g`, `autocart`, and the `ck*`/`ct*`/`mkm*` family);
treat anything not in the list above as unreviewed, and note that some of it
embeds live credentials.

- **bantool** — a registry of `scraperOption{constructor, flags}` for every
  target, including the per-game ones: six `*_riftbound` targets (cardmarket,
  cardtrader, coolstuffinc, starcitygames, tcg_index, tcg_market) and seven
  `*_lorcana` ones — the same six plus `strikezone_lorcana`, which has no
  Riftbound counterpart. Selection via `-scrapers`/`-sellers`/`-vendors`;
  `-format` json/csv/ndjson (each also with an `.xz` variant); output through
  `github.com/mtgban/simplecloud` to local/B2/GCS/S3/HTTP; optional HMAC
  signing (`BAN_SECRET`); all credentials via env vars (godotenv autoload).
  It blank-imports `mtgmatcher/games`, which is what lets `-datastore` accept
  a file for any of the three games without further configuration. Init
  closures set `scraper.LogCallback = GlobalLogCallback` as a **direct field
  assignment on the concrete pointer** in more than forty places — the binding
  constraint on any `BaseScraper` refactor (the field must stay exported and
  embedding-reachable).
- **manapoolOrders** — Mana Pool buyer-order CSV dumps.
- **mkmPriceGuide** — Cardmarket price-guide export.
- **boosterGen / boosterList** — booster simulation and sealed introspection
  over the mtgmatcher sealed API.
- **tcgid4scryfall** — TCGplayer id → Scryfall id mapping export.

**CI** (`.github/workflows/`). `ci.yml` provisions the three datastores (§2.7)
and then gates on three steps in order: **Check formatting** (fails on any
`gofmt -l` output), **Vet** (`go vet ./...`), and `go test ./... -v`. There is
one `bantool-<target>.yml` per scraper target — including the per-game
variants such as `bantool-cardmarket_lorcana.yml` and
`bantool-tcg_market_riftbound.yml` — each triggered by cron plus
`workflow_dispatch`/`repository_dispatch`, and each delegating to the reusable
`run-bantool.yml` with `target`, `game` and `datastore-filepath` inputs.
`run-bantool.yml` uploads to `b2://mtgban-dumps/<game>/<target>` and then pings
a signed `http://<game>.mtgban.com/api/load/<target>` URL so the server reloads
the fresh snapshot. Magic targets prepend a `cache-datastore` job and pass a
cached local path; Riftbound targets skip caching entirely and pass a `b2://`
path that bantool reads directly (which is why they need the datastore B2
keys), and they run on 12-hour crons under a queued concurrency group rather
than the main Magic stores' hourly schedule. No Makefile or Docker — plain
`go build` per `cmd/` subdirectory.

**Key dependencies**: goquery/colly (HTML), retryablehttp + cleanhttp
(HTTP), simplecloud (storage abstraction), go-ndjson, weightedrand (boosters),
montanaflynn/stats (EV), golang.org/x/text (normalization), uarand (UA
rotation), plus the in-house `go-cardkingdom` and `go-tcgplayer` clients.

---

## 5. Design through-lines

> The load-bearing decisions below are recorded as ADRs in
> [`docs/adr/`](docs/adr/) with full context and alternatives: UUID-as-key
> (ADR-0001) and the immutable global backend (ADR-0002).

1. **Everything keys on the mtgmatcher UUID** — scrapers are thin
   translators; correctness lives in one place. (By convention, not
   type-enforced; sealed inserts the product UUID directly.)
2. **One pipeline, pluggable rules** — `Match()` is a single skeleton and
   every game-specific decision is a `GameRules` hook. Adding a game means
   adding a package, not adding a branch to shared code; and because core
   never imports a game package, a Magic-only concept cannot leak into the
   Lorcana path by accident.
3. **Games self-register, consumers opt in** — the `database/sql` idiom.
   `RegisterGame` from an `init()`, a blank import to activate, `Open` when
   the game is known and auto-detection is unnecessary. A binary links only
   the games it imports.
4. **Sorted-record invariants instead of queries** — `entries[0] == NM` is
   produced by `add()`'s sort and consumed by `Arbit`; the CSV writers rely on
   the ordering too.
5. **Tables before code** — new-set support is data (`VariantsTable`, the
   edition aliases, the promo elements); per-set callbacks are the escape
   hatch, and they live in the game package rather than in core.
6. **Graceful degradation on dirty input** — `output()` finish clamping, the
   Id-path validation/reset, non-strict CSV loading, `ErrUnsupported` as a
   silent-skip channel distinct from real errors.
7. **Injected logging + bounded worker pools** — uniform operational
   behavior; the WorkerPool migration of the legacy colly trio is the
   remaining standardization gap.
8. **Build-once, read-only state** — the matcher backend is an immutable
   global *by convention*. The contract is unenforced and violated by the
   consumer's runtime reload endpoint (§2.1); `Open()` plus an
   `atomic.Pointer[Backend]` is the supported way out for consumers that need
   hot swaps.

## 6. Extending the system

**Adding a store**: create a package with the four-file layout, implement
`Seller` and/or `Vendor` (and `Market`/`Trader` if it has sub-sellers), fetch
with `WorkerPool` + retryablehttp, write a `preprocess.go` that builds
`InputCard`s and handles the store's naming quirks, register a `scraperOption`
in bantool, and add a GitHub Actions workflow. Set the right `ScraperInfo`
flags (`MetadataOnly`, `NoQuantityInventory`, `SealedMode`, `CreditMultiplier`,
`Family`, `Game`). The hard part is always preprocessing — which is why
mtgmatcher's typed errors, variant tables, and per-set callbacks exist.

**Adding a game**: create `mtgmatcher/<game>/` with a `Load(io.Reader)
(*mtgmatcher.Backend, error)` that converts the source data (populating
`Sets`, `UUIDs`, `Hashes`, `CanonicalNames`, `ExternalIdentifiers` and
`FoilUUIDs`, then calling `IndexSets()` and `SetRules()`), a `rules.go`
implementing `GameRules`, a `register.go` whose `init()` calls
`RegisterGame`, and a replay suite gated on a `<GAME>_PATH` environment
variable with a regeneration flag. Add the game to `mtgmatcher/games`, add a
`Game` constant in `mtgban`, and make `Load` reject inputs it does not
recognize so auto-detection can move past it. Existing storefronts often come
along cheaply: a TCGplayer category is one entry in
`tcgplayer.SupportedGames`, and cardmarket / cardtrader / coolstuffinc /
starcitygames all select games by id.

---

## 7. Primary consumer — `mtgban-website` (usage reference)

The reference embedder demonstrates the intended production topology:
**scraping and serving are decoupled.** bantool scrapes and uploads per-store
`Seller`/`Vendor` JSON; the website loads those snapshots and the matcher
datastore, and never runs scrapers in-process. Canonical patterns:

- **Activate a game, then load the datastore once at startup** — the matcher
  can match nothing until a game package is linked in, so blank-import
  `mtgmatcher/games` (or just the games you serve) and then call
  `mtgmatcher.LoadDatastore(reader)` streamed from a `simplecloud` bucket,
  firing async cache builds afterwards. When the game is known,
  `mtgmatcher.Open("magic", reader)` skips auto-detection and hands back a
  `*Backend` you own. A signature-verified `/api/load/datastore` endpoint can
  reload the global at runtime (see the §2.1 race caveat, and prefer an
  `atomic.Pointer[Backend]` over the global if you do this).
- **Consume pre-scraped JSON** — `mtgban.ReadSellerFromJSON` /
  `ReadVendorFromJSON` per `game/name/kind/shorthand`. The live sets sit
  behind `atomic.Pointer[[]mtgban.Seller]` / `[[]mtgban.Vendor]` for lock-free
  reads with single-writer publish — the correct concurrency pattern for a
  long-running server over swappable snapshots.
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
  `GetVendors()`; sort the `[]ArbitEntry`. Apply `CreditMultiplier` yourself
  if you want store-credit pricing — `Arbit` does not.
- **Buylist pricing reducer** — range `GetVendors()`, filter by `SealedMode`/
  shorthand, fold `vendor.Buylist()` entries into a price map
  (`getVendorPrices`).
- **Sealed introspection** — `GetSealedUUIDs`/`GetDecklist`/`GetPicksForSealed`/
  `SealedIsRandom`/`SealedHasDecklist` (the website surfaces booster/deck flags
  but delegates generation to the matcher). It does not call `BoosterGen`
  directly — for that, the embedded `cmd/` tools are the example.
- **CSV export** — `mtgban.WriteBuylistToCSV(records, creditMultiplier, w)`
  straight to an HTTP writer.
