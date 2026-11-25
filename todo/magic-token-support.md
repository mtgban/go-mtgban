# Magic token support: what is still open

Magic token support shipped in #186 through #625; each PR's commit messages
carry its measurements, and the two-sided sheets were routed store by
store in #1544 through #1556. Four things remain.

- **TELD Goat // Food, an upstream MTGJSON error to report.** Among `ELD`'s
  tokens, TCGplayer product `200320` pairs Goat
  (`c60ebff2-0afe-5a7e-b011-cb3156ff8207`) with Food #16
  (`dcf079a4-9833-5527-b318-9318ef6491d8`), the same pair `200319` names.
  The Food on that sheet is #17 (`5f8204d1-ab7a-5182-b905-725704116a7b`).
  Still wrong in the 2026-09-30 AllPrintings. Vittorio submits these
  reports himself.
- **Listings named after one token that was only printed on two-sided
  sheets.** Nothing in the name says which sheet, so they stay on that
  face: 151 at Card Trader, about 130 at Hareruya and 82 at Mint Card as
  of 2026-11-24.
- **Cardmarket still refuses about 1,500 two-sided products** (as of
  2026-11-24): about 600 have no pairing in the datastore, 364 a pairing
  outside the edition's token set, 182 sit on a shelf with no token set,
  and 359 disagree with the pairing on numbers or names.
- **Sheets the loader does not mint**, for want of a `tokenProducts`
  record: the Secret Lair Cat 27 // 28 and the Walker pairs.

## Method notes

- **Drift control, always.** A live-vendor measurement runs base, branch,
  then base again, back to back, and the two base runs must be
  byte-identical before the branch number means anything.
- **An id beats free text only when it is the exact listing's own id**,
  cross-checked against the resolved printing's `tcgplayerProductId`, not
  assumed. A vendor's id disagreeing with its own wording is not
  automatically the vendor's bug.
- **Full-catalog scans are cheap enough to just run.** Whether any live
  product has a given shape is answerable in minutes by walking all ~114k
  Magic products once (`ListAllProducts` in a loop, filtered client-side).
  Do that before deleting anything as dead code.
