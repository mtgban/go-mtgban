# Loader cleanup

How to change the per-game datastore loaders, or what datastore-gen
publishes for them, and prove the change did only what it says. Written after
the 2026-09 alignment of the eight non-Magic loaders (#768 to #823).
`harnesses.md` beside this file has the templates, and `gen_dump.py`,
`dump.sh` and `jdiff.py` are the backend-equivalence harness itself.

Sync go-mtgban, datastore-gen and mtgban-website before measuring anything.
Measure against the published datastores (`b2://mtgban-datastore/<game>/<game>.json.xz`),
or against the copies your `.env` names once they're refreshed. A stale local
build measures a file nobody reads.

## 1. Audit what is published against what is read

A loader's struct lists what the loader reads, not what the builder writes.

- List every key the published file carries, at every level, with how many
  records fill it. Set that against every `json` tag in the loader package
  (template A). A key nothing decodes is one of three things:
  - **upstream's** (Riot's gallery, LorcanaJSON): leave it alone;
  - **a fact a loader should read**: FaB and Lorcana stamped every card
    English while their builders tagged 21 exclusives with a language;
  - **ours and unread**: a candidate to drop (§4).
- Tabulate which `Card` fields each loader's literal sets (template A). A
  difference should follow what each game publishes. One that doesn't is the
  finding.
- Before calling a field dead, or safe to change, find every reader:
  - go-mtgban outside the loader: scrapers read `IsOversized`, `Identifiers`
    and `PromoTypes`;
  - mtgban-website: `git grep` on its `origin/master`;
  - datastore-gen's own tools and self-checks.

## 2. Prove a refactor changes no backend

- Run `dump.sh` on master and on the branch, then diff. It dumps every
  fixture's `Backend` and hashes each published datastore's. Run master twice
  first and check the two runs match: Lorcana's `AllSets` once came out in
  map order.
- Gate each commit on its own, in a temporary worktree:
  - build, vet, gofmt (1.25), revive and staticcheck;
  - the affected packages' tests;
  - the dump, diffed against the previous commit's.

  That way each commit message's "identical" is checked, not only the
  branch's.
- Prove a change that should move one field by masking that field on both
  sides (`ZZ_MASK_*`, one per field in `gen_dump.py`; no `reflect`). The
  masked dumps must match exactly, and the unmasked diff names what moved
  (`jdiff.py`).
- Move fixtures onto a new shape in place; don't add test helpers to shared
  files. The dump shows a moved fixture loads to the same backend. A variant
  a test derives with `strings.Replace` can legitimately differ, so read it.
- Load a builder change with both loaders. Point `P` at a directory where
  that game's `<game>.json` is the local build and the rest link to the
  published files.

## 3. Prove a matching change on the scrapers' own paths

A hand-written `Match()` call measures a path no scraper takes. Replay what
each scraper sends, on master and on the branch, and grade every changed
line by the vendor's own id.

- **TCGplayer names**: `TestReplayCatalogNames`. Set one `<GAME>_PATH` at a
  time: the last game replayed overwrites `REPLAY_OUT`. The uuid carries the
  product id to grade by.
- **Cardmarket**: template C. It walks the id map and the day's product list
  through `resolveMapped`, as `Index.walkCatalog` does, without the
  CardTrader bridge. A product the bridge answers never reaches the name
  path, so a no-bridge diff is the larger set. Record the raw answer before
  twin marking as well: `twinsAmong` can hide a raw change that the bridged
  `yugiohWorded` path still reaches.
- **Star City Games**: template D. It resolves every product of the catalog
  export through `resolveProduct`: one download, needing `SCG_API_KEY`.
- A listing field that only some scrapers send (a language, a rarity tag)
  can only move their listings. Name those scrapers, and where you ran no
  replay, say in the PR why the change can only help.
- Every reader of a changed field counts. `IsOversized` feeds core's
  oversize gate and cardmarket's `plausiblePrinting`, and cardmarket's
  resolver asks for "Oversized" from a product's rarity.

## 4. Changing what a datastore publishes

Moving a published field takes four steps across three repositories, in
this order:

1. **datastore-gen publishes the new name beside the old.** go-mtgban master
   must load that file identically (§2, with `P`). Publish it: the `publish`
   workflow runs at 12:37 UTC, or can be dispatched with `game:`.
2. **The go-mtgban loader switches.** Its CI reads the published file, so it
   fails until step 1 is published. After the publish, re-run the whole
   workflow, not only the failed jobs, so the cache job re-keys on the new
   B2 object.
   - Locally, refresh `~/src/datastore-gen/output` with datastore-gen's
     `.github/scripts/fetch-datastores.sh` (pass `OUTPUT_DIR` from a
     worktree); the pre-push hook reads it.
   - To push before the publish, give the worktree a gitignored `.env` that
     points that game at the local build.
3. **The site releases a go-mtgban tag carrying the switch.**
4. **datastore-gen drops the old name.** A publish asks `<game>.mtgban.com`
   to reload, for every game in `RELOAD_GAMES`. A drop published before
   step 3 is therefore read at once by the site's older loader.

Every step taken early loses data without an error: `AddSealed` skips a
product whose set the file lacks, and a loader reading a dropped id finds
none. Riftbound's move to the common names ran in exactly this order:
datastore-gen#94, go-mtgban#814, the site release, then datastore-gen#95.
Put the order in each PR's body, and open the PRs ready for review.
