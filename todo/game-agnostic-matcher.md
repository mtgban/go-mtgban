# Plan: Game-agnostic mtgmatcher (extract `mtgmatcher/magic`)

Make `mtgmatcher` identify cards **regardless of game**. Today the core matcher
is ~4,500 lines of Magic *data* plus ~200 Magic *logic* dispatch points baked
directly into the matching pipeline. Goal: a thin game-agnostic core that calls
**game hooks if present**, with all Magic data + logic relocated to a
`mtgmatcher/magic` package (and Lorcana to `mtgmatcher/lorcana`).

**Explicit goals:**
1. Thin game-agnostic core; Magic data+logic in `mtgmatcher/magic`, Lorcana in
   `mtgmatcher/lorcana`; auto-dispatch via registered hooks.
2. **Per-game test suites.** A Lorcana matcher-replay suite mirroring Magic's
   `matcher_test_data.json`, with `go test ./...` exercising **every** available
   datastore (not just Magic) — the prerequisite that makes goal #3 safe.
3. **Delete `SimpleSearch`.** Every scraper — Magic *and* Lorcana — matches via a
   single `mtgmatcher.Match()`, with **no per-game branching in scraper loops**.
4. **`mtgban-website` changes near-zero** — keep the package-level API and every
   symbol it references; the only required change is a blank import to register
   a game.

**Invariant across every phase:** the *existing* suite stays green at the end of
each step — `go test ./mtgmatcher/... ./mtgban/...` plus the preprocess tests —
and the Magic `matcher_test_data.json` replay passes with **no `-u`
regeneration**. This is a pure, behavior-preserving refactor: a phase that
*forces* a regen means matching drifted — stop and fix, don't accept the diff.
Tests beside moved code (`variants_test.go`, `editions_test.go`, …) move *with*
it into `magic/` and keep their assertions unchanged. The §6 Lorcana suite is
*added* coverage, never a substitute for keeping Magic green.

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done

---

## 0. Where we are (evidence)

**Coupling reality** (master, flat `mtgmatcher/`):
- **Pure Magic data** — clean to move: `editions.go` (`EditionTable`, ~500),
  `variants.go` (`VariantsTable`/`allVariants`/`multiPromosTable`, 4,651 lines /
  123 KB), `table.go` data (`setAllowedForTokens`, `sldJPNLangDupes`,
  `productsWithOnlyFoils`, …), the Magic half of `replacer.go`
  (`replacerStrings`), the `PromoType*`/`Finish*`/`FrameEffect*` constants.
- **Magic logic with hardcoded names** — must become hooks: `Match()` prefilter
  (`"Red Herring"`, `"Bind // Liberate"`, …), `adjustName()` (Ikoria/Hawkins/
  Un-set renames + `AlternateProps`), `adjustEdition()` (**90+ hardcoded
  card-name switch**, lines 729–1365), `filterPrintings()` + `filterCards()`
  (50+ set-code/tag dispatches), `callbacks.go` (`promoTypeElements`,
  `simple/complex/numberFilterCallbacks`, ~50 filter funcs), the ~40
  `InputCard.is*()` predicates in `card.go`, and the set-code switches in
  `(AllPrintings).Load()` (`backend.go:445–567`).
- **Genuinely game-agnostic** — stays: `MatchId`, `output`, `hasPrinting`, the
  `Backend` storage struct, and the `Normalize/Equals/Contains` *functions*
  (the *replacer data* they use is Magic-specific).

**The only game-dispatch today** is the `io.TeeReader` try-Magic-then-Lorcana
fallback in `LoadDatastore` (`backend.go:1300`). After load, **nothing records
which game is active** — `Card`/`CardObject`/`Backend` have no game field.
Lorcana already matches via a *different* path (`SimpleSearch`, name+number+
foil), not the Magic `Match()` pipeline — so the two games already have
divergent strategies.

