# Magic token support: what is still open

Magic token support shipped in #186 through #625; each PR's commit messages
carry its measurements. Four things remain.

- **TELD Goat // Food, an upstream MTGJSON error to report.** Among `ELD`'s
  tokens, TCGplayer product `200320` pairs Goat
  (`c60ebff2-0afe-5a7e-b011-cb3156ff8207`) with Food #16
  (`dcf079a4-9833-5527-b318-9318ef6491d8`), the same pair `200319` names.
  The Food on that sheet is #17 (`5f8204d1-ab7a-5182-b905-725704116a7b`).
  Still wrong in the 2026-09-30 AllPrintings. Vittorio submits these
  reports himself.
- **Two design decisions, not built** (#621):
  - a pairing entity for the 89 Star City Games composite-sku listings
    (as of 2026-09-15) that have no upstream `tokenProducts` id to price
    from;
  - relaxing `Backend.TokenPairIndex`'s one id per entity, to recover the
    remaining "losing sibling" ids.
- **CardTrader's Token blueprints**: about 62% (as of 2026-09-15) carry no
  id and no collector number, so nothing can resolve them. Revisit only if
  CardTrader starts publishing one.

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
