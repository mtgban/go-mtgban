# Refactor Plan

A prioritized, evidence-backed cleanup plan for go-mtgban. Each item notes
**why**, **scope**, **risk**, and a checklist. Ordered roughly by
value-to-effort. Scope is **committed code only** unless explicitly noted —
untracked WIP (`synthetic/`, `mvpsportsandgames/`, `tcgplayer/complete.go`,
`tcgplayer/custom.go`, ~22 `cmd/` tools) is triaged in §0.3 but otherwise
out of scope.

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done

---

## Prioritization matrix

Each item scored **Impact** (dev-velocity / ops drag), **Risk** (cost if left
unfixed), and **Effort** (1 = trivial, 5 = large), with
`Priority = (Impact + Risk) × (6 − Effort)` — higher = do sooner. This is a
triage lens only; the **dependency sequencing** at the bottom still governs
execution order (e.g. §2.3 before §2.2; §3.1 characterization tests before the
money-path dedups).

| Item | Debt type | I | R | E | Priority |
|------|-----------|---|---|---|----------|
| 0.7 Profitability formula *(code done)* | Code / money | 3 | 4 | 1 | **35** |
| 0.5 CI gofmt/vet gate | Infrastructure | 3 | 3 | 1 | **30** |
| 0.6 bantool credential-arg bug | Code / correctness | 2 | 4 | 1 | **30** |
| 0.8 abugames Solr efficiency *(new)* | Performance | 4 | 2 | 1 | **30** |
| 0.3 untracked WIP + pre-commit secret gate | Doc / security | 3 | 4 | 2 | 28 |
| 0.9 dependency / vuln audit *(new)* | Dependency | 2 | 3 | 1 | 25 |
| 3.4 CI runs the data-backed suite *(new)* | Test / infra | 3 | 3 | 2 | 24 |
| 3.1 analysis-layer characterization tests | Test | 3 | 4 | 3 | 21 |
| 4.4 global-backend reload race | Architecture | 2 | 5 | 3 | 21 |
| 0.2 gitignore / stray artifacts | Infrastructure | 2 | 2 | 1 | 20 |
| 2.3 `add()` dedup | Code | 2 | 4 | 3 | 18 |
| 0.1 gofmt three files | Code | 2 | 1 | 1 | 15 |
| 2.1 WorkerPool migration | Code | 2 | 3 | 3 | 15 |
| 3.2 preprocess tests | Test | 2 | 3 | 3 | 15 |
| 3.3 mtgmatcher API tests | Test | 2 | 3 | 3 | 15 |
| 1.1 BaseScraper extraction | Code | 4 | 3 | 4 | 14 |
| 2.2 Arbit/Mismatch dedup | Code | 3 | 4 | 4 | 14 |
| 1.2 shared HTTP client | Code | 2 | 2 | 3 | 12 |
| 2.4 Combine dedup | Code | 2 | 2 | 3 | 12 |
| 0.4 mp2ckbl supabase config | Doc | 1 | 1 | 1 | 10 |
| 2.5 json.go collapse | Code | 1 | 1 | 1 | 10 |
| 4.1 adjustEdition table extract | Code | 3 | 2 | 4 | 10 |
| 4.2 filterPrintings split | Code | 2 | 1 | 3 | 9 |
| 1.3 responseChan unify | Code | 1 | 1 | 2 | 8 |
| 4.3 variants.go split | Code | 1 | 1 | 2 | 8 |

---

## P0 — Hygiene (cheap, do first)

### 0.1 `gofmt` the three unformatted files
**Why:** `gofmt -l $(git ls-files '*.go')` currently flags exactly:
`mtgban/arbit.go`, `cmd/manapoolOrders/main.go`, `cmd/mp2ckbl/main.go`.
CI/agents should start from a clean base.
**Risk:** none (pure formatting).

- [ ] `gofmt -w mtgban/arbit.go` (misaligned `resolvedOpts` struct)
- [ ] `gofmt -w cmd/manapoolOrders/main.go`
- [ ] `gofmt -w cmd/mp2ckbl/main.go`
- [ ] Confirm `gofmt -l $(git ls-files '*.go')` prints nothing

