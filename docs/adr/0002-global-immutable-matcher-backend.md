# ADR-0002: Global, immutable-after-load, unsynchronized matcher backend

**Status:** Accepted — with an open follow-up (live-reload race)
**Date:** 2026-06-28
**Deciders:** Maintainer (Vittorio Giovara)

## Context

`mtgmatcher` resolves identity against a backend built from a ~600 MB MTGJSON
dataset (`LoadAllPrintings`, with a Lorcana fallback). The backend
(`defaultBackend`) is a package-global value, populated once at startup by
`LoadDatastore` / `SetGlobalDatastore`. Match calls are read-only and extremely
hot — every scraped row hits them across many goroutines.

## Decision

Keep the backend a **process global, immutable after load, with no locking.**
Concurrency-safety rests on a *contract*: the datastore is loaded once during
startup, then only read. `cardBackend` is an unexported value type, so only the
package can construct or assign it. This buys lock-free, allocation-free reads
on the hottest path.

## Alternatives considered

- **Mutex/RWMutex around the backend.** Rejected for the steady state: it adds
  contention to the hottest read path to defend against a write that, by
  contract, never happens after startup.
- **Pass the backend explicitly** instead of a global. Cleaner, but a large API
  change across every `mtgmatcher` entry point and every caller.

## Consequences

- **Easier:** lock-free reads; trivial call sites.
- **Harder / risk:** the read-only contract is **currently violated.** The
  reference consumer (`mtgban-website`) exposes an authenticated
  `/api/load/datastore` endpoint that re-runs `LoadDatastore` on a *live*
  server, reassigning the unsynchronized global while HTTP handlers read it — a
  data race. The same consumer already hot-swaps its *scraper* sets safely via
  `atomic.Pointer[[]Seller]`; the matcher backend has no equivalent only
  because it is an unexported value type.

## Action items (the open decision — refactor.md §4.4)

1. [ ] **Option A (preferred):** store the backend behind an
   `atomic.Pointer[cardBackend]`; `LoadDatastore`/`SetGlobalDatastore` publish a
   new pointer atomically; accessors load it once per call. Lock-free reads
   *and* a safe live swap, mirroring the consumer's scraper-set pattern.
2. [ ] **Option B:** keep the value global but require reloads to quiesce reads
   (maintenance window), and document that loudly.
3. [ ] Until resolved, do not add new in-process reload paths.
