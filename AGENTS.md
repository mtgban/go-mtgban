# AGENTS.md

Guidance for AI coding agents working on **go-mtgban**, a trading-card
market-data platform: scrape store inventories and buylists, normalize every
listing to a canonical card identity, then compute arbitrage across stores.
Three games are supported today — Magic: The Gathering, Lorcana, and
Riftbound. Read `SPECIFICATIONS.md` for the full architecture and `docs/adr/`
for the reasoning behind the load-bearing decisions.

## The one rule that matters

**Everything keys on the `mtgmatcher` UUID.** Scrapers are thin translators;
correctness lives in `mtgmatcher`. If a card matches wrong, fix it in
`mtgmatcher` — usually in a data table, and for Magic inside the
`mtgmatcher/magic` sub-package — not in the scraper.

## Layout

```
mtgban/                interfaces (Scraper/Seller/Vendor), records,
                       Arbit/Mismatch, CSV I/O, WorkerPool
mtgmatcher/            game-agnostic core: Backend, Match()/MatchId(), the
                       GameRules seam, the game registry, the search API,
                       the shared EditionTable/VariantsTable, and the Magic
                       replay suite
mtgmatcher/magic/      Magic rules, MTGJSON loader, promo/frame vocabulary
mtgmatcher/lorcana/    Lorcana rules, loader, replay corpus
mtgmatcher/riftbound/  Riftbound rules, loader, replay corpus
mtgmatcher/games/      meta-package that blank-imports all three games
<store>/               one package per store (tcgplayer, cardkingdom,
                       cardmarket, ...)
cmd/                   tools; cmd/bantool is the production orchestrator
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
in `mtgban/`, in `mtgmatcher/` and its `lorcana` and `riftbound`
sub-packages, and in several scraper packages (`abugames`, `cardkingdom`,
`starcitygames`, `tcgplayer`). A subset run can pass while CI fails.
`mtgmatcher/magic` has no test files of its own — it is covered indirectly,
through the core Magic replay suite.

### Datastores

The matcher suites exercise real data, one datastore per game, located through
environment variables:

- `ALLPRINTINGS5_PATH` — MTGJSON `AllPrintings5.json`. Feeds the core
  `mtgmatcher` suite and `mtgmatcher/magic`, both of which load it through
  `magic.Load`, as well as the abugames, cardkingdom and starcitygames
  scraper suites.
- `LORCANA_PATH` — the LorcanaJSON all-cards file, plain uncompressed JSON.
  Feeds `mtgmatcher/lorcana`.
- `RIFTBOUND_PATH` — the Riftbound datastore, built by
  `github.com/mtgban/riftbound-datastore`. Feeds `mtgmatcher/riftbound`.

The asymmetry is deliberate but sharp-edged. Both Magic-backed suites — core
`mtgmatcher` and `mtgmatcher/magic` — fail fast: their `TestMain` calls
`log.Fatalln("Need ALLPRINTINGS5_PATH variable set to run this suite")` when
it is unset, which takes down the whole package's test binary, including the
tests that need no datastore at all such as the replacer and utility tests.
The Lorcana and Riftbound suites call `t.Skip` instead, so a contributor
without either dump still gets a green (if thinner) run.

Use absolute paths for all three variables. A relative path is resolved
against the directory of the package under test, so a single relative value
cannot serve suites that sit at different depths in the tree — and they do:
`ALLPRINTINGS5_PATH` is read from `mtgmatcher` one level down and from
`mtgmatcher/magic` two. CI passes all three as absolute paths for that
reason.

CI restores all three datastores from `actions/cache` before testing. Magic
and Lorcana are fetched from the URLs in the `DATASTORE_MAGIC` and
`DATASTORE_LORCANA` repository variables; Riftbound has no public URL and is
pulled from a private B2 bucket, keyed on the object's own metadata because B2
serves no HTTP etag.

### The three golden suites

Each game owns a replay corpus of *(input card → expected verdict)* pairs,
and each has its own regeneration flag:

The flag belongs to the test binary, not to `go test`, so the package has to
come first — `go test -u ./mtgmatcher/magic/` fails with "no Go files in" the
repository root, having read `-u` as a `go` flag and dropped the path.

```sh
# mtgmatcher/magic     -> testdata/magic_test_data.json
go test ./mtgmatcher/magic/ -run TestMatch -u

# mtgmatcher/lorcana   -> testdata/lorcana_test_data.json
go test ./mtgmatcher/lorcana/ -update-lorcana