**Prior work to carry forward.** Two stale branches are the *same* refactor
twice: `claude/goofy-jones` (Apr, 14 commits) and **`newbackend` (May, 27
commits — newer & more complete)**. Both: export `cardBackend → Backend`
(public), add `Backend` *methods* to the api/match/filter/callbacks/lorcana
functions, add `NewBackend`, and move the **loaders** into `mtgmatcher/mtgjson`
+ `mtgmatcher/lorcana`. `newbackend` additionally makes `Load` return `*Backend`
directly and `SetGlobalDatastore(*Backend)`. **Neither moved the Magic
matching data/logic or added a hook mechanism** — that's the new work here.

> **Key dependency:** the instance-method refactor (`Backend` receiver methods)
> is the *enabler*. `callbacks.go` filters call back into `GetSet`/`MatchInSet`/
> `SearchEquals`; once those are methods on `*Backend`, the `magic` package can
> import core for the `Backend` type and call them **without an import cycle**
> (magic → core only; core → magic never; magic is wired in at runtime by the
> loader). This is why §1 must land before §3–4.

---

## Target architecture

```
mtgmatcher/            game-agnostic core
  backend.go             Backend struct + storage + generic lookups (methods)
  match.go               thin Match() that dispatches to Backend.rules
  rules.go               GameRules interface + registry (RegisterGame)
  api.go / card.go       MatchId, output, generic InputCard, string helpers
mtgmatcher/magic/        the Magic GameRules + all Magic data & logic
  loader.go              AllPrintings → *Backend (load-time Magic transforms)
  rules.go               implements GameRules (adjustName/edition/filter*)
  editions.go variants.go callbacks.go tables.go replacer.go
mtgmatcher/lorcana/      the Lorcana GameRules (SimpleSearch-style match)
```

Core carries a `GameRules` on the `Backend` and **calls each hook only if the
game supplies it** — the "do all these calls automatically if present" goal:

```go
// core
type GameRules interface {
    Normalize(string) string                              // name folding (replacer)
    Prefilter(*InputCard)                                 // pre-match name fixups
    AdjustName(*Backend, *InputCard)                      // canonical/face/flavor/reskin
    AdjustEdition(*Backend, *InputCard)                   // edition resolution + per-card fixups
    FilterPrintings(*Backend, *InputCard, []string) []string  // narrow candidate sets
    FilterCards(*Backend, *InputCard, map[string][]Card) []Card // variant/number/promo
}
// ONE pipeline for every game. Match() ALWAYS runs the shared skeleton;
// game-specific behavior comes ONLY from the registered hooks (a nil hook is a
// no-op). No per-game override, no SimpleSearch: Lorcana resolves through the
// same Match() as Magic — its hooks are simply thin (no edition aliases, no
// variant table, no promo types), and the skeleton's name+number+foil
// disambiguation (what SimpleSearch did) carries it.
func (b *Backend) Match(in *InputCard) (string, error) {
    // id lookup -> Prefilter -> AdjustName -> AdjustEdition -> Printings4Card
    //   -> FilterPrintings -> FilterCards -> verdict   (hooks skipped if nil)
}
```

Registration follows the **`database/sql` / `image.RegisterFormat` idiom** —
register an implementation under a string name, activate it with a blank import.
Core never imports the game packages (no cycle); the consumer chooses which
games to compile in:

```go
// core — name is a separate string arg, exactly like sql.Register(name, driver)
type Game struct {
    Detect func(io.Reader) bool              // cheap format probe (MTGJSON vs Lorcana)
    Load   func(io.Reader) (*Backend, error) // attaches the game's GameRules to the Backend
}
func RegisterGame(name string, g Game)        // panics on duplicate name, like sql.Register
func LoadDatastore(io.Reader) error           // probes registered Detect funcs (image.Decode style)
func Open(name string, r io.Reader) (*Backend, error) // explicit by name (sql.Open style)

// mtgmatcher/magic — self-registers in init()
func init() {
    mtgmatcher.RegisterGame("magic", mtgmatcher.Game{Detect: detect, Load: Load})
}

// consumer (bantool / website) — blank import activates the games
import (
    "github.com/mtgban/go-mtgban/mtgmatcher"
    _ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
    _ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)
```