### 0.2 Close `.gitignore` gaps & untrack stray artifacts
**Why:** compiled binaries and datastore JSON sit untracked in the tree
(`bantool`, `cmd/*/​<binary>`, top-level `manapoolOrders` / `mkmhtml2csv` /
`mp2ckbl` / `omnitool-3g`, `cmd/mp2ckbl/allprintings5.json`). The current
`.gitignore` only covers `*.exe`/`*.so`/`*.csv`, not native Go binaries or
`*.json` datastores.
**Risk:** low; verify nothing intended is ignored before committing.

- [ ] Add ignore patterns: compiled cmd binaries (e.g. `/cmd/*/[a-z]*` with no
      extension, or enumerate them), `allprintings5.json` /
      `AllPrintings*.json`, `identifiers.json`, top-level tool binaries
- [ ] Confirm no committed file matches the new patterns:
      `git ls-files | git check-ignore --stdin --no-index` returns nothing
- [ ] Remove the stray binaries/JSON from the working tree

### 0.3 Decide fate of untracked WIP
**Why:** several new packages/files are untracked and excluded from review:
`synthetic/`, `mvpsportsandgames/`, `tcgplayer/complete.go`,
`tcgplayer/custom.go`, and ~22 `cmd/` tools. They drift from the committed
codebase and confuse tooling.
**Risk:** medium — `complete.go`/`custom.go` may be load-bearing for current
scraping.

- [ ] Triage each: commit, or delete if abandoned
- [ ] For anything committed, apply the rest of this plan to it
- [ ] **Audit for hardcoded credentials before committing** — these tools
      are **untracked** (not in the index), so this is a *pre-commit* gate,
      not a committed-secret emergency. `cmd/omnitool-3g/main.go:61–87` holds
      live secrets (MKM app token/secret, a `cspherePassword`, a full
      CardTrader production JWT, `scgBearer`, `banKey`, `csSessionToken`);
      `cmd/autocart` and several `mkm*`/`ct*` tools similar. Move to env vars
      / `.env` before any `git add`. Note: `sb_publishable_*` prefix in
      Supabase URLs (see §0.4) is by design a *public* anon key, not a secret.
- [ ] **`amazon/api.go` is committed and in-scope but unplanned.** It is a
      complete Amazon Creators-API client (OAuth2 `SearchItems`/`GetItems`)
      with **no scraper struct / no `Load`/`Inventory`/`Info`** — dead stub
      blocked on Associates credentials. Decide: keep with a clear
      package/file doc comment ("client only, scraper TODO"), or move behind a
      build tag, so it doesn't masquerade as a live scraper. (`cmd/amazonsearch`
      is its test harness; tracked.)
- [ ] **`mvpsportsandgames/` (untracked) does not compile against the
      interface** — `Inventory()` returns `(InventoryRecord, error)`, not the
      bare `InventoryRecord` that `mtgban.Seller` requires (`mvp.go:76`).
      Triage is "delete, or fix the signature," not just "commit or delete."

### 0.4 mp2ckbl: consolidate Supabase endpoint config
**Why:** `cmd/mp2ckbl/main.go` hardcodes `supabaseAPIURL` and
`supabaseAPIKey` as package-level constants (lines 49–50). The key is a
Supabase *publishable* anon key (`sb_publishable_*` prefix) and is safe to
embed client-side, so this is **not** a credential leak — but the pair is
de-facto third-party-vendor config that should live next to other env-driven
config for consistency.
**Risk:** none.

- [ ] Move the URL + publishable key to env vars (`MANAPOOL_SUPABASE_URL`,
      `MANAPOOL_SUPABASE_KEY`) with the current values as defaults, OR
      explicitly comment why they're hardcoded (publishable, stable endpoint)

