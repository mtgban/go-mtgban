# Foreign Black Border and 4th Edition FBB languages

Both sets were printed in several languages, and mtgjson files each card
once, under one of them, with the others in `foreignData`.

| Set | Printed in | mtgjson's card | Sold as (`forcedLanguages`) | Other languages (`languageSets`) |
|---|---|---|---|---|
| FBB | French, German, Italian | French | Italian | FBBFR, FBBDE |
| 4BB | Chinese Traditional, Japanese, Korean, Portuguese (Brazil), Spanish | Spanish | Japanese | 4BBES, 4BBPT, 4BBKO, 4BBZHT |

- The base set takes the language US storefronts mostly sell: the per-set
  loop of `matcher/magic/mtgjson.go` gives each card that language's name,
  image and Scryfall id. A card never printed in it keeps its own (FBB's
  Forest #306a exists in German alone).
- Every other language is a set of its own, minted by `duplicate` like
  LEGITA, holding only the cards printed in it. A copy's uuid is the one
  mtgjson gives that language in `foreignData`, so `MatchIDLanguage` finds it
  from the base card, as the TCGplayer and Scryfall id paths do.
- A listing reaches a language set by naming the base set and its language
  (`languageSet` in `matcher/magic/rules.go`), or the language set by name.

Scryfall never scanned some of these printings and serves a stamped English
placeholder for them; mtgjson carries no scan status to avoid it.

## Why `mtgjsonId` stays mtgjson's card uuid

A base card sold in a language other than mtgjson's keeps mtgjson's card in
`Identifiers["mtgjsonId"]`: the French FBB or Spanish 4BB printing. So does a
copy, for the card it was made from. Every reader of the key means mtgjson's
card:

- The loader files it in `IDSpaceMTGJSON`, the index a vendor's mtgjson
  uuid is resolved through, and tells a copy from its original by it.
- The site's price API returns it in its `mtgjson` id mode, where a consumer
  would look it up in AllPrintings' card list, which holds no foreign uuids.
