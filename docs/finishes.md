# Finish names

A datastore game names a finish the way TCGplayer prices the printing, with
case and separators dropped and the plain printing called `nonfoil`:
`FinishSlug("Cold Foil")` is `coldfoil`, `FinishSlug("Normal")` is
`nonfoil`. That name keys `Card.FoilUUIDs` and is `CardObject.Finish`. It is
read off the `finish` field a datastore entry publishes; nothing reads a
finish off a uuid or spells a uuid from a finish. datastore-gen ends every
uuid in the same spelling, but a consumer must not rely on it.

## The table

`mtgmatcher.Finishes` holds every printing TCGplayer's catalogs define for
the eight datastore games, checked against the catalog dumps on 2026-09-25.

| slug | TCGplayer | run | treatment | foil |
|---|---|---|---|---|
| `nonfoil` | Normal | – | `nonfoil` | no |
| `foil` | Foil | – | `foil` | yes |
| `holofoil` | Holofoil | – | `holofoil` | yes |
| `reverseholofoil` | Reverse Holofoil | – | `reverseholofoil` | yes |
| `coldfoil` | Cold Foil | – | `coldfoil` | yes |
| `rainbowfoil` | Rainbow Foil | – | `rainbowfoil` | yes |
| `1stedition` | 1st Edition | 1st Edition | `nonfoil` | no |
| `unlimited` | Unlimited | Unlimited | `nonfoil` | no |
| `limited` | Limited | Limited | `nonfoil` | no |
| `1steditionholofoil` | 1st Edition Holofoil | 1st Edition | `holofoil` | yes |
| `unlimitedholofoil` | Unlimited Holofoil | Unlimited | `holofoil` | yes |
| `1steditionnormal` | 1st Edition Normal | 1st Edition | `nonfoil` | no |
| `1steditionrainbowfoil` | 1st Edition Rainbow Foil | 1st Edition | `rainbowfoil` | yes |
| `1steditioncoldfoil` | 1st Edition Cold Foil | 1st Edition | `coldfoil` | yes |
| `unlimitededitionnormal` | Unlimited Edition Normal | Unlimited | `nonfoil` | no |
| `unlimitededitionrainbowfoil` | Unlimited Edition Rainbow Foil | Unlimited | `rainbowfoil` | yes |

Each TCGplayer name reads back through `FinishSlug` to its slug, which is
how "Normal" becomes `nonfoil`: from its row, not from a spelling of its
own. A printing TCGplayer adds later loads under its own name and is taken
for a foil; `TestPublishedFinishesHaveARow` reports it until it has a row.

## What answers a finish name

`FinishUUID` reads the name through `FinishSlug`, the same for every game, and
looks it up among the printing's own finishes. `nonfoil` and `foil` also name
the printings the bare flags answer with, which `DefaultPrinting` picks from
the table: of the foils or of the rest, the treatment listed first (Rainbow
Foil, Cold Foil, Holofoil, Reverse Holofoil), in its plainest run (none,
Unlimited, 1st Edition, Limited). So "Foil" reaches Gundam's Holofoil and
Lorcana's Cold Foil. A finish the printing is not sold in answers with one
differing from it by print run alone:

- a name naming no run, on a product sold only in runs, takes the unlimited
  run, then the first edition ("Holofoil" on a Base Set card);
- a run named on a card never printed in it is dropped ("1st Edition Rainbow
  Foil" on a card printed once), where the card is not sold in that run on
  another product: Base Set's unlimited Alakazam is plain Holofoil, and its
  1st Edition is the Shadowless product's;
- one run never answers for another, and a finish the datastore sells nowhere
  is not answered at all.

Anything else is refused: a storefront's own spelling ("Reverse Holo",
"Holographic") is its scraper's to translate. A scraper building a name from
a shelf's run and a treatment goes through `PrintingFinish`, which drops the
run where TCGplayer prices no such printing ("Unlimited Edition Cold Foil").