### 0.5 Add a CI `gofmt`/`go vet` gate (root-cause fix for 0.1)
**Why:** `.github/workflows/ci.yml` runs **only** `go test ./... -v` (line 47)
— no format or vet gate. That missing gate is *why* unformatted files
(§0.1) keep slipping in. 0.1 fixes the symptom; this fixes the cause.
**Risk:** none (additive CI step). Do it in the **same PR** as 0.1 so it
can't regress.

- [ ] Add a step running `test -z "$(gofmt -l $(git ls-files '*.go'))"` (fail
      on any unformatted file) and `go vet ./...` to `ci.yml`
- [ ] Confirm the suite is green only after 0.1 lands

### 0.6 Fix `bantool initializeBucket` credential arg duplication (committed bug)
**Why:** `cmd/bantool/main.go` initializes the TCG-SKU bucket as
`initializeBucket(tcgSKUPath, os.Getenv("B2_KEY_ID_DATASTORE"),
os.Getenv("B2_KEY_ID_DATASTORE"))` at lines **287, 419, 459, 490** — the
**key-id is passed as both the access-key and the secret.** Compare the
correct datastore-bucket call at line 975:
`initializeBucket(*datastoreOpt, B2_KEY_ID_DATASTORE, B2_APP_KEY_DATASTORE)`.
This is committed, in-scope, and silently wrong (the SKU fetch authenticates
with a malformed secret).
**Risk:** low to fix, but it changes which env var is read — confirm
`B2_APP_KEY_DATASTORE` is the intended secret before committing; can't be
unit-tested without B2 creds.

- [ ] Replace the duplicated `B2_KEY_ID_DATASTORE` second arg with
      `B2_APP_KEY_DATASTORE` at lines 287/419/459/490 (or confirm the SKU
      bucket genuinely shares a single-credential auth and document why)

