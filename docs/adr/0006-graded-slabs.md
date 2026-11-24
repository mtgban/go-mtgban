# ADR-0006: Graded slabs carry their label; graded stores are flagged singles

**Status:** Accepted
**Date:** 2026-11-22

## Context

Four stores sell professionally graded copies ("slabs": PSA 10, BGS 9.5,
CGC Pristine 10) and we publish them today:

| Store | How it reaches a dump today | Graders read |
|---|---|---|
| Card Kingdom | its own scraper, `cardkingdom_graded` (`CKGraded`) | PSA, BGS, CGC |
| ABU Games | sub-seller of `abugames`, "ABU Games Graded" (`ABUGraded`) | PSA, BGS, CGC |
| Cool Stuff Inc | sub-seller of `coolstuffinc`, "Cool Stuff Inc Graded" (`CSIGraded`), eight games | PSA, BGS, CGC |
| Hareruya | sub-seller of `hareruya`, "Hareruya Graded" (`HAGraded`) | PSA, BGS, CGC |

Every one of them folds the slab into a condition through
`mtgban.SlabCondition` (PSA 9 is NM, BGS 8 is SP) and keeps nothing else.
The grader, the score, a half point (BGS 9.5 reads as 9) and the premium
labels (CGC Pristine 10, BGS Black Label 10) are gone by the time the entry
is added. `InventoryEntry` has no place to hold them, and no scraper writes
them into `CustomFields` or `InstanceID`. Card Kingdom's graded entries
carry no `InstanceID` either, so two slabs of one card at one price and
condition merge into one entry of quantity two and lose the second's link.

Coverage is partial. Cool Stuff Inc reads only numeric PSA, BGS and CGC
labels: TAG slabs, CGC Pristine and any score outside the fold table are
forced to NM and filed under "Cool Stuff Inc (unique)". ABU lists slabs whose
features say "Graded" and name no grader, and others in a "Less than 8"
bucket with a grader and no score; both are graded by ABU's own shelf
condition. CardTrader publishes a `graded` flag per product: the singles
scraper drops every such listing, and `cardtrader_sealed`, in eight games,
does not filter it at all, so a slabbed pack is priced as raw sealed. Card
Game Corner, Mint Card, MTG Seattle and two unregistered scrapers drop slabs
on sight. No buylist reads a slab: ABU's slab branch ends before its buylist
code, and no other buylist feed names a grader.

On mtgban-website a store's category is read only from the dump's
`ScraperInfo.SealedMode`; nothing in the site's configuration names it.
Outside tests it is read 58 times across about twenty files: arbit's "Keep categories separate",
the v1 and v2 API sections, BASE_ACCESS eligibility, uploads, alerts,
settings grouping, Sleepers. A graded store has `SealedMode == false`, so it
is a singles store everywhere. Search lists a PSA 10 under "Condition: NM"
beside raw copies, which is what we want to keep. But Global and Reverse
also compare a $400 slab against a $4 raw NM, `/data/prices` can pick a slab
as a card's best NM, and an NM price alert can fire on one.

We want a Graded section on the site that searches only slabs and shows the
grader and score instead of NM/SP, while the singles view keeps showing
graded stores as it does today, with the grader and score beside the store
name.

## Decision

### 1. A slab keeps its label on the entry

`InventoryEntry` gains

```go
// Slab is a grading service's label on one copy. Grader and Score are
// empty when the store does not say them.
type Slab struct {
	Grader string  `json:"grader,omitempty"` // "PSA", "BGS", "CGC", "TAG"
	Score  float64 `json:"score,omitempty"`  // 1 to 10, in half steps
	Label  string  `json:"label,omitempty"`  // "Pristine", "Black Label"
}

Slab *Slab `json:"slab,omitempty"`
```

A nil `Slab` is a raw copy; a non-nil one is a slab, even an empty one (ABU's
graderless "Graded" rows). The pointer costs eight bytes on every entry of
every store, where an inline struct would cost forty.

