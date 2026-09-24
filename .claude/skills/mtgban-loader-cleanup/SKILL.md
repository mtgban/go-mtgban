---
name: mtgban-loader-cleanup
description: Change or audit go-mtgban's per-game datastore loaders, or what datastore-gen publishes for them, and prove the change did only what it claims. Use when asked to clean up or align the loaders, make them behave the same, move a loader onto different published field names, add, rename or drop a datastore field, check what the datastores publish against what the loaders read, or show that a loader or builder refactor changes no backend. Covers the published-versus-decoded audit, the backend-equivalence harness with per-field masks and locally built datastores, replays through the TCGplayer, Cardmarket and Star City Games paths, and the four-step datastore-gen, go-mtgban, site, datastore-gen order a field change needs.
---

# mtgban loader cleanup

The method lives in
[`docs/agents/loader-cleanup/README.md`](../../../docs/agents/loader-cleanup/README.md),
with `harnesses.md` and the harness scripts beside it. Read that file now and
follow it. This repo keeps one copy of the method there rather than a
duplicate here, so every agent working in it reads the same document.
