# Log: Magic token support and the TCGplayer id chase

Written for whichever agent picks this up next. Covers the whole arc from
"tokens are silently dropped" through "the id overrides are one table" and
lists what is still open. Read the PRs themselves for full detail (commit
messages carry the measurements); this is the map, not the territory.

## The discovery that started it

`adjustTokens` nilled `Tokens` for every set failing a 55-entry
`okForTokens` allowlist, dropping **8,432 of 9,072** token entries across
254 sets — only 640 across 79 sets survived. The cost was invisible because
every vendor fails on a dropped token *silently*: instrumenting Card
Kingdom's quiet `ErrUnsupported` exit alone showed **3,874 of 150,089 price
rows (2.6% of its whole Magic feed)** vanishing with nothing in any log.
SCG's own workaround failed the same way. Log audits kept missing this
because there was nothing to find in a log — the rows just never appeared.

That's the standing lesson of this whole effort: **the biggest losses never
show up in a log.** Every fix below started from measuring a live vendor
feed or the datastore directly, not from reading error output.

## What shipped, in dependency order

### PR #186 — `mtgmatcher/magic: Carry every Magic token` (merged, `ea5576ee`)

The core change. Files tokens under mtgjson's own `tokenSetCode` (a
synthesized `"<set> Tokens"` set) instead of the 55-entry allowlist, which
is deleted. A token whose name a real card also answers to gets filed as
`"<name> Token"` (106 names collide; the plain bucket keeps the card).
Magic goldens stayed byte-identical (4 hand-edited entries, never `-u`).

Measured live on seven vendors, each run base → branch → base again to
control for feed drift (two false alarms taught this discipline — SCG
churned 137/154 on the *same build* 20 minutes apart once, unrelated to the
change): **~13,600 rows recovered, zero real losses**, across CK/ManaPool/
CardTrader/Cardmarket/SCG/Hareruya/CSI.

Full detail and the earlier measurement work: `[[project-magic-tokens]]`
in agent memory, or read the PR body directly.

### PR #375 — oversized-printing detection (merged, `ffc95038`)

Found while chasing a reported bug: `Undercity (Oversized)` from Commander
Legends: Battle for Baldur's Gate priced as the *ordinary* dungeon token.
Root cause: the old "is this oversized-and-refused" rule matched by
`Contains(edition, "Commander")`, and "Commander Legends: Battle for
Baldur's Gate" trips that by accident.

Rewrote it to ask the datastore directly — does this **card** have an
oversized printing anywhere, not does this **edition**'s name contain a
magic word. Measured on the full oversized corpus (1,157 rows): 0 lost,
160 newly resolving.

### PR #377 — quiet the TCGplayer catalog-walk noise (merged, `c4e33a53`)

`cmd/tcgid4scryfall` (see below) raised 639 warnings walking the whole
catalog; 285 were inserts the datastore never carries at all (helper
cards, countdown-kit letter cards, Jumpstart theme cards, World
Championship biography/blank cards) — refused by name now. 95 were
emblems the catalog spells backwards, `"Emblem - X"` instead of `"X
Emblem"` — read that shape and try both spellings. 639 → 76.

### PR #383 — the Oversize Cards shelf, two ways broken (merged, `fcb06054`)

TCGplayer sells every oversized card from **one** group standing for a
dozen real sets, so the matcher was narrowing to *every* printing of the
same card, and preprocess was throwing away the wording that named the
right one (`Interplanar Tunnel (Planechase Anthology)` vs `(Planechase
2012)`, same number, different sets) in favor of a useless catalog number.
Fixed both ends. All 14 of the shelf's unmatched products started
resolving; measured 1,257 → 1,311 on a 2,196-row corpus, 0 lost.

### PR #384 — Japanese promo token sheets (merged, `347fe53b`)

