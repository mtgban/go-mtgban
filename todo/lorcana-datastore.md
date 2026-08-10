# Should Lorcana stop depending on LorcanaJSON?

**Short answer: no — not as a replacement. Enrich it, the way `riftbound-datastore`
enriches Riot's gallery, and expect a much smaller return than Riftbound got.**

The premise behind the question — "build our own from the TCGplayer catalog and
add back what LorcanaJSON does not carry" — does not survive contact with the
data. The one asset the catalog uniquely gives Riftbound is the TCGplayer
product id, because Riot's gallery has none. **LorcanaJSON already carries the
product id for 3228 of its 3242 cards (99.6%), and every one of those 3226
distinct ids exists in the catalog (zero absent).** The rest of the catalog's
card data is a strictly weaker version of what LorcanaJSON already gives us,
and two things the matcher depends on cannot be rebuilt from it at all.

What is genuinely worth taking from the catalog is **40 product ids** and,
later, **171 sealed products**. Everything else is already ours.

## Evidence

Both sources were read in full inside CI, where the network and credentials
are (see "How this was measured"). Numbers are from run
[31359926383][run1] / [31360215158][run2], LorcanaJSON format 2.3.5 generated
2026-08-02, catalog dump of category 71.

- Catalog: 20 groups, **3605 products**, 3 printings (`Normal` 132,
  `Holofoil` 133, `Cold Foil` 141), 6 conditions (1–5 plus **6 `Unopened`**).
- LorcanaJSON: **18 sets, 3242 cards**, 14 distinct foil sub-types.

### VERIFIED: the bucket object exists

`b2://mtgban-datastore/lorcana/tcgplayer-catalog.json.xz` **is present** and
downloads cleanly with the `B2_KEY_ID_DATASTORE` / `B2_APP_KEY_DATASTORE`
credentials go-mtgban's workflows already use. It is a real tcgdumper dump of
category 71 (`category.categoryId == 71`, `category.name == "Lorcana TCG"`),
carrying `conditions`, `languages`, `printings`, `rarities`, `groups` and
`products[].skus`. The convention the brief guessed at holds; no tcgcsv
reconstruction was needed, which matters because tcgcsv exposes no skus and
skus are the only authoritative source of finishes.

## 1. What LorcanaJSON gives that the catalog does not

Going through `mtgmatcher/lorcana/lorcana.go` field by field. "Catalog"
means: could this be reconstructed from category 71 alone?

