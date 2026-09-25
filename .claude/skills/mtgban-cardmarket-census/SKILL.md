---
name: mtgban-cardmarket-census
description: Walk every Cardmarket product of one game (or all seven on Cardmarket) through the Index scraper's own resolution and grade each landing against the datastore's Cardmarket ids, CardTrader's links and the published prices. Use when asked to census or audit a game's Cardmarket catalog, "do the same for" another game or rarity, check whether Cardmarket prices land on the right printings, find Cardmarket products priced onto the wrong card or left unpriced, check CardTrader's links for a game, or collect datastore gaps to hand to datastore-gen. Covers fetching the inputs, the production-faithful catalog walk, reading the grading report, classifying each finding into the layer that fixes it, sizing a rule before writing it, and proving the fix with a per-product walk diff.
---

# mtgban Cardmarket census

The method lives in
[`docs/agents/cardmarket-census/README.md`](../../../docs/agents/cardmarket-census/README.md),
with `harnesses.md`, `census.sh` and `grade.py` beside it. Read that file now
and follow it. This repo keeps one copy of the method there rather than a
duplicate here, so every agent working in it reads the same document.
