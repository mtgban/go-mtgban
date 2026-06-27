# ADR-0005: abugames Solr request shape

**Status:** Accepted — implemented in `abugames/api.go`
**Date:** 2026-06-28
**Deciders:** Maintainer (Vittorio Giovara)

## Context

abugames is the slowest scraper. Its catalog is a Solr core
(`data.abugames.com`) of ~213k grouped products; the scraper pages through it
with `group=true&group.field=product_id`. Timing the endpoint showed the cost
is request *shape*, not concurrency: a page costs ~3.8 s / 6 MB, of which ~2.9 s
is Solr computing `group.ngroups` — a value the per-page code never reads.

## Decision

Keep `group=true` (the scraper needs per-product condition docs), but reshape
the per-page request:

- **Drop `group.ngroups=true` from page requests**; keep it only on the initial
  count call (`GetTotalItems`). Measured 3.76 s → 0.89 s/page, identical data.
- **Add an `fl` field list** of the 17 fields actually decoded (docs carry 56).
  Measured 6.08 MB → 0.66 MB/page.
- **Raise `rows` 200 → ~1000** (1,068 → ~214 requests).

Net ≈ 10× faster, ~9× less data.

## Options Considered

### Solr `cursorMark` (textbook deep-paging fix)

| Dimension | Assessment |
|-----------|------------|
| Complexity | High |
| Fit | **Incompatible with `group=true`** |

`cursorMark` is the right tool for `start/rows` deep paging (which is O(start)),
but it cannot combine with result grouping. Using it would mean dropping
grouping, querying flat docs sorted by `product_id`, and **re-grouping
client-side** — a real rewrite. Rejected for now; the ngroups/fl/rows fix
captures nearly all the win at near-zero risk.

### Raise concurrency only

Rejected as a band-aid: each request stays expensive (~3.8 s) and the server
does heavy per-request grouping work, so more parallel deep queries risks
contention / rate-limiting. Fix request shape first; `defaultConcurrency = 4`
barely matters once a page is sub-second.

## Consequences

- **Easier:** abugames runs in minutes, not tens of minutes; far less data to
  JSON-decode per worker.
- **Harder / residual:** deep paging (`start` ~213k) is still O(start) (~3.9 s
  cold at the tail), bounded to ~214 pages at rows=1000. Erasing it needs the
  ungrouped + `cursorMark` rewrite above, tracked separately.

## Action items (refactor.md §0.8)

1. [ ] `GetProduct`: strip `group.ngroups`, add `fl`; keep ngroups on
   `GetTotalItems`.
2. [ ] `maxEntryPerRequest` 200 → ~1000.
3. [ ] Verify inventory/buylist counts vs. a recorded run.