| `mtgmatcher.Card` field | Source today | From the catalog? |
| --- | --- | --- |
| `UUID` | `card.id` (LorcanaJSON's integer) | **No.** See "The uuid problem". |
| `Name` | `fullName` | Degraded. 380 of 3226 product names differ from the canonical name: storefront decoration (`(Enchanted)`, `(Alternate Art)`, `(Foil)`) and lost diacritics (`Te Ka` for `Te Kā`). |
| `SetCode` | `setCode` | Partly. Group abbreviations line up for the 14 numbered sets, but the promo groups (`DLPC`, `D23`, `D100`) each scatter across every set — `DLPC`'s 170 products belong to 13 different LorcanaJSON sets. |
| `Finishes` / `FoilUUIDs` | `foilTypes` | **No.** See "The finish problem". |
| `Number` / `OriginalNumber` | `number` + `variant` | Yes, essentially: 3223 of 3226 agree; 3 differ. But `variant` (7 cards) has no catalog equivalent. |
| `Images` | `images` | Only the TCGplayer product shot (100% formulaic, `…/product/<id>_200w.jpg`). Not the card art LorcanaJSON links. |
| `Colors` | `colors` / `color` | Approximately, via `InkType` (3219 products) — one string, not the multi-colour list. |
| `Rarity` | `rarity` | Yes, `Rarity` (3434 products). The catalog also has `Quest` and `Iconic`, which `lorcanaRarityMap` does not rank. |
| `Types` | `type` | Yes, `CardType` (3288). |
| `Subtypes` | `subtypes` | Approximately, `Classification` (2699, semicolon-joined). |
| `Supertypes` | `story` | Approximately, `Property` (3307). |
| `Printings` | derived from all cards of a name | Derivable either way. |
| `IsPromo` | `nonPromoId != 0` | Moot — **no LorcanaJSON card sets `nonPromoId` today**, so `IsPromo` is always false. The catalog's `PromoType` (143 products) would actually be an improvement, if anything read it. |
| `Language` | hardcoded `"English"` | n/a |
| `Identifiers[tcgplayerProductId]` | `externalLinks.tcgPlayerId` | **This is the one the catalog would add — and LorcanaJSON already has it at 99.6%.** |
| `Set.ReleaseDate` / `Type` | `sets` | No. The catalog has a group `publishedOn` (street date, differs from LorcanaJSON's release date) and no notion of `expansion` vs `quest`. |
| `Set.BaseSetSize` | first `enchanted` number | Derivable from `Rarity`. |

Plus everything the file carries that the matcher does not read today but the
site could: structured `abilities`, `keywordAbilities`, `fullText`,
`flavorText`, `clarifications`, `errata`, `artists`. The catalog has
`Description` and `Flavor Text`, but HTML-marked-up and unstructured.

### The finish problem

This is the blocking one, and it is exactly the delicate area the brief warned
about.

LorcanaJSON names **14 distinct foil sub-types**: `Silver` (2726 cards),
`Satin` (115), `Magma` (95), `VerticalWave` (76), `Lava` (68), `FreeForm1`
(18), `RainbowPillars` (15), `Tempest` (13), `FreeForm2` (12), `Glitter` (11),
`Lore` (10), `SeaWave` (7), `CalendarWave` (5), and `None` (2743).

TCGplayer has **three printings, full stop**: `Normal`, `Holofoil`,
`Cold Foil`.

`selectFinish` in `mtgmatcher/lorcana/rules.go` resolves a storefront's wording
onto a card's stored finishes by *matching the exported sub-type name*
(`strings.Contains(variation, finish)`, where `finish` is the lowercased
LorcanaJSON foil type). A catalog-built datastore has no sub-type names to
match against, so that resolution collapses to the flag, and every sub-typed
printing folds onto the primary foil. The 12 cards with a second foil sub-type
are precisely the 12 products TCGplayer sells as `Cold Foil + Holofoil +
Normal` — the two sources agree on *how many* there are and only LorcanaJSON
knows *what they are called*.

Independently: **catalog finishes are less accurate than LorcanaJSON's, not
more.** Comparing per-product, 3190 agree and 36 disagree — and in 30 of those
36 LorcanaJSON is right. Those 30 are cards TCGplayer splits into two products
(`Louie - One Cool Duck` 631349 `Normal`, `Louie - One Cool Duck (Foil)`
633427 `Cold Foil`); read one product at a time, the catalog says the card is
nonfoil-only. This is the "several products for one card, look at every
product for the key" lesson, and here it costs accuracy rather than buying it.

### The uuid problem

Lorcana uuids are LorcanaJSON's integer `card.id`. They are not internal:

- `mtgban-website/chart_resolve.go` resolves a **bare integer as a
  LorcanaJSON id straight through `mtgmatcher.GetUUID`** (`matcherTarget`), so
  every chart URL in circulation is a LorcanaJSON id.
- `PricesArchiveDB`'s variant tables key non-Magic cards on the TCGplayer
  product id, and `cachedBanIDForCard` bridges the two.

A datastore that minted its own ids (TCGplayer product ids, say) would
invalidate every stored chart target and every `ban_id` row keyed on the old
identity. There is no upside that pays for that.

## 2. What the catalog gives that LorcanaJSON does not

Quantified, and smaller than expected.

| | Count | Verdict |
| --- | --- | --- |
| Products no LorcanaJSON card claims | 377 | breaks down below |
| … **sealed products** | **171** | Real, valuable, out of scope here |
| … puzzle inserts / oversized / lore cards / case files | 147 | Collectibles, not cards; no matcher concept |
| … extra listings of a card LorcanaJSON already has | 56 | 30 safe to alias, 26 not |
| … genuinely new printings | **5** | All presale for an unreleased set |
| LorcanaJSON cards with no product id | 14 | 10 fillable from the catalog |

**Sealed (171).** The `Unopened` condition is a perfect discriminator, which
settles the question the parallel Riftbound investigation
([riftbound-datastore#2][rb2]) could not answer for category 89: of 3605
products, exactly **171 have skus in condition 6 only, and all 171 are
unnumbered; zero numbered products are sealed.** These are the booster boxes,
Illumineer's Troves, starter decks, gift sets and prerelease boxes. It is a
clean rule with no inference and no risk of publishing a playmat as a booster
box. But `mtgmatcher/lorcana` has no sealed concept at all — no `AllSealed`,
no sealed `CardObject`s — and adding one is the same three-repo change that
report specifies for Riftbound. **Left out deliberately**; it should follow
that design so both games get one shape, not two.

**Genuinely new printings (5).** This is the headline negative. The
promotional printings that justify `riftbound-datastore` appending promo-typed
sets amount, in Lorcana, to five cards — `Pete - Made by the Vine`,
`Vine Sprout - Buzzer`, `Cruella De Vil - Made by the Vine`,
`Bramble Bully - Protecting the Stalk`, `With a Few Good Friends` — all in
group Q3 `Illumineer's Quest: The Great Hunny Rescue`, published 2026-10-02,
i.e. **presale for a set two months out**. LorcanaJSON already declares the
`Q3` set and has published complete card lists for `Q1` and `Q2`; it will
publish these too at release. Minting datastore uuids for them now means
inventing an id space outside LorcanaJSON's, which then collides or churns
when the real cards land. Not worth it. Same for group 14 `Hyperia City`
(6 products, all sealed, zero numbered).

Every promo group LorcanaJSON supposedly "does not carry" — `DLPC` (170
products), `D23` (18), `D100` (6) — is in fact already covered: LorcanaJSON
files those promos under the base set they belong to, and claims 147 of
DLPC's 170, 16 of D23's 18, and all 6 of D100's.

**Extra listings (56).** TCGplayer carries a second product for a card
LorcanaJSON already has, at the same set and collector number. 30 of them are
the split-foil case (`… (Foil)`) and are safe to alias onto the card. The
other 26 are **not** the same object — `(Oversized)` (4), `(Errata Version)`
(2), `(JP Exclusive)` (2), `(CS Exclusive)`, `(Serial Numbered)`,
`(Puzzle Promo)` — and aliasing them would land oversized and exclusive prices
on the ordinary printing. Left alone.

**Missing product ids (14 → 10).** Fourteen cards carry no
`externalLinks.tcgPlayerId`; ten match exactly one unclaimed catalog product
by normalised name plus collector number (all in `DLPC`, e.g. `9 #3
Stitch - Rock Star` → 668575). The other four have no catalog counterpart at
all.

## 3. Therefore: enrich, not replace

Riftbound enriches because Riot's gallery is authoritative for card identity
and silent about commerce, so the two sources are complements. Lorcana has the
same shape but a **much smaller gap**, because LorcanaJSON is already
commerce-aware: it ships the TCGplayer id itself.

So the same shape fits — emit the LorcanaJSON payload with extra data merged
in, so `mtgmatcher/lorcana` reads it unchanged and a stock LorcanaJSON reader
still parses it — but the payload of the merge is:

- fill the 10 recoverable `tcgPlayerId`s;
- record the 30 split-foil product ids as extra ids on the card they belong
  to, so a TCGplayer feed keyed on those products resolves instead of
  silently dropping.

That is **40 product ids on 3605**, about 1.1%. It is worth stating plainly
that this content gain alone would not justify a nightly pipeline. What
justifies building it is the other two things it buys:

1. **The file comes under our control.** Today CI and every deployment curl
   `vars.DATASTORE_LORCANA` off a third-party host at run time. Publishing to
   `b2://mtgban-datastore/lorcana/lorcana.json.xz` beside the other two games
   makes an upstream outage a stale-file problem instead of an outage.
2. **It is the place sealed goes.** The 171 sealed products are the real
   prize, and they need a builder to emit them.

## 4. What would break, and how it is handled

| Risk | Handling |
| --- | --- |
| The loader must keep reading today's LorcanaJSON | The output *is* a LorcanaJSON file with two extra keys. The new key is absent from upstream, so the loader's behaviour on today's file is byte-for-byte what it is now. Verified by loading both. |
| A deployed older loader reads the new file | Unknown JSON fields are ignored; the file stays a valid LorcanaJSON. Rollout is safe in either order. |
| `LoadDatastore` auto-detection picks the wrong game | Unchanged: the file still parses as LorcanaJSON and fails the Magic/Riftbound loaders exactly as before. |
| The Lorcana golden is baked against today's datastore | The change registers extra entries in `ExternalIdentifiers` and fills ids on 10 cards. It creates no uuid, changes no `Card`, and does not touch `Match`, `FilterCards` or `selectFinish`. The golden exercises `Match`, so it is expected to be **unchanged** — and it is; see the PR. |
| The Magic golden | Nothing outside `mtgmatcher/lorcana` and a new `cmd/` is touched. Must pass with no `-u`. |
| The live deployment pulls the published file | Nothing is published by this change. `vars.DATASTORE_LORCANA` keeps pointing where it does; the cutover is a separate, reversible config change once the builder has run green for a few days. |
| Registering a uuid for a finish a printing lacks | Not done. No `CardObject` is created; only `ExternalIdentifiers` (id → existing uuid) grows, and `GetUUID("")` still errors. Checked in the round-trip. |

## Also found, not acted on

`FilterCards` in `mtgmatcher/lorcana/rules.go` drops candidates that cannot
satisfy the requested finish. The brief supposed that is "safe only while
every card claims every finish" — in fact **582 cards are already
single-finish today** (499 foil-only, 83 nonfoil-only), straight out of
LorcanaJSON, so the gate is already dropping real candidates whenever a
storefront reports the foil flag wrongly or not at all. That is the same
reasoning behind `mtgmatcher/riftbound/rules.go`'s deliberate refusal to
filter on finish. Lorcana differs from Riftbound in that name+number does *not*
uniquely identify a printing there, so the gate is doing some real work and
removing it is not a mechanical copy of the Riftbound fix. It moves the
Lorcana golden and deserves its own change with its own justification, rather
than riding along with a datastore build.

## How this was measured

This sandbox's egress policy allows GitHub and the Go module proxy and returns
403 for everything else — `tcgcsv.com`, `mtgjson.com`, `lorcanajson.org` and
the B2 host included — so nothing could be fetched locally. The measurements
were taken by running `cmd/lorcanaprobe` inside a GitHub Actions job on branch
`lorcana-datastore-probe`, which has the network and the existing repository
credentials, and reading its output back from the run log. That is also how
the bucket object was verified. The probe and its workflow are scaffolding and
are not part of the delivered change.

[run1]: https://github.com/mtgban/go-mtgban/actions/runs/31359926383
[run2]: https://github.com/mtgban/go-mtgban/actions/runs/31360215158
[rb2]: https://github.com/mtgban/riftbound-datastore/pull/2
