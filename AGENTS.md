# AGENTS.md

Guidance for AI coding agents working on **go-mtgban**, a trading-card
market-data platform: scrape store inventories and buylists, normalize every
listing to a canonical card identity, then compute arbitrage across stores.
Nine games are supported today — Magic: The Gathering, Lorcana, Riftbound,
One Piece, Yu-Gi-Oh, Flesh and Blood, Pokemon, Gundam, and Palworld. Read
`SPECIFICATIONS.md` for the full architecture and `docs/adr/` for the
reasoning behind the load-bearing decisions.

## The one rule that matters

**Everything keys on the `mtgmatcher` UUID.** Scrapers are thin translators;
correctness lives in `mtgmatcher`. If a card matches wrong, fix it in
`mtgmatcher` — usually in a data table, and for Magic inside the
`mtgmatcher/magic` sub-package — not in the scraper.

The exception is a vendor's own identifier being wrong, or shared between
two genuinely different products — that's the vendor's data to distrust,
which makes it the scraper's fix, not `mtgmatcher`'s.
`starcitygames/README.md` works through that whole defect class end to end:
how it's found, the fix shapes (a general rule, a closed table, a
refusal), and the repeatable method for catching the next one.

The opposite exception is a vendor that publishes **no** identifier at
all: Cool Stuff Inc's buylist and retail feeds carry none, for any of the
eight games it prices, so every resolution is wording- and shelf-driven
and the shelf itself is sometimes wrong. `coolstuffinc/README.md` names
the recurring shape found three times independently across three
games — a promo shelf holding a printing that collides with a real set's,
told apart only by a note or bracket the code has to read — and where it
has, and hasn't yet, been fixed.

