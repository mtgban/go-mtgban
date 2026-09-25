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

| slug | TCGplayer | label | run | treatment | foil |
|---|---|---|---|---|---|
| `nonfoil` | Normal | Normal | – | `nonfoil` | no |
| `foil` | Foil | Foil | – | `foil` | yes |
| `holofoil` | Holofoil | Holofoil | – | `holofoil` | yes |
| `reverseholofoil` | Reverse Holofoil | Reverse Holofoil | – | `reverseholofoil` | yes |
| `coldfoil` | Cold Foil | Cold Foil | – | `coldfoil` | yes |
| `rainbowfoil` | Rainbow Foil | Rainbow Foil | – | `rainbowfoil` | yes |
| `1stedition` | 1st Edition | 1st Edition | 1st Edition | `nonfoil` | no |
| `unlimited` | Unlimited | Unlimited | Unlimited | `nonfoil` | no |
| `limited` | Limited | Limited | Limited | `nonfoil` | no |
| `1steditionholofoil` | 1st Edition Holofoil | same | 1st Edition | `holofoil` | yes |
| `unlimitedholofoil` | Unlimited Holofoil | same | Unlimited | `holofoil` | yes |
| `1steditionnormal` | 1st Edition Normal | same | 1st Edition | `nonfoil` | no |
| `1steditionrainbowfoil` | 1st Edition Rainbow Foil | same | 1st Edition | `rainbowfoil` | yes |
| `1steditioncoldfoil` | 1st Edition Cold Foil | same | 1st Edition | `coldfoil` | yes |
| `unlimitededitionnormal` | Unlimited Edition Normal | same | Unlimited | `nonfoil` | no |
| `unlimitededitionrainbowfoil` | Unlimited Edition Rainbow Foil | same | Unlimited | `rainbowfoil` | yes |

Both names read back through `FinishSlug`, so a label has to keep its slug's
words. A printing TCGplayer adds later loads under its own name and is taken
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

Lorcana's treatments (Satin, Rainbow Pillars) are promo types, not finishes:
a listing naming one in its wording reaches the printing that carries it,
and one sending it as the finish falls through to the wording.

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