Datasets are self-describing, so **auto-detect** (`LoadDatastore` probing
`Detect` funcs in registration order) is the primary path — this is `image`'s
model more than `sql`'s required-name `Open`; keep `Open(name, …)` as the
explicit escape hatch. **Gotcha:** a consumer that blank-imports *no* game gets
"no game registered" — by design (core alone is inert), so return a clear error,
and make sure `cmd/bantool` + the website add the blank imports during cutover.

---

## Phases

### 1. Reconcile the prior branch onto master  *(prerequisite, biggest mechanical lift)*
**Why:** `newbackend` is the instance-`Backend` foundation everything else needs,
but it's ~since May 7 and master has ~67 commits of `mtgmatcher` churn on top.
**Risk:** high — conflicts across `backend.go`/`api.go`/`mtgmatcher.go`/`filter.go`.

- [ ] Pick `newbackend` (newer/more complete than `goofy-jones`); rebase/replay
      it onto current master, resolving against the SLD/variants/editions
      changes that landed since
- [ ] Keep `defaultBackend` as a thin global wrapper over an instance so the
      existing global-function API (and the website consumer) stays source-compatible
- [ ] Land **only** the mechanical "Backend instance + methods + loader
      sub-packages" change, no behavior change; `go test ./mtgmatcher/...` green
      with no `matcher_test_data.json` regen
- [ ] This also resolves ADR-0002 (the unsynchronized global) — an instance
      backend behind an `atomic.Pointer` swap becomes natural

### 2. Introduce the `GameRules` seam (no code moves yet)
**Why:** establish the hook boundary while Magic logic is still in core, so the
move in §3–4 is mechanical and individually testable.

- [ ] Define `GameRules` (+ optional `GameMatcher`) and a `Backend.rules` field
- [ ] Wrap the *existing* core functions as a `magicRules` value that lives in
      core for now and is attached by the loader — `Match()` calls
      `b.rules.AdjustEdition(...)` etc. instead of the free functions
- [ ] Behavior identical; tests green. This is the riskiest *behavioral* step —
      do it under the full matcher suite and diff a recorded `Match` run

### 3. Move the pure Magic **data** to `mtgmatcher/magic`  *(the easy 20% you asked for)*
**Why:** clean moves, low import risk; immediately shrinks core.

- [ ] `editions.go` (`EditionTable`) → `magic/`; core edition resolution is now a
      `magicRules.AdjustEdition` concern (see §4)
- [ ] `variants.go` (123 KB) → `magic/`; the single `filter.go:785`
      `VariantsTable[...]` lookup becomes a `FilterCards` hook call
- [ ] `table.go` Magic data (`setAllowedForTokens`, `sldJPNLangDupes`,
      `productsWithOnlyFoils`, `missingP[AE]LPtags`) → `magic/`; keep the
      game-neutral `LanguageCode2LanguageTag` family in core
- [ ] `PromoType*`/`Finish*`/`FrameEffect*` consts → `magic/` (re-export the few
      genuinely shared ones, e.g. finish suffixes used by `output()`)

### 4. Move the Magic **logic** to `mtgmatcher/magic`  *(the hard 80%)*
**Why:** this is where the matcher actually becomes game-agnostic. Needs §1's
instance methods so callbacks can reach `b.MatchInSet`/`b.GetSet` cycle-free.

- [ ] `adjustEdition()` (the 90+ card switch) → `magicRules.AdjustEdition`. Pair
      with the §4.1 refactor-plan item (extract the per-card switch to a table)
      so it moves as data, not a 600-line method
- [ ] `adjustName()` + the `Match()` prefilter → `magicRules.AdjustName`/`Prefilter`
- [ ] `filterPrintings()` / `filterCards()` → `magicRules.FilterPrintings`/`FilterCards`
- [ ] `callbacks.go` (**hardest** — tight `Card`/`InputCard` coupling +
      `GetSet`/`SearchEquals`/`MatchInSet` callbacks) → `magic/`; resolved by §1
