# Language sets: foreign borders of Revised and 4th Edition, Renaissance

These sets were printed in several languages, and mtgjson files each card
once, under one of them, with the others in `foreignData`.

| Set | Printed in | mtgjson's card | Sold as (`forcedLanguages`) | Other languages (`languageSets`) |
|---|---|---|---|---|
| FBB | French, German, Italian | French | Italian | FBBFR, FBBDE |
| 4BB | Chinese Traditional, Japanese, Korean, Portuguese (Brazil), Spanish | Spanish | Japanese | 4BBES, 4BBPT, 4BBKO, 4BBZHT |
| REN | French, German | French | French | RENDE |
| 3ED | English, French, German, Italian | English | English | FWB (Italian), FWBFR, FWBDE |

- FBB and 4BB take the language US storefronts mostly sell: the per-set
  loop of `matcher/magic/mtgjson.go` gives each card that language's name
  and image. A card never printed in it keeps its own (FBB's Forest #306a
  exists in German alone).
- Every other language is a set of its own, minted by `duplicate` like
  LEGITA, holding only the cards printed in it. Its parent is the base set,
  so the site stacks it with the base under the base's own parent (Revised,
  Fourth Edition). A copy's uuid is the one mtgjson gives that language in
  `foreignData`, so `MatchIDLanguage` finds it from the base card, as the
  TCGplayer and Scryfall id paths do. The base card's own entry names the
  base, so a copy finds its way back too, except to 3ED: mtgjson lists no
  English entry for its cards.
- A listing reaches a language set by naming the base set and its language
  (`languageSet` in `matcher/magic/rules.go`), or the language set by name.
  One whose edition names neither is never sent there by its language alone
  (`isLanguageSet` in `matcher/magic/candidates.go`).
- Foreign Revised is white-bordered, sold apart as Foreign White Border.
  mtgjson has an FWB set holding its sealed products alone, which keeps them
  and takes Revised's Italian cards; FWBFR and FWBDE are made from it. No
  vendor sells them under Revised's products, so they carry none of its
  vendor ids, and TCGplayer's products are linked by name
  (`tcgplayer/cataloglinks.go`). Foreign white-bordered 4th Edition is not
  loaded.

Scryfall never scanned some of these printings and serves a stamped English
placeholder for them; mtgjson carries no scan status to avoid it.

## Why `mtgjsonId` and `scryfallId` stay mtgjson's card

A base card sold in a language other than mtgjson's keeps mtgjson's card in
`Identifiers["mtgjsonId"]` and `["scryfallId"]`: the French FBB or Spanish
4BB printing. A copy keeps it in `mtgjsonId`, for the card it was made from,
and takes its own language's `scryfallId`. Vendors name the card by mtgjson's
ids in every language (Card Trader's blueprints carry the Scryfall one) and
the listing's language picks the printing, so moving either to the base's
sold language would send those listings to the copy instead. Every reader of
`mtgjsonId` means mtgjson's card:

- The loader files it in `IDSpaceMTGJSON`, the index a vendor's mtgjson
  uuid is resolved through, and tells a copy from its original by it.
- The site's price API returns it in its `mtgjson` id mode, where a consumer
  would look it up in AllPrintings' card list, which holds no foreign uuids.

## Sealed products

mtgjson lists FBB's and FWB's boosters, boxes and starter decks without
saying what the pack and the starter deck hold. Both open like Revised's
(`packedLike` in `matcher/magic/table.go`), from the Italian set alone, and
only that set's cards name them in `SourceProducts`: the French and German
sets name no product.