`cardmarket/README.md` is a different kind of read: not a defect class,
but the design of `Market` (the live-listing singles scraper built
alongside `Sealed`'s existing one) and the investigation trail behind
it — a per-token concurrency limit measured directly rather than assumed,
listings that turned out not to be strictly price-ascending once replayed
against real data, and an offline pre-filter whose price snapshot is
published one host per game, not shared, which the loader originally got
wrong in a way that read as "nothing passed the filter" rather than "the
fetch hit the wrong server." Read it before touching any of the three
Cardmarket scrapers or the `resolver` they share.

`sealedev/README.md` covers the one "scraper" with no site behind it: it
prices sealed product by what opening it is worth, against the MTGBAN price
API rather than a storefront. Read it before changing anything an EV reads,
because two of its price sources are *estimated* rather than quoted — TCG
Direct (net) where the buylist has no number or an implausible one, and
Cardmarket for the ~86% of the catalog its market scraper never polls — so
an EV can move between runs without any price having moved. It also names
what silently never counts toward a total (serialized, cosmic foil,
unfixed Secret Lair bonuses, anything under the bulk threshold) and why
that floor is deliberately the same for every source.

## Layout

```
mtgban/                    interfaces (Scraper/Seller/Vendor), records,
                           Arbit/Mismatch, CSV I/O, WorkerPool
mtgmatcher/                game-agnostic core: Backend, b.Match/b.MatchID,
                           the GameRules seam, the game registry, the
                           search API, the shared EditionTable/
                           VariantsTable, and the Magic replay suite
mtgmatcher/magic/          Magic rules, MTGJSON loader, promo/frame vocabulary
mtgmatcher/lorcana/        Lorcana rules, loader, replay corpus
mtgmatcher/riftbound/      Riftbound rules, loader, replay corpus
mtgmatcher/fleshandblood/  Flesh and Blood rules, loader, replay corpus
mtgmatcher/gundam/         Gundam rules, loader, replay corpus
mtgmatcher/onepiece/       One Piece rules, loader, replay corpus
mtgmatcher/palworld/       Palworld rules, loader, replay corpus
mtgmatcher/pokemon/        Pokemon rules, loader (no replay corpus yet)
mtgmatcher/yugioh/         Yu-Gi-Oh rules, loader, replay corpus
mtgmatcher/games/          meta-package that blank-imports all nine games
<store>/                   one package per store (tcgplayer, cardkingdom,
                           cardmarket, ...)
cmd/                       tools; cmd/bantool is the production orchestrator
```

Magic's edition aliases, variants, replay corpus and promo dates live in
`mtgmatcher/magic`. Its `CandidateSets` hook owns promo sibling expansion,
Secret Lair widening and edition-selection policy; `FinalizeCandidates` owns
the World Championship trim. Core materializes the selected sets and applies
language filtering after the final game hook.

The dependency runs one way: a game package imports core `mtgmatcher`, and no
non-test file in core imports a game package. That one-directional rule is
what keeps the core game-agnostic, and it is worth remembering before reaching
for a Magic symbol from core — the import would cycle. The Magic replay suite
gets away with importing `magic` only because it is an external test package
(`package mtgmatcher_test`), which is outside the import graph of the library
itself.

## Build, test, format

```sh
go build ./...              # must stay green
gofmt -l .                  # must print nothing
go vet ./...
go test ./... -v
```

Run all four before committing. CI (`.github/workflows/ci.yml`) runs the last
three — there is no separate build step, but vet and test compile the whole
module anyway — and the formatting check is a hard gate: the job lists the
offending files and exits non-zero, so a stray unformatted file fails the
build rather than merely drawing a review comment. The tree is gofmt-clean
today; keep it that way.

Do not narrow the test or vet invocation to a subset of packages. Tests live
in `mtgban/`, in `mtgmatcher/` and every one of its nine `mtgmatcher/<game>`
sub-packages, and in about a third of the scraper packages (`abugames`,
`cardkingdom`, `cardmarket`, `cardtrader`, `coolstuffinc`, `gamenerdz`,
`hareruya`, `starcitygames`, `tcgplayer`, and others — check for a `*_test.go`
file before assuming a package has none). A subset run can pass while CI
fails. `mtgmatcher/magic` carries the largest test suite of any package in
the repo (two dozen `*_test.go` files); the core Magic *replay* suite is a
separate thing, still living in `mtgmatcher`'s own test package (see "The
golden suites" below).

### Datastores

The matcher suites exercise real data, one datastore per game, located through
environment variables:

- `ALLPRINTINGS5_PATH` — MTGJSON `AllPrintings5.json`. Feeds the core
  `mtgmatcher` suite and `mtgmatcher/magic`, both of which load it through
  `magic.Load`, as well as every scraper suite that needs a real Magic
  datastore to test its `preprocess.go` (abugames, cardkingdom, cardmarket,
  cardtrader, gamenerdz, hareruya, magiccorner, manapool, mintcard,
  sealedev, starcitygames, tcgplayer, vegassingles — grep a package for the
  literal skip message before assuming it is or is not among them). Magic
  is the one game whose datastore is not built by `datastore-gen`.
- `LORCANA_PATH`, `RIFTBOUND_PATH`, `ONEPIECE_PATH`, `YUGIOH_PATH`,
  `FLESHANDBLOOD_PATH`, `POKEMON_PATH`, `GUNDAM_PATH`, and `PALWORLD_PATH` —
  one datastore per remaining game, all built by
  `github.com/mtgban/datastore-gen`. Each feeds its matching
  `mtgmatcher/<game>` suite (`mtgmatcher/lorcana`, `mtgmatcher/riftbound`,
  and so on).

There is no fail-fast/skip asymmetry between games any more — there used to
be, when only Magic, Lorcana and Riftbound existed, and older prose (this
file's own history included) still describes Magic's `TestMain` calling
`log.Fatalln` and taking the whole binary down when `ALLPRINTINGS5_PATH` is
unset. That call was removed. Every suite, Magic and core `mtgmatcher`
included, now loads its datastore lazily behind a `sync.Once`-guarded
`realDatastore(t)` helper and calls `t.Skip("Need <VAR> set to run this
test")` on the tests that need it, so a contributor missing every one of the
nine datastores still gets a green, if much thinner, `go test ./...` run.
`mtgmatcher/magic`'s `TestMain` still calls `log.Fatalln`, but only if its own
golden `testdata/magic_test_data.json` fails to open or parse — a repo
integrity fault, not a missing-env-var one.

Use absolute paths for every variable. A relative path is resolved against
the directory of the package under test, so a single relative value cannot
serve suites that sit at different depths in the tree — and they do:
`ALLPRINTINGS5_PATH` is read from `mtgmatcher` one level down and from
`mtgmatcher/magic` two. CI passes every one as an absolute path, per game,
for that reason.

**Locally**, `go test` does not source `.env` on its own — only
`cmd/bantool` does, via a blank `godotenv/autoload` import in its own
`main.go`. Export the variables yourself before running a suite that needs
one, e.g. `set -a; . .env; set +a; go test ./mtgmatcher/...`, or
`source .env && go test ./...` if your shell's `source` does the same. CI
needs none of this: it exports every `<GAME>_PATH` directly as job env
(below), so this is a local-checkout step only.

CI restores all nine datastores from `actions/cache` before testing, one
`cache-<game>` job per game. Only Magic has a public URL: `cache-datastore`
calls the reusable `cache-file.yml` with `vars.DATASTORE_MAGIC`. Every other
game — Lorcana included, which used to be the other public-URL exception —
is pulled from the private `mtgban-datastore` B2 bucket and cached under a
key built from the object's own metadata, since B2 serves no HTTP etag.

`internal/vocabulary` runs two checks across the datastore-gen boundary, each
gated on the `<GAME>_PATH` variables above and skipping the games whose
variable is unset. `TestLoadersReadWhatIsPublished` holds every loader's
promo-type word table to what the published datastore actually states, so a
loader that folds or drops a word cannot drift unnoticed.
`TestReplayCatalogNames` matches every product name in a `REPLAY_CATALOG`
TCGplayer catalog dump and writes the verdicts to `REPLAY_OUT`, for diffing
two checkouts against each other rather than against a fixed assertion.

### The golden suites

Eight of the nine games own a replay corpus of *(input card → expected
verdict)* pairs under their own `testdata/` directory, each with its own
regeneration flag; only Pokemon has none yet. The flag belongs to the test
binary, not to `go test`, so the package has to come first — `go test -u
./mtgmatcher/magic/` fails with "no Go files in" the repository root, having
read `-u` as a `go` flag and dropped the path.

```sh
# mtgmatcher/magic     -> testdata/magic_test_data.json
go test ./mtgmatcher/magic/ -run TestMatch -u

# every other game     -> testdata/<game>_test_data.json
go test ./mtgmatcher/lorcana/     -update-lorcana
go test ./mtgmatcher/riftbound/   -update-riftbound
go test ./mtgmatcher/onepiece/    -update-onepiece
go test ./mtgmatcher/yugioh/      -update-yugioh
go test ./mtgmatcher/fleshandblood/ -update-fleshandblood
go test ./mtgmatcher/gundam/      -update-gundam
go test ./mtgmatcher/palworld/    -update-palworld
```

The Magic regeneration flag is the odd one out: it is the bare `-u` it has
always been, rather than `-update-magic`. Every other game follows
`-update-<game>` exactly.

Regenerate only after an *intentional* matching change, and read the
resulting diff line by line. All seven non-Magic regenerators carry two
extra safety nets Magic's lacks: they refuse to flip a case between success
and error (a verdict-class flip fails the test and leaves the golden file
untouched), and they re-insert their own hand-authored seed cases — each
package has its own `<game>Seeds`, e.g. `lorcanaSeeds`, `riftboundSeeds` —
which pin the contract edges the sampled corpus cannot reach. The Magic
regenerator has neither guard; it silently rewrites the expected uuid of any
case that now resolves differently and trusts you to read the diff.

**The Magic corpus is an invariant, not a scoreboard.** Making the matcher
game-agnostic was meant to preserve pre-refactor Magic behavior exactly,
quirks included — `magic.Rules.FilterCards` still short-circuits a lone
candidate specifically to preserve the historical behavior of the
pre-`GameRules` pipeline. Refactoring must therefore leave
`magic/testdata/magic_test_data.json` byte-identical. If a change forces a
regeneration,
Magic matching has drifted: stop and find the cause. Do not accept the diff
as a new baseline.

## Conventions

- **gofmt always.** CI enforces it; `gofmt -l .` must print nothing.
- **No global loggers.** Each scraper takes a
  `LogCallback mtgban.LogCallbackFunc` and logs through a tagged `printf`
  helper (`[TAG] `-prefixed). The matcher logs through the `Logger` on the
  `Backend` it is asked through (`b.Logf`), and is quiet while that is nil.
- **Insert via the `Add*` family**, never by appending to the map directly.
  `Add`/`AddRelaxed`/`AddStrict`/`AddUnique` enforce defaults (NM, qty 1),
  validate conditions against `FullGradeTags`, merge duplicates, and keep each
  slice sorted. `Arbit` depends on that sort leaving the NM entry at
  `entries[0]`. `AddUnique` is the strictest gate and **ignores price**.
- **`ErrUnsupported` is a silent-skip signal**, not a failure. In a scraper's
  preprocess loop: skip `ErrUnsupported`, but log `AliasingError` (call
  `.Probe()` to dump the candidates) and every other error as a data-quality
  alarm.
- **Concurrency = `mtgban.WorkerPool`.** New fetch code uses it; do not
  hand-roll goroutine/channel pools. Cancelling the context stops dispatch but
  lets in-flight workers finish.

## mtgmatcher: the game seam

### GameRules

`Match()` is one pipeline shared by every game. The steps that differ per game
are dispatched through the `GameRules` interface in `mtgmatcher/rules.go`, 14
methods in all: `Prefilter`, `AdjustName`, `AdjustEdition`, `AliasEdition`,
`FilterPrintings`, `CandidateSets`, `FinalizeCandidates`, `FilterCards`,
`IsUnsupported`, `IsSpecificUnsupported`, `MissingPromoTag`, `IsToken`,
`CanonicalFinish`, and `PlainNumber` (read `rules.go` itself — each method
carries a paragraph explaining what it owns and why). A game's loader
attaches its implementation with `Backend.SetRules` when it builds the
`Backend`; a `Backend` that never got rules returns `ErrDatastoreEmpty` from
`Match` rather than panicking.

Two properties of the seam matter when writing a hook. First, core retains
language handling and shared token/oversize admission, but games own candidate
edition selection and final ambiguity policy. `DefaultRules.CandidateSets`
tries exact names, partial names, then all editions; `PromoWildcard` requests
all editions. Magic overrides this policy and trims World Championship cards
before the core language filter, preserving its historical ordering. Second,
hooks receive `InputCard` by pointer and may mutate it; those mutations persist
through the rest of the pipeline and remain visible to the caller.

`FilterCards` receives a map, which Go iterates in random order. An
implementation is responsible for producing deterministic output when more
than one candidate survives, because that slice feeds the user-visible
aliasing diagnostics.

### Games register themselves

Each game package ships `rules.go` (its `GameRules` implementation), a
`Load(io.Reader) (*mtgmatcher.Backend, error)` function, and a `register.go`
whose `init()` calls `mtgmatcher.RegisterGame(name, Load)` — the
`database/sql` driver idiom — under the name of its own package: `magic`,
`lorcana`, `riftbound`, `onepiece`, `yugioh`, `fleshandblood`, `pokemon`,
`gundam`, and `palworld`.

The consequence is that **a game only exists if something imports it**. A
consumer blank-imports the games it needs, or blank-imports
`mtgmatcher/games` to get all of them (`cmd/bantool` does the latter); asking
`Open` for a game nothing registered is an error naming the games that are.

### Datastore loading

`mtgmatcher.Open(name, reader)` loads the named game's datastore: it runs
exactly one loader and hands back the `*Backend`, stamped with the game it
was loaded as (`b.Game`). There is no global datastore: every lookup is a
method on the backend a caller holds, a scraper is built on one
(`mtgban.NewScraper(b, name, opts...)`) and matches against it alone,
and a test builds the backend it needs and passes it. `RegisteredGames()`
lists what is currently linked in. `internal/datastore.Read(game, path)` is
`Open` over a path that may be a file, an `http(s)://` URL or a `b2://`
object, `.xz` or not; a suite reads its game's file that way once, in a
helper that hands the backend to each test.

There is no auto-detection. The caller always knows the game — bantool reads
it off the registry key its target sits under, a test off the package it sits
in — and the loader that tried every registered game in turn decoded
AllPrintings three times over before reaching Magic's, behind a buffer of the
whole file.

`Backend` is exported and every lookup is one of its methods (`b.Match`,
`b.GetUUID`, `b.GetSetByName`, ...); there are no package-level functions
of the same name, and no backend the package keeps for you.

### Tables before code

New-set and new-promo support is almost always **data**, not logic:

- edition name aliases → `mtgmatcher/editions.go` (`EditionTable`), still
  core-level and shared;
- card↔number disambiguation → `mtgmatcher/magic/variants.go`
  (`VariantsTable`), which scrapers reach as `magic.VariantsTable`;
- Magic promo detection rules → `mtgmatcher/magic/callbacks.go`
  (`promoTypeElements`);
- per-set special cases that genuinely need code → the filter callbacks in
  `magic/callbacks.go`, or the load-time patch tables in `magic/mtgjson.go`
  (the `switch set.Code` blocks) together with the companion data in
  `magic/table.go`.

Adding a `switch` case to a `Rules` hook is the last resort. The terminal
per-card `switch inCard.Name` at the end of `magic.Rules.AdjustEdition` is
itself a standing "tables before code" violation; do not grow it without a
good reason. Read the `Match` pipeline in `mtgmatcher/mtgmatcher.go`, and the
ordering notes in `SPECIFICATIONS.md`, before editing any stage.

Every non-Magic game needs far less of this, and shares the shape: each
embeds `mtgmatcher.DefaultRules` (`Rules struct{ DefaultRules }`) and
overrides only what it actually needs different. All eight rely on
`DefaultRules` — a real no-op — for
`FilterPrintings`, `FinalizeCandidates`, `MissingPromoTag` and `IsToken`, and
all eight implement their own `Prefilter`, `AdjustName`, `AdjustEdition`,
`AliasEdition`, `FilterCards`, `CanonicalFinish` and `PlainNumber`, which is
where a game's actual vocabulary — its editions, its number shapes, its
finish names — lives. A few games additionally override one hook for a
narrow, real check: Lorcana and Yu-Gi-Oh override `IsUnsupported` (Lorcana
drops puzzle-insert and cruise-promo products; Yu-Gi-Oh drops storefront
character-art cards that carry no collector number), and Pokemon overrides
`CandidateSets` to fold `*Promos` shelves into its loose-edition pass. The
real, shared work across all eight is edition and number normalization in
`Prefilter`/`AdjustName`/`AdjustEdition`/`AliasEdition` and the
number-and-finish disambiguation in `FilterCards`.

### Search API

There is no `SimpleSearch` — it was removed when Lorcana stopped having a
separate matching path, and every scraper now goes through `Match()`. The core
lookup surface is in `mtgmatcher/api.go`: `GetUUIDs`, `GetUUIDsInSet`,
`GetSealedUUIDsInSet`, `Names`, and the `Search*` family.

## Adding a scraper

1. New package with the standard layout: `<store>.go` (struct plus
   `Load`/`Inventory`/`Buylist`/`Info`), `api.go` (client and auth),
   `preprocess.go` (store text → `InputCard` → `b.Match()`), optional
   `sealed.go`.
2. Embed the common fields (`LogCallback`, `MaxConcurrency`,
   `DisableRetail`/`DisableBuylist`, inventory/buylist plus timestamps) and
   keep the `*mtgmatcher.Backend` the constructor was handed in an unexported
   field — follow `starcitygames` for an API-backed store or `mtgseattle` for
   an HTML-scraped one.
3. Fetch with `WorkerPool` plus `retryablehttp` (`LinearJitterBackoff`).
4. Add a `register.go` whose `init()` calls `mtgban.Register(name, games,
   constructor)` — `name` is the external flag the store has always been
   known by, the store's own name alone under `mtgban.GameMagic` and
   `<store>_<game>` under every other game (`coolstuffinc_pokemon`,
   `cardtrader_gundam`, `starcitygames_sealed_lorcana`), and `games` is
   every game the scraper prices. Blank-import the package in
   `cmd/bantool/main.go`: `targets()` walks `mtgban.AllGames` ×
   `mtgban.Registered(game)` to build the flag table, so a registered
   scraper needs no entry written there by hand, only the import. Add one
   `bantool-<store>_<game>.yml` workflow per target —
   `cmd/bantool/workflows_test.go`'s
   `TestEveryTargetIsScheduledByItsOwnWorkflow` fails the build if a
   registered target has no workflow scheduling it, or a workflow names a
   target that is not registered.
5. Set the right `ScraperInfo` flags: `MetadataOnly`, `NoQuantityInventory`,
   `SealedMode`, `CreditMultiplier`, `Family`, and `Game` — every scraper sets
   `Game` explicitly now, `mtgban.GameMagic` included; nothing reads as Magic
   by default.
6. The constructor takes the datastore first and nothing naming a game:
   `NewScraper(b *mtgmatcher.Backend, ...) (*T, error)`. A scraper that prices
   more than one game reads which one with `mtgban.GameOf(b)`, so it cannot be
   told one game and matched against another's datastore. The vendor's own
   naming for its games — slugs, catalog ids, department numbers — stays
   exported, because the package's own API helpers take one (`Search`,
   `SCGBuylistURL`, `NewGNClient`); what a caller no longer needs it for is
   building a scraper. One `map[mtgban.Game]<vendor value>` per package sits
   between the two and both converts and validates, and a game the map does
   not hold is refused at the constructor — as is a datastore that names no
   game at all, which `GameOf` rejects before the map is asked, rather than
   anything silently defaulting to Magic. Store the typed game on the struct
   so `Info()` reads `info.Game = x.game` rather than switching a vendor value
   back into one; keep every read that drives a run on the vendor value.

### Adding a game

A game is added in `mtgban` first and reaches the scrapers from there:

1. Add the `mtgban.Game` constant in `mtgban/mtgban.go` and list it in
   `mtgban.AllGames`. Both are pinned by `mtgban/game_test.go`, which also
   pins the string it is published as — that value is a wire format, carried
   in every dump, so pick it once and do not rename it later.
2. Register the matcher side: a loader, a `GameRules` implementation and a
   `register.go` under `mtgmatcher/<game>/`, plus the blank import in
   `mtgmatcher/games/games.go`.
3. Per storefront that carries it: one constant naming the vendor's own
   spelling beside that package's existing ones, and one line in its
   `<recv>Games` map. Nothing else in the scraper changes — the switches that
   used to translate a vendor id back into a game are gone.
4. Per scraper that should run it: add the game to that store's own
   `register.go` (its `mtgban.Register` games list) alongside the map entry
   from step 3, and add a `bantool-<store>_<game>.yml` workflow. Nothing is
   written in `cmd/bantool/main.go` itself — `targets()` derives the entry
   from the registry.
5. Wire the game's datastore into `.github/workflows/ci.yml` — a cache job and
   a `test-<game>` job — and add its path variable to
   `internal/vocabulary/read.go`'s `Games`.

For TCGplayer specifically, the per-game scrapers `TCGGame` and `TCGGameIndex`
are built from the `tcgplayer` package's `tcgGames` map, which associates a
game with the TCGplayer category serving it. Adding a game there is one table
entry plus the bantool options and workflows. Magic is deliberately absent from
that map: it is identified by SKU and has its own scrapers.

## Gotchas

- A `Backend` is immutable once loaded: its maps, slices, cards and rules
  are shared by every scraper built on it and must not be mutated. Replacing
  a datastore means loading a new backend and building new scrapers on it;
  nothing is published process-wide. A report reads the backend it is
  handed as its first argument, and a nil one returns no rows at all;
  `ArbitOpts` only filters. See
  `docs/adr/0004-localized-matcher-and-scraper-registry.md`.
- Everything that reads a datastore is a method on `*Backend`, including
  the `magic.Has*Printing` helpers (they take the backend first).
  `ExtractNumber` and `ExtractNumberAny` read no datastore and stay
  package-level functions, along with `Normalize`, `Title`, `ExtractYear`,
  `SplitVariants` and the rest.
  Keep new identification lookups on `b`, including in callbacks.
- The Magic promo-type constants (`PromoTypeBoosterfun`, `PromoTypeBuyABox`,
  `PromoTypePrerelease`, `PromoTypePromoPack`, `PromoTypeThickDisplay` and the
  rest) live only in `mtgmatcher/magic`. Core keeps no shim copies, so code
  that resolved them from `mtgmatcher` fails to compile until it imports
  `magic` — deliberately, since core cannot import `magic` and a duplicated
  constant would drift silently where a build error names the symbol.
- Foil and etched flags coming from scrapers are often wrong. `output()`
  clamps them against the printing's real finishes — trust the matcher, not
  the input.
- `Normalize()` has deliberate *protection* entries that map a string to
  itself (`"waste land"`, `"vs"`). Changing the replacer table can silently
  re-alias unrelated cards; run the full matcher suite after any edit there.
- The insert-time sort invariant matters: `Arbit` assumes `entries[0]` is the
  NM entry, and that ordering is a *side effect* of the sort in `add()`. A
  change to that sort breaks `Arbit` silently. Pin the ordering in
  `mtgban/base_test.go` before touching it.
- `Mismatch`'s `defaultGradeMap` covers NM/SP/MP/HP plus `PO` at 0. A
  condition that is unknown, or that maps to zero, causes the pair to be
  *skipped* — not zero-priced — because a zero factor cannot be divided by.
  The reference's own grade is divided out before the probe's is applied, so a
  non-NM reference is not compared against a rescaled copy of itself.
- `Card.Legalities` is populated only by the MTGJSON loader, so it is nil for
  every non-Magic card, all eight other games alike.
- `WriteBuylistToCSV` is the one CSV writer taking a middle `creditMultiplier`
  argument; `GetExchangeRate` returns the *reciprocal* — a multiply-to-USD
  factor.

## Git / commits

- **No `Co-Authored-By` lines** in commit messages.
- **Wrap commit subject and body at 80 columns.**
- Commit messages follow the existing `area: imperative summary` style, e.g.
  `mtgmatcher: dedup hashes via per-norm membership set`. Game sub-packages
  use their full path as the area, e.g. `mtgmatcher/riftbound: ...`.
- Do not commit compiled binaries, datastore JSON, or scraped CSVs. The
  current `.gitignore` catches `*.exe`/`*.dll`/`*.so`/`*.dylib`/`*.test` and
  `*.csv`, but **not** extensionless Go binaries and **not** datastore
  `*.json` files, so check `git status` before staging.

## Task-specific guides

- **Scraper log noise, refusals, or a suspicious cross-vendor / buy-vs-ask
  price**: read `docs/agents/noise-sweep/README.md` before starting. It
  covers ranking targets from CI refusal lines and site-captured spreads,
  replaying a vendor's feed through the production path rather than
  hand-built `Match()` calls, classifying each shape into scraper table /
  matcher rule / datastore gap, and the report format the work is graded by.
