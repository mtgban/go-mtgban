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

- The override is `forcedLanguages` in `mtgmatcher/magic/table.go`, applied
  in the per-set loop of `mtgjson.go`: US storefronts mostly sell Italian
  FBB and Japanese 4BB. A card takes the language only if it was printed in
  it (FBB's Forest #306a exists in German alone).
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
- The base cards show their forced language's image since #769, but the
  name the site quotes beside the flag is still mtgjson's `flavorName`,
  which for these sets holds its own printing's name ("Animation de mur",
  "Poción de alabastro"). Flavor and printed names are indexed for matching,
  so changing them needs a replay of its own.
- Scryfall never scanned many of these printings and serves a stamped
  English placeholder instead. Measured when #769 landed: 24 cards gave up a
  real French or Spanish scan for an Italian or Japanese placeholder, and
  mtgjson carries no scan status to avoid it.

## Why `mtgjsonId` stays mtgjson's card uuid

A card sold in a language other than mtgjson's own takes that language's
image through `originalScryfallId`: every FBB and 4BB card, and the language
copies (LEGITA, DRKITA, the SLD and PURL Japanese ones). Its
`Identifiers["mtgjsonId"]` still holds mtgjson's card: the French FBB or
Spanish 4BB printing, or the English card a copy was made from. The uuid
mtgjson 5.3 gives the language itself, `foreignData[].uuid`, is not decoded
at all: the loader's `ForeignData` struct has no field for it.

It stays that way because every reader of the key means mtgjson's card:

- `tcgplayer.go` looks TCGplayer's skus up by it, and mtgjson files the skus
  of every language under the card. Swapping it would cost FBB and 4BB
  their TCGplayer prices (TCG Market priced 168 and 249 of them on
  2026-09-24).
- The loader files it in `IDSpaceMTGJSON`, the index a vendor's mtgjson
  uuid is resolved through.
- The site's price API returns it in its `mtgjson` id mode, where a
  consumer would look it up in AllPrintings' card list, which holds no
  foreign uuids.

To address it later: decode `foreignData[].uuid`, keep it under a key of its
own beside `originalScryfallId` (say `originalMtgjsonId`), file it in
`IDSpaceMTGJSON` too so a vendor sending it resolves, and decide per reader
which of the two it wants. Mind the shared `Identifiers` map above before
writing anything per-language into it.

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
