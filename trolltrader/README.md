# Troll Trader Cards

A UK Magic store on Shopify, priced in pounds. Its storefront names every
printing by MTGJSON set code and collector number, so matching asks for the
printing by set and number and refuses anything that lands elsewhere.

## Retail

Shopify's `products.json`, one collection at a time. The collection holding
every single (`trading-card-singles`, about 96,000 products) cannot be
walked whole: Shopify refuses a page past 25,000 products. Every other
collection is walked instead, each set's own and the shelves beside them,
and a product seen twice is counted once. Products carry no stock count,
only whether a variant is available.

Shopify answers a burst of about 60 requests from one address, then about
one a second, and refuses the rest with a 429 that names no wait. Every
request waits on one ticker, so a run takes about 17 minutes.

- The set code is the second field of every variant's SKU,
  `cc-<set>-<slug>-<number>-<grade>-<finish>`.
- The number is the first parenthesized group of the title, `Name (89)`,
  whatever qualifier follows (` - Surge-Foil`, ` (Display Commander)`).
  An art variant reads `(Female - 88a)`; the part after the dash is the
  number.
- Where the title's number does not land, the SKU's is tried. A title can
  name only the art, `(Van Camp)`, or carry the wrong number,
  `Gran-Gran (1)` for PW25 14. Newer SKUs keep the slug's hyphens and add a
  hash after the number, `cc-pw25-palladium-myr-6-<hash>-nm-nf`, and keep
  only the last part of a number like `USG-172`.
- Grades follow the store's own guide: Excellent is SP, Light Played MP,
  Played HP, Poor PO. A few listings say Good, which the guide leaves out;
  it sits between Excellent and Light Played, and reads as MP.

The store's numbering differs from MTGJSON's in ways the match accepts:

- a foil the datastore numbers apart (`89★`) is numbered as the plain card;
- an etched foil carries an `e` the datastore does not (`273e`);
- a foil can carry a letter for its treatment, `317r` ripple, `9s` surge,
  `2109f` galaxy, and is read as the `★` printing only;
- a playtest card keeps the number it prints, which MTGJSON replaces
  (MB2 `265` is MTGJSON's 501);
- Final Fantasy Commander's surge foils are filed under a code of their
  own, `FICS`;
- tokens and emblems are filed under their set's code, where the datastore
  keeps them in the token set, and two-sided tokens are numbered by both
  faces, `22/5`;
- a Secret Lair crossover is titled by its flavor name first,
  `Amunet, Tyrants' End - Queen Marchesa`.

## Buylist

Not scraped. The buylist is a Card Companion app behind the store's Shopify
app proxy, `/tools/cc_buylist/buylist/search`, with no listing: a search
takes a set code and text and returns at most 24 cards of the store's
catalog, priced or not, matching the text as a SQL `LIKE` pattern or as an
exact collector number, with no page past them. Covering every set this way
takes about 15,000 searches. The app proxy shares the storefront's limit and,
past it, holds a search open rather than refusing it, so searches cannot run
side by side; one at a time they average about two seconds, which puts a
full walk near nine hours.
