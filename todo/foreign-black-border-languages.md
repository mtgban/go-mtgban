# Todo: language copies for Foreign Black Border and 4th Edition FBB

Written for whichever agent picks this up next. Nothing here is built yet;
this is the design agreed on 2026-09-24, when "mtgmatcher/magic: keep
numbers on whole-set copies" stopped tagging the numbers of LEGITA, DRKITA
and 4EDALT and kept the tag only for copies filed next to their original.

## Where things stand

Both sets were printed in several languages, but the backend carries each
card once, in one language, and refuses every other on purpose.

| Set | Printed in (mtgjson `foreignData`) | mtgjson's own card language | Loader forces |
|---|---|---|---|
| FBB | French, German, Italian | French | Italian |
| 4BB | Chinese Traditional, Japanese, Korean, Portuguese (Brazil), Spanish | Spanish | Japanese |

- The override is the `switch set.Code` in `mtgmatcher/magic/mtgjson.go`
  (`case "FBB"`, `case "4BB"`): US storefronts mostly sell Italian FBB and
  Japanese 4BB.
- The refusals are pinned: Cool Stuff Inc refuses the other languages in
  `preprocess()` (#610), and SCG's `TestResolveProductForeignSets` asserts
  German FBB and Korean / Chinese Traditional 4BB stay unmatched.
- Since mtgjson 5.3 every `foreignData` entry carries its own `uuid`,
  `identifiers.scryfallId` and `skuIds`, so each language is addressable.

## The design

1. Mint the missing languages as copies in the same set with
   `duplicateCards`, the way SLD and PURL mint their Japanese copies. The
   base keeps its forced language, so every uuid in use today keeps its
   meaning; a copy is `<uuid>_<tag>`.
2. Keep the tag on the copy's number too ("1fra"). Magic has no duplicate
   (set, number) pair today and these read the pair as a key: the site's
   `name|set|number` price-API key (`api_banprice.go`), `printingsAt`
   (`redirect.go`), upload's `MatchWithNumber` (`upload.go`), and SCG's
   twin lookups (`MatchWithNumber("", set, number)` in `catalog.go`).
3. Add the tags to `langs` in `mtgjson.go`, e.g. FRA and DEU for FBB, KOR,
   ZHT, SPA and POR for 4BB. Pick them once: the uuid tag is persisted by
   every consumer.
4. `FilterCards` appends "jpn" to the number when the listing is Japanese
   (`case inCard.IsJPN()`). Replace that branch with a lookup from the
   listing's language to its tag, the inverse of `langs`.
5. Widen `originalOverCopy` (`candidates.go`). Today it drops a copy only
   when it shares its original's language (the 4EDALT case); make a copy
   win only when the listing asks for its language. Without that, a FBB
   listing naming no language keeps every copy, and core's language check
   (no language means English) then drops all of them.

## Traps found while planning

- `duplicateCards` copies the card struct, which shares the `Identifiers`
  map with its base. Every copy's `originalScryfallId` already lands on its
  base: all 310 English LEG cards, 119 DRK cards and the 54 SLD bases carry
  their copy's. It is harmless only because the image builder is the one
  reader and runs before the copy is minted. Clone the map before writing
  anything per-language into it, or the last language minted overwrites the
  base's ids.
- `duplicateCards` finds a copy's printed name and Scryfall id by exact
  `foreignData.Language`. mtgjson says "Portuguese (Brazil)"; core's tag is
  "Portuguese" (`mtgmatcher/table.go`), so the lookup would miss.
- The base images already show the wrong language: FBB shows the French
  printing and 4BB the Spanish one, because the loader forces `Language`
  but not `originalScryfallId`. Tracked as its own follow-up.

## What to measure before merging

- A per-listing diff on SCG, Cool Stuff Inc, TCGplayer index, Cardmarket
  and CardTrader: nothing that resolves today may move, and the listings in
  the new languages should land on their copies.
- TCGplayer's sku loop in `tcgplayer.go` keeps a sku only when its
  language is the card's, so it should price the copies unchanged. That
  has not been checked.
- Cardmarket and CardTrader send the language as a listing attribute;
  check each scraper passes it on as `InputCard.Language`.
- The site's price history keys on `uuid[:36]` plus language, so copies
  get rows of their own without a schema change.
