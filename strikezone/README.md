# Strike Zone

Strike Zone's shelves are crawled as HTML category pages. Magic rows go
through `preprocess` (store wording to `InputCard`) and `Match`; Lorcana,
Pokemon, Yu-Gi-Oh and Flesh and Blood rows go through `preprocessDetails` or
`lorcanaListing`. The store publishes no identifier for any of them, so every
landing is wording- and shelf-driven.

## When the match is ambiguous

`processRow` hands an `AliasingError` to `resolveAliasing` (`aliasing.go`).

- For Magic, `contradictions` is a table of checks over one candidate and the
  listing: a premium foil the wording never names, a borderless printing never
  called borderless, a flavor name the product does not write, a finish the
  printing was never made in. A candidate any row rules out is dropped.
- Several survivors are narrowed to the one in the set the shelf (or the edition
  `preprocess` made of it) is named for. Every game takes this step, which is
  how the Lorcana promo shelf lands.
- A lone survivor on a `Promos:` or `Promo Pack:` shelf must be a promo, or the
  tie stays unresolved. Without that check the finish row sends Scute Swarm on
  the Media shelf to its Zendikar Rising printing.

Rejected: reading a promo shelf as a selector for every ambiguous card (it chose
Vito and Go for the Throat wrongly), and a row for a treatment word the listing
names but the printing lacks (no listing in the 2026-10-05 capture reaches it).

## Secret Lair bare names

A bare Secret Lair name filed under several drops is refused unless the finish
on sale is printed in only one of them (`secretLairDrops`), in which case the
listing is pinned to that drop's number. The names still refused (158 keys in
the capture) have more than one drop in the finish, which no wording on the
page tells apart.

## Known limits

- Secret Lair, The List and Mystery Booster errors are not logged: the shelf
  files a card under several printings the wording cannot tell apart.
- Several Promos shelves list a card whose promo printings differ only by a
  stamp the store does not write (Vito, Go for the Throat, Knight Exemplar,
  Lathliss).
- The loader strips `doubleexposure` from the Duskmourn nonfoil printings, so
  the table rows name those cards by number rather than by treatment.