### 0.7 Profitability formula — code corrected to match spec — DONE
**Why:** the spec (Koda's brain-teaser doc) defines
`PI = (Difference / (Sell Price + k)) * log10(1 + Spread) * Units^(1/2)`. The
`ArbitEntry.Profitability` doc comment matched that spec, but the **code had
drifted**: it used `math.Log` (natural log, not log10) and `Pow(qty, 0.25)`
(fourth root, not square root) at both call sites (Arbit + Mismatch). The
fourth root is an unambiguous spec violation; the log base only rescales PI by
a constant (order-preserving for ranking) but shifts the absolute
`MinProfitability` gate.
**Risk:** money-facing. The log10 switch makes a given `MinProfitability` ~2.3×
stricter than under ln; consumers must retune (the website's `MinProfitable =
4.0` ≈ `1.74` on the new scale to preserve behavior).

- [x] Fix both call sites: `math.Log` → `math.Log10`, `Pow(qty,0.25)` →
      `math.Sqrt(qty)`; restore the comment to the spec formula
- [x] Retune `MinProfitability` in consumers — DONE in `mtgban-website`:
      `arbit.go` `MinProfitable` 4.0 → 1.74 (= 4.0 / ln(10)), and the inline
      `upload.go:1090` copy switched to `math.Log10` + `math.Sqrt`; `sleep.go`'s
      tier formula left on `ln` deliberately (integer buckets) with an inline
      note. (upload's PI is sort-only, so only `arbit.go`'s threshold retuned.)
- [ ] Decide the library default for `k` (`ProfitabilityConstant`): spec
      reference is 10, current default is 0 (no stabilization); the website
      passes its own (2 / 10), so production is unaffected, but an unconfigured
      caller gets the un-stabilized formula
- [ ] When §3.1 lands, add a golden test pinning the formula (log10, sqrt, the
      chosen `k` default) so code and comment stay tied

### 0.8 abugames: stop paying the Solr grouping cost on every page
**Why:** `abugames/` is the slowest scraper, and timing `data.abugames.com`
shows three compounding request-shape problems — none captured in this plan:
1. **`group.ngroups=true` on every page.** Baked into `abuBaseUrl` /
   `abuBaseUrlFull` (`api.go:66,69`), it makes Solr enumerate all ~213k groups
   *per request* (~2.9 s of the ~3.8 s/page), yet per-page code never reads the
   count — `GetProduct` consumes `.Groups`, not `.Count` (`abugames.go:59`).
   It is needed **only** on the initial `GetTotalItems`. Dropping it measured
   3.76 s → 0.89 s/page (~4.2×), identical data.
2. **No `fl` field list.** Docs carry 56 fields; the scraper reads 17
   (`ABUCard`, `api.go:16`). Adding `fl` cut a page 6.08 MB → 0.66 MB (~9×
   less to transfer and JSON-decode in each worker).
3. **`rows=200` too small** (`maxEntryPerRequest`, `api.go:64`). 213,446
   groups = 1,068 requests; `rows=1000` → 214 requests, amortizing the
   per-request overhead.

Combined ≈ **10× faster** (≈20–30 min → ≈2–3 min at concurrency 4) and ~9×
less data. `defaultConcurrency = 4` is a red herring once each request is
sub-second — fix request shape before touching concurrency.
**Risk:** low — request shape only; response decoding is unchanged.
**Scope:** `abugames/api.go` (+ one `fl` const). The sealed path uses an
ungrouped `numFound` query, so it has no ngroups problem and gains only
marginally from `fl`.

- [x] `GetProduct`: strip `group.ngroups` (kept on `GetTotalItems`) and add
      `fl=` for the 17 fields `ABUCard` decodes — done in `abugames/api.go`
- [x] Bump `maxEntryPerRequest` 200 → 1000
- [ ] Spot-check inventory/buylist counts vs. a recorded run (a live smoke test
      confirmed the count path + page decode: 1000 groups in 0.57 s vs. 200 in
      ~3.8 s; full Load-count diff still TODO); confirm deep pages still return
- [ ] *(deeper, optional)* residual cost is Solr deep paging (O(start));
      `cursorMark` can't combine with `group=true`, so only a flat ungrouped
      query + client-side grouping erases it — track as a larger item

### 0.9 Dependency currency / vulnerability audit
**Why:** the plan verifies `go mod tidy` is clean but never checks dependency
*currency* or known CVEs — a debt category of its own. Nothing runs
`govulncheck`, and the AWS SDK bump (`f2f8dd5b`) was manual/reactive.
**Risk:** low to run; the payoff is catching a vulnerable transitive dep before
it ships.

- [ ] Run `govulncheck ./...` once and triage findings
- [ ] Add `govulncheck` to the §0.5 CI gate (alongside `gofmt`/`go vet`)
- [ ] Note a cadence for `go get -u` review (deps are otherwise updated only
      when something breaks)

---

## P1 — Reduce scraper boilerplate (highest structural payoff)

Evidence collected across the 23 committed scraper packages:

| Pattern                        | Count | Notes                                |
|--------------------------------|-------|--------------------------------------|
| Identical `SetConfig` body     | 10    | coolstuffinc, hareruya, mtgseattle, ninetyfive, starcitygames (×2), strikezone, trollandtoad, magiccorner |
| Per-scraper `printf` helper    | ~41   | tag-only difference (`[CSI]`, `[95]`, `[HA]`, …) |
| `MaxConcurrency` field         | 34    | nearly every scraper                 |
| `LogCallback` field            | 29    | matches the printf helpers           |
| `inventory` + `inventoryDate`  | 27/21 |                                      |
| `buylist` + `buylistDate`      | 16/13 |                                      |
| `DisableRetail`/`DisableBuylist`| 10/10|                                      |
| Trivial `Inventory()`/`Buylist()` getters | ~60 | one-line return     |

### 1.1 Extract an embeddable `BaseScraper`
**Why:** the table above. Each modern scraper duplicates ~30 lines of the same
scaffolding; net savings ≈ 400 LOC per scraper × 20 modern scrapers.
**Scope:** add to `mtgban/`; migrate scrapers incrementally.
**Risk:** medium — touches every scraper; do it one package at a time behind a
green build. Bantool writes to `scraper.LogCallback = ...` directly, so the
field must remain exported and reachable through embedding.

- [ ] Design an embeddable struct in `mtgban` (e.g. `BaseScraper`) holding
      `LogCallback`, `MaxConcurrency`, `DisableRetail`/`DisableBuylist`,
      `inventory`/`buylist`, `inventoryDate`/`buylistDate`, plus a
      `Printf(tag, fmt, ...)` method and a default `SetConfig`
- [ ] Decide embedding vs. composition — confirm
      `scraper.LogCallback = ...` in `cmd/bantool/main.go` still compiles
- [ ] Migrate one modern scraper as the reference (`ninetyfive` or
      `mtgseattle`)
- [ ] Roll out to remaining scrapers, one commit per package
- [ ] Delete the dead per-scraper `SetConfig`/`printf` duplicates

### 1.2 Shared HTTP-client constructor
**Why:** ~14 scrapers use a bare `http.Client{}`, 8 use
`cleanhttp.DefaultClient()`, 3 wire up `retryablehttp` + `LinearJitterBackoff`
fully. The wiring is the same modulo retry tuning.
**Scope:** helper in `mtgban` (e.g. `NewRetryClient(opts)`); opt-in.
**Risk:** low–medium — preserve each scraper's deliberate tuning (cardmarket
and cardsphere are intentionally gentle; mtgstocks rotates UA via
`uarand.GetRandom()`; hareruya does a custom 403 backoff).

- [ ] Add a configurable client constructor in `mtgban`
- [ ] Migrate scrapers that use the vanilla pattern
- [ ] Leave bespoke clients (mtgstocks UA rotation, hareruya 403 backoff) as-is
      or expose hooks (`RoundTripper`, `CheckRetry`) for them

### 1.3 Unify the per-scraper `responseChan` pattern
**Why:** ~11 scrapers define a `responseChan`/`resultChan`/`respChan` struct
carrying `cardId` + entries. Names and field order vary slightly (cardmarket
adds `ogId`, starcitygames adds `pageURL`), adding cognitive friction without
real semantic difference.
**Risk:** low — local to each scraper.

- [ ] Decide whether a `mtgban.ScrapeResult[T any]` generic is worth it, or
      whether standardizing the *name* and field order is enough
- [ ] Apply uniformly during the §1.1 BaseScraper rollout

---

## P2 — Finish in-flight standardization

### 2.1 Complete the `WorkerPool` migration
**Why:** spec/commit history shows a `WorkerPool` standardization underway.
Confirmed: `trollandtoad/`, `wizardscupboard/`, `strikezone/` still hand-roll
concurrency (no `WorkerPool` reference; bare `go func(){…}()` + channel
patterns).
**Risk:** medium — legacy scrapers are the least tested; verify output parity
against a recorded run.

- [ ] Port `trollandtoad` to `mtgban.WorkerPool`
- [ ] Port `wizardscupboard`
- [ ] Port `strikezone`
- [ ] Spot-check inventory/buylist counts before vs. after for each (record a
      pre-refactor snapshot via `bantool -format json` and diff)

### 2.2 De-duplicate `Arbit` / `Mismatch`
**Why:** `mtgban/arbit.go` has `Arbit` (lines 253–386, ~134 lines) and
`Mismatch` (lines 393–494, ~102 lines) sharing a large core: `resolveOpts`,
`filterCard`, the condition/price/qty filter loop, spread/difference math,
threshold checks, min-qty computation, and the identical profitability
formula (`(difference / (price + r.profitabilityConstant)) * math.Log(1+spread)`).
**Scope:** internal to `mtgban/arbit.go`.
**Risk:** medium — both feed money decisions; behavior must be identical.
Excellent candidate for a characterization test first (see §3.1).

- [ ] Write table tests pinning current `Arbit` and `Mismatch` outputs
- [ ] Extract the shared entry-comparison/threshold/profitability core
- [ ] Re-express `Arbit` (buylist) and `Mismatch` (reference, grade ladder) on
      top of it
- [ ] Tests stay green

### 2.3 De-duplicate the two `add()` paths in `mtgban/base.go`
**Why:** `InventoryRecord.add` (lines 15–73) and `BuylistRecord.add` (lines
103–145) share the default-fill, condition-validation, duplicate-merge, and
sort-after-append structure. They differ in strictness levels (4-tier int vs
bool), price field name, and sort direction (ascending vs descending). A
single generic over a constrained entry interface would collapse them.
**Risk:** medium — these enforce invariants every scraper depends on; the
existing `mtgban/base_test.go` covers `Add*` and must stay green.
**Scope:** `mtgban/base.go`.

- [ ] Define an interface (or generic constraint) capturing the fields the
      add path needs: `Conditions`, `Quantity`, `Price/BuyPrice`, seller name
- [ ] Collapse both methods to a single generic helper; keep the four/two
      public `Add*` entry points
- [ ] `mtgban/base_test.go` stays green; add cases for `Bundle`/`URL` edges

### 2.4 De-duplicate `CombineInventories` / `CombineBuylists`
**Why:** `mtgban/combine.go` is 93 lines; both functions share the outer
loop, the `root.Names`/`root.Entries` setup, and the result-struct assembly.
They differ only in: NM filter on inventory, `Price` vs
`BuyPrice * CreditMultiplier`, and inclusion of `Ratio` on buylist.
**Risk:** low — feeds CSV-comparison output, exercised operationally.

- [ ] Extract the shared accumulator
- [ ] Parameterize on a `(GenericEntry) -> (price, ratio, include)` callback
      per direction
- [ ] Verify CSV outputs byte-identical against a recorded run

### 2.5 Minor: collapse `json.go` Write/Read pairs
**Why:** `WriteScraperToJSON`, `WriteSellerToJSON`, `WriteVendorToJSON` (and
the `ReadSeller`/`ReadVendor` pair) repeat scaffolding for ~20 LOC of
savings. Cosmetic but trivial.
**Risk:** none.

- [ ] One private helper builds the `scraperJSON` struct from any `Scraper`;
      the three public Write functions become one-liners
- [ ] `ReadSellerFromJSON`/`ReadVendorFromJSON` share a private decoder

---

## P3 — Test coverage where it pays

> **Strategy (see SPECIFICATIONS.md §2.7).** The test pyramid is inverted by
> the MTGJSON data dependency: the matcher is a data-backed replay, while the
> money path needs no dataset yet has no tests. Order of attack: data-free
> **money-path** golden tests first (they run in CI and guard the
> highest-consequence code), then characterization tests *before* the
> §2.2–§2.4 dedups, then scraper breadth. §3.4 makes CI actually run the
> data-backed suite instead of skipping it.

### 3.1 Characterization tests for the analysis layer
**Why:** `Arbit`/`Mismatch`/`Pennystock`/`CombineInventories`/
`CombineBuylists` encode the core business logic and have no direct tests
(`mtgban/base_test.go` only covers `Add*`). They are also the target of §2.2
and §2.4.
**Risk:** none (additive).

- [ ] `arbit_test.go`: golden cases covering rate/credit, condition matching,
      spread/profitability thresholds, foil/RL/edition filters, qty math
- [ ] `Mismatch` cases incl. the `defaultGradeMap` (NM=1, SP=0.8, MP=0.6,
      HP=0.4) ladder
- [ ] `Pennystock` rarity/threshold/border-exclusion cases
- [ ] `combine_test.go`: NM-only inventory filter; `useCredit` math on buylist

### 3.2 Preprocess tests for the messiest scrapers
**Why:** the test surface today is `cardkingdom/preprocess_test.go` and
`starcitygames/preprocess_test.go` (2 of 23 scrapers). The gnarliest name
mangling — cardmarket promo tables, cardtrader id fallbacks, tcgplayer
variant splitting — has no characterization tests.
**Risk:** none (additive); needs `ALLPRINTINGS5_PATH`.

- [ ] Add `preprocess_test.go` for `cardmarket`
- [ ] Add for `cardtrader`
- [ ] Add for `tcgplayer`
- [ ] (cmd tools remain untested — acceptable)

### 3.3 `mtgmatcher` API-surface tests beyond replay
**Why:** `mtgmatcher/matcher_test.go` (196 lines) is a replay of
`matcher_test_data.json`. `api_test.go` (142 lines) is sparse. `BoosterGen`,
`GetPicksForSealed`, `GetProbabilitiesForSealed`, and the sealed reverse
index have no unit tests aimed at edge cases (reroll thresholds, balance
colors, duplicate prevention, variable contents).
**Risk:** none (additive).

- [ ] `BoosterGen` cases: weighted-sheet draw correctness; balance-colors
      paths; reroll-threshold logic
- [ ] `GetPicksForSealed` recursion: pack-of-packs, variable contents,
      decklist override paths

### 3.4 Make CI exercise the data-backed suite (not skip it)
**Why:** CI runs `go test ./... -v` with no `ALLPRINTINGS5_PATH`, so all
matcher/preprocess tests that need the dataset **skip silently** — green CI
currently certifies far less than it appears to. Cheap to fix and it unlocks
the entire existing matcher suite in CI.
**Risk:** low; the dataset is large (~600 MB), so cache it (don't fetch each
run).
**Scope:** `.github/workflows/ci.yml` (pairs with §0.5).

- [ ] Cache/restore an `AllPrintings5.json` (keyed on MTGJSON version) and set
      `ALLPRINTINGS5_PATH` for the test step
- [ ] Fail (or at least surface a skip count) when tests skip for a missing
      dataset, so the gap can't silently return
- [ ] Confirm the matcher replay actually runs in CI after the change

---

## P4 — Readability / maintainability (low urgency)

### 4.1 Extract the per-card edition switch in `adjustEdition`
**Why:** `adjustEdition` in `mtgmatcher/mtgmatcher.go` is 637 lines (line 729
to 1365). The final `default:` branch at line 1099 opens a ~245-line switch
on `inCard.Name` that hardcodes per-card edition/variation fixups for ~38
cards (Rhox, Balduvian Horde, Disenchant, Nalathni Dragon, Reya Dawnbringer,
…). This is exactly the "tables before code" pattern violated.
**Risk:** low–medium — the data is fiddly; cover with `matcher_test_data.json`
regression cases before extracting.
**Scope:** new file `mtgmatcher/edition_overrides.go` (or similar) with the
table; replace the switch with a table lookup + predicate dispatch.

- [ ] Catalog every case under the `default:` switch (line 1100–~1343)
- [ ] Design a `cardEditionOverride` entry that captures predicates
      (`isJudge()`, `isGenericPromo()`, `isArena()`, …) and the resulting
      `edition`/`variation` assignments
- [ ] Move data to a new file; `adjustEdition` becomes a table lookup
- [ ] `matcher_test.go` stays green; regenerate expectations only if
      genuinely changed behavior

### 4.2 Split `filterPrintings` helpers
**Why:** `mtgmatcher/filter.go` is 1116 lines and `filterPrintings` is a
single very long function with repetitive sub-blocks (FNM/Judge/Arena/MagicFest
patterns all follow the same shape).
**Risk:** low (mechanical), but churns blame history — only if it genuinely
helps.

- [ ] Extract `filterByPromoType`, `filterByEditionTag` helpers for the
      repeated FNM/Judge/Arena/release/launch blocks
- [ ] Keep the date-constant decisions in one place

### 4.3 Decide whether to split `variants.go`
**Why:** `mtgmatcher/variants.go` is 4651 lines of pure data. Hard to
navigate, but splitting will churn blame.
**Risk:** low (mechanical).

- [ ] Decide: split by set family / era, or leave as-is and rely on editor
      folding / search

### 4.4 Document — and consider enforcing — the global-backend contract
**Why:** the `defaultBackend` singleton is immutable-after-load and
unsynchronized by design, but that contract is implicit **and currently
violated**. The reference consumer (`mtgban-website`) exposes an
authenticated `/api/load/datastore` endpoint (`api.go:872`, `main.go:1076`)
that re-runs `mtgmatcher.LoadDatastore` on a live server, reassigning the
unsynchronized global *struct value* `defaultBackend` while HTTP handlers
concurrently read it — a data race. Notably, the same consumer hot-swaps its
*scraper* sets safely via `atomic.Pointer[[]mtgban.Seller]` (`load.go:37–44`);
the matcher backend has no equivalent because `cardBackend` is an unexported
value type.
**Risk:** documenting is none; making it swappable is low–medium (touches the
load path + all `api.go` accessors).

- [ ] Add a doc comment on `SetGlobalDatastore`/`defaultBackend` stating the
      read-only-after-load invariant (now also in `AGENTS.md` "Gotchas" and
      `SPECIFICATIONS.md` §2.1)
- [ ] **Decide between two real options** (the prior "just document it" stance
      is insufficient now that a live reload path exists):
      (a) store `*cardBackend` behind an `atomic.Pointer` and have
      `LoadDatastore`/`SetGlobalDatastore` publish a new pointer atomically,
      mirroring the consumer's scraper-set pattern; accessors load the pointer
      once per call. Cheapest correct fix.
      (b) keep the value global but require reloads to quiesce reads (maintenance
      window), and document that loudly.

---

## Explicitly NOT doing (verified clean / out of scope)

- `go.mod` indirect markers — checked, they are correct (`go mod tidy` clean).
- `go build ./...` — passes.
- `go vet` on core packages — clean.
- ~~Adding a mutex to the matcher backend — unnecessary given the read-only
  contract.~~ **Revised (§4.4):** the read-only contract is *violated* by the
  consumer's live `/api/load/datastore` reload, so an atomic-pointer swap (not
  a mutex) is now a justified option, not pure over-engineering.
- Per-store data-table reorganization in `mtgmatcher/callbacks.go` — already
  cleanly organized as per-set callbacks; size is data, not complexity.
- Backend load-time patches (FBB/4BB language overrides, STA/PLST strips,
  SLD per-number fixes, CMB1/CMB2 playtest renames) — already cleanly
  localized in a switch-on-set-code; no refactor needed.
- `extensions/`, untracked tools/modules — out of review scope per §0.3.

---

## Suggested sequencing

1. **P0** in one housekeeping PR: format (§0.1) **+ the CI gofmt/vet gate
   (§0.5) in the same PR** so it can't regress, + gitignore (§0.2) + WIP triage
   (§0.3, incl. the `amazon/api.go` and `mvpsportsandgames` decisions) +
   supabase constants (§0.4) + the bantool credential-bug fix (§0.6).
2. **§3.1** characterization tests **first and broad** — and critically,
   **extend `base_test.go` to pin the `entries[0] == NM` ordering + sort
   direction (and a `PO`-zeroing `Mismatch` case) before touching `add()`.**
   The NM-first contract flows `add()` → `Arbit`/`CombineBuylists`, so an
   `add()` change can break the consumers with green Arbit tests that never
   exercise a multi-condition slice.
3. Then the money-path dedups **in dependency order**: **§2.3** add()
   unification → **§2.2** Arbit/Mismatch → **§2.4** Combine — *not* the
   original 2.2-first order, because Arbit's NM assumption is downstream of
   `add()`'s sort. Each behind the green characterization suite.
4. **§1.1** BaseScraper rollout (one PR per package, reference `ninetyfive`),
   then **§1.2** HTTP-client helper, then **§2.1** legacy WorkerPool migration
   (easier once BaseScraper scaffolding exists).
5. **§4.1** edition-override table extraction (data-driven win on
   `mtgmatcher`); **§4.4** decide the atomic-swap vs. document-only call.
6. **§3.2**, **§3.3**, **§2.5**, **§4.2** opportunistically; **drop/defer §1.3
   and §4.3** (pure churn).
