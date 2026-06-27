# ADR-0002: Global, immutable-after-load, unsynchronized matcher backend

**Status:** Accepted — with an open follow-up (live-reload race)
**Date:** 2026-06-28 (amended 2026-08-08 for the game-agnostic matcher)
**Deciders:** Maintainer (Vittorio Giovara)

> **Amendment note.** The decision stands: the global matcher backend is still
> a plain package variable, read without locks, immutable by contract after
> load. What changed is the *justification*. The backend type is now exported
> and built outside the package, so the original "unexported type" safety
> argument no longer applies, and the contract has widened to bind callers as
> well. Superseded statements are marked where they occur.

## Context

`mtgmatcher` resolves identity against a `Backend` built from a game's
datastore. Loading is now a `database/sql`-style registry rather than a single
hardcoded format: a game package registers a `GameLoader`
(`func(io.Reader) (*Backend, error)`) under a name from its `init()` via
`RegisterGame`, and a consumer activates games with blank imports —
`mtgmatcher/magic`, `mtgmatcher/lorcana`, `mtgmatcher/riftbound`, or
`mtgmatcher/games` for all three. `LoadDatastore` auto-detects the format by
trying every registered loader in registration order and installs the first
that succeeds as the global backend; `LoadDatastoreFile` wraps it over a path;
`Open(name, reader)` loads one named game explicitly and returns the `*Backend`
*without* installing it globally. Only the Magic datastore is large — the
MTGJSON `AllPrintings` file, around 600 MB; the Lorcana and Riftbound payloads
are small JSON files. *(Superseded: the original text described
`LoadAllPrintings` with "a Lorcana fallback"; neither exists today.)*

Match calls are read-only and extremely hot — every scraped row hits them
across many goroutines.

## Decision

Keep the backend a **process global, immutable after load, with no locking.**
Concurrency-safety rests on a *contract* in two phases: a game's loader builds
a `Backend` and mutates it freely while constructing it (filling the maps,
calling `IndexSets`, attaching rules with `SetRules`), then publishes it
through `SetGlobalDatastore`; after publication it is only read. This buys
lock-free, allocation-free reads on the hottest path.

**The contract is now enforced by documentation alone.** `Backend` is an
exported type. Construction happens *outside* the core package by design: all
three loaders are a `Load(io.Reader) (*mtgmatcher.Backend, error)` in their own
game package, assembling a `mtgmatcher.Backend` field by field and attaching
their hooks through the exported `SetRules`. They must live outside core, or
core would have to import the game packages and cycle. Only the `rules` field
remains unexported. Nothing in the type system prevents a caller from
assigning to a published backend's fields. *(Superseded: the original text
justified safety by `cardBackend` being an unexported value type that only the
package could construct or assign.)*

**The contract also binds consumers, not just the package.** `Backend.UUIDs` is
`map[string]*CardObject`, and `GetUUID` hands the same pointer to every caller
— the field's own comment states that the values are shared with every caller
of `GetUUID` and must never be modified after the load completes.
`GetUUIDsInSet` and `GetSealedUUIDsInSet` return slices that alias the
backend's per-set index. In the Lorcana and Riftbound loaders, every card
sharing a name also shares one `Printings` backing array. This aliasing is
precisely what makes reads allocation-free as well as lock-free, and it is the
part of the contract most easily broken from the outside: one caller writing
through a returned pointer or slice corrupts the datastore for the whole
process.

A second piece of unsynchronized global state arrived with the registry: the
`registeredGames` slice appended by `RegisterGame`. It is safe for the same
reason and by the same kind of contract — registration happens only from game
packages' `init()` functions, before `main` runs and before any goroutine can
read the slice. `RegisterGame` panics on a nil loader or a duplicate name, and
it is not safe for concurrent or post-`init` use.

Finally, a `Backend` with no rules attached degrades gracefully instead of
panicking: `Match` returns `ErrDatastoreEmpty` when `b.rules` is nil, the
id-lookup path nil-checks the rules before calling `MissingPromoTag`, and
`GetSetByName` skips the `AdjustEdition` step. A hand-built or partially
initialized backend is therefore usable for plain lookups.

