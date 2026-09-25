# Lorcana Panorama foils

30 Lorcana cards are sold in two arts:

| set | cards |
|---|---|
| The Reign of Jafar | 5 |
| Winterspell | 4 |
| Wilds Unknown | 15 |
| Attack of the Vine! | 6 |

Each has the regular art and a Panorama foil, whose art joins its
neighbours'. Every vendor sells the Panorama as a product of its own:

| | regular art | Panorama |
|---|---|---|
| TCGplayer | main product, Normal skus only | "… (Foil)" product, Cold Foil skus only |
| CardTrader | regular blueprint (`037`) | "Panorama" blueprint (`037p`) |
| Cardmarket | V.1 | V.2 |

datastore-gen folds each Panorama into the regular card's row. Its Cold Foil
uuid (`2752_coldfoil`) is the Panorama, and its TCGplayer product is listed in
`tcgPlayerExtraIds`.

## What the loader does

`mtgmatcher/lorcana` splits such a row in two:

- **The plain card** keeps the Normal printing, the main product and the
  Cardmarket id (the V.1).
- **A ★ twin** takes the foil printing, the extra product and the Panorama's
  own Cardmarket id (the V.2), which datastore-gen publishes in
  `cardmarketExtraIds` (datastore-gen#110). It is numbered
  `37★`, and `PlainNumber` strips the ★ so a listing numbered 37 still
  reaches it.

Every uuid stays where it was.

The twin shows the Panorama's own art. LorcanaJSON publishes it as
`images.fullFoil`, on exactly these 30 cards. It is borderless where `full`
keeps the frame. The twin gets a copy of the map with `full` and `thumbnail`
both set to `fullFoil`. The plain card keeps upstream's map unchanged.

`fullFoil` fills `thumbnail` too because upstream has no foil thumbnail, and
the regular one is the framed art. It is 1468×2048 where a thumbnail is
367×512, but the site renders most pages from the thumbnail with no fallback
to `full` (`uuid2card`), so an empty key would show nothing.

The regular art is then sold in no foil. The vendors do list a few regular
foils: LorcanaJSON gives the card `foilTypes: Silver`, and CardTrader carries
foil listings on the regular blueprints. TCGplayer sells none, and the
datastore has no uuid for one.

So a foil claim on the regular card is left unpriced, rather than clamped onto
the plain card or put on the Panorama. That is:

- CardTrader's `processProducts` skipping it;
- Cardmarket's foil slot equalling the plain card;
- TCGplayer's existing rule for market-only price rows.

## Measured when this landed

Inputs, 2026-09-25:

- go-mtgban `6de4ab985`;
- datastore `lorcana.json` 2026-09-25 v1;
- the published Cardmarket and TCGplayer catalogs;
- the CardTrader Lorcana feed, captured at 04:13 UTC.

Results by path, from master to this change:

| path | replayed | what moved |
|---|---|---|
| backend | 6,157 uuids | uuid set identical. 180 lines change, all inside the 30 cards. 30 set cards are added (the twins). |
| TCGplayer names | 3,484 | none |
| TCGplayer price rows (the 60 products) | 180 rows | 7 market-only "Cold Foil" rows on regular products (€0.39–6) stop pricing the Panorama. Every live row is unchanged. |
| Cardmarket catalog (with bridge) | 3,372 | 30 V.1 foil slots empty out. The V.2 stays on the twin. |
| CardTrader listings | 435,399 | 131 foil listings on 28 regular blueprints go unpriced. |
| Star City Games catalog | 5,865 | none |

Cool Stuff Inc and Game Nerdz match Lorcana by name only. The name path
reaches the same uuid for every finish, as the TCGplayer and Star City Games
replays show.

On the website:

- TCGplayer links for the twin now go to the Panorama's own product.
- The twin's Cardmarket link goes to the V.2 once the datastore carries
  `cardmarketExtraIds`. Before that the twin had no Cardmarket id, and its
  link was a name search.
- The twin's image, full size and thumbnail, is the Panorama's.

## Measured when the twin got its image

Inputs, 2026-09-25: go-mtgban `e446d4f77` and datastore `lorcana.json`
2026-09-25 v1.

- **backend**: 6,157 card uuids, the same set on both sides. With `Images`
  masked (`ZZ_MASK_IMAGES`), every fixture and all eight published backends
  dump identically. Unmasked, 30 CardObjects move: the 30 twins, in `full`
  and `thumbnail` only. Every plain card keeps upstream's map.
- **TCGplayer names**: 3,484 replayed, identical. Nothing in go-mtgban
  reads `Images`, so no other scraper path can move.