Graders come from a closed list in `mtgban`, the ones the four stores name
today (PSA, BGS, CGC, TAG), and scores are 1 to 10 in half steps; a scraper
maps the store's wording onto those, the way a `<tag>Condition` helper maps
condition words before `ParseCondition`. `Label` is free text and is kept
and shown as the store printed it: a new premium label must not cost the
listing. A score the store does not give (ABU's "Less than 8") is zero, and
the bucket goes in `Label`.

Cert numbers and subgrades are left out. ABU's subgrades are copied between
slabs, Card Kingdom's title number is not known to be a cert, and Hareruya's
autographed slabs are dropped before grading; each can be added later as an
`omitempty` field without touching a reader.

### 2. The singles view of a slab is a fold, and the scraper does it

`SlabCondition(grader, score)` becomes `Slab.Fold() (Condition, bool)`, over
the same per-grader table keyed by the whole score (half points round down,
as they do now). A label that means a top score sets the score too (CGC
"Pristine" is 10). A slab with no score folds only by a label the table
names (CGC "Authentic Altered" is PO, as today); any other does not fold.
The scraper calls it and sets `Conditions`, as all four do today.
`Conditions` stays what every singles consumer reads: `Mismatch`, the
insert order, the API, the site's condition buckets. Where the scraper
knows better than the table (ABU's graderless and "Less than 8" rows,
graded by shelf condition) it sets `Conditions` itself.

A slab the scraper cannot read - a grader outside the list, or a score or
label the table does not fold - is still published, as NM, in the graded
seller, with a `Slab` holding what could be read and the store's wording in
`Label`; a grader outside the list leaves `Grader` empty. The scraper logs it
loudly with that wording (`unsupported slab "SGC 10", published as NM`). The
listing is not lost while support for a new grader is added, and the log
line is what says it is due. This replaces today's mix, where Card Kingdom,
ABU and Hareruya drop such a slab and Cool Stuff Inc files it as a raw NM
copy under CSIUnique.

`add` gains one rule: an entry whose `Slab` has a non-empty `Grader` outside
the list, or a non-zero `Score` that is not 1-10 in half steps, or that has a
`Slab` and an empty `Conditions`, is refused with `ErrInvalidCondition`,
ahead of the default that turns an empty condition into NM. A slab is never
silently NM: the scraper chooses NM for an unread slab and says so. `add`
keeps entries whose slabs differ apart, a slab beside a raw copy included,
before any duplicate check, so the slab is part of the key under every `Add`:
only the same slab twice can merge, or be refused as a duplicate. Card
Kingdom's graded scraper also sets `InstanceID` from the product id it
already reads, so two slabs with the same label stay two offers, each with
its own link.

### 3. A graded store is a singles store with a flag

`ScraperInfo` gains

```go
// The seller lists graded slabs only
GradedMode bool `json:"graded,omitempty"`
```

beside `SealedMode`, not in place of it. A graded store is still a singles
store - it lists single-card ids and folds to NM/SP - so every site branch
on `SealedMode` keeps treating it as singles, which is what search wants.

The flag describes a published seller, not a scraper's whole record. The
marketplace parents (`abugames`, `coolstuffinc`, `hareruya`) mix raw copies
and slabs under one `Info()`; their `InfoForScraper` sets `GradedMode` for
the graded sub-seller, and `UnfoldScrapers` already writes each sub-seller
with that `Info`. A slab and a raw copy are never published in one seller:
every entry of a seller with `GradedMode` has a `Slab`, and no entry of any
other seller does. `Add` cannot see `ScraperInfo`, so each of the four
scrapers pins this in its own test.

A graded store is named for its store, the way a sealed one already is:
Card Kingdom's sealed scraper is "Card Kingdom" beside the singles one, and
the site tells them apart by `SealedMode`. So "Card Kingdom Graded" becomes
"Card Kingdom" with `GradedMode`, and likewise for ABU, Cool Stuff Inc and
Hareruya; the slab beside the name says the rest. The shorthand stays as it
is (`CKGraded`, `ABUGraded`, `CSIGraded`, `HAGraded`): it is the key for
dumps, configuration, saved settings and alerts, and must stay unique. The
marketplace parents keep "Cool Stuff Inc Graded" and the like as the
internal `SellerName` that splits their record; only the `Name` that
`InfoForScraper` publishes changes. Wherever the site lists stores of both
kinds side by side it labels the graded one, as `storeLabel` does for
sealed, and its one lookup by name (the redirect fallback,
`findSellerInventoryByName`) matches `GradedMode` as it matches
`SealedMode`.

Graded sealed product is dropped, not modelled: `cardtrader_sealed` skips
`product.Graded` as the singles scraper does. Two booleans make a store that
is both representable; no scraper sets both, and the site reads
`SealedMode` first.

### 4. The Graded section on the site

A Graded page beside `/sealed` searches only `GradedMode` sellers and shows
grader, score and label in place of the condition buckets, grouped by
grader, then score descending, then label; a slab whose store names no
grader or score shows as "Graded" with its label, last.

The Graded page has no buylist: no store buys slabs. Its Buyers column
shows last-sold data only, with no switch to a buylist, the way sealed
search shows its buylist only. Those sales are graded sales, keyed by card
and slab: a card's raw sales from the recent-sales feed never show there,
since a raw copy's price says nothing about a PSA 10's. No source of graded
sales exists yet, so the pane shows its empty state ("No sales on record")
until one is added.

Singles search keeps listing graded stores under their folded condition, as
today, and names the slab beside the store: "Card Kingdom - PSA 10",
"ABU Games - CGC Pristine 10". Uploads and settings keep listing graded
stores with the singles ones, labelled. The API's singles endpoints leave
them out (decision 5); a graded endpoint is not part of this decision, since
the API's per-condition shape (NM to PO) has no place for a grader and
score.

### 5. Slabs leave the price comparisons

This lands in its own PRs, after 1-4. A slab and a raw copy of one card are
not the same goods, so a spread between them is never real. The callers leave
graded sellers out; `Arbit` and `Mismatch` compare the sellers they are given
and do not look at `Slab`. No published seller mixes the two (decision 3), so
a caller reads `GradedMode` on each seller, with marketplace parents unfolded,
and never `Slab` on each entry. On the site one predicate,
`rawSingles(info) = !info.SealedMode && !info.GradedMode`, picks the stores
for:

| Consumer | Graded stores |
|---|---|
| Singles search, `/search` | kept, folded condition, slab named beside the store |
| Graded page | only these |
| Arbit, Global, Reverse | skipped |
| `/data/prices` best NM | skipped |
| Price alerts | skipped |
| Sleepers | skipped |
| API v1/v2 singles endpoints and store lists, BASE_ACCESS | skipped |
| Upload, settings pickers | kept, labelled as graded |
| Price history stash | not affected: it stashes only the shorthands its datasets name, none graded |
| Carts | not affected: graded splits already get no button |

## Alternatives

| Option | Benefit | Cost |
|---|---|---|
| Replace `SealedMode` with `Category string` (`singles`/`sealed`/`graded`) | One field; a sealed-and-graded store cannot be written | A graded store stops being singles, the opposite of what search wants; all 58 site reads change, and each `!= sealed` branch must decide again where graded goes; old dumps need a migration rule |
| Spell the slab in `Conditions` ("PSA 10") | No new field | Breaks the closed grade set every consumer relies on: `add`'s validation, the insert order, `Mismatch`'s grade map, the site's condition buckets, the alerts table's CHECK constraint |
| Grader and score in `CustomFields` | No type change | Stringly typed, every reader re-parses it, the grader list lives nowhere |
| A canonical `Score string` checked against the fold table | One field, sorts as text | Every grader × half point × label must be listed, and a new label drops the listing instead of showing it |
| No store flag; the site looks for entries with a `Slab` | Nothing new on `ScraperInfo` | A store's category would flip on a day it lists nothing, and the site needs it at load time for menus and arbit's store lists |
| A separate graded record type and interface | Full freedom of shape | A slab is still a single card on the same uuid; new readers, writers and unfold paths for four stores |
| Keep the library; add the four shorthands to the site's arbit and search block lists | Fixes the comparisons today, no schema change | Leaves the label lost, so there is nothing to build the Graded page from; and a list in config drifts from the stores |
| `Arbit` and `Mismatch` skip an entry with a `Slab` | Every caller is covered at once | Puts a choice of sellers inside the reports, entry by entry, when the seller's flag already answers it |

## Consequences

The library change is additive for anyone reading dumps: `slab` and
`graded` are new JSON keys, omitted when empty, and `decodeRecord` ignores
keys it does not know. A site on the old library reads new dumps, and a site
on the new library reads an old dump as raw singles, so the Graded page and
the exclusions take effect store by store, as each graded store runs on the
new version. bantool's NDJSON output embeds `InventoryEntry` and gains the
`slab` key the same way. The CSV writers get no column: a graded seller
written as CSV loses its slabs on reload, and the site reads JSON.

The one source break is `SlabCondition` giving way to `Slab.Fold()`; its five
callers are in this repository, and it is removed only after each has moved.
bantool builds from master, so the new keys reach production dumps as each
store's change merges, before any site bump reads them.

Every row whose wording names a grader becomes a slab in the graded seller.
Cool Stuff Inc's TAG and CGC Pristine slabs, and any it cannot read, move
from CSIUnique to CSIGraded, which leaves CSIUnique the non-grader markers
("Shadowless", "No Set Symbol"). Card Kingdom, ABU and Hareruya start
publishing, as NM with a loud log line, the slabs they drop today (ABU's
"BGS Quad 10" among them). Each count comes from one crawl before the change
lands. CardTrader's graded singles become publishable; CardTrader splits its
sellers by class (CT, CT0, CT1DR), so that means a graded sub-seller per
class that has slabs.

The site's existing `gradedStore` helper means "files prices by condition"
and has nothing to do with slabs; it gets a new name before `GradedMode`
lands beside it.

## Action items

- [x] go-mtgban: `Slab`, the grader list, `Slab.Fold()`, `add`'s refusal,
      `ScraperInfo.GradedMode`; `SlabCondition` removed once its callers
      have moved.
- [ ] go-mtgban: Card Kingdom, ABU, Cool Stuff Inc and Hareruya set `Slab`
      and `GradedMode`, and publish an unread slab as NM with a loud log;
      Card Kingdom sets `InstanceID`; CSI's slabs all move to CSIGraded.
- [ ] go-mtgban: `cardtrader_sealed` skips graded products.
- [ ] go-mtgban: CardTrader graded singles, once a raw record shows what the
      feed says about the grade.
- [ ] mtgban-website: rename `gradedStore`; graded labels in pickers and
      the redirect lookup; the Graded page; the slab beside the store name
      in singles search.
- [ ] go-mtgban, after the site labels them: graded stores named for their
      store.
- [ ] Decision 5, at a later stage: mtgban-website's `rawSingles` predicate,
      the API included.
- [ ] Docs: `SPECIFICATIONS.md` and `coolstuffinc/README.md` name all four
      graded sellers.