- [ ] The ~40 `InputCard.is*()` Magic predicates → `magic/` (they encode Magic
      tag/promo semantics). Keep generic accessors (`isFoil`/`isEtched`) in core
- [ ] Magic set-code transforms in `(AllPrintings).Load()` (`backend.go:445–567`,
      `skipSet`, sealed-foil detection, rarity/commander maps) → the
      `magic/loader.go` load path

### 5. Cross-cutting: normalization, Lorcana, registry
- [ ] **Replacer**: make `Normalize` use the game's registered replacer
      (`b.rules.Normalize`); core keeps the generic `Equals/Contains` shells.
      Falls back to identity when no game/replacer is set
- [ ] **Lorcana**: wrap `SimpleSearch` as `lorcana`'s `GameMatcher.Match`; decide
      whether to keep the exported `SimpleSearch` (strikezone/starcitygames call
      it) or route them through `Match`
- [ ] **Registry + auto-detect**: `RegisterGame` from `magic`/`lorcana` `init()`;
      `LoadDatastore` probes registered `Detect` funcs (replacing the hardcoded
      TeeReader fallback). Stamp the active game onto the `Backend`

---

### 6. Multi-game matcher test harness  *(goal #2 — prerequisite for §7)*
**Why:** Magic has `matcher_test.go` replaying `matcher_test_data.json` against a
real AllPrintings (`ALLPRINTINGS5_PATH`). Lorcana has **no** matcher test — its
matching (`SimpleSearch`) is unverified. Before routing Lorcana through the
unified `Match()` and deleting `SimpleSearch` (§7), stand up a Lorcana replay
proving `Match()` resolves Lorcana cards; and make `go test ./...` exercise
**every** game's datastore, not just Magic's.
**Risk:** none (additive); needs each game's dataset env var.

- [ ] Generalize the replay harness so each game package carries its own
      `matcher_test.go` + `testdata/<game>_test_data.json` (input → expected
      UUID/error), gated on that game's dataset env var (`ALLPRINTINGS5_PATH` for
      Magic, e.g. `LORCANAJSON_PATH` for Lorcana), skipping cleanly when unset —
      mirroring the Magic harness and its per-game `-u` regenerate flag
- [ ] Seed the **Lorcana** `testdata`: the name+number+foil cases that go through
      `SimpleSearch` today (incl. ambiguous/alias cases) → expected Lorcana
      UUIDs, so the unified `Match()` is proven *before* §7 deletes it
- [ ] `go test ./...` now runs every game's suite automatically; CI provisions
      all datasets (ties to refactor.md §3.4 — without the env vars they skip)
- [ ] Seed the Lorcana suite as early as the dataset + seam allow so it also
      guards the §3–§5 moves, not only §7

### 7. Delete `SimpleSearch`; unify every scraper on `Match()`  *(goal #3)*
**Why:** `SimpleSearch(name, number, foil)` is a parallel match path with ~14
call sites across ~9 packages — Lorcana (`tcgplayer/lorcana`, `lorcanaindex`,
`cardtrader` `LorcanaFoil`, the strikezone/scg Lorcana branches) **and** as a
generic simple matcher (`coolstuffinc`, `ninetyfive`, `cardmarket`,
`trollandtoad/generic`). It forces per-game branching in scraper loops
(`strikezone sz.game == GameLorcana`, `starcitygames GameLorcana`). One
`Match()` for all games removes that split.
**Risk:** medium — `Match()` must reproduce SimpleSearch's name+number+foil
disambiguation across printings (it already half-does this in `filterCards`);
verify each repointed scraper's match rate doesn't drop.

- [ ] Make the unified `Match()` skeleton resolve a name+number(+foil)
      `InputCard` across printings with thin/absent edition — absorb
      SimpleSearch's job (this *is* the game-neutral core of the pipeline)
