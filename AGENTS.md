# AGENTS.md

Guidance for AI coding agents working on **go-mtgban**, a Magic: The Gathering
market-data platform (scrape store inventories/buylists → normalize to a
canonical card identity → compute arbitrage). Read `SPECIFICATIONS.md` for the
full architecture; `todo/refactor.md` for the prioritized cleanup plan.

## The one rule that matters

**Everything keys on the `mtgmatcher` UUID.** Scrapers are thin translators;
correctness lives in `mtgmatcher`. If a card matches wrong, fix it in
`mtgmatcher` (usually a data table), not in the scraper.

## Layout

```
mtgban/      interfaces (Scraper/Seller/Vendor), records, Arbit/Mismatch, I/O, WorkerPool
mtgmatcher/  card identity: Match()/MatchId(), DataStore loaders, variant tables, sealed API
<store>/     one package per store (tcgplayer, cardkingdom, cardmarket, ...)
cmd/         tools; cmd/bantool is the production orchestrator
```

## Build, test, format

```sh
go build ./...                          # must stay green
go test ./mtgban/... ./mtgmatcher/...   # the real test suite lives here
gofmt -l $(git ls-files '*.go')         # must print nothing before committing
go vet ./mtgban/... ./mtgmatcher/...
```

- **CI runs `go test ./... -v` only — there is no `gofmt`/`go vet` gate**
  (`.github/workflows/ci.yml`). Run them yourself before committing; nothing
  upstream will catch a formatting/vet regression.
- The real test surface today: `mtgmatcher/*_test.go` (matcher, replacer,
  variants, editions, api, backend, utils), `mtgban/base_test.go`, and
  preprocess tests for `cardkingdom` and `starcitygames`. Everything else is
  validated operationally.
- `mtgmatcher` tests require a real MTGJSON dataset: set
  `ALLPRINTINGS5_PATH=/path/to/AllPrintings5.json`. Without it those tests skip.
- `mtgmatcher/matcher_test.go` is data-driven (`matcher_test_data.json`). After
  an *intentional* matching change, regenerate expectations with `go test -run
  TestMatch -u ./mtgmatcher/` and **review the diff** — never blindly accept it.

## Conventions

- **gofmt always.** Currently `gofmt -l` flags exactly `mtgban/arbit.go`
  (misaligned `resolvedOpts` struct), `cmd/manapoolOrders/main.go`, and
  `cmd/mp2ckbl/main.go` (see `todo/refactor.md` §0.1). Don't add more.
- **No global loggers.** Each scraper takes a `LogCallback mtgban.LogCallbackFunc`
  and logs through a tagged `printf` helper (`[TAG] `-prefixed). `mtgmatcher`
  logs to `io.Discard` unless `SetGlobalLogger` is called.
- **Insert via the `Add*` family**, never by appending to the map directly —
  `Add`/`AddRelaxed`/`AddStrict`/`AddUnique` enforce defaults (NM, qty 1),
  validate conditions, merge duplicates, and keep each slice sorted. Downstream
  code (notably `Arbit` and `CombineBuylists`) assumes `entries[0]` is the NM
  entry. `AddUnique` is the strictest gate and **ignores price**.
- **`ErrUnsupported` is a silent-skip signal**, not a failure. In a scraper's
  preprocess loop: skip `ErrUnsupported`, but log `AliasingError` (call
  `.Probe()` to dump candidates) and other errors as data-quality alarms.
- **Concurrency = `mtgban.WorkerPool`.** New fetch code uses it; don't hand-roll
  goroutine/channel pools. Context cancellation stops dispatch but lets in-flight
  workers finish.

## mtgmatcher: tables before code

New-set / new-promo support is almost always **data**, not logic:
- card↔number disambiguation → `variants.go` (`VariantsTable`)
- edition name aliases → `editions.go` (`EditionTable`)
- promo detection rules → `callbacks.go` (`promoTypeElements`)
- per-set special cases that truly need code → `callbacks.go` callbacks or the
  load-time patch tables in `backend.go` (the `switch set.Code` block) plus
  the companion data in `table.go`

