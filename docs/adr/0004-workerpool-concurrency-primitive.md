# ADR-0004: `WorkerPool` is the single concurrency primitive

**Status:** Accepted
**Date:** 2026-06-28
**Deciders:** Maintainer (Vittorio Giovara)

## Context

Scrapers fan out hundreds–thousands of HTTP requests and CPU-bound `Match()`
calls. Early scrapers hand-rolled goroutine+channel pools with subtly different
shapes (buffering, cancellation, result draining) — which is where concurrency
bugs hide.

## Decision

New fetch code uses `mtgban.WorkerPool[T,R]` — bounded workers consuming a work
channel, each pushing results to a results channel drained by a **single
consumer** on the caller's goroutine. Context cancellation **stops dispatch but
lets in-flight workers finish**, so partial results are still consumed. Don't
hand-roll goroutine/channel pools.

## Alternatives considered

- **`errgroup` / `x/sync/semaphore`.** Fine primitives, but each call site
  still re-implements result draining and the "finish in-flight on cancel"
  policy. A single typed pool standardises both.
- **Unbounded goroutines.** Rejected: stores rate-limit and ban; bounded
  concurrency (typically 2–8) is required.

## Consequences

- **Easier:** uniform cancellation and result handling; the single-consumer
  drain means the `consume` callback needs no locking (e.g. `AddRelaxed` is
  serialised for free).
- **Harder:** three legacy scrapers (`trollandtoad`, `wizardscupboard`,
  `strikezone`) predate it and still hand-roll concurrency — migration is
  refactor.md §2.1.

## Action items

1. [ ] Port the three legacy scrapers to `WorkerPool` — refactor.md §2.1.
2. [ ] Preserve each scraper's deliberate retry/backoff tuning when migrating.
