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

## What's still open

Updated 2026-09-14, after a second pass through this list.

- **Gala Greeters, 10 of 11 language listings — fixed, not yet pushed.**
  All 11 ids folded into `productOverrides` (English `268367` plus the
  10 others, each mapped to its own-language `SNC` printing — the
  datastore does carry all 11). Committed locally as
  `gala-greeters-languages` (`3c918f01`); full gate green.

- **`PUNK` ("Black Lotus Unknown Planechase") — fixed, not yet pushed.**
  A live catalog walk (2026-09-14) confirmed TCGplayer sells ~46 of its
  52 plane cards for real, so the "not on sale anywhere" `skipSet` entry
  was stale. Checked collision safety properly before touching it: PUNK
  shares exactly two names with cards outside itself — `No Way Out`
  (also `MID`/`DBL`, the risk this item was originally deferred over)
  and `Artist Alley` (also the still-skipped `UNK`, untouched by this
  change). Probed both through `Match()` directly: `No Way Out` resolves
  by edition with zero ambiguity either way (PUNK when the edition says
  Planechase, MID when it says Innistrad: Midnight Hunt), and with no
  edition given at all it correctly reports aliasing rather than
  guessing — same safe-failure shape as before PUNK was unskipped, just
  with a third candidate in the list. Committed locally as
  `punk-frontcards-skipset`; full gate green.

- **The "Front Cards" `skipSet` suffix — investigated, deliberately left
  alone.** `FJMP` (13 of 46 divider cards live) and `FTMC` (1 of 5, TMNT's
  `Bosses + Events`) do have real TCGplayer ids, same shape as PUNK. But
  unlike PUNK, unskipping them buys nothing: every card in both sets
  carries `layout: "front_card"`, and `isUnsupported` already has a
  standing, code-independent guard — `c.Contains("Front Card")` — that a
  genuine live scrape of one of these products trips on its own,
  regardless of whether the set is loaded. Confirmed by probing FJMP's
  `Phyrexian` (id `218188`) directly: unskipping it does not make the
  live product match (still refused, exactly as before), it only adds a
  second, spurious `Phyrexian` candidate to plain name lookups — real
  ambiguity with no corresponding fix. Left `skipSet` untouched for both;
  if a future card here turns out not to carry `layout: front_card`,
  redo this check before assuming the same reasoning applies.

- **`Ertai, the Corrupted` PLST aliasing — root-caused, not fixed.**
  `Ertai, the Corrupted (Alt. Art Foil)` from Planeshift aliases against
  **two** candidates: the correct `PLS #107★` and a spurious `PLST
  #PLS-107` (The List's plain reprint of the same card). Traced all the
  way down: an "Alt. Art" wording that gets fully trimmed away during
  name-fixup sets `InputCard.BeyondBaseSet = true` (a deliberate "widen
  the search" signal for cases where an alt-art genuinely lives in
  SLD/PLST), which admits the List candidate; `listEditionCheck`'s own
  stand-down logic (`the edition already names this printing's origin
  set → don't reject`) then lets it survive, because nothing there knows
  the base set already produced one confident answer via its own
  `simpleFilterCallbacks` entry (`altArtCheck` for PLS). Confirmed this
  is narrow, not systemic: a *plain* Planeshift card with a PLST reprint
  resolves fine (checked three at random) — `BeyondBaseSet` only fires
  when there's nothing left after trimming. This needs its own PR with a
  full corpus measurement (the shape of the fix likely touches
  `listEditionCheck` broadly, which serves every base-set-vs-List
  disambiguation, not just this one card) — don't rush it.

- **`PUNK` ("Black Lotus Unknown Planechase") and the "Front Cards"
  exclusion — same root shape as the two retired refusals, not touched.**
  `skipSet` still hardcodes `PUNK` as "not on sale anywhere" (23 of the
  Oversize-shelf warnings) even though TCGplayer now sells it for real.
  Same story for any set whose name ends in `"Front Cards"` (a TMNT
  divider-card product, `FTMC`, sells one of its five cards — `Bosses +
  Events` — on TCGplayer despite the whole category being excluded).
  Both are deferred specifically because unskipping either risks a real
  name collision: `PUNK`'s "No Way Out" plane shares a name with a real,
  commonly-priced card (`MID`/`DBL`). Don't flip either `skipSet` entry
  without checking collision safety the way the original oversized-shelf
  work did, and re-measuring the full corpus.

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

- **The walk count — fresh run taken 2026-09-14, after PUNK/Gala
  Greeters/Front-Cards above (not yet pushed).** 645 CSV rows total, not
  comparable to the old "48" figure — that number predates full token
  coverage resolving cleanly enough to *reach* the comparison step, so
  it undercounted. Split out: 74 are `The List` (excluded per the rule
  above), 566 are ordinary upstream-missing-id rows (Scryfall/MTGJSON
  has no id yet — overwhelmingly emblem/token sheets, the same
  submission backlog as before, just bigger now that more of them
  resolve at all), and 5 are same-uuid duplicate-listing artifacts of
  this *tool's* own `AddStrict` + concurrent paging (two live TCGplayer
  products for one printing race to be "the" id compared against the
  datastore's; not a matcher bug — same shape as the already-documented
  `Battlefield Forge`/`SLD` duplicate). **Zero real matcher bugs in this
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
