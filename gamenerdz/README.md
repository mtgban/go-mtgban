# Game Nerdz

The storefront publishes two feeds per game. Retail carries the catalog's
TCGplayer id for nearly every product; the buylist carries none, so its wording
is what lands, and the retail ids are what that reading is measured against.
`resolveProduct` reads Magic retail by id first and every other row by wording.

## Guards on the wording rules

Four guards keep the buylist rules from widening:

- The typed-energy and Greek respellings apply only where the storefront's own
  name is unknown to the catalog and the respelled one is known. Applied
  unguarded, the energy rule rewrote `Basic Darkness Energy` and
  `Aromatic Grass Energy` and lost 98 landings.
- The numberless basic-energy retry is kept only where it lands a printing the
  catalog files with no number.
- A One Piece listing with no card code is kept only where it lands a printing
  numbered `LEADER`; its label alone is too weak to trust anywhere else.
- The sized-number retry runs only after a refusal, on a dashed listing, and is
  taken only when it lands the shelf's own set at that number. Stripping the
  size always lost five Miscellaneous Cards cosmos promos.

Converting every `[...]` to `(...)` is wrong as well: `Dragapult - 091/192
[Rebel Clash]` names an origin set, so only `[Winner]` is read.

## Known remaining

- Fairy Energy in Kalos Starter Set has no row in the Pokemon datastore.
- Aquapolis listings written over a plain number (`Drowzee 74/147`) name two
  printings (74a, 74b) and stay refused.
- `Monkey.D.Luffy (Release Event Leader) (P)` stays refused. Its label alone
  lands the OP-PR P-135 Green character, while the vendor's retail id 634531
  is a six-colour Leader the datastore does not carry.
- `Gum-Gum Mole Pistol (Premium Card Collection -Best Selection Vol.5)` has no
  card code and is not a leader, so it stays refused.
