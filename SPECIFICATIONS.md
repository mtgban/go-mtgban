# go-mtgban — Architecture & Development Specification

**Module**: `github.com/mtgban/go-mtgban` (Go 1.26)
**License**: dual AGPLv3 + commercial (see [LICENSING.md](LICENSING.md))

go-mtgban is a trading-card market-data platform: it scrapes retail
inventories and buylists from a couple of dozen card stores and marketplaces,
normalizes every listing to a canonical card identity, and computes arbitrage
opportunities between them. Magic: The Gathering is the primary game; eight
more are supported alongside it — Disney Lorcana, Riftbound (the League of
Legends TCG), One Piece, Yu-Gi-Oh, Flesh and Blood, Pokemon, Gundam and
Palworld — each with its own datastore, its own matching rules, and its own
set of scraper targets.

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
  + mtgmatcher/{magic,lorcana,riftbound,onepiece,     per-game loaders & rules
      yugioh,fleshandblood,pokemon,gundam,palworld}    (nine, one per game)
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
GenericEntry   Pricing() float64; Condition() Condition; Qty() int
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
type-assertion.

### 1.2 Records and entries

`InventoryRecord = map[string][]InventoryEntry` and
`BuylistRecord = map[string][]BuylistEntry` — both keyed by the mtgmatcher
UUID. The key is a string by convention; the `Add*` path does **not**
type-check it against the backend (sealed scrapers, for instance, insert the
product UUID directly without calling `Match()` — see §3).

- `InventoryEntry`: `Quantity`, `Conditions` (a `Condition`, named by `NM`
  through `PO`), `Price` (USD), `URL`, `SellerName`, `Bundle` (part of a
  direct-shipping hub), `OriginalID` (store product id), `InstanceID` (SKU),
  `CustomFields map[string]string`, `ExtraValues map[string]float64`.
- `BuylistEntry`: swaps `Price` for `BuyPrice` + `PriceRatio` (buy/sell
  ratio, a desirability signal) and `SellerName` for `VendorName`.

A scraper reads a store's grade text with `ParseCondition`, which knows the
spellings stores share and returns `ErrInvalidCondition` for anything else.

**Insertion semantics** (`add()` in `mtgban/base.go`) — the de-dup engine
every scraper relies on. Defaults are applied first, and the two sides differ:
both default an empty condition to `NM`, but only the inventory side
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
both. `TestSort` in `base_test.go` pins the order; run it after touching
the add path.

