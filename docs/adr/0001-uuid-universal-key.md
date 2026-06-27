# ADR-0001: The mtgmatcher UUID is the universal key

**Status:** Accepted
**Date:** 2026-06-28
**Deciders:** Maintainer (Vittorio Giovara)

## Context

go-mtgban ingests inventory/buylist data from ~23 stores, each with its own
naming, set labels, foil/finish conventions, promo tags, and internal IDs.
Downstream analysis (arbitrage, price comparison, sealed EV) only works if "the
same card" from two different stores collapses to one identity. Two broad
designs exist: let each scraper emit its own normalized identity, or route
every store record through one central identity resolver.

## Decision

Every record keys on a single canonical identity — the **mtgmatcher UUID** (the
MTGJSON UUID, or a Lorcana equivalent). Scrapers are **thin translators**: they
parse store text into an `InputCard` and call `mtgmatcher.Match()` /
`MatchId()`. All correctness about *what card this is* lives in `mtgmatcher`
(its variant/edition tables, normalization, and the `Match()` pipeline), never
in the scraper.

## Alternatives considered

- **Per-scraper identity** — each scraper resolves its own canonical name/set.
  Rejected: N copies of the hardest logic, drifting independently; a matching
  bug would need fixing in many places and would silently differ across stores.
- **Match on `(name, set, number)` tuples.** Rejected: promos, variants, foils,
  languages, and reprints make a tuple ambiguous; the UUID already encodes all
  of it.

## Consequences

- **Easier:** adding a store is mostly preprocessing — identity is free; a
  matching fix in `mtgmatcher` corrects every store at once.
- **Harder:** `mtgmatcher` is a large, central, data-heavy package and the
  single point of failure for correctness — hence its regression replay
  (`matcher_test_data.json`) and the "tables before code" rule.
- **Rule of thumb:** if a card matches wrong, fix it in `mtgmatcher` (usually a
  data table), not in the scraper. New-set/promo support is almost always data
  (`variants.go`, `editions.go`, `callbacks.go`), not logic.

## Action items

1. [ ] Keep new identity logic in `mtgmatcher` data tables, not scrapers
   (enforced by review; see SPECIFICATIONS.md §2.5 and AGENTS.md).