- [ ] Repoint the ~14 callers to build an `InputCard` and call `Match()`:
      strikezone, starcitygames, coolstuffinc, ninetyfive, cardtrader,
      tcgplayer/{lorcana,lorcanaindex}, cardmarket, trollandtoad/generic
- [ ] Remove the `GameLorcana` branches from the strikezone/starcitygames Load loops
- [ ] Delete `SimpleSearch` (api.go); its disambiguation now lives inside `Match()`
- [ ] Regression: compare per-scraper match counts before/after each repoint

---

## Consumer compatibility (mtgban-website must change ~nothing) — goal #4

The website calls a broad slice of the **package-level** API — measured top
usages: `GetUUID` (67), `CardObject` (47), `GetSet` (26), `Title` (16),
`GetAllSets`, `MatchId`, `GetUUIDs`, `SearchEquals`, `Match`, `ExternalUUID`,
`Printings4Card`, `InputCard`, `CardReleaseDate`, `AllPromoTypes`, `AllNames`,
`AliasingError`, and the Magic **`PromoType*` constants** (`PromoTypeBoosterfun`,
`PromoTypePromoPack`, `PromoTypeBuyABox`, `PromoTypePrerelease`,
`PromoTypeThickDisplay`). It does **not** use `SimpleSearch`. Hard constraints:

- [ ] **Keep every one of those symbols at `mtgmatcher.` package level.** The
      instance refactor (§1) keeps thin global wrappers; the `PromoType*`
      constants are Magic data that moves to `magic/`, so **re-export them from
      core** (alias) — `mtgmatcher.PromoTypeBoosterfun` must still resolve.
- [ ] **`LoadDatastore(io.Reader) error` keeps its exact signature** — the
      website's `/api/load/datastore` reload path depends on it.
- [ ] The **only** required website change: add the blank import(s)
      (`import _ ".../mtgmatcher/magic"`, plus `/lorcana` if it serves Lorcana)
      so a game registers; without one, `LoadDatastore` → "no game registered".
- [ ] Build the website against the refactored module before merge as the
      acceptance gate for "changed very little".

## Risks & mitigations

- **The per-game replay suites are the safety net (§6) — see the Invariant.**
  Every phase ends with green Magic *and* Lorcana suites (`ALLPRINTINGS5_PATH` +
  the Lorcana dataset) and **no `-u` regeneration** of `matcher_test_data.json`;
  a forced regen is the tripwire that behavior drifted. CI doesn't set the
  datasets (refactor.md §3.4), so run them locally each phase.
- **Import cycles** (`callbacks.go` ↔ core helpers): the §1 instance methods are
  the unlock — `magic` imports core for `Backend`; core never imports `magic`.
- **Scale**: variants.go 123 KB + callbacks.go 33 KB + editions.go 32 KB are big
  mechanical moves; do one file per commit, build green between each.
- **Pipeline is Magic-shaped.** The shared skeleton (edition→set→number→promo)
  is itself Magic semantics; Lorcana uses the `GameMatcher` full-override rather
  than no-op hooks. Don't force Lorcana through the Magic skeleton.
- **Don't pick `goofy-jones`** — it's the older twin of `newbackend`; carrying
  both forward wastes the rebase.

## Suggested order

§1 (rebase `newbackend`) → §2 (seam, behavior-preserving) → §3 (data moves,
mechanical) → §4 (logic moves, one subsystem per commit, callbacks last) → §5
(normalize + registry) → §6 (multi-game test harness; seed the Lorcana suite as
early as the dataset allows so it also guards §3–§5) → §7 (delete `SimpleSearch`,
unify scrapers on `Match()`). §1 and §4 are the heavy lifts; §7 is gated on the
unified `Match()` from §2/§4 being game-neutral **and** the §6 Lorcana suite.
Everything between is mechanical and individually test-gated. The
website-compatibility checklist is a merge gate, not a phase.