That is the finish a caller names. A listing naming its printing only in its
wording ("1st Edition Holo", CardTrader's "Alpha") is read by its game for a
run and a treatment, each in TCGplayer's words, and `NamedFinish` answers the
plainest printing sold in both. One treatment never answers for another there
either: "Holo" does not reach a Reverse Holofoil.

Lorcana's treatments (Satin, Rainbow Pillars) are promo types, not finishes:
a listing naming one in its wording reaches the printing that carries it,
and one sending it as the finish falls through to the wording.

## A product id answers for its product

A uuid names one printing, and `MatchIDFinish` answers it from that
printing's own finishes, as above. A vendor's id names a product, which can
hold more than one printing, so it is answered from the whole product:

- A finish the printing the id files at is not sold in comes from a set-mate
  sold under the same product: one carrying the same id and no other id of
  that kind. 7th to 10th Edition file a card's foil as a printing of its own,
  numbered with a star, and TCGplayer sells the two as one product: 3040 is
  Raise Dead #157 as Normal and #157★ as Foil, whichever of the two the id
  files at. Aether Revolt's Alley Strangler shares 126455 with its starter
  deck printing #52†, and only #52 is sold in foil.
- A star foil carrying no id of that kind is its card's product's foil, as
  `tcg_index` files it: 95037 Foil is Fate Reforged's Crux of Fate #65★. An
  etched star twin is not, being sold under an etched product of its own.
- An id filed at an etched printing answers Foil with it: TCGplayer sells an
  etched foil as a product of its own under the printing Foil, so 233370 Foil
  is Swords to Plowshares' etched #10, and 233369 Foil its plain foil.

A set-mate carrying another id of the kind has a product of its own and is
never reached: the surge foil the loader files apart from Meteor Golem
carries the card's `tcgplayerAlternativeFoilProductId`, and 698282, the
surge foil's product, still sells no Normal. Where two set-mates would answer
differently the finish is refused; no datastore row reaches that today.

### Measured when product ids landed

With go-mtgban df3c4efc8 as the base and allprintings5.json 5.3.0+20260927:

- Every English near-mint sku in MTGJSON's TcgplayerSkus 5.3.0+20260905,
  keyed by product and printing as a listing is, and graded against the uuid
  MTGJSON files it under in the sku's own finish (145,872 keys of cards):

  | | right | wrong | refused |
  |---|---|---|---|
  | `Match` with the printing as the finish, before | 143,444 | 287 | 2,141 |
  | the same, after | 145,847 | 3 | 22 |
  | the SYP scraper's name-then-flags lookup, before | 145,847 | 3 | 22 |
  | the same, after | 145,850 | 0 | 22 |
  | `tcg_index`'s product map, unchanged | 145,850 | 0 | 22 |

  The 287 were etched products answered with the plain foil (Strixhaven
  Mystical Archive 126, Modern Horizons 2 113, Modern Horizons 1 Timeshifts
  40, Secret Lair 8), and the 2,119 "unknown finish" refusals the star foils,
  the etched products and Alley Strangler's foil. The 3 left are Secret Lair
  #159 to #161: Scryfall gives each
  card its etched twin's product id and MTGJSON copies it, so the id files at
  the card. The 22 are ids the datastore carries no printing for. SYP's 3
  were the flags' twin: 7th Edition's Chinese alt-arts #157★s and #161s,
  and Alley Strangler #52†.
- Every MTGJSON, Scryfall and TCGplayer id on the Magic datastore asked for
  each finish and both flags, and every id through `ConvertID` (2.32M
  answers): 4,552 refusals now land, 284 etched products move from the plain
  foil to the etched one, nothing that landed is refused, and no flag or
  `ConvertID` answer moved. The eight datastore games published 2026-09-28
  answer every id and finish as before.

## Measured when this landed

On the datastores published 2026-09-24 and 25, frozen for the comparison, with
datastore-gen#97's Lorcana and Gundam, and go-mtgban 45c4b189a as the base:

- `CardObject.Finish` changed on 10,784 uuids and nowhere else: Flesh and
  Blood's Normal (7,011) `normal` to `nonfoil`, Gundam's Holofoil (1,046)
  `foil` to `holofoil`, Lorcana's Cold Foil (2,727) `foil` to `coldfoil`. No
  foil flag moved, and every `nonfoil`/`foil` default points where it did.
- Every uuid asked for 37 finish names and both flags, and every product id
  for both flags (5.68M answers): no flag or product-id answer moved, One
  Piece, Palworld and Riftbound are unchanged, and the run rule reproduces
  every Flesh and Blood alias answer. Refused now: "Unlimited Edition Cold
  Foil" (Flesh and Blood, 1,916), Gundam's "Holo" and "Holographic",
  Pokemon's "Holo", "Reverse Holo", "1st Edition Holo" and "Unlimited Holo",
  Lorcana's "None", its treatments as finish names, "Holofoil" on the 2,715
  printings sold in no Holofoil, and "Cold Foil" on the 463 sold in Holofoil
  alone. Answered now: Pokemon's run names across runs (29,100 "1st
  Edition", 29,101 "Unlimited", 13,015 and 13,016 of the two Holofoil runs,
  361 "Holofoil"), and Yu-Gi-Oh's "Normal" and "Foil" as the flags answer
  them.
- The catalog replay (101,395 names, all eight games) answers identically.
