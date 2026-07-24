# ADR-0003: Profitability Index formula & calibration

**Status:** Accepted — code corrected; one open sub-decision (`k` default)
**Date:** 2026-06-28
**Deciders:** Maintainer (Vittorio Giovara)

## Context

`Arbit` / `Mismatch` rank opportunities by a Profitability Index (PI). A design
spec defines it; the code had silently drifted on two terms, and the choice of
log base has a money-facing side effect on threshold calibration.

## Decision

PI is, at both call sites (`Arbit` and `Mismatch`):

    PI = (Difference / (Sell Price + k)) · log10(1 + Spread) · sqrt(Units)

- **log10**, not natural log.
- **sqrt(Units)**, applied only when Units > 1 — not the fourth root.
- **k** = `ProfitabilityConstant`, a denominator stabiliser that stops cheap
  cards from dominating (spec reference value 10).

## Alternatives considered / rationale

- **Fourth root vs sqrt of Units.** The code used `Pow(qty, 0.25)`; the spec
  says square root. Unambiguous spec violation → fixed to `math.Sqrt`. (This
  changes ranking between rows of differing quantity.)
- **ln vs log10 of Spread.** `log10(x) = ln(x)/ln(10)`, a constant ~0.434×
  rescale — **order-preserving for ranking** but it shifts the absolute
  `MinProfitability` gate by ~2.3×. Chose log10 to match the spec; **the gate
  must be retuned per consumer** (done: website `MinProfitable` 4.0 → 1.74).

## Consequences

- Library and the website's inline copy (`upload.go`) now agree.
- Any consumer comparing PI to a fixed threshold must rescale it; consumers
  that only *sort* by PI are unaffected (the log base is monotonic).
- **Open:** the library default for `k` is **0** (no stabilisation) while the
  spec reference is **10**. Production is unaffected (the website passes its own
  2 / 10), but an unconfigured caller silently gets the un-stabilised formula.

## Action items

1. [ ] Decide the library `ProfitabilityConstant` default (0 vs 10) — refactor.md §0.7.
2. [ ] Add a golden test pinning log10 + sqrt + the chosen `k` — refactor.md §3.1.