Adding a `switch` case in `Match()`/`adjustEdition()` is the last resort. The
`Match()` pipeline order (id → name surgery → canonicalize → edition → set
selection → card disambiguation → verdict) is documented in `SPECIFICATIONS.md`
§2.4; understand it before editing. (`adjustEdition`'s terminal per-card switch
is itself a "tables before code" violation queued for extraction —
`todo/refactor.md` §4.1.)

### Datastore loading

`mtgmatcher.LoadDatastore(reader)` is the entry point. It tries
`LoadAllPrintings` (MTGJSON) and **falls back to `LoadLorcana`** via an
`io.TeeReader` replay; both satisfy `type DataStore interface { Load()
cardBackend }`, and the result is assigned directly to the package-global
`defaultBackend`. `SetGlobalDatastore` is a *separate* entry for swapping a
pre-built backend. `cardBackend` is unexported, so only the package can
implement `DataStore`.

## Adding a scraper

1. New package with the standard layout: `<store>.go` (struct + `Load`/
   `Inventory`/`Buylist`/`Info`), `api.go` (client/auth), `preprocess.go`
   (store text → `InputCard` → `Match()`), optional `sealed.go`.
2. Embed the common fields (`LogCallback`, `MaxConcurrency`, `DisableRetail`/
   `DisableBuylist`, inventory/buylist + timestamps) — follow `ninetyfive`
   (API) or `mtgseattle` (HTML).
3. Fetch with `WorkerPool` + `retryablehttp` (LinearJitterBackoff).
4. Register a `scraperOption` in `cmd/bantool` and add a
   `.github/workflows/bantool-<store>.yml`.
5. Set the right `ScraperInfo` flags: `MetadataOnly`, `NoQuantityInventory`,
   `SealedMode`, `CreditMultiplier`, `Family`, `Game`.

**Do not copy as templates:**
- `trollandtoad`, `wizardscupboard`, `strikezone` — legacy `gocolly` +
  hand-rolled concurrency, predate `WorkerPool` (migration is `todo/refactor.md`
  §2.1).
- `amazon/` — a stub (`api.go` Creators-API client only, no scraper struct,
  blocked on credentials).
- `mvpsportsandgames/` — its `Inventory()` returns `(record, error)` and does
  **not** satisfy `mtgban.Seller`; WIP/orphan.

## Gotchas

- The `mtgmatcher` backend (`defaultBackend`) is a **package global, immutable
  after load by convention, and unsynchronized.** Safe for concurrent reads —
  *but only if it isn't reloaded.* The reference consumer reassigns it at
  runtime via `/api/load/datastore` with no locking, which is a latent data
  race; don't add an in-process reload without making the swap atomic.
- Foil/etched flags from scrapers are often wrong — `output()` clamps them
  against the printing's real finishes. Trust the matcher, not the input.
- `Normalize()` has deliberate *protection* entries (e.g. `"waste land"`,
  `"vs"`). Changing the replacer table can silently re-alias unrelated cards;
  run the full matcher test suite after any edit there.
- Insert-time sort invariants matter: `Arbit` and `CombineBuylists` assume
  `entries[0]` is NM, while `CombineInventories` hard-filters `Conditions !=
  "NM"`. The NM-first ordering is produced *as a side effect of the sort in
  `add()`* — a change to `add()`'s sort silently breaks both consumers. Pin
  ordering in `base_test.go` before touching it.
- `Mismatch`'s `defaultGradeMap` only has NM/SP/MP/HP; any other condition
  (including `PO`) maps to multiplier 0 and zeroes the price.
- `Card.Legalities` is populated only from MTGJSON — it is **nil for Lorcana**.
- `WriteBuylistToCSV` is the one CSV writer taking a middle `creditMultiplier`
  argument; `GetExchangeRate` returns the *reciprocal* (a multiply-to-USD
  factor).

## Git / commits

- **No `Co-Authored-By` lines** in commit messages.
- **Wrap commit subject and body at 80 columns.**
- Commit messages follow the existing `area: imperative summary` style
  (e.g. `mtgmatcher: dedup hashes via per-norm membership set`).
- Don't commit compiled binaries, datastore JSON, or scraped CSVs (the current
  `.gitignore` misses extensionless Go binaries and `*.json` datastores — see
  `todo/refactor.md` §0.2).
