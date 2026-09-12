# ADR-0003: Atomically publish immutable backend snapshots

**Status:** Accepted
**Date:** 2026-09-12
**Supersedes:** ADR-0002's unsynchronized global publication

## Context

The matcher exposes both independent backends and package-level convenience
functions. Assigning the global Backend struct while those functions read it
races and can combine fields from different generations. Changing only the
publication mechanism is insufficient for reports that perform several global
lookups: separate calls can legitimately observe different generations.

Magic's identification callbacks must also remain scoped to their Backend.
The List's Game Day exception still searched the global name index before
looking up its results in the supplied backend.

The website has removed its runtime reload endpoint. This decision establishes
the library's publication contract for callers that replace a datastore; it
does not restore or require a website reload endpoint.

## Decision

Publish a shallow copy through atomic.Pointer[Backend]. Each package-level
operation loads that pointer once and calls methods on the captured backend.
An uninitialized pointer behaves as an empty datastore. A missing sealed index
is built on the copy before publication, without mutating the supplied value.

GlobalDatastore returns a shallow copy of the captured snapshot. Its maps,
slices, card pointers and rules remain shared and immutable; it is not a deep
clone. Reassigning fields on the copy does not change the published headers.

Arbit and Mismatch capture one backend per report, with ArbitOpts.Backend for
callers owning an explicit backend. Pennystock also captures once. Custom
callbacks and result rendering requiring the same generation must use that
backend explicitly. Concurrent publishers use last-store-wins semantics;
freshness ordering belongs to the consumer.

All Magic identification lookups use their supplied backend, including the
token lookup in Backend.IsGenericPromo. InputCard.IsGenericPromo remains a
global convenience method, so game rules call the backend method. Exported Magic
Has*Printing convenience wrappers continue to use the default intentionally.

## Options considered

| Option | Benefit | Cost |
|---|---|---|
| Keep startup-only assignment | No publication machinery | Runtime replacement remains unsafe |
| Guard the backend with an RWMutex | Safe coordinated access | Every read needs locking; returned pointers still need immutability |
| Atomic immutable snapshots | One atomic load per operation, old readers finish safely | Requires explicit scope across calls and immutable nested data |
| Remove the global API | Explicit ownership everywhere | Breaks existing scrapers and consumers unnecessarily |

## Consequences

Publication is safe concurrently with reads and with other publishers of an
immutable input. Existing package-level signatures remain available. Keeping
an old snapshot alive also retains its maps until the last reader releases it.
Atomic publication does not synchronize mutation of exposed maps or changes to
the global logger. It does not coordinate a consumer's separate price caches
or choose which of two concurrently loaded datasets is newer.

## Action items

- [x] Publish snapshots atomically and retain empty-datastore behavior.
- [x] Scope The List's Game Day lookup to its backend.
- [x] Capture the backend for an entire analysis report.
- [x] Add concurrent publication and mid-report replacement regressions.
- [x] Run datastore-free CI tests under the race detector.