Six sets print a "Japanese Promo Tokens" sheet (`WDMU`, `WFIN`, `WMKM`,
`WMOM`, `WONE`, `WWOE`). Sold two ways: beside every other set's promos
(wording names the set — already handled) or under the set it came with
(wording names *nothing*, only the edition says which sheet — wasn't
handled, so `March of the Machine`'s sheet was unmatched). One rule now
covers both, and any future seventh sheet.

### PR #387 — a token edition says which of two names it means (merged, `0d4dc671`)

A token whose name carries no "Token" of its own is filed under a key
that adds one (`"battoken"`), leaving the *plain* key to whatever card
normalizes the same way. Normalization drops the hyphen, so
Unsanctioned's `Bat-` owned the bucket `Bat` asks for: `Bat | Bloomburrow
Tokens | 10` priced an Unsanctioned card. Ask the token key first when
the edition names a set of tokens. Separately: the "token leaked here"
refusal fires on edition *or* variation, but its exception only ever
checked the edition — `Angel Token | Dominaria United` worked,
`Angel | Dominaria United | Token` didn't. Fixed both; 188,751-row replay,
0 lost, 37 moved off `Bat-`/`Rhino-` onto the right token.

**This PR had a self-caught regression worth remembering.** The first
version of the token-key fix skipped a name-fixup step that was
responsible for triggering the edition filter further down — so a token
with exactly one printing got served for *any* token edition a listing
named, the same bug shape the PR was fixing, just relocated. An
adversarial review workflow caught it before merge. The lesson: when a
fix changes *which branch* of a match resolves a name, check what else
that branch was quietly doing besides the thing you meant to fix.

### PR #388 — the Standard Showdown shelf names its own year (merged, `e09d8940`)

Same shape as Oversize Cards: one shelf, many sets, this time keyed by
*year* (`PSS1`/`PSS2`/.../`PW26`/`PSPL`/`PPRO`...) with a hardcoded
per-year list that goes stale every year TCGplayer adds a new one. Tried
the generic fix first — "ask which promo set carries this card" — and it
was **much worse**: moved 2,739 answers and lost 7, because the automatic
answer overrode a fallback that was already correct for most of the
shelf. The hardcoded list, updated for the current year, was the right
answer despite needing yearly upkeep. Worth remembering when the shelf
breaks again next year: don't re-attempt the generic fix without
re-running that same measurement.

### The `tcgid4scryfall` tool and the upstream id chase

`cmd/tcgid4scryfall` walks the *entire* TCGplayer catalog (~114k Magic
products), runs each through `Preprocess` + `Match`, and for anything that
resolves compares the matched printing's `tcgplayerProductId` against
Scryfall's. Anywhere they disagree (including "Scryfall has none at all")
goes in a CSV: `name,set,cn,scryfall_id,old_tcgplayer_id,new_tcgplayer_id`.
A full walk takes 4–8 minutes once the tool builds cleanly.

**Two CSVs have been generated and handed to the user for upstream
submission** (Scryfall, since MTGJSON mirrors it): the first ~583-row
batch (mid-September), and further batches after each round of matcher
fixes reduced the noise. As of the last check, **uptake is partial**: the
`WDMU` (Dominaria United Japanese Promo Tokens) sheet's 5 ids all landed
and verified live on Scryfall; most of the rest (the bulk `RVR`/`MOM`/
`BLB`/`MID` token sheets, `WWOE`/`WMKM`/`WMOM`) had not, last measured.
**Re-check before trusting any "still missing" claim** — this is
Scryfall's queue, not ours, and it moves on its own schedule. The
datastore's own build date (`meta.version` in `allprintings5.json`)
doesn't mean anything moved; check a few known-missing ids against the
live Scryfall API directly.

When re-running the walk, exclude/expect two known non-bugs rather than
chasing them again:
- **The Secret Lair x MSCHF Zeta Set** (fixed in #581, see below) used to
  be 86% of all warnings — if it reappears at that volume, the edition
  mapping regressed.
- **"The List" mismatches**: cards that don't exist yet at a given List
  number because Scryfall hasn't scanned that specific reprint in — not a
  bug on our side, nothing to fix, just don't count it against "is the
  walk clean."

### PR #581 — the Zeta Set edition mapping (merged, `d05a82af`)

One TCGplayer group, `"Secret Lair x MSCHF: The Zeta Set"`, was never
mapped to the set the datastore calls `"The Zeta Set"` — **305 of 353**
warnings in one walk, from one missing `EditionTable` entry. Checked
first whether this was a "cards missing upstream" case (per the user's
usual framing for things like The List) — it wasn't: the datastore
already carried the right `tcgplayerProductId` for every affected
printing. Purely our own gap. One line; 353 → 48.

Also checked, and it's worth remembering as the answer if this question
comes up again: **there is no generic "Secret Lair x BRAND" prefix
resolver** anywhere in the matcher. The one generic Secret-Lair rule that
exists (`Contains(edition, "Secret Lair")` in `aliasEdition`) is a
deliberate no-op, there specifically so odd-named drops fall through to
`EditionTable` rather than get mangled by decoration-trimming. A future
"Secret Lair x <next brand>" needs its own `EditionTable` entry, the same
way this one did.

### PR #583 — `productOverrides`, one table for every id-keyed fix (merged, `cb85e7ef`)

Three commits, landed together:

1. **Six more catalog-Number-lies-by-id cases** — `tcgplayer/preprocess.go`
   now has a `productOverrides map[int]struct{ Edition, Number string }`
   keyed by TCGplayer's own product id, for listings where the catalog's
   `Number` field counts a position within a promo wave rather than the
   real collector number (Zidane/Wood Elves/Tom-Bert-William/Turtle Lair
   from WPN-and-Gateway/Unique-and-Miscellaneous Promos; two Mythic
   Edition Construct tokens misnumbered `001`/`003` against the real
   sheet's `R1`/`G3`).

2. **Every prior id-override table folded into that one map.** `cardIDs`,
   `tokenIDs`, and a hand-written `if edition == "L13" { if productID ==
   X { variant = Y } }` special case all answered the same question —
   "which printing does this id really name" — from three different
   points in `Preprocess`, for no reason that turned out to matter. All
   26 of the old entries were checked against the live catalog before
   and after the fold: same 26 uuids, byte for byte.

3. **Two stale hardcoded refusals retired, six more one-offs folded.**
   `Glissa, the Traitor`'s "Mirrodin Besieged" wording used to be
   refused outright; turned out the datastore has it right (P11 #96),
   but the *wording itself doesn't disambiguate* it from the other
   oversized Glissa (`OCM1`) — the shelf just always fell back to OCM1
   regardless of what was asked. Pinned by id. `Bruna, Light of
   Alabaster`'s refusal checked `variant == "Commander 2018"`, a
   condition no live product has ever tripped — TCGplayer puts that text
   in the *group*, not the variant, and the switch never looked there.
   Confirmed via a full-catalog scan (114k products) that no live
   listing carries the wording the refusal checked for, before deleting
   it as dead code. Five more `cardName → fixed edition+number` one-offs
   on the Unique-and-Miscellaneous-Promos shelf folded the same way,
   each verified to be the *sole* listing under that name on that shelf
   before folding (folding a name rule that covers multiple ids, e.g. a
   foil and nonfoil variant, into one id entry would silently drop the
   others — checked for exactly that before touching any of them).

**Pattern for future one-off audits**: grep `preprocess.go` for
`case "<Card Name>":` blocks that set a fixed edition/number unconditionally
— those are exactly as narrow as an id override already, just spelled as a
name switch. The ones that stay as name rules on purpose are the ones
*conditioned on variant text* in a way a single id can't capture (Serra
Angel's two different real editions depending on wording, for instance) —
don't fold those; a name rule generalizes to future listings, an id entry
doesn't.

### PR #605 — combined entities for two-sided token sheets (open, not yet merged, `derived-token-pairs`)

A second thread of work, distinct from the id-chase above: mtgjson carries
a `tokenProducts`/`tokenParts` field on many token cards, never parsed
here, naming the TCGplayer product for a two-sided token *sheet* — one
physical card printing two different token faces back to back, sold as
one product. TCGplayer sells every distinct pairing as its own listing
(Double Masters 2022's Eldrazi Scion alone pairs with nine different
partners, nine separate products); mtgjson never mints a combined entity
for these the way it does for the fixed pairings (`layout:
"double_faced_token"`, e.g. Commander Legends: Battle for Baldur's Gate's
Undercity // The Initiative) — only the two independent single-faced
tokens, linked solely by that unparsed field. Three commits:

1. **`mtgmatcher/magic/tokenpairs.go` (new) mints the combined entity.**
   `deriveTokenPairs` walks every `tokenProducts` entry, dedupes by the
   unordered pair of uuids, and mints one `Card` per pairing surviving an
   exclusion ladder: a face mtgjson never gave its own uuid (a bare
   `faceId`/`faceName` stub), a self-pair, a uuid the file no longer
   carries, an id claimed by more than one distinct pairing, an id a real
   printing's own `tcgplayerProductId` already owns. 7,146 raw two-part
   entries → **3,094 derived entities** (today's datastore). Resolves
   **only by its own TCGplayer product id** (`Match(&InputCard{ID:
   "278823"})`), never by name — deliberate: the combined name is not
   unique inside its own set (a single-faced token can pair with several
   different partners on one sheet, ~280 same-set name collisions
   measured), so a name-based path would alias against the wrong pairing.
   Never appended to `set.Cards`/`set.Tokens`, never added to
   `Hashes`/`CanonicalNames`, no dedicated index either — each entity
   carries `Identifiers["derivedTokenPair"] = "true"`, filtered on demand
   from the existing `UUIDs` map (an earlier draft *did* add a
   parallel `AllDerivedUUIDs`/`SetDerivedUUIDs` index; removed after
   noticing it was written but read nowhere outside its own test — one
   fact, one place). Golden matcher corpus untouched. Went through a
   hostile adversarial review before landing (findings and fixes in the
   commit itself) — one real bug caught (a diagnostic counter
   double-walked two overlapping card lists, 2x inflating every count;
   the *derived output* was unaffected, already deduped by the pairing
   maps) plus several limitations documented rather than silently
   shipped (below).

2. **`tcgplayer/tcgplayer.go`'s `Market.Load` priced these wrong until
   now.** It looks up a card's skus by `Identifiers["mtgjsonId"]` and
   sends every one through, filtered by a check against the card's own
   `tcgplayerProductId` — silently inert for a token with no id of its
   own, which is exactly what a two-sided sheet's *face* is. Verified
   directly against the live sku file (`tcgskus.json`, 112,660 entries):
   Eldrazi Scion's own sku list spanned 9 distinct product ids, every one
   of its 9 partners', none its own. Fixed with a paired edit: the
   existing per-card walk now skips a derived-claimed id, and a second
   walk prices each set's derived entities from their own ids directly
   (looked up via `tokenPairPartA`, falling back to `tokenPairPartB` —
   mtgjson files a pairing's skus under either or both faces with no
   guaranteed convention), scoped to the specific finish each uuid
   represents. Re-verified against the same live file after the fix:
   Eldrazi Scion's absorbed-product count goes from 9 to 0, and the
   derived entity for id 278823 resolves its own 5 matching skus.

3. **`tcgplayer/preprocess.go` refused these products twice over,
   `tcgplayer/index.go` never got to price them at all.** Two separate
   dead refusals, both predating anywhere to send a match: `isToken &&
   strings.Contains(product.CleanName, "Double")` caught it by name
   ("Boar // Eldrazi Scion Double-Sided Token"), and further down
   `strings.Contains(RawProductNumber(product), "//")` caught whatever
   the first missed by its printed number ("A // B"). Both now resolve
   by id first (`mtgmatcher.ConvertID`), refusing only what the datastore
   doesn't carry. Caught the *second* refusal only by testing against the
   **live API** rather than trusting the first passing fixture — all
   three test ids (Eldrazi Scion // Boar, Illusion // Skeleton, Goat //
   Food) failed with "duplicate" on the first attempt, from the check
   that came *first* in the file, not the one already fixed. `index.go`'s
   price guide scraper gets the same second-walk treatment as
   `Market.Load`, built once per `Load()` call rather than per set.

4. **`cardkingdom/preprocess.go` now resolves CK's own two-sided token
   listings to these derived entities where CK sells the same physical
   pairing.** Built and verified against one fetch of CK's real
   pricelist (151,082 products, never repeated); see the Card Kingdom
   entry under "What's still open" below for the full writeup — **751 of
   CK's 1,039 two-sided token listings (72%)** now resolve exactly, via
   CK's own `scryfall_id` (or, for one shape it never carries one for, a
   sku-derived set/number) anchoring one face and its listing's own
   wording disambiguating among that face's known pairings, never from
   the name alone. (Figure corrected twice since first measured: 791 by
   commit 5's foil-resolution bug, then 788 → 751 by the hostile-review
   pass's `TokenPairIndex` collision fix - see "Hostile review" below.
   Both corrections moved the number *down*: each was a case this
   feature had been silently getting away with, not a new capability
   lost.)

5. **Hardened the matching primitive for a second vendor, and fixed a
   foil bug it surfaced in CK's own version along the way.** The
   matcher (`TokenPairIndex`/`NormalizeTokenFace`/`SplitTokenPairName`/
   `MatchTokenPairing`/`MatchTokenPairingBySetNumber`) lives in
   `mtgmatcher/magic/tokenpairs.go` itself, appended to the same file
   that mints what it reads, from commit 4 on - not a second
   similarly-named file of its own. Nothing about the matcher is
   CK-specific, and it is not generic enough to belong in the parent
   `mtgmatcher` package. Investigating whether Star City Games needed
   the same support (below) meant actually exercising it from a second
   caller for the first time, which is what surfaced the real bugs
   below.

   Building the equivalent SCG check surfaced a real, live bug in
   commit 4's own CK code: `MatchTokenPairing` returned a bare
   `InputCard{ID: id}` with no `Foil` field, so a *foil* CK listing
   resolving to a derived pairing silently got the *nonfoil* uuid's
   price - **52 of 72 foil CK two-sided listings (72%) were silently
   mispriced** this way before the fix (measured against the same CK
   snapshot, comparing the resolved uuid's own `Foil` bit against CK's
   `is_foil`). `MatchTokenPairing`/`MatchTokenPairingBySetNumber` now
   take a `foil bool` and refuse (return `""`) rather than silently
   substitute the wrong finish's price when the derived pairing's own
   uuid was never sold in the requested finish - the same "sheet was
   never sold foil" doctrine `TestPreprocessTokenFoilRefused` already
   pins elsewhere in this file, just enforced inside the shared matcher
   itself so every caller gets it for free rather than having to
   remember the check. Re-verified against the full CK corpus through
   the real `Preprocess()`+`Match()`
   path: 0 finish mismatches, the (at-the-time) corrected 788/1,039
   figure above (3 of the previous 791 were the wrongly-priced foil
   cases, now correctly refused - `Preprocess()` errors go from 14 to
   17). **That 788 was itself still wrong** - see "Hostile review"
   below for the `TokenPairIndex` collision bug this pass didn't catch,
   correcting the figure again to 751.

**Known limitations, documented rather than fixed here** — read before
trusting `tcgplayerProductIds` or the pricing change blindly:
- A handful of the 34 entities carrying more than one usable id look, on
  inspection, like they may be mtgjson mis-mappings rather than confirmed
  duplicate listings of the same physical product (ELD's Goat/Food ids
  run suspiciously sequential with a gap).
- 77 pairings get no usable id at all because mtgjson already gave their
  id to one of their own single-faced tokens as that token's own
  `tcgplayerProductId` — those 77 keep absorbing the two-sided product's
  price exactly as before this PR; not regressed, not fixed either.
- ~128 ids were refused as claimed by more than one pairing; most (127)
  are the same physical pairing duplicated across sibling set entries
  (`DMU`/`WDMU`, `MID`/`MIC`) rather than genuinely different objects —
  refusing is the safe direction, but a future pass could recover most of
  them with a smarter merge.
- Cross-set pairings (a real shape: 36 of 39 distinct set-code
  combinations in the corpus are a direct `parentCode` edge, e.g. `MIC` →
  `MID`) display a `Number` field mixed from two different sets' own
  numbering — harmless, never reachable by number, but potentially
  misleading to a display consumer.
- **The pricing change (commit 2) has not been run against the live API
  in a drift-controlled base → branch → base comparison** — verified
  statically against the real local sku file and confirmed correct at
  the sku level, but that is not the same thing as a live measurement,
  and this project holds every other live-pricing change to that
  standard before trusting a row-count delta.

## What's still open

Updated 2026-09-14 (evening), after landing PR #605 (three commits: mint,
price, unrefuse — see above). The prior pass's items are compressed below
to one line each now that they're resolved and merged; full detail is in
the PR itself and the "What shipped" section above.

**Resolved this cycle, merged**: Gala Greeters' 10 missing languages
(#591); `PUNK` unskipped, collision-checked safe (#592); `Ertai, the
Corrupted` PLST aliasing (#599); `OAFR`/`OCLB` unskipped plus a real
`FilterPrintings` regression caught and fixed narrowly (#596); the
`crossSetProductIDs` id-collision guard, plus two upstream id-collision
reports (Dungeon of the Mad Mage, Sakura-Tribe Elder — still need
submitting to MTGJSON/Scryfall, see prior entry above for the exact
scryfallIds) (#598). The "Front Cards" `skipSet` suffix was investigated
and deliberately left alone (already safe via an independent
`isUnsupported` guard; unskipping only adds ambiguity).

- **Combined two-sided token entities — open, not yet merged (#605, see
  "What shipped" above for the full writeup).** The live pricing change
  in it (commit 2) still needs a drift-controlled live-API run before
  trusting it; several other limitations are documented in the PR rather
  than fixed (multi-id entities that may be mtgjson mis-mappings, 77
  pairings with no usable id, ~128 refused ids that are mostly
  recoverable, cosmetic cross-set numbering).

- **Cross-vendor name-based matching, Card Kingdom — investigated and
  fixed 2026-09-14 (#605).** One fetch of CK's own published pricelist
  API (`https://api.cardkingdom.com/api/v2/pricelist`, the same endpoint
  the production scraper already uses, 151,082 products - not a
  page-by-page crawl, and not repeated) was enough to build and verify
  this without needing a second vendor visit.

  CK does sell two-sided token pairings - **1,039 listings** matching
  `cardkingdom/preprocess.go`'s own existing trigger condition (a name
  containing `" // "` or `" - "`, paired with "Token" in the name or a
  token-wrapped SKU prefix), e.g. `"Angel Token - Cat Token"` (Commander
  2018, sku `TC18-003`).

  A first pass at measuring overlap with #605's derived entities found
  none, by checking CK's `scryfall_id`-resolved uuid for the
  `derivedTokenPair` marker directly - but that marker is set *only* on
  the synthetic combined-uuid entity, never on either real face's own
  identifiers, and CK's `scryfall_id` always resolves to a real single
  face (a derived entity has no scryfallId of its own). That check was
  structurally guaranteed to read zero regardless of any real overlap.
  Rebuilding it correctly - checking membership in the set of uuids that
  appear as *either* half (`tokenPairPartA`/`tokenPairPartB`) of any
  derived pairing, then requiring both CK's own `scryfall_id` face and
  the second name segment from CK's own listing (normalized) to jointly
  match one derived entity's two parts - found **752 of the 1,039
  (72%)** resolve to an exact, safely-identifiable #605 derived entity.

  **Fix**: the matcher, exported from the start since nothing about it
  is CK-specific, is appended directly to `mtgmatcher/magic/tokenpairs.go`
  (the same file that mints the derived entities it reads, rather than a
  second, confusingly similarly-named file of its own) and builds a
  `TokenPairIndex` once (against the loaded datastore, `sync.OnceValue`
  - lazy, since `GlobalDatastore` is populated by the running program
  after mtgjson is parsed, not at Go's own package-init time) mapping
  each face uuid that participates in a derived pairing to every other
  face CK has been seen naming it against, keyed by that other face's
  own name normalized (`NormalizeTokenFace`: no artist parenthetical,
  no `" Token"` suffix, case-folded) to the derived entity's
  tcgplayerProductId. `MatchTokenPairing(scryfallID, listingName)`
  looks up CK's own `scryfall_id` for one face and the other half of
  CK's own listing (`SplitTokenPairName`) against that index, before
  the old first-face-collapse fallback runs. The scryfallId anchors one
  face precisely; the listing's own wording is used only to disambiguate
  among the (typically few) pairings that
  specific face already has - never as a blind name-only match, which
  is exactly the risk (landing on the wrong side of an unrelated
  same-named token) this whole feature exists to avoid. Where no exact
  match is found, it falls through unchanged to the existing
  first-face-collapse behavior. Four direct unit tests added alongside
  the existing `tokenpairs_test.go` suite, independent of any vendor
  package: `TestNormalizeTokenFace`/`TestSplitTokenPairName` (pure
  string logic), `TestMatchTokenPairingAnchorsEitherHalf` and
  `TestMatchTokenPairingBySetNumber` (against the same real pairings
  this file's own vendor-level pinning tests already use as ground
  truth).

  **Three follow-up fixes, same day** (categorized the initial 287
  unmatched listings, then investigated the three worth chasing with a
  parallel multi-agent pass - see workflow run `wf_7e73c370-170` -
  followed by direct verification of every finding against the real
  snapshot and datastore before landing anything):
  - `normalizeTokenFace` stripped the `" Token"` suffix *before* the
    artist parenthetical, so a listing shaped `"X Token (Artist)"` never
    had the suffix trimmed at all - `TrimSuffix` never finds `" Token"`
    at the end of a string that actually ends in `" (Artist)"`. Fixed by
    reordering the two strips (parenthetical first). Recovers cases like
    `"Eldrazi Spawn Token (Briclot) - Eldrazi Token (Proce)"` (Planechase
    Anthology) with zero effect on anything already matching - pinned
    with `TestPreprocessTokenPairingParenthetical`.
  - `matchTokenPairing` always assumed CK's `scryfall_id` anchors the
    *first* half of its own listing name and the second half names the
    partner - but CK sometimes lists the id against the second half
    instead (`"Cat Token - Cat Warrior Token"` anchors to the Cat
    Warrior face, not Cat). Fixed by comparing the resolved face's own
    real name against both halves and looking up whichever one is *not*
    its own name, falling back to the original first-anchors convention
    when neither or both match (the self-named-both-halves case, e.g.
    `"Ooze Token (Swanland) - Ooze Token (Swanland)"`, where guessing
    would be unsafe) - pinned with
    `TestPreprocessTokenPairingSecondHalfAnchored`.
  - CK never publishes a `scryfall_id` for a `"Mystery Booster/The
    List"` listing that bundles two independently-numbered token-sheet
    entries into one retail sku (`MTMKC-0012`, `MTMH3-0002`, ...) -
    structurally, not as a data gap: no vendor ever sells that exact
    ad-hoc CK-chosen pairing as one physical/digital product, so no id
    could exist to publish. Added `matchTokenPairingBySetNumber`, a
    sibling that anchors the first face via
    `mtgmatcher.MatchInSetNumber(name, setCode, number)` against the
    sku's own already-correctly-derived filing set and number instead -
    the same function and the same "exactly one match or don't guess"
    discipline every other sku-driven resolution in this file already
    trusts, just handed a real anchor instead of only free text for
    `Match()` to interpret. Scoped narrowly to the one shape verified
    safe (an `"MT"`-prefixed sku under the `"Mystery Booster/The List"`
    edition) rather than generalized to every scryfall_id-less
    two-sided listing - a sibling Secret Lair shape (`TSLD-2446`) was
    checked and found *not* safe to fold in.

    Followed up 2026-09-15: `TSLD-2446` looked like a sku-parsing bug at
    first (the setCode-fallback branch appends `card.Variation` onto an
    already-set `variation` unconditionally, producing a garbled
    `"2446 2446 // 003"` for these two listings specifically), and that
    duplication is real - but it turns out to be cosmetic, not the
    actual blocker. Neither of the two printings this listing names
    (`"Phyrexian Minion"`, and separately `"Pegasus"`/`"Angel"` as its
    partner) exists anywhere in the loaded datastore at all - not under
    `SLD` (which carries 3,355 cards and 69 tokens, none matching, none
    numbered anywhere near `2446`/`003`/`0002`), not under any other set.
    `2446` is almost certainly Card Kingdom's own internal catalog
    number for this Secret Lair drop, not a real Magic collector number
    - mtgjson simply hasn't catalogued this drop's tokens yet (the local
    datastore used for this check was built the same day, so this isn't
    staleness). No sku-parsing fix, however clean, can resolve a face
    that isn't in the data - confirmed by checking the other 7
    two-sided Secret Lair token listings in the corpus (`FTSLD-148`
    through `FSLD-1516`), every one of which already carries a working
    `scryfall_id` and resolves through the existing path; `TSLD-2446` is
    the only one that doesn't, and it's the only one hitting the
    concatenation branch at all in this entire corpus. Left alone rather
    than fixed cosmetically with nothing to verify it against - revisit
    once mtgjson catalogues this drop's tokens, at which point it's
    likely to resolve through the existing `matchTokenPairing` (once
    it gets a `scryfall_id`) or `matchTokenPairingBySetNumber` paths with
    no further code change, and the concatenation bug would need fixing
    then, not before there's a real case to verify against - pinned with
    `TestPreprocessTokenPairingBySetNumber` for the shape that *is* safe.
  - Separately (not a two-sided-token-matching fix, found while tracing
    the 16 listings where `Preprocess` errored outright): two Warhammer
    40,000 surge-foil skus (`SFT40K-015`, `SFT40K-016`) were refused
    because the bare number they name is filed nonfoil-only in `T40K` -
    but the set separately catalogues a `★`-suffixed duplicate of the
    exact same token as the foil printing, which nothing ever tried.
    Two `skuFixupTable` entries redirect to the starred number, in the
    same style as the table's existing `★` entries. The other 14 of the
    16 errors are confirmed correct, intentional refusals (the same
    "sheet was never sold foil" doctrine `TestPreprocessTokenFoilRefused`
    already pins) - pinned with `TestPreprocessSurgeFoilStarredDuplicate`.

  Combined effect, verified end to end against the real saved snapshot
  through the actual `Preprocess()` path: **752 → 791 of 1,039 (76%)**
  resolve exactly, with zero regressions (every previously-resolving
  case, including the pinned Angel/Cat example, still resolves to the
  same entity), and `Preprocess()` errors on this corpus drop from 16 to
  14.

  The remaining ~248 mostly don't have a #605 entity to resolve to at
  all, for reasons a matching-mechanism change can't paper over: ~110
  are a real single-faced card/token that already carries its own
  `tcgplayerProductId` directly (no derived entity is ever minted for
  those - the same "mtgjson already gave the pairing's id to one face"
  shape #605 documented excluding 77 TCGplayer pairings for), ~79 are a
  face that plain doesn't participate in any derived pairing (CK's own,
  often older catalog - Commander 2014, Modern Event Deck, `F12` FNM
  promos - and TCGplayer's booster-sheet `tokenProducts` data are partly
  disjoint sets), ~27 are a face that *does* participate in a pairing
  but CK's own named partner for that exact ad-hoc listing simply isn't
  one TCGplayer ever bundled as a product (confirmed structurally, not
  a wording gap - guessing among that face's other known partners by
  similarity is exactly the ambiguity this feature exists to avoid),
  ~20 are the `"Mystery Booster/The List"` shape that now safely anchors
  by sku but still has no matching TCGplayer-derived partner, a further
  handful are the `TSLD-2446` listing (confirmed to name printings
  mtgjson hasn't catalogued at all yet - not a fixable parsing bug, see
  the follow-up note above) and one genuine 3-face listing
  `matchTokenPairingBySetNumber` correctly declines to touch, and 14 are
  the correct-refusal `ErrUnsupported`
  cases above. Those all keep using the pre-existing
  first-face-collapse/`MatchID` fallback, which was already correct for
  them.

  The per-vendor "no gap" caveat above turned out not to hold - see
  the Star City Games entry right below, whose overlap is even larger.

- **Cross-vendor name-based matching, Star City Games — investigated
  and fixed 2026-09-15 (#605).** Same method as Card Kingdom above: one
  fetch of SCG's own bulk catalog export
  (`https://api.starcitygames.com/hawksearch/catalog/download/json`,
  the same endpoint the production scraper already uses, 178,035
  products across every game SCG sells - not a page-by-page crawl, and
  not repeated).

  SCG sells two-sided token pairings too - **3,769 listings** matching
  the same trigger shape (a name containing `" // "` with `"Token"` in
  it - SCG's own wording never uses `" - "` for this, confirmed against
  the full catalog: the only three hits were sleeve/deck-box products
  whose marketing copy happens to contain both a dash and the word
  "Token"), e.g. `"{Angel of Sanctions Token} // {Drake Token}"`
  (Amonkhet, sku `SGL-MTG-AKH-T1T18-ENN`). SCG wraps each face's name in
  its own `{curly braces}`, otherwise the shape is the same one CK's
  matcher already handles.

  SCG's own catalog carries a `scryfall_id` field for **3,484 of the
  3,769 (92%)** - a materially higher rate than CK's 1,013/1,039 - and
  checking it against #605's derived pairings the same way as CK found
  **3,108 of the 3,769 (82%) resolve to an exact, safely-identifiable
  derived entity**, using the shared `magic.MatchTokenPairing`
  (see commit 5 above) unchanged, needing nothing SCG-specific beyond
  stripping its own brace wrapping - already covered by
  `NormalizeTokenFace`'s generic (and, for every non-brace-using vendor,
  inert) brace strip.

  **Before this, SCG's own resolution wasn't refusing these listings -
  it was silently mispricing them.** Unlike CK (whose old collapse
  fallback never touched `scryfall_id` for a two-sided name at all),
  SCG's `resolveProductID` already tries `mtgmatcher.MatchID` on
  `p.ScryfallID` directly for every product, two-sided or not - which
  for a two-sided listing resolves cleanly to *one face's own single
  card/token*, oblivious to the rest of the listing's own name, and
  returns successfully. So the "Skip tokens and similar" swallow in
  `starcitygames.go` (triggered only on a resolution *error*) never
  fired for these 3,108: they were already priced today, just as the
  wrong product (one face of a two-sided card, not the two-sided card
  itself) rather than as unresolved.

  **Fix**: in `starcitygames/catalog.go`'s `resolveProductID`, the new
  two-sided-token check runs before the plain identifier loop that was
  already resolving these to the wrong single face. (The existing
  promo-shelf table, `promoShelfPrintings`, is unaffected either way -
  its own `MatchWithNumber("", set, number)` call already finds
  mtgjson's own combined "X // Y" printing whole where one exists, the
  same way `MatchNativeTokenPair` below does explicitly; its two
  two-sided League Tokens entries were *already* resolving to the full
  pairing before any of this session's changes, not "one face only" as
  an earlier draft of this note incorrectly assumed without checking.)
  `magic.MatchTokenPairing(p.ScryfallID, p.Name, foil)`
  resolves the pairing's own id; `mtgmatcher.MatchID(tcgID, foil,
  etched)` turns that into the correctly-foil-aware uuid (a bare
  `mtgmatcher.ConvertID` here would have reproduced CK's own
  now-fixed foil bug from scratch - caught by re-verifying the fix
  against the real snapshot with foil awareness specifically, the same
  discipline that caught CK's version). Verified end to end against the
  real SCG snapshot through the actual `resolveProductID` function (not
  a standalone probe): 3,108/3,769 resolve, 0 finish mismatches across
  1,861 foil and 1,247 nonfoil exact matches. Two new pinning tests
  (`TestResolveTwoSidedTokenPairing`, nonfoil and foil cases).

  The remaining listings break down the same way CK's did: ~214 face
  uuids don't participate in any derived pairing at all (TCGplayer's
  own `tokenProducts` feed has no sheet data for that specific printing
  yet), ~162 face uuids *do* participate in a pairing but SCG's own
  named partner for that exact listing isn't one TCGplayer ever bundled
  as a product (a genuine data gap, not a wording gap - confirmed the
  same way as CK's equivalent bucket), and 285 carried no usable
  `scryfall_id` at all - **recovered separately, see the follow-up
  below** (mostly older FNM/Guild Kit promo listings whose sku carries
  a composite, non-collector-number scheme - `FNM_XLN_T11`,
  `T05_GK1_T05` - that would need its own per-shape investigation the
  way CK's `"MT"`-prefixed Mystery Booster/List skus got, not attempted
  here). Those all keep resolving the way they did before this fix -
  correctly for most (the plain single-id or promo-shelf paths), a
  known imprecision for the rest.

  **Follow-up, same day: recovered 130 of the 285 no-scryfall_id
  listings.** Their sku's own third segment sorts them into shapes:

  - ~208 name their own filing set directly in the sku (`GK1-T01`,
    `MID-T16_AFR_T14`, ...), unlike the Mystery Booster/List shape
    below. Two distinct, safe mechanisms recover about 130 of these:
    - Some aren't a synthetic tokenProducts-derived pairing at all -
      mtgjson already files them as one ordinary printing of its own,
      under a combined `"X // Y"` name, the same way it files a fixed
      double-faced token like Undercity // The Initiative (Guild
      Kit's `"Copy // Horror"` at `TGK1` #1). Added
      `magic.MatchNativeTokenPair`, reconstructing that name from
      the vendor's own two-part listing (trying both face orders,
      since a vendor's own order does not always agree with
      mtgjson's - `"Weird Token // Goblin Token"` names the same
      printing as mtgjson's own `"Goblin // Weird"`) and resolving it
      by ordinary set+number lookup, no derived entity involved.
    - The rest resolve through the existing
      `magic.MatchTokenPairingBySetNumber` (cardkingdom's own
      sku-anchored fallback, exercised by a second vendor for the
      first time), anchoring the first face alone by set and number.

    Building this on SCG's own `{brace}`-wrapped face names surfaced a
    real bug in the moved function: it never stripped a vendor's brace
    wrapping from its anchor-face candidates before searching,
    unnoticed until now because cardkingdom (its only caller so far)
    never wraps its own names in anything. Fixed by sharing the same
    wrapping-strip logic `NormalizeTokenFace` already used, in both a
    suffix-kept and a suffix-stripped form: most token Card.Names drop
    the `" Token"` suffix a vendor spells, but not all (e.g.
    `"Kobolds of Kher Keep Token"`, disambiguated from the real
    nontoken card by that same suffix) - CK's own existing
    `TestPreprocessTokenPairingBySetNumber` pins exactly this case,
    and caught the regression before it shipped when an initial,
    over-simplified fix dropped the suffix-kept candidate entirely.
  - ~51 (Mystery Booster/List) embed two unrelated sets' own tokens in
    one sku with no single filing set to anchor from - checked (tried
    anchoring by either embedded set/number pair) and confirmed a
    genuine, structural gap: the two faces come from different
    original products entirely, so by construction TCGplayer never
    bundled them as one physical/digital product, and there is no
    derived pairing to find regardless of matching mechanism.
  - ~19 (Secret Lair Drop embedded, FNM/League promo) share the same
    root cause as the already-investigated-and-closed `TSLD-2446`
    finding (cardkingdom) or the League Tokens table's own documented
    limitation - not new problems, not chased further.
  - ~7 (Warhammer 40K/Doctor Who Commander) likely share CK's own
    "sheet was never sold foil" correct-refusal doctrine - not chased
    further given the small count and the established pattern.

  Verified end to end against the real snapshot through the actual
  `resolveProductID` function: **130 of the 285 now resolve** (3,108 →
  3,239 of 3,769 two-sided listings total, 0 finish mismatches, at the
  time). **That 3,239 was wrong** - the same `TokenPairIndex` collision
  bug the hostile review caught (below) affected SCG far more than CK:
  431 of these 3,239 were an arbitrary pick among several correctly-
  colliding partners, not a verified match. Corrected total: **2,808 of
  3,769 (74%)**. Two new pinning tests, one per mechanism
  (`TestResolveNativeTokenPair`, `TestResolveTokenPairingBySetNumber`),
  plus two direct unit tests in `tokenpairs_test.go` for the matcher
  changes this required, independent of any vendor package:
  `TestNormalizeTokenFaceStripsBraceWrapping` (SCG's own curly-brace
  wrapping) and `TestMatchNativeTokenPair` (Guild Kit's real "Copy //
  Horror" at TGK1 #1, both face orders).

- **Hostile review of PR #605, 2026-09-15 — two real bugs found and
  fixed, one found and rejected, one investigated and left alone.**
  Assessed each claim against the real code and real data before
  acting on any of it; several were valid, one was a genuine
  misunderstanding of how the code actually works, and one flagged a
  real but already-disclosed data-quality risk that doesn't call for a
  code change.

  **Fixed — `TokenPairIndex` silently dropped a colliding pairing
  (real, the most consequential finding by far).** A face commonly
  pairs with several different partners across a sheet - that
  plurality is the entire reason a vendor's own wording has to
  disambiguate at all - and two of those partners can normalize to the
  identical key: measured against today's datastore, 5,783 derived
  pairings touch 1,865 distinct faces across 5,768 `(face, key)` slots,
  and **252 of those faces (13.5%)** carry at least one colliding slot
  (297 of the slots themselves, 5.1% - a single face can collide on
  more than one key across different sheets, which is why the slot
  count runs higher than the face count). "Bear" alone pairs with
  **four** differently-numbered "Food" tokens across Throne of
  Eldraine's own sheets, all colliding on `"food"`. The index was a
  plain `map[string]string`, last write wins, no error, no log -
  whichever pairing got processed last in (randomized) map-iteration
  order silently became the only one reachable, and a vendor listing
  naming one of the *other* colliding partners would resolve to the
  wrong physical product under the survivor's id. `TokenPairIndex()`
  now tracks a collision the moment it sees two different ids for the
  same `(face, key)` and blanks that entry rather than picking one -
  the caller's own fallback handles it from there, the same "don't
  know, refuse" discipline this feature has followed everywhere else.

  This was not a theoretical risk once measured against real data: **37
  of CK's 788 and 431 of SCG's 3,239** previously-reported matches were
  an arbitrary pick among several colliding partners, not a verified
  one. Corrected totals: CK **751/1,039 (72%)**, SCG **2,808/3,769
  (74%)**. Two new unit tests in `mtgmatcher/magic/tokenpairs_test.go`
  (`TestTokenPairIndexCollision`, pinning the Bear/Food case directly)
  and one SCG pinning test swapped for a collision-safe example
  (`TestResolveTwoSidedTokenPairing`'s foil case moved from Bear/Food to
  Soldier/Rebel, which pairs with exactly one partner).

  **Fixed — a foil request could be silently validated against a
  finish the pairing itself was never sold in (real, narrower blast
  radius).** `unionFinishes` (the very first commit) deliberately takes
  the union of both faces' own finish lists, so `MatchIDFinish` doesn't
  error on a finish only one face happens to carry - a sound choice for
  its *original* purpose, made before this feature's foil-safety check
  (`tokenPairingFinishOK`, added investigating Star City Games) existed
  to lean on it. That check trusted the derived card's own (unioned)
  `HasFinish("foil")`, which answers "did either face exist in foil
  somewhere," not "was this specific two-sided PRODUCT sold in foil" -
  reintroducing, one layer up, exactly the silent wrong-finish
  substitution this whole check exists to prevent. `tokenPairingFinishOK`
  now requires **both** source faces (`tokenPairPartA` and
  `tokenPairPartB`) to independently carry the requested finish, a
  provably narrower bar than the union. Pinned with a second new test
  (`TestMatchTokenPairingRequiresBothFacesInRequestedFinish`, using a
  real TKHM pair where Boar was never sold foil but its sheet partner
  Spirit was - the union said foil, the fix correctly refuses).

  **Fixed — a misleading error string.** `tcgplayer/preprocess.go`
  returned `errors.New("duplicate")` at both of its two-sided-token
  refusal sites when a product id had no derived entity - not a
  duplicate, and an operator grepping logs for "duplicate" to find a
  real duplicate-listing bug would chase the wrong root cause. Renamed
  to `"no derived pairing for this id"` at both sites.

  **Rejected — "multi-id entities pick `ids[0]` and pray, a coin flip
  with a sort."** This claim misreads what `usableIDs` actually holds.
  Every id in it was independently filtered to ids that *only ever*
  appeared associated with this one specific `(uuidA, uuidB)` pairing
  in mtgjson's own `tokenProducts` data (`len(idPairs[id]) > 1` already
  excludes an id claimed by more than one pairing, a separate,
  already-existing rung of the exclusion ladder). They are not
  competing candidates for what the pairing *is* - confirmed directly:
  `mtgmatcher.ConvertID` on every usable id for a multi-id entity
  returns the identical uuid (already pinned by
  `TestDerivedTokenPairResolvesByProductID`'s "multi-id pairing, first
  id" / "second id" cases), and `tcgplayer.go`'s own pricing walk prices
  from the whole `tcgplayerProductIds` list, not just `ids[0]`. Picking
  `ids[0]` for the singular `Identifiers["tcgplayerProductId"]` field is
  a display/convention choice, not an identity guess.

  The underlying worry is real, though, just mis-stated: a "handful of
  the 34" multi-id entities may be mtgjson mis-mappings rather than
  confirmed duplicate listings of the same physical product (documented
  in commit 1's own message before this review ever happened - ELD's
  Goat/Food ids run suspiciously sequential with a gap). If one of
  those ids is genuinely wrong, pricing off it is wrong regardless of
  which one gets called `ids[0]`, since all of them get priced. That is
  an upstream data-quality question already disclosed, not a logic bug
  this code can fix by choosing differently among ids that are - by
  every signal this code has access to - equally valid. Quarantining
  all 34 from pricing to protect against the suspected handful was
  considered and rejected: it would trade away ~30 likely-genuine
  matches to guard against noise nobody has actually confirmed yet, in
  the one part of the corpus small enough (1% of 3,094 derived
  entities) that a future pass verifying each of the 34 by hand is
  realistic - a better fix than a blanket, mostly-unnecessary refusal.

  **Investigated, left alone — `" - "` as a face separator.** The worry:
  a token-ish CK listing whose name uses `" - "` for something that
  isn't a second face still enters the pairing path, since the trigger
  is `Contains(" // ") || Contains(" - ")`. Checked the full real CK
  corpus (164 listings currently reach `isTwoSidedToken` via `" - "`
  with no `" // "` present) - zero are false positives; every one is a
  genuine two-sided pairing. More importantly, the design is safe even
  in a hypothetical false positive, not just lucky today: `matchTokenPairing`
  requires a `scryfall_id` match against a *curated* index before it
  ever returns non-empty, so a false split's nonsense "second half"
  essentially never finds a matching key; and CK's own fallback collapse
  only fires when `MatchInSetNumber` on the *original, uncollapsed* name
  already found zero matches - so a real card whose real name happens to
  contain `" - "` and resolves fine under its full name is never touched
  at all. No code change; this is a case where the layered fallback
  design the whole feature already relies on turns out to cover a risk
  that looked real on first read.

  **Left as a design note, no change made — derived entities are only
  reachable by id or the `Identifiers["derivedTokenPair"]` marker, not
  through `set.Cards`/`Hashes`/`AllUUIDs`.** Deliberate from commit 1
  (a name-reachable derived entity would alias against the wrong
  pairing, pinned by `TestDerivedTokenPairsDoNotTouchTheNameIndex`), and
  every consumer added since (TCG Market/Index, CK, SCG) has correctly
  filtered on the same marker rather than assuming enumerability through
  an existing index - so the concern that "the next consumer will
  forget" hasn't materialized across three additions so far. Worth
  watching, not urgent enough on its own to justify a new public API
  surface in a bug-fix pass; revisit if a fourth consumer actually gets
  this wrong.

  **Not addressed — "this is four PRs duct-taped together, split it."**
  A fair process point about review scope, not a code defect, and not
  something to resolve unilaterally mid-review - see the conversation
  for the decision.

- **`Case of the Lost Witness` / `Oracle of the Alpha` / `Perforator
  Crocodile` (Mystery Booster Commander Edition, `MBC`) — the user has
  said they believe this is an upstream (MTGJSON/Scryfall) data issue and
  will report it themselves.** Don't duplicate that report. For context
  if it resurfaces here anyway: these three of `MBC`'s 80 cards are
  silently dropped by the `if card.IsOnlineOnly { continue }` filter in
  the loader (they carry `isOnlineOnly: true` / `availability: ["arena"]`
  in the raw data) even though the physical set is real, unreleased as of
  this writing (`releaseDate: 2026-11-09`), and TCGplayer already lists
  real product ids for all three.

- **`LoadTCGSKUs` dedup filter — investigated 2026-09-14, do NOT build
  this.** The plan on record here (a load-time filter dropping any
  productId spanning more than one uuid) was measured against the local
  sku file before writing any code, and ruled out as badly overbroad:
  10,985 of 112,626 productIds span >1 distinct uuid, but only 3,378 of
  those touch a token-layout uuid at all — the other 7,607 are ordinary
  double-faced/split/adventure/transform *cards*, where one physical
  TCGplayer product legitimately supplying both mtgjson face-entries is
  correct, not a bug. Narrowing to the token-only 3,378: 84 are a
  double-faced token's two faces sharing one product (also legitimate,
  same shape as the card case), and the remaining 3,199 are TCGplayer's
  own double-sided token products (`Keimi`/`Spirit` on one product id,
  etc.) — also legitimate. **Zero** cases were found of a token id
  colliding with an unrelated real card, the shape that would actually
  be a bug. On top of that: Vittorio closed the three PRs that already
  tried to fix token pricing/dedupe from the scraper side (#461, #472,
  #463) with "the magic tokens need some datastore work which is
  separate from this" — see `[[feedback_magic_tokens_datastore_first]]`
  in project memory. A `LoadTCGSKUs` filter is the same category of fix
  in a different function. Don't build it here; if the datastore-side
  token model work ever starts, that's where this belongs.

  **Update, PR #605**: that datastore-side work started - #605 mints a
  proper combined entity for the token-pairing shape this item measured
  (`mtgmatcher/magic/tokenpairs.go`) rather than filtering it away, and
  wires `tcgplayer.go`'s pricing to it. This item's "don't build a
  dedup filter" verdict still stands on its own terms (a blind filter
  at `LoadTCGSKUs` would have been the wrong shape of fix regardless of
  where the real fix eventually landed) - it is not contradicted by
  #605, just superseded as the place this data shape gets handled.

- **The walk count — fresh run taken 2026-09-14, before PUNK/Gala
  Greeters/OAFR/Ertai above.** 645 CSV rows total, not comparable to the
  old "48" figure — that number predates full token coverage resolving
  cleanly enough to *reach* the comparison step, so it undercounted.
  Split out: 74 are `The List` (excluded per the rule above), 566 are
  ordinary upstream-missing-id rows (Scryfall/MTGJSON has no id yet —
  overwhelmingly emblem/token sheets, the same submission backlog as
  before, just bigger now that more of them resolve at all), and 5 are
  same-uuid duplicate-listing artifacts of this *tool's* own
  `AddStrict` + concurrent paging (two live TCGplayer products for one
  printing race to be "the" id compared against the datastore's; not a
  matcher bug — same shape as the already-documented `Battlefield
  Forge`/`SLD` duplicate). **Zero real matcher bugs in this
  walk.** Re-run fresh before quoting a number to anyone, as always —
  this one predates the PUNK/Gala Greeters commits above.

## Method notes worth keeping

- **Drift control, always.** Any live-vendor measurement runs base →
  branch → base again, back to back, and the two base runs must be
  byte-identical before the branch number means anything. Two false
  alarms in this project's history came from skipping this.
- **An id disagreeing with a vendor's own wording is not automatically
  the vendor's bug** (see the general credential in project memory,
  `[[feedback_vendor_ids_are_theirs]]`), but for id-vs-wording mismatches
  *within our own preprocess/matcher code* the fix is almost always to
  trust whichever one is more specific — an id beats free text, but only
  the id *actually on the exact listing in question*, cross-checked
  against the resolved printing's own `tcgplayerProductId`, not assumed.
- **Full-catalog scans are cheap enough to just run.** Several fixes
  above depended on knowing "no live product currently has this shape" —
  that's answerable in minutes by walking all ~114k Magic products once
  (`ListAllProducts` in a loop, filtered client-side) rather than
  guessing from a stale log or a hunch. Do that before deleting anything
  as dead code.