# mtgmatcher/riftbound -> testdata/riftbound_test_data.json
go test ./mtgmatcher/riftbound/ -update-riftbound
```

All three corpora sit under a `testdata/` directory beside the rules they
exercise. The Magic regeneration flag is the odd one out: it is the bare
`-u` it has always been, rather than being named after its game.

Regenerate only after an *intentional* matching change, and read the resulting
diff line by line. The Lorcana and Riftbound regenerators carry two extra
safety nets: they refuse to flip a case between success and error (a
verdict-class flip fails the test and leaves the golden file untouched), and
they re-insert their hand-authored seed cases — `lorcanaSeeds` and
`riftboundSeeds` — which pin the contract edges the sampled corpus cannot
reach. The Magic regenerator has neither guard; it silently rewrites the
expected uuid of any case that now resolves differently and trusts you to
read the diff.

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
  helper (`[TAG] `-prefixed). `mtgmatcher` logs to `io.Discard` unless
  `SetGlobalLogger` is called.
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
are dispatched through the `GameRules` interface in `mtgmatcher/rules.go`:
`Prefilter`, `AdjustName`, `AdjustEdition`, `FilterPrintings`, `CandidateSets`,
`FilterCards`, `FinalizeCandidates`, `IsUnsupported`, `IsSpecificUnsupported`,
and `MissingPromoTag` (see `rules.go` for the complete interface). A game's
loader attaches its implementation with `Backend.SetRules` when it builds the
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
`database/sql` driver idiom — under the names `magic`, `lorcana`, and
`riftbound`.

The consequence is that **a game only exists if something imports it**. A
consumer blank-imports the games it needs, or blank-imports
`mtgmatcher/games` to get all of them (`cmd/bantool` does the latter); asking
`Open` for a game nothing registered is an error naming the games that are.

### Datastore loading

`mtgmatcher.Open(name, reader)` loads the named game's datastore: it runs
exactly one loader and hands back the `*Backend` without touching the global,
which `SetGlobalDatastore` installs. `RegisteredGames()` lists what is
currently linked in. `internal/datastore.Read(game, path)` is the same over a
path that may be a file, an `http(s)://` URL or a `b2://` object, `.xz` or
not; a suite reads its game's file that way in its TestMain, or in a helper
of its own for another game's.

There is no auto-detection. The caller always knows the game — bantool reads
it off the scraper it runs, a test off the package it sits in — and the
loader that tried every registered game in turn decoded AllPrintings three
times over before reaching Magic's, behind a buffer of the whole file.

`Backend` is exported and carries instance methods (`b.Match`, `b.GetUUID`,
`b.GetSetByName`, ...); the package-level functions of the same name are thin
wrappers over the global backend.

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

Lorcana and Riftbound need far less of this. Both identify a card by name plus
collector number plus finish, so half their hooks are literal no-ops —
`FilterPrintings` returns the editions untouched and `IsUnsupported`,
`IsSpecificUnsupported` and `MissingPromoTag` all return `false`. The real
work is edition and number normalization in `Prefilter`/`AdjustName`/
`AdjustEdition` and the number-and-finish disambiguation in `FilterCards`.

### Search API

There is no `SimpleSearch` — it was removed when Lorcana stopped having a
separate matching path, and every scraper now goes through `Match()`. The core
lookup surface is in `mtgmatcher/api.go`: `GetUUIDs`, `GetUUIDsInSet`,
`GetSealedUUIDsInSet`, `AllNames`, and the `Search*` family.

## Adding a scraper

1. New package with the standard layout: `<store>.go` (struct plus
   `Load`/`Inventory`/`Buylist`/`Info`), `api.go` (client and auth),
   `preprocess.go` (store text → `InputCard` → `Match()`), optional
   `sealed.go`.
2. Embed the common fields (`LogCallback`, `MaxConcurrency`,
   `DisableRetail`/`DisableBuylist`, inventory/buylist plus timestamps) —
   follow `ninetyfive` for an API-backed store or `mtgseattle` for an
   HTML-scraped one.
3. Fetch with `WorkerPool` plus `retryablehttp` (`LinearJitterBackoff`).
4. Register a `scraperOption` in `cmd/bantool` and add a
   `.github/workflows/bantool-<store>.yml`. Non-Magic scrapers get one option
   and one workflow per game, named `<store>_lorcana` / `<store>_riftbound`.
5. Set the right `ScraperInfo` flags: `MetadataOnly`, `NoQuantityInventory`,
   `SealedMode`, `CreditMultiplier`, `Family`, and `Game` (`mtgban.GameMagic`
   is the empty string, so a non-Magic scraper must set `Game` explicitly).

For TCGplayer specifically, the per-game scrapers `TCGGame` and `TCGGameIndex`
are built from the `tcgplayer.SupportedGames` map, which associates a game tag
with the TCGplayer category serving it. Adding a game there is one table entry
plus the bantool options and workflows. Magic is deliberately absent from that
map: it is identified by SKU and has its own scrapers.

**Do not copy as templates:** `trollandtoad`, `wizardscupboard`, and
`strikezone` still use `gocolly` with hand-rolled concurrency; they predate
`WorkerPool` and should be migrated, not imitated.

## Gotchas

- The global matcher backend is published through `atomic.Pointer[Backend]`.
  Each package-level operation captures one immutable snapshot. Several calls
  can span publications: capture `GlobalDatastore()` for related lookups, or
  pass `ArbitOpts.Backend` for a report using an explicit backend. The snapshot
  is a shallow copy; maps, slices, cards and rules remain shared and must not
  be mutated after publication. See `docs/adr/0003-atomic-backend-snapshots.md`.
- Magic identification callbacks use the supplied backend. Only the exported
  `magic.Has*Printing` convenience wrappers intentionally consult the global.
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
  both Lorcana and Riftbound.
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