`ScraperInfo` carries identity (`Name`, `Shorthand`, `CountryFlag`, and
`Game`) plus behavior flags consumed by the analysis layer. `Game` is of type
`mtgmatcher.Game`, the game's own lowercase name, the one its datastore loader
registers under: the constants `mtgmatcher.GameMagic`, `GameLorcana`,
`GameRiftbound` and the rest (`mtgmatcher.AllGames` lists all nine).
Every scraper sets `Game` explicitly, `GameMagic` included: the zero value
names no real game, so an unset field is a bug to fix rather than a reading
of Magic. Giving it a type of its own is what settles which naming a scraper is *built*
from: a multi-game scraper reads `b.Game` off its datastore, converts it to the
vendor's own naming through one unexported map, and reports `Game` back from
the backend. The vendor's spellings stay exported — each package's API
helpers take one — but nothing outside has to know them to ask for a game. The scrapers that serve more than one game are
cardmarket, cardtrader, coolstuffinc, gamenerdz, miniaturemarket,
starcitygames, strikezone, tcgplayer's `TCGGame`/`TCGGameIndex`/`TCGSYPList`,
and vegassingles. The behavior flags are `MetadataOnly` (index
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

`Arbit(b, opts, vendor, seller)` then, for each buylist card present in the
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

`Mismatch(b, opts, reference, probe)` is the seller-vs-seller analogue with
identical filter scaffolding, but it compares across grades instead of
requiring an exact condition match. `defaultGradeMap` is the ladder
`NM=1, SP=0.8, MP=0.6, HP=0.4, PO=0`, and the reference price is scaled by
`invGrade / refGrade` — dividing out the reference entry's own grade before
applying the probe's, so a non-NM reference is not compared against a
rescaled copy of itself. A pair whose grade is missing from the map or is
`≤ 0` (which `PO` is) is skipped with `continue` rather than silently priced
at zero. The result carries `ReferenceEntry` instead of `BuylistEntry`, and
`NoQuantityInventory` bypasses the qty gate here too.

`mtgban` imports no game package; a report that reads one game's rarities,
borders or promo types lives with its caller.

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
writers auto-selecting the header from the data shape, and loaders
(`LoadInventoryFromCSV(b, r, flags...)`, `LoadBuylistFromCSV`) skipping bad
rows when passed `false` instead of stopping at the first one.
**`WriteBuylistToCSV(b, buylist, creditMuliplier float64, w)` is the one
writer with a multiplier argument** (all others are `(b, data, w)`); the param
is spelled `creditMuliplier` (missing "t") in source, and it computes the
Trade Price column as `BuyPrice × creditMuliplier`.

`mtgban/utils.go` supplies `GetExchangeRates(ctx, opts...)` and
`GetExchangeRate(ctx, currency, opts...)` (fawazahmed0 currency CDN,
`@latest`/unpinned), which return the **reciprocal**, i.e. a
*multiply-to-USD* factor, not the raw quoted rate.

---

## 2. `mtgmatcher/` — the matching engine

The hard problem: store listings name cards inconsistently
("Lightning Bolt (Borderless) - MH2 *F*"), while prices must key on an exact
printing. mtgmatcher resolves free text to a unique UUID across hundreds of
sets and dozens of promo classes.

The package is **game-agnostic core plus one package per game**. Core owns the
`Backend` data model, the `Match()` skeleton, normalization, the search API and
the sealed-product API; the nine `mtgmatcher/<game>` packages (`magic`,
`lorcana`, `riftbound`, `onepiece`, `yugioh`, `fleshandblood`, `pokemon`,
`gundam`, `palworld`) each own their datastore loader and the game-specific
identification logic that core dispatches through the `GameRules` interface.
The dependency runs one way only — the game packages import core, core never
imports a game package — which is what keeps the vocabularies from bleeding
into each other.

### 2.1 Games, loaders, and the `Backend`

**Registration.** Loaders register themselves in the style of `database/sql`
drivers (`mtgmatcher/datastore.go`):

```go
type GameLoader func(io.Reader) (*Backend, error)

func RegisterGame(name Game, load GameLoader)  // panics on nil or duplicate
func RegisteredGames() []Game                  // registration order
```

Each game package has a `register.go` whose `init()` calls `RegisterGame`,
under the name of its own package — `"magic"`, `"lorcana"`, `"riftbound"`,
`"onepiece"`, `"yugioh"`, `"fleshandblood"`, `"pokemon"`, `"gundam"`,
`"palworld"` — so a consumer activates a game with a blank import and pays
for nothing it does not use:

```go
import _ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
```

`mtgmatcher/games` is a meta-package that blank-imports every game, for
consumers that want them all with a single import; `cmd/bantool` does
exactly that. The trade-off is spelled out in its doc comment: it links every
game and its transitive dependencies into the binary.

**Loading.** The caller names the game:

```go
func Open(name Game, reader io.Reader) (*Backend, error)
```

`Open` loads exactly the named game and returns the `Backend`, stamped with
the name it was loaded as (`b.Game`). There is no global datastore to install
it into: a caller holds the backend and asks it. Asking for a game nothing
registered fails with an error naming the games that are.
There is no auto-detection: the loader that once tried every registered game
in turn decoded AllPrintings through three foreign decoders before Magic's,
behind a buffer of the whole file, and was removed. bantool reads the game
off the registry key its target sits under; a suite names its own in the
helper that loads it.

**No global backend.** Every lookup is a method on `*Backend`; the package
keeps no datastore of its own and no package-level logger (`b.Logger`, read
through `b.Logf`, is the caller's to set). A scraper is built on a backend
with `mtgban.NewScraper(b, name, opts...)` and matches against that one
alone; two games can be priced in one process by two backends. A backend is
immutable once loaded: maps, slices, card pointers and rules are shared by
everything built on it. Replacing a datastore is loading another backend and
building new scrapers on it. See
[ADR-0004](docs/adr/0004-localized-matcher-and-scraper-registry.md), which
supersedes the global publication of ADR-0002 and ADR-0003.

**Backend as a type.** `Open()` returns an independent `*Backend`. Magic's
identification callbacks search that backend, including The List's Game Day
exception and the token lookup in its generic-promo check; its exported
`Has*Printing` helpers take the backend first. A process serving several
games keeps one backend per game and asks each.

`Arbit`, `Mismatch` and the CSV readers and writers take the
backend as their first parameter; `ArbitOpts` carries optional filters and
nothing else, and a nil backend gives a nil report. Custom callbacks doing
auxiliary lookups use the same backend, and so does a caller naming the card
behind an `ArbitEntry`, since the entry stores a card ID rather than its
datastore.

Every accessor is an instance method, `GetUUIDsInSet`, `GetSealedUUIDsInSet`
and `Names` (the former `AllNames`) included; `AllPromoTypes` is a field.
`ExtractNumber` reads no datastore and stays a package-level function.

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
  (LEGITA/DRKITA/4EDALT and SLD/PURL JPN language dupes). Every copy's UUID
  carries its tag (`_ita`, `_alt`, `_jpn`); only a copy filed in its
  original's set carries it on the number too, and a copy in the original's
  language loses a tie to it (`FinalizeCandidates`).
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
`github.com/mtgban/datastore-gen`, which stamps every printing with its
TCGplayer product id and appends the promotional printings the gallery does
not carry; that repository publishes a ready-made file daily to a private
bucket. Two consequences show up in the rules: the datastore builder
marks the sets it appends with `Type == "promo"`, and Riftbound's `FilterCards`
refuses those printings unless the input edition itself resolves to a promo set
(promos reuse the main sets' collector numbers, so they would otherwise alias
every base card). And because the gallery exports no finish information at all,
every card is registered as available in *both* finishes — matching how
TCGplayer lists Riftbound singles as Normal and Foil.

**Indexes on `Backend`** (`mtgmatcher/backend.go`): `AllSets` and `Sets`
(code → `*Set`); `NormalizedSets` (normalized set name → `*Set`); `UUIDs`
(UUID → `*CardObject`); `CanonicalNames` (normalized → canonical); `Tokens`;
`Hashes` (normalized name → UUID list); `ExternalIdentifiers` (per
`IDSpace`: Scryfall/TCG/etched id → UUID, and the Cardmarket ids One Piece,
Lorcana and Pokemon publish, reachable only through `ConvertID`); `AlternateProps` (flavor
names); the sorted name and sealed-name arrays backing prefix/contains/regexp
search; `SetUUIDs` and `SetSealedUUIDs` (per-set sorted UUID buckets);
`AllPromoTypes`, `SLDDeckNames`, `CommanderKeywordMap`; the partitioned
`AllUUIDs`/`AllSealedUUIDs`; and the unexported `rules GameRules` that a
loader attaches with `SetRules`.

Three of those deserve their own note:

- **`UUIDs` holds pointers.** `map[string]*CardObject`, and `GetUUID` hands
  the pointer straight back. The object is shared with every other caller and
  **must never be modified after the load completes** — both the field and the
  accessor say so in-source. Copy before mutating.
- **`NormalizedSets` is built by the exported `IndexSets()`**, which every
  loader must call once its `Sets` are populated (hence exported: the loaders
  live in other packages). It visits set codes in sorted order so that two
  sets normalizing to the same name resolve deterministically — lowest code
  wins — replacing a linear rescan that followed random map order.
- **`SetUUIDs` is built by the exported `IndexSetUUIDs()`**, its `IndexSets()`
  counterpart: every loader must call it once `UUIDs` and `AllUUIDs` are
  populated. Nothing else fills this bucket — unlike `SetSealedUUIDs`, which
  `AddSealed` builds incrementally as each sealed product is filed, there is
  no per-card add path shared across games, so a loader that never calls
  `IndexSetUUIDs()` leaves `SetUUIDs` permanently nil with no error to show
  for it. That is exactly what happened to eight of the nine game loaders
  before this method existed (§6).

**UUID scheme and finishes.** The source datastore's UUID identifies a
printing; a card that exists in several finishes registers each one
explicitly in `Card.FoilUUIDs`, a finish → UUID map the loaders populate.
Magic keeps the historical suffixes there (`_f` for foil, `_e` for etched, and
split foil printings also carry `★`/`†` number suffixes). The datastore games
key every finish by the name TCGplayer prices it under (`FinishSlug`:
`nonfoil`, `coldfoil`, `1steditionholofoil`), read off the finish the entry
publishes and never off its uuid; `nonfoil` and `foil` also name the printings
a caller's bare flags answer with. `Card.Finish` records that name for the
specific stored UUID, which is what keeps two entries apart when the `Foil`
boolean alone cannot. `mtgmatcher.Finishes` is the table of those names, their
TCGplayer spellings, print runs and foilness; `docs/finishes.md` has the rules
around it.
These UUIDs are first-class: resolve them only via `GetUUID`/`ConvertID`.

**Data model.** `Card`, `CardObject`, `Set` and `SealedProduct` are core types
declared in `mtgmatcher/backend.go`; the MTGJSON-shaped `AllPrintings`
structures are the Magic loader's own, in `mtgmatcher/magic/mtgjson.go`.
`CardObject` is a `Card` plus the resolved `Edition`/`Foil`/`Etched`/`Sealed`.
`Card` carries the MTGJSON field set, including `Legalities map[string]string`
(JSON tag `legalities`, format → legality) — populated only by the Magic
loader and **nil for every other game's** cards, so consumers must handle
that. It also carries the cross-game additions described above:
`FoilUUIDs`, `Finish`, `Images` (at minimum a `"full"` and a `"thumbnail"`
URL) and `PlainNumber` (the collector number as a person writes it, the game's
marks and decorations off; `OriginalNumber` before v0.8.3).

**No compatibility shims.** The Magic promo-type constants live only in
`mtgmatcher/magic`; core keeps no re-declared copies of them. Downstream code
that used to resolve `PromoTypeBoosterfun` and friends from `mtgmatcher`
imports the `magic` package instead. Core cannot import `magic` (it would
cycle), so a shim would have had to duplicate the values rather than alias
them, and a duplicated constant that silently drifts is worse than a build
error that names the symbol.

### 2.2 Normalization (`mtgmatcher/replacer.go`, `mtgmatcher/utils.go`)

`Normalize()` trims, lowercases and runs a single `strings.Replacer` that
deletes spaces, punctuation, brackets, quotes, dashes, invisible formatting
marks and the `/` and `|` separators, drops `" and "`, `" to "` and `" & "`,
folds accented letters to plain ASCII and shortens a run of underscores (a
blank in a card name) to five. Three *protection* entries map a name to
itself so it keeps what the rest would strip (`"waste land"` stays distinct
from "Wasteland", `"trial and error"` from "Trial // Error", and
`"goblin // soldier"` keeps its separator). `Equals`/`Contains`/`HasPrefix`
are normalized comparisons used everywhere. Editing this table can silently
re-alias unrelated cards; **run the full matcher test suite after any change.**

`ExtractNumber()` pulls the first collector number `< 1993` from a string
(1993 separates numbers from years), refusing strings containing month
names (dates) and ordinals (`30th`); it preserves
single-letter suffixes lowercased (`123s` prerelease, `123p` promo pack) and
understands PLST's `SET-123` format. `ExtractYear()` handles `'06`/`M15`
style abbreviations. `SplitVariants()` splits the parenthesized fields stores
use to distinguish printings while protecting legitimately-parenthesized
names such as *Erase (Not the Urza's Legacy One)* and *B.F.M.*; eight of
the nine games' `Prefilter` hooks call it — every one but Pokemon's.

Magic's promo dates live in `mtgmatcher/magic/mtgjson.go`, including
`BuyABoxInExpansionSetsDate` (2018-04) and `PromosForEverybodyYay` (2019-10).
Magic's candidate-set policy consults them; core no longer knows which dates
admit promo siblings. Callers of the former core symbols must import `magic`.
The other date thresholds and `PromoType*`/`FrameEffect*`/`BorderColor*`
vocabulary also remain in that game package.

### 2.3 Input and ID matching

`InputCard` (`mtgmatcher/card.go`) is
`{Id, Name, Variation, Edition, Foil, Language}` plus three flags that are
exported because the per-game rules packages set and read them across the
package boundary: `BeyondBaseSet` and `OriginalName` are internal matcher
state (`json:"-"`), while `PromoWildcard` is part of the serialized input
(`json:"PromoWildcard,omitempty"`). It carries two exported `Is*()` predicates,
`IsFoil()` and `IsEtched()`, built on normalized comparisons. The promo wording
a game reads lives in that game's package, Magic's in
`mtgmatcher/magic/inputcard.go`.

`(b *Backend) MatchID(inputID string, finishes ...bool)`: `finishes[0]` = foil,
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

`Match` is a `Backend` method (`mtgmatcher/mtgmatcher.go`). It owns the
skeleton; game-specific stages are dispatched through
the `GameRules` value the loader attached. `mtgmatcher/rules.go` defines
its 11 methods:

```go
type GameRules interface {
    Prefilter(b *Backend, inCard *InputCard)
    AdjustName(b *Backend, inCard *InputCard)
    AdjustEdition(b *Backend, inCard *InputCard)
    AliasEdition(b *Backend, edition string) string
    FilterPrintings(b *Backend, inCard *InputCard, editions []string) []string
    CandidateSets(b *Backend, inCard *InputCard, editions []string) []string
    FinalizeCandidates(b *Backend, inCard *InputCard, cards []Card) []Card
    FilterCards(b *Backend, inCard *InputCard, cardSet map[string][]Card) []Card
    IsUnsupported(b *Backend, inCard *InputCard, co *CardObject, stage Stage) bool
    IsToken(b *Backend, name string) bool
    PlainNumber(number string) string
}
```

`AliasEdition` spells an edition string the way the datastore names its set,
card-free, for `GetSetByName`'s last resort; `IsToken` names a token by
wording alone, for a game whose checklists and rules tips describe one
without a token type of its own; `PlainNumber` (described above, under "Data
model") names a game's collector-number shorthand. Finish names are no
game's hook: every one is read through `FinishSlug`.

Two contracts matter when implementing it. **Hooks receive the `InputCard` by
pointer and may mutate it**; mutations persist for the rest of the pipeline
*and* are visible to the caller after `Match` returns — the Magic
`AdjustEdition` sets `PromoWildcard` and `BeyondBaseSet` that way, and later
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
2. **Id fast path** — if `Id` is set, `b.MatchID()`; the hit is *validated*.
   A wrong language resets the input to the resolved card's fields and falls
   through to full matching; a token id in a non-default language returns
   `ErrUnsupported`; and `rules.IsUnsupported` at `StageAnswer` rejects
   prerelease / promo-pack / serialized claims the resolved card does not
   carry (upstream tags lag releases), and for Pokemon a metal-card claim.
   This runs before the rules are known non-nil, so the hook call is
   guarded.
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
4. **Unsupported gate** — `rules.IsUnsupported` at `StageWording`, checked
   before name resolution on the storefront's own wording.
5. **Canonicalization** — `CanonicalNames[Normalize(name)]`; on miss,
   `rules.AdjustName` (Magic: typo/token/number fixups and flavor-name
   resolution via `AlternateProps`; Lorcana and Riftbound: a prefix fallback
   for feeds that truncate "Character - Title" names, plus Riftbound's
   champion-first legend remapping) and retry. Final miss → `ErrUnsupported`
   for tokens/oversize, else `ErrCardDoesNotExist`.
6. **Edition adjustment** — `rules.AdjustEdition`. For Magic this is the
   long ladder in `mtgmatcher/magic/rules.go`: `EditionTable`
   aliases ("Alpha" → "Limited Edition Alpha", Universes Beyond names, …),
   variation-implies-edition rules ("Invocation" → Amonkhet Invocations),
   Commander-product parsing, and a terminal `default: switch inCard.Name` of
   per-card fixups. Lorcana and Riftbound instead trim the storefront noise
   ("Disney Lorcana: …", "Riftbound: League of Legends …", a trailing
   "Singles"). Core then applies its own hard `ErrUnsupported` gates for
   custom token sets and most oversize cards, and asks `rules.IsUnsupported`
   again at `StageEdition`, now with the canonical name and adjusted
   edition.
7. **Set selection** — `Printings4Card()`, then `rules.FilterPrintings` when
   more than one printing survives *or* the original name ended in "Token"
   (single-printing token names still need filtering). An empty result is
   `ErrCardNotInEdition`, or `ErrUnsupported` for tokens/oversize. Then a
   call to `rules.CandidateSets` chooses the set codes, which core materializes
   into `cardSet map[setCode][]Card` via `MatchInSet()`.

   `DefaultRules` tries exact normalized edition names, then partial names,
   then all printings. `PromoWildcard` keeps all printings, as used by Gundam
   and One Piece. It applies no Magic promo or World Championship policy.

   Pokemon overrides the loose pass to retain its promo-shelf fallback: when
   generic promo wording names no exact edition, `*Promos` shelves join loose
   edition matches before the fallback to all printings. This prevents an
   unresolved promo edition from resolving straight to an ordinary printing.

   Pokemon's `FilterPrintings` holds a listing saying oversized to the sets
   printing the card oversized at its number, and one naming a metal card to
   the sets holding its metal printing. TCGplayer files both on shelves (Jumbo
   Cards, Miscellaneous Cards & Products) that the card's own set, which is
   what a storefront writes, never reaches.

   Magic owns its historical three passes in `magic/candidates.go`: exact
   edition matches can enroll the `P<code>` promo sibling (or the base set in
   reverse); loose matches admit generic promos and recent bundle/BaB sets;
   and the final fallback admits all printings. Japanese wording suppresses
   sibling expansion, World Championship skips loose matching, and Secret
   Lair or `PromoWildcard` skips both narrowing passes. A lone printing still
   bypasses expansion. Date thresholds retain strict before/after boundaries.
8. **Card-level disambiguation** — `rules.FilterCards`, run
   **unconditionally**. This is a deliberate change from the pre-`GameRules`
   pipeline, which returned a lone candidate without validating it: Lorcana
   enforces the collector number in this hook, and the old shortcut let a
   wrong-numbered card through. Magic preserves the historical behavior
   *inside* its own hook — a single card in a single set is returned as-is,
   so a lone candidate still matches even when the variation carries junk —
   but that is now the game's choice rather than the skeleton's.
   `rules.FinalizeCandidates` then applies final game policy. Magic keeps only
   the first World Championship candidate; other games retain ambiguity.
   This hook runs **before** core's language filter, preserving Magic's
   historical ordering even when the first candidate is in another language.
9. **Verdict** — 0 cards: `ErrCardWrongVariant` (or `ErrCardMissingVariant`
   if no variation was given, `ErrUnsupported` if a language was involved);
   1 card: `output()` plus a final `rules.IsUnsupported` at `StageAnswer`;
   2+: `AliasingError`, whose `Probe()` returns all candidate UUIDs —
   consumers log these as data-quality alarms (and, like `mtgban-website`,
   may pick the newest printing from `Probe()`).

Error taxonomy (`mtgmatcher/utils.go`): `ErrDatastoreEmpty`,
`ErrCardUnknownID`, `ErrCardDoesNotExist`, `ErrCardNotInEdition`,
`ErrCardWrongVariant`, `ErrCardMissingVariant`, `ErrCardWrongFinish`,
`ErrCardUnnamedFinish`, `ErrUnsupported`, and `AliasingError` (`ErrAliasing`
is an empty one). `ErrUnsupported` doubles as a silent-skip channel *and* a
found-but-invalid-promo-tag signal.

### 2.5 The per-game rules packages

Each game package has the same shape: a `Load()` datastore converter, a
`rules.go` holding the `GameRules` implementation, a `register.go` whose
`init()` calls `RegisterGame`, and (all but Pokemon) a golden replay suite
kept inside the package. Lorcana and Palworld stay at three source files;
the other six add a `table.go` and/or a `promolabels.go` (the promo-type
word list a game's storefronts spell out, checked against the published
datastore by `internal/vocabulary`) as their own vocabulary grew, up to
Pokemon's six. Magic needs considerably more (`callbacks.go`, `table.go`,
`wrappers.go`, a `doc.go`, `variants.go`), and its replay suite, `TestMatch`
over `testdata/magic_test_data.json`, is `mtgmatcher/magic/matcher_test.go`
(§2.7). What differs between the games is how much identification logic
each needs.

**Magic** (`mtgmatcher/magic/rules.go`) carries essentially all of it.
`FilterPrintings` eliminates whole sets using the input's promo
predicates against set type, release dates and name patterns — dedicated,
repetitive blocks for prerelease vs promo-pack, release/launch promos, BaB,
bundles, Secret Lair vs Mystery List, WCD, MagicFest, Duel Decks, 30th
Anniversary, judge promos, and a wildcard-promo mode. `FilterCards`
disambiguates within sets, consulting in order:

1. the hand-curated `VariantsTable` (`mtgmatcher/magic/variants.go`, pure
   data: set → card → variant tag → collector number, alongside
   `MultiPromosTable`), with the `EditionTable` aliases beside it in
   `mtgmatcher/magic/editions.go`;
2. `ExtractNumber` with its suffix semantics;
3. for a Secret Lair listing naming a flavor name, the printings sold under
   it; naming neither a number nor a flavor name, the card's own unflavored
   printing when there is exactly one (a number and its ★ foil) — both
   ahead of the passes below, which would otherwise veto the printing for a
   treatment or a finish the listing never mentioned;
4. promo-type validation through the `promoTypeElements` table (each entry:
   tag strings, an optional `TagFunc`, an activation date, wildcard
   eligibility);
5. per-set `simpleFilterCallbacks` / `complexFilterCallbacks` /
   `numberFilterCallbacks` (`mtgmatcher/magic/callbacks.go`) for sets whose
   disambiguation needs real code;
6. per-set promo dedup via `MultiPromosTable`, then finish/frame
   separation — etched (gated on `SeparateFinishCollectorNumberDate`),
   borderless, extended art (gated on `PromosForEverybodyYay`) and showcase —
   each pass discarding its result rather than the candidates when it would
   filter everything away.

This three-tier design — **data tables first, generic number/promo logic
second, per-set code last** — is the Magic package's core maintenance pattern:
most new-set support lands as table entries, not code.

One narrow language case runs just ahead of stage 3: a listing naming the
Phyrexian language keeps only the Phyrexian-language printings when there are
any, since stage 3 and the passes after it read such a printing's flavor name
(the card's own, in Phyrexian script), showcase frame and dual finish as things
the listing left unsaid.

**The other seven** need far less of that, and share a common shape: each
game's `Rules` type embeds `mtgmatcher.DefaultRules` and overrides only the
hooks its own catalog forces. `FinalizeCandidates` and `IsToken` fall
through to `DefaultRules` — a genuine no-op — for all of them, and
`FilterPrintings` for all but Pokemon. A couple override one hook further
where their catalog has a real, narrow case to handle: Lorcana and Yu-Gi-Oh
give `IsUnsupported` real logic at `StageWording` (Lorcana drops
puzzle-insert and cruise-promo products; Yu-Gi-Oh drops the storefront's own
character-art cards, which carry no collector number and no catalog row), and
Pokemon overrides `CandidateSets` to fold `*Promos` shelves into the loose
pass before falling back to every printing. Pokemon also holds oversized and
metal-card listings to those printings in `FilterPrintings` (§2.4, stage 7),
and refuses in `IsUnsupported` at `StageAnswer` a metal-card listing that
answered with a card that is not one. The real, shared work across all
seven is name + collector number + finish narrowing in `FilterCards`, with
the edition breaking ties when it resolves. The interesting details are the
ones each game's own data forces: Lorcana honors the name hash rather than
the edition-keyed `cardSet` values so that case-variant spellings (three real
pairs exist) stay reachable, and it strips leading zeros from numbers while
keeping a genuine `"0"` reachable; Riftbound canonicalizes numbers out of the
public code ("OGN-066a/298" → "66a") and refuses promo sets unless explicitly
targeted; One Piece, Yu-Gi-Oh, Flesh and Blood and Gundam each strip a
different shape of padding and set-code prefix off the ordinal a person
actually types (`PlainNumber`, §2.1's "Data model") rather than the
catalog's own spelling.

### 2.6 Sealed products & search API (`mtgmatcher/api.go`)

Lookups: `GetUUID`, `GetSet`, `GetSetByName`, `GetAllSets`, `GetUUIDs`/
`GetSealedUUIDs`, the per-set `GetUUIDsInSet`/`GetSealedUUIDsInSet` (backed by
the `SetUUIDs`/`SetSealedUUIDs` buckets — the result aliases the index and must
not be modified), `Printings4Card`, `CardReleaseDate`, `ConvertID`,
`AllPromoTypes`, `Names(form NameForm, sealed)` and `NameIsToken`.

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
case and punctuation, so "Mr. 1 (Daz.Bonez)" and "Mr.1 (Daz.Bonez)" are
distinct cards sharing a bucket), and only then falls back to the first
entry, which is the right answer for alias-only buckets such as flavor names.

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

**What exists.** Eight of the nine games have a data-driven golden replay
suite that matches a corpus of real inputs against expected UUIDs or errors,
gated on an environment variable pointing at a real datastore, with a flag
that regenerates expectations after intentional changes — only Pokemon has
none yet:

| Game | Suite | Corpus | Env var | Regenerate |
|---|---|---|---|---|
| Magic | `mtgmatcher/magic/matcher_test.go` | `mtgmatcher/magic/testdata/magic_test_data.json` | `ALLPRINTINGS5_PATH` | `-u` |
| Lorcana | `mtgmatcher/lorcana/matcher_test.go` | `mtgmatcher/lorcana/testdata/lorcana_test_data.json` | `LORCANA_PATH` | `-update-lorcana` |
| Riftbound | `mtgmatcher/riftbound/matcher_test.go` | `mtgmatcher/riftbound/testdata/riftbound_test_data.json` | `RIFTBOUND_PATH` | `-update-riftbound` |
| One Piece | `mtgmatcher/onepiece/matcher_test.go` | `mtgmatcher/onepiece/testdata/onepiece_test_data.json` | `ONEPIECE_PATH` | `-update-onepiece` |
| Yu-Gi-Oh | `mtgmatcher/yugioh/matcher_test.go` | `mtgmatcher/yugioh/testdata/yugioh_test_data.json` | `YUGIOH_PATH` | `-update-yugioh` |
| Flesh and Blood | `mtgmatcher/fleshandblood/matcher_test.go` | `mtgmatcher/fleshandblood/testdata/fleshandblood_test_data.json` | `FLESHANDBLOOD_PATH` | `-update-fleshandblood` |
| Gundam | `mtgmatcher/gundam/matcher_test.go` | `mtgmatcher/gundam/testdata/gundam_test_data.json` | `GUNDAM_PATH` | `-update-gundam` |
| Palworld | `mtgmatcher/palworld/matcher_test.go` | `mtgmatcher/palworld/testdata/palworld_test_data.json` | `PALWORLD_PATH` | `-update-palworld` |
| Pokemon | — none yet — | — | `POKEMON_PATH` | — |

These are the regression harness for the heuristic tables. Run the relevant
regeneration after a *deliberate* matching change and **review the diff** —
never blindly accept it. All seven non-Magic suites additionally carry
hand-authored seed cases whose expected verdicts are baked by the regeneration
rather than hard-coded, with a `"negative:"` description prefix declaring the
author's intent so the regeneration fails loudly when the outcome class
changes.

Every suite behaves alike when its datastore is missing, Magic included:
each loads its backend lazily behind a `sync.Once`-guarded helper and calls
`t.Skip("Need <VAR> set to run this test")` on the tests that need it.
Magic's `TestMain` calls `log.Fatalln` only if its own golden
`testdata/magic_test_data.json` fails to open or parse, a repo integrity
fault rather than a missing-datastore one. Set the relevant `<GAME>_PATH`
variables before running `go test ./mtgmatcher/...` to exercise more than
the datastore-free tests.

Unit tests cover normalization, number/year extraction, variants-table
integrity, the set index, the search surface, `HasPrinting`, CSV write-error
propagation (`mtgban/csv_error_test.go`), the `Add*` family (in
`mtgban/base_test.go`), and `Arbit` and `Mismatch` (in the three
`mtgban/arbit*_test.go` files). `mtgmatcher/rules_test.go` is the one that
pins the pipeline contract: it builds a `Backend` by hand (so `SetRules` was
never called) and asserts that every rules-dependent entry point returns
`ErrDatastoreEmpty` rather than panicking on the nil hooks.

**The pyramid is inverted here.** Card identity is only meaningful against a
large real dataset (AllPrintings alone is a few hundred MB), so the *largest*
test surface — the matcher — is a data-backed integration replay rather than a
unit test, while the most business-critical code, the money path, needs **no**
external data and is tested on synthetic records. The strategy follows from
that asymmetry:

| Layer | Targets | Test type | Needs dataset? | Today |
|-------|---------|-----------|----------------|-------|
| **Money path** (top risk) | `Arbit`, `Mismatch`, `add()` invariants, profitability formula | unit / golden on synthetic records | **No**, runs in CI | 31 tests in `arbit*_test.go`, plus `Add*` and sort in `base_test.go` |
| **Matcher** (data integrity) | `Match`/`MatchID`, normalization, variants/editions, sealed API | data-backed regression replay | **Yes** — one per game | replay + unit |
| **Scraper preprocess** (breadth) | per-store title → `InputCard` → `Match` | table tests on captured fixtures | partial | 19 of 24 packages |

Principles: (1) **the money path is unit-tested**, with in-test records and
no datastore dependency; (2)
**characterization before refactor** — pin `Arbit`/`Mismatch`/`add()` outputs
*before* changing them, and refactor under green; (3) **assert invariants, not
just functions** — the `entries[0] == NM` ordering is a sort side effect that
`Arbit` and the CSV writers depend on, and `TestSort` in `mtgban/base_test.go`
pins it directly; (4) **scrapers:
breadth over depth** — a few fixture table tests for the gnarliest
preprocessors catch the realistic break; `cardmarket` and `cardtrader` have
grown well past table-test fixtures into dozens of narrow, named `*_test.go`
files each. `abugames`, `cardkingdom`, `cardmarket`, `gamenerdz`,
`magiccorner`, `mintcard`, `starcitygames` and `tcgplayer` have a
`preprocess_test.go` to copy the table-test shape from; about a dozen more
packages carry tests of some other shape (a `*_test.go` covering a specific
fix) without one, and `arcanafrisia`, `cardsphere`, `mtgstocks`,
`secretdeskorrigans` and `toamagic` have none.

**CI provisions all nine datastores.** `.github/workflows/ci.yml` runs one
`cache-<game>` job per game. Only `cache-datastore` (Magic) uses the reusable
`cache-file.yml` against a public URL (`vars.DATASTORE_MAGIC`). Every other
game, `cache-lorcana` included, pulls its `.json.xz` from the private
`mtgban-datastore` B2 bucket (built by `datastore-gen`) and caches it under a
key built from the object's own metadata, since B2 serves no HTTP etag. Each
`test-<game>` job then exports only its own game's `<GAME>_PATH`:
`test-magic` runs `go test ./... -v` over the whole tree, and the other eight
run a scoped list: their `mtgmatcher/<game>` suite, a few scraper packages
and `internal/vocabulary`. A data-backed suite outside its game's list skips
green in every job.

---

## 3. Scraper packages

Idealized shape: `NewScraper(b, creds...)`, a `*mtgmatcher.Backend` first,
always, returning a struct that holds `b` alongside unexported fields that
its `register.go` sets from `mtgban.Options` (`logCallback`,
`maxConcurrency`, optional `partner`/`affiliate`), `disableRetail`/
`disableBuylist` set through `SetConfig`, and `inventory`/`buylist` +
`inventoryDate`/`buylistDate`. Most packages' `Load(ctx)` fans out via
`mtgban.WorkerPool`, usually over `retryablehttp` clients; tcgplayer's Magic
scrapers (`index.go`, `tcgplayer.go`, `sealed.go`) hand-roll their own
pools. Of the clients, only the unregistered cardsphere and mtgstocks set
`LinearJitterBackoff`, and mtgseattle `RateLimitLinearJitterBackoff`. A
`preprocess.go` translates store naming into `InputCard` + `b.Match()`,
skipping `ErrUnsupported`, logging `AliasingError`s; results inserted via
the `Add*` family. Every scraper has a tagged `printf` helper
(`x.logCallback("[TAG] "+format, a...)`), and a `mtgban.NewHTTPClient` it
fetches through reports its retries there
(`mtgban.WithHTTPLogCallback(x.printf)`). File
convention: `<store>.go` / `api.go` / `preprocess.go` / optional `sealed.go`
(a *separate* scraper struct with its own `SealedMode` `Info`, holding the
same `b`).

**The preprocess → `b.Match()` contract** (per listing): build an
`InputCard`, call `b.Match()` — or `b.MatchID(externalID, foil, etched)`
when the store exposes a Scryfall or TCGplayer id. Those two are the only
entry points: every scraper's free-text path runs through the same `Match`
pipeline on the backend it was built on, so a matching improvement lands
for all of them at once and no scraper carries a private shortcut, and no
scraper can be told one game while matching against another's datastore —
`b.Game` is the one source of truth for which game a backend
prices. On `ErrUnsupported` silently `continue`; on `AliasingError` log and
`Probe()`; on other errors log with context (many scrapers suppress
known-noisy editions first); then insert with `Add*`. `PriceRatio` is
computed by reading back `inventory[cardId]` before inserting the buylist
row.

### API-based

| Package | Service & auth | Notes |
|---|---|---|
| `tcgplayer` | OAuth via `go-tcgplayer` + cookie-authed marketplace APIs | Largest: Market/Index/Sealed/SYP-list/per-seller scrapers, plus the table-driven single-game pair (see below); SKU map keyed by UUID; TCG Direct modeled as a Vendor with net-after-fees pricing |
| `cardmarket` | `Index` takes no credentials; `Market` and `Sealed` use OAuth 1.0 HMAC-SHA1 (gentle retry) | `Index` is a **Market** (`MarketNames → MKM Low/Trend`, `MetadataOnly`, `Family="MKM"`); `Market` prices live listings and `Sealed` sealed product; EUR→USD; Magic plus seven more games (all but Palworld), each built from an `mtgmatcher.Game` and mapped to Cardmarket's id inside the package |
| `cardtrader` | Bearer token | `Market` (**Market**, 3 seller tiers, `Family="CT"`, `CountryFlag="EU"`); all eight non-Magic games, each built from an `mtgmatcher.Game` and mapped to Card Trader's id inside the package; `Sealed` mirror; bulk upload + cart APIs |
| `cardkingdom` | Public pricelist via `go-cardkingdom` (file/URL-fed, no own client) | Full 4-condition buylist with price ratios; `CreditMultiplier 1.3`; singles + `sealed.go` + `graded.go` are three scrapers |
| `manapool` | Public JSON API | Three scrapers: `Manapool` (aggregate, `MatchID` by Scryfall id, `NoQuantityInventory`), `Index` (`MetadataOnly`) and `Sealed` |
| `merlion` | Public buylist CSV | Riftbound buylist-only; matches by the TCGplayer id in each product link, and reads the finish out of the condition column |
| `arcanafrisia` | Public buylist endpoint | Buylist-only EU vendor, shorthand `AF`; matches by Scryfall id and maps the store's NM/EX/GD grades onto NM/SP/MP |
| `cardsphere` | Session cookie (gentle 3s) | Unregistered, not run by bantool; buylist-only; `BuyPrice ×0.87` fee **and** `CreditMultiplier 1.1` |
| `mtgstocks` | Public API, fixed `curl` User-Agent | Unregistered, not run by bantool; MetadataOnly index (average/market interests) |

**The tcgplayer single-game scrapers.** `TCGGame` (retail) and `TCGGameIndex`
(index pricing) in `tcgplayer/game.go` and `gameindex.go` are built from one
table:

```go
var tcgGames = map[mtgmatcher.Game]int{
    mtgmatcher.GameLorcana:       tcgplayer.CategoryLorcana,
    mtgmatcher.GameRiftbound:     tcgplayer.CategoryRiftbound,
    mtgmatcher.GameOnePiece:      tcgplayer.CategoryOnePiece,
    mtgmatcher.GameYuGiOh:        tcgplayer.CategoryYuGiOh,
    mtgmatcher.GameFleshAndBlood: tcgplayer.CategoryFleshAndBlood,
    mtgmatcher.GamePokemon:       tcgplayer.CategoryPokemon,
    mtgmatcher.GameGundam:        tcgplayer.CategoryGundam,
    mtgmatcher.GamePalworld:      tcgplayer.CategoryPalworld,
}
```

Every non-Magic game is in the table today; supporting a tenth would be one
more entry here, provided the matcher has a datastore for it. Magic is
deliberately absent: it is identified by SKU and has its own scrapers. Both
game scrapers pass the printing name through the `InputCard.Variation`
field alongside the collector number, so the game's rules can tell foil
sub-types apart (this is what makes Lorcana's "Holofoil" convention
resolvable — see §2.3).

### HTML / crawler

`starcitygames` (HawkSearch/Meilisearch APIs, serialized detection, sealed,
plus Lorcana, Riftbound and Flesh and Blood, which its `scgGames` table maps
onto SCG's numeric game ids), `coolstuffinc` (seven of the eight non-Magic
games — every one but Flesh and Blood — `CreditMultiplier 1.25`),
`hareruya` (JPY, **bespoke 403 → 5-min backoff**), `magiccorner` (EUR,
Italian), `abugames` (Solr, MINT-aware grading, `InfoForScraper`), `mtgseattle`
(`CreditMultiplier 1.33`), `mintcard` (rides TCG SKUs,
`CreditMultiplier 1.1`), `vegassingles`, `manaleak` (GBP),
`miniaturemarket` (sealed-only),
and the unregistered `secretdeskorrigans` (CAD, French) and `toamagic`
(Spanish), which bantool does not run.

`sealedev` builds sealed-EV "scrapers" from mtgmatcher probabilities or
5,000-run booster simulations priced against the MTGBAN API, emitting EV
entries with dispersion stats (std-dev/IQR), `Family="EV"`, `SealedMode`, and
`MetadataOnly` toggled per sub-scraper. Two of its sources are estimated
rather than read: TCG Direct (net) where the buylist is missing, and
Cardmarket for the ~86% of the catalog its market scraper never polls, which
`sealedev/mkm.go` scales from the published guide's Trend column using a
calibration fitted from the same snapshot being priced. That fit is per game
and not portable — Magic's multiplier runs 0.43-1.24 across the price range
where Yu-Gi-Oh's runs 2.2-3.7 — and Pokemon and Yu-Gi-Oh do not estimate well
enough to publish, for reasons upstream of the scaling.

**Not templates.** Some directories in a working tree are untracked WIP and
are not part of the committed module — `synthetic/` (a *computed* buylist with
no site behind it, synthesizing prices from TCG/CK/SCG, `MetadataOnly`) and
`mvpsportsandgames/`, whose non-conforming `Inventory() (record, error)` does
**not** satisfy `mtgban.Seller`. For new work, copy `starcitygames` (API) or
`mtgseattle` (HTML) — never an untracked orphan.

---

## 4. Tooling — `cmd/` and CI

Committed tools: **bantool** (the production orchestrator), **boosterGen**,
**boosterList**, **manapoolOrders**, and **tcgid4scryfall**
(TCG id → Scryfall id export). A long tail of further tools exists only as
untracked working-tree WIP (`manapoolSeller`, `mkmhtml2csv`, `mp2ckbl`,
`amazonsearch`, `omnitool-3g`, `autocart`, and the `ck*`/`ct*`/`mkm*` family);
treat anything not in the list above as unreviewed, and note that some of it
embeds live credentials.

- **bantool** — a table of `scraperOption{flags}`, one per target, derived
  from the scraper registry rather than written out by hand: `targets()` in
  `cmd/bantool/main.go` walks `mtgmatcher.AllGames` × `mtgban.Registered(game)`,
  each scraper package having filed its own keys with `mtgban`'s registry
  from `init`, so bantool builds no scrapers of its own and an entry is a
  name and its flags and nothing else. It runs well past a hundred targets
  once every scraper's own per-game and `_sealed` variants are counted (14
  `*_riftbound` targets, 14 `*_lorcana` ones, and the rest of the eight
  non-Magic games besides). The table is a
  `map[mtgmatcher.Game]map[string]*scraperOption`: the game is the outer key and
  the store's own name the inner one, so a target's game is which sub-map
  holds it rather than something composed into or re-derived from its name
  at runtime. A required `-game` selects the sub-map; `-store` (plus
  `-sellers`/`-vendors` for a single-half run) takes a comma list of the
  bare names that sub-map holds, so a store is never told which game it
  belongs to twice. `resolveGame` and `enableStore` refuse an unknown game
  or store up front, listing what is actually registered, rather than
  leaving `mtgban.NewScraper` to fail deeper in. A target whose entry
  already answers for one half alone is refused rather than overwritten
  when `-sellers`/`-vendors` names it too. A target and the `game` input of
  the workflow scheduling it are pinned against each other by
  `cmd/bantool/workflows_test.go`, one `bantool-<game>-<store>.yml` file per
  target (`-store tcg_market`, which is what `run-bantool.yml` invokes).
  `-format` json/csv/ndjson
  (each also with an `.xz` variant); output through
  `github.com/mtgban/simplecloud` to local/B2/GCS/S3/HTTP; optional HMAC
  signing (`BAN_SECRET`); all credentials via env vars (godotenv autoload).
  It blank-imports `mtgmatcher/games`, which is what lets `-datastore` accept
  a file for any of the nine games without further configuration. The log
  callback is one option among the rest: `scraperOptions` puts
  `mtgban.WithLogCallback(log.Printf)` on every target it builds, alongside
  `WithAuthenticator`, the concurrency cap, the half a target is held to and
  the partner code its key's family reads from the environment. Each
  registered constructor in a package's own `register.go` then reads what
  it needs out of `mtgban.Options` (`opts.LogCallback`, `opts.Affiliate`,
  `opts.TargetEdition`, `opts.MaxConcurrency`, `opts.Secret`) and sets the
  scraper's unexported fields from it, `scraper.logCallback = opts.LogCallback`
  alone being 37 assignments across the 20 `register.go` files.
- **manapoolOrders** — Mana Pool buyer-order CSV dumps.
- **boosterGen / boosterList** — booster simulation and sealed introspection
  over the mtgmatcher sealed API.
- **tcgid4scryfall** — TCGplayer id → Scryfall id mapping export.

**CI** (`.github/workflows/`). `ci.yml` runs a datastore-free style job on its
own — **Check formatting** (`gofmt -s -l .`), **Vet** (`go vet ./...`),
`go test -race ./...`, **revive** and **staticcheck** — alongside nine
per-game jobs that each restore their own datastore from cache (§2.7);
Magic's job runs `go test ./... -v` over the whole tree, and the other eight
each run a scoped list: their `mtgmatcher/<game>` suite, a few scraper
packages and `internal/vocabulary`. Nightly, `race-magic` reruns the whole
tree under `-race` with the Magic datastore, one package at a time, and writes
each package's coverage to the run's summary. There are well past a hundred
`bantool-<game>-<store>.yml` files, one per scraper target, such as
`bantool-lorcana-cardmarket.yml` and `bantool-riftbound-tcg_market.yml`,
each triggered by cron plus
`workflow_dispatch`/`repository_dispatch`, and each delegating to the reusable
`run-bantool.yml` with `store`, `game` and `datastore-filepath` inputs.
A dispatch of `<game>-<store>` reruns one store, the type the site sends,
and `<game>-all` runs every store of that game in one call, e.g. `gh api
repos/mtgban/go-mtgban/dispatches -f event_type=lorcana-all`.
`.github/workflows/README.md` describes the workflows in full.
`run-bantool.yml` uploads to `b2://mtgban-dumps/<game>/<store>` and then pings
a signed `http://<game>.mtgban.com/api/load/<store>` URL so the server reloads
the fresh snapshot. Magic targets prepend a `cache-datastore` job and pass a
cached local path; every other game's targets skip caching entirely and pass
a `b2://` path that bantool reads directly (which is why they need the
datastore B2 keys). Cron cadence and concurrency grouping are set per
scraper target, not by a Magic-vs-everyone-else rule — `cardmarket`'s own
targets, Magic's included, run twice daily under a `queue: max` concurrency
group regardless of game, and other scrapers use their own schedule. No
Makefile or Docker — plain `go build` per `cmd/` subdirectory.

**Key dependencies**: goquery (HTML), retryablehttp + cleanhttp
(HTTP), simplecloud (storage abstraction), weightedrand (boosters),
montanaflynn/stats (EV), golang.org/x/text (normalization), plus the in-house
`go-cardkingdom`, `go-cardmarket` and `go-tcgplayer` clients.

---

## 5. Design through-lines

> The load-bearing decisions below are recorded as ADRs in
> [`docs/adr/`](docs/adr/) with full context and alternatives: UUID-as-key
> (ADR-0001), the original global backend (ADR-0002), atomic snapshot
> publication (ADR-0003, superseding ADR-0002), and the localized matcher and
> scraper registry (ADR-0004, superseding both).

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
   behavior across scrapers via `WorkerPool` and injected log callbacks.
8. **No global, no publication** — loaders build independent backends and
   hand them to callers directly; there is nothing ambient to swap and
   nothing to publish. Several related operations capture a backend
   explicitly and pass it through; its nested maps, slices and pointers
   must remain immutable once built, since every scraper and report built
   on it shares them. See ADR-0004.

## 6. Extending the system

**Adding a store**: create a package with the four-file layout, its scraper
holding the `*mtgmatcher.Backend` it is built on; implement `Seller` and/or
`Vendor` (and `Market`/`Trader` if it has sub-sellers), fetch with
`WorkerPool` + retryablehttp, write a `preprocess.go` that builds
`InputCard`s and calls `b.Match`/`b.MatchID`, handling the store's naming
quirks. Add a `register.go` whose `init()` calls `mtgban.Register(name,
games, constructor)` under the key name bantool has always used for the
target and every game the scraper prices, then blank-import the package in
`cmd/bantool/main.go` — `targets()` walks `mtgmatcher.AllGames` ×
`mtgban.Registered(game)`, so a registered scraper is immediately a valid
`-store` value for its games, with no entry to write by hand. Add the
matching GitHub Actions workflow: `cmd/bantool/workflows_test.go`'s
`TestEveryTargetIsScheduledByItsOwnWorkflow` enforces the pairing in both
directions, failing the build if a registered target has no
`bantool-<game>-<store>.yml` workflow scheduling it, or a workflow names a
target that is not registered. It also checks that each file's
`repository_dispatch` types are exactly `<game>-<store>, <game>-all`. Set the
right `ScraperInfo` flags (`MetadataOnly`, `NoQuantityInventory`,
`SealedMode`, `CreditMultiplier`, `Family`, `Game`). The hard part is always
preprocessing — which is why mtgmatcher's typed errors, variant tables, and
per-set callbacks exist.

**Adding a game**: create `mtgmatcher/<game>/` with a `Load(io.Reader)
(*mtgmatcher.Backend, error)` that converts the source data (populating
`Sets`, `UUIDs`, `AllUUIDs`, `AllSets`, `Hashes`, `CanonicalNames`,
`ExternalIdentifiers` and `FoilUUIDs`, filing sealed products through
`AddSealed`/`SortSealed`, then calling `IndexSets()` and `IndexSetUUIDs()`
once `UUIDs`/`AllUUIDs` are in and `SetRules()`), a `rules.go` implementing
`GameRules`, a `register.go` whose `init()` calls `RegisterGame`, and a
replay suite gated on a `<GAME>_PATH` environment variable with a
regeneration flag. **Every `Backend` field a loader is responsible for
needs an explicit line setting it — there is no default that makes a zero
map or slice merely "smaller"; a nil `SetUUIDs` looks identical to an empty
one until a caller reads it and gets nothing back.** `IndexSetUUIDs()` was
added to close exactly this gap: eight of the nine loaders built `AllUUIDs`
and `UUIDs` correctly but had nothing to call, so `SetUUIDs` stayed nil and
`GetUUIDsInSet` silently answered empty for every set of every game but
Magic, whose loader alone built the same bucketing by hand — which
mtgban-website's edition-only searches (`s:CODE`, seeded from that index
alone when there is no text to search) read as "no results" rather than
"index not built," for every non-Magic deployment, until a test that
actually loads a real datastore (not a hand-built fixture standing in for
one) caught it. When you add a new field to `Backend` that an index or a
lookup depends on,
grep every game package for the sibling field it is meant to travel with
(`AllUUIDs`, `SetSealedUUIDs`, …) and confirm each one sets the new field
too, or add a shared setter every loader calls (`IndexSets`,
`IndexSetUUIDs`) rather than trusting nine separate hand-written loops to
stay in sync. Add the game to `mtgmatcher/games`, add a `Game` constant in
`mtgmatcher/game.go` and list it in `mtgmatcher.AllGames`, and make `Load`
refuse anything that is not its game's datastore envelope. Existing storefronts
often come cheaply: a TCGplayer category is one entry in `tcgplayer`'s
`tcgGames`, and cardmarket / cardtrader / coolstuffinc / starcitygames each
need one constant naming the storefront's own spelling plus one line in their
`map[mtgmatcher.Game]<vendor value>`. See the *Adding a game* checklist in
`AGENTS.md` for the bantool options, workflows and CI jobs that go with it.

---

## 7. Primary consumer — `mtgban-website` (usage reference)

The reference embedder demonstrates the intended production topology:
**scraping and serving are decoupled.** bantool scrapes and uploads per-store
`Seller`/`Vendor` JSON; the website loads those snapshots and the matcher
datastore, and never runs scrapers in-process. Canonical patterns:

- **Activate a game, then load the datastore once at startup** — the matcher
  can match nothing until a game package is linked in, so blank-import
  `mtgmatcher/games` (or just the games you serve) and then call
  `mtgmatcher.Open(game, reader)` streamed from a `simplecloud` bucket and
  keep the `*Backend` it returns: every lookup is a method on it, and a
  scraper is built on it with `mtgban.NewScraper`. Replacing the datastore
  is loading another backend; the old one stays valid for whatever still
  holds it, and must not be mutated (§2.1).
- **Consume pre-scraped JSON** — `mtgban.ReadSellerFromJSON` /
  `ReadVendorFromJSON` per `game/name/kind/shorthand`. The live sets sit
  behind `atomic.Pointer[[]mtgban.Seller]` / `[[]mtgban.Vendor]` for lock-free
  reads with single-writer publish — the correct concurrency pattern for a
  long-running server over swappable snapshots.
- **Build a store from records** — `mtgban.NewSellerFromInventory` /
  `NewVendorFromBuylist` when you already hold an `InventoryRecord` /
  `BuylistRecord`.
- **Match user input → UUID, handle aliasing** — `b.Match(&InputCard)` on the
  backend that holds the game, with `errors.As(err, &AliasingError)` →
  `Probe()` to pick the newest printing (the upload flow);
  `b.MatchID(scryfallID, foil, etched)` for the external-id fast path.
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
- **CSV export**: `mtgban.WriteBuylistToCSV(b, records, creditMultiplier, w)`
  straight to an HTTP writer.
