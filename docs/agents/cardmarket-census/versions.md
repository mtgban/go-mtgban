# Versions the id map cannot place

`cardmarket/versions.go` holds `versionPrintings`, which names the printing
sold by Magic products the id map gets wrong. Each is a "(V.N)" version of a
card on a World Championship Deck, Pro Tour 1996, Chronicles: Japanese or
Arabian Nights shelf. MTGJSON's map, which copies Scryfall's
`cardmarket_id`, links each one to one of three things:
- no printing on its own shelf;
- a sibling version's printing;
- several printings at once.

Built on 2026-09-25 against AllPrintings 5.3.0+20260924.

## Why a table

The name route cannot help. Cardmarket gives these products no collector
number, and its V.N order follows no rule: of the 96 basic-land versions
where both sources agree, only 41 are in number order. Matching by name
landed every version on one printing, for example all eight of Sim Han How's
Forests on shh328. Because the index keeps the cheapest entry, each such row
published the lowest price among several versions.

Two sources link a Cardmarket product to a printing: Scryfall (and so
MTGJSON) and CardTrader's blueprints. Across the 280 WCD versions they agree
on 132 and disagree on 54; for 93 more, CardTrader alone gives an answer.

## What placed each row

The tag after each row's name says what placed it:

| tag | meaning |
|---|---|
| image | Cardmarket's product image was matched to the Scryfall card by eye |
| swap | the other half of a two-version swap an image settled |
| sideboard | V.1 is the main deck and V.2 the sideboard, as in all 6 pairs both sources agree on. CardTrader follows that order in all 15 disputes, Scryfall in none |
| CardTrader | CardTrader's collector number, which no other product's row contests |
| left over | the one row of its group left once the other versions are placed |
| map lists two, four | the map files the product on several printings. CardTrader and the sibling priced on the other one agree which is this product's |

The images settled 35 disputes: 33 went CardTrader's way and two went
Scryfall's (Antoine Ruel's Island V.4 is ar335, Mark Le Pine's Mountain
V.1 is mlp343). A CardTrader row is therefore very likely right. Before
changing a row, check Cardmarket's image at
`https://product-images.s3.cardmarket.com/1/<shelf code>/<id>/<id>.jpg`.
The shelf code is WCD9 to WCD37 in Cardmarket's expansion-id order.

## Keeping the table

- **Redundant rows.** A row the map later gets right is redundant, not
  wrong. Delete it only once the map carries the same printing.
- **New versions.** A version the table lacks goes back to the map and then
  to the name route. A new WCD version would collapse again, so place it
  here from its image.