## Alternatives considered

- **Mutex/RWMutex around the backend.** Rejected for the steady state: it adds
  contention to the hottest read path to defend against a write that, by
  contract, never happens after startup. The argument is more load-bearing now
  that `GetUUID` returns shared pointers — the reads it protects allocate
  nothing at all.
- **Pass the backend explicitly** instead of a global. Originally rejected as a
  large API change across every `mtgmatcher` entry point and every caller —
  **since adopted in API form.** Nearly every entry point now exists as a
  method on `*Backend` (`b.Match`, `b.MatchId`, `b.GetUUID`, `b.GetSet`,
  `b.SearchEquals`, and most of `api.go`), with the package-level function kept
  as a thin wrapper that delegates to `defaultBackend`. A handful of accessors
  were never given a method and still read the global directly —
  `GetUUIDsInSet`, `GetSealedUUIDsInSet`, `AllPromoTypes`, `AllNames` and
  `HasPrinting` — so the conversion is not quite complete. `Open` returns a
  `*Backend` that is never installed globally, so explicit-backend usage is
  fully supported today and the global is a compatibility default rather than
  the only mode. Since the global holds exactly one game at a time, a process
  that needs several games must use per-game backends anyway.

## Consequences

- **Easier:** lock-free, allocation-free reads; trivial call sites; and,
  through `Open`, several independent backends in one process.
- **Harder / risk:** the read-only contract is **currently violated.** The
  reference consumer (`mtgban-website`) exposes an authenticated
  `/api/load/datastore` endpoint that re-runs `LoadDatastore` on a *live*
  server while HTTP handlers read the global — a data race. The precise
  mechanism is worth stating, because the original diagnosis has been
  overtaken: `SetGlobalDatastore` performs `defaultBackend = *b`, a non-atomic
  multi-word copy of a large struct, and that copy races with lock-free
  readers reading the same variable. A reader crossing the copy can observe a
  torn value — a mixture of the old and new struct headers, one game's map
  paired with another game's slice — which is worse than a stale read, because
  the fields it sees were never consistent with each other. There is no `sync`
  or `sync/atomic` use anywhere in `mtgmatcher`, in core or in any game
  package.
  *(Superseded: the original text attributed the missing safe swap to the
  backend being an unexported value type, contrasting it with the consumer's
  `atomic.Pointer[[]Seller]` scraper hot-swap. A consumer that keeps its own
  `*Backend` from `Open` can already do exactly that swap; what has no safe
  swap is the package global.)*
- **Peak memory:** `LoadDatastore` buffers the entire datastore with
  `io.ReadAll` so it can retry each registered loader over the same bytes. On
  the ~600 MB Magic file that is a peak-memory cost the old single-format
  streaming loader did not pay. `Open` avoids it, since a named loader
  consumes the reader directly.

## Action items (the open decision)

1. [ ] **Option A (preferred):** store the global behind an
   `atomic.Pointer[Backend]`; `LoadDatastore` / `SetGlobalDatastore` publish a
   new pointer atomically; the package-level wrappers load it once per call.
   Lock-free reads *and* a safe live swap. Now largely mechanical: most
   accessors are already a method on `*Backend` fronted by a wrapper that
   reads the global exactly once, and the few that still touch
   `defaultBackend` directly need the same one-line change.
2. [ ] **Option B:** keep the value global but require reloads to quiesce reads
   (maintenance window), and document that loudly.
3. [ ] **Option C (available today, no core change):** the consumer holds its
   own `*Backend` obtained from `Open`, swaps it behind its own
   `atomic.Pointer`, and never reloads the global at all. This is the only
   shape that also works for a process serving more than one game.
4. [ ] Until resolved, do not add new in-process reload paths. This has held:
   the global is still published only by `SetGlobalDatastore` and by
   `LoadDatastore`, which calls it — `LoadDatastoreFile` is a path wrapper over
   `LoadDatastore`, and `Open` never touches the global at all.
