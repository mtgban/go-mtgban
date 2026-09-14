---
name: mtgban-noise-sweep
description: Find and fix bad card matches in go-mtgban. Three detectors - refusal lines in a CI run ("unknown card name", "unknown variant", "aliasing detected", "duplicate entry"); suspicious cross-vendor spreads captured from the /arbit, /reverse and /global pages on the mtgban site; and a scraper buying and selling one card at nearly the same price (bantool's "bought at N% or more of their asking price" warning). Use when asked to clean up scraper log noise, get a game/vendor to 0 errors, triage refusals, audit spreads or arbitrage that look too good, investigate a suspicious buylist-vs-retail price, or work through the ci-sweep backlog. Covers ranking targets, replaying the feed through the production path, classifying each shape into the right layer, and landing the fix.
---

# mtgban scraper noise sweep

The full method lives in
[`docs/agents/noise-sweep/README.md`](../../../docs/agents/noise-sweep/README.md),
with `detectors.md` and `harnesses.md` beside it. Read that file now and
follow it — this repo keeps one copy of the method there, not a duplicate
here, so every agent working in this repository (Claude Code, Codex, or a
human) reads the same document rather than two that can drift apart.
