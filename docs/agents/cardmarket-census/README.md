# Cardmarket census

A census walks every Cardmarket product of one game through the Index
scraper's own resolution, then grades each landing against two other
sources: the datastore's published Cardmarket ids and CardTrader's links. It
finds three kinds of fault:

- products priced onto the wrong printing, which is live on the site;
- products left unpriced whose printing the datastore does carry;
- printings the datastore should carry and does not.

The noise sweep (`../noise-sweep/`) starts from the refusal lines a run
prints. A census also catches the silent faults, which print nothing.
`harnesses.md` holds the Go probes, `census.sh` fetches and walks, and
`grade.py` reports. Read this whole document first.

Eight games are on Cardmarket: Magic, Lorcana, Riftbound, One Piece,
Yu-Gi-Oh, Flesh and Blood, Pokemon and Gundam. Palworld is not: its
downloads stop at game 25, Sorcery, and CardTrader prices Palworld instead.

## 1. Sync, and work in your own worktree

Fetch and branch from a fresh `origin/master`. `census.sh` writes probe
files into `cardmarket/` of the checkout it runs in, and removes them on
exit, so never run it in a shared checkout. Master moves under a long
census: before diffing a branch, rerun the master walk on the branch's own
base. An old baseline shows other people's merges as your change.

## 2. Fetch and walk

```sh
docs/agents/cardmarket-census/census.sh <game>          # walk-master.tsv
python3 docs/agents/cardmarket-census/grade.py <game>   # the report
```

It reads `.env` (`ENV_FILE`) and writes under `$OUT/<game>`, by default
`~/src/claude-scratchpad/cardmarket-census/<game>`. What it fetches:

| input | source |
|---|---|
| datastore | `b2://mtgban-datastore/<game>/<game>.json.xz`; Magic: `$ALLPRINTINGS5_PATH` |
| id map | `b2://mtgban-datastore/<game>/cardmarket_catalog.json.xz`; Magic: MTGJSON's `CardmarketIdentifiers.json.xz` (the workflows' `MKMIDS_MAGIC`) |
| product list, price guide | Cardmarket's own downloads, as `Index.Load` fetches them |
| CardTrader blueprints and the bridge | `cardtrader.BlueprintsForGame`, the bridge built as bantool's `cardtraderBridge` builds it |
| published dumps | `b2://mtgban-dumps/<game>/{tcg_index,cardmarket,cardtrader}_<game>/retail/`; Magic has no `_<game>` suffix |

The walk applies what `walkCatalog` does, bridge and game passes included
(`harnesses.md`). Magic walks without the bridge, because bantool passes
none, but `grade.py` still grades Magic against it. Inputs are fetched
once; delete `data/` or `dumps/` to refresh them, and note the datastore's
`meta` date in anything you report.

## 3. Read the report

Each landing is graded against its anchor. The anchor is the datastore's
own Cardmarket id where it has one, and CardTrader's link otherwise:

- **`-agree`** or **`no-anchor`**: nothing to see unless another section
  names the product.
- **`DISAGREE`**: the walk and the anchor name different printings. Either
  side can be wrong (§4).
- **foreign, refused, twin, error**: unpriced, by shelf and by the guide's
  trend. `bridged` counts the products CardTrader links to a row the
  datastore carries. A foreign or refused product there is the first
  thing to check.
- **printings priced by more than one product**: one of them publishes a
  price for a card it does not sell, or the two are one printing sold twice.
- **CardTrader program names another row**: CardTrader's version text
  (for example "Release Event Winner") matches a sibling row better than
  the one the product landed on.
- **price outliers**: a landing whose trend sits 4x from TCGplayer's market.
  Mostly real EU/US gaps and thin markets. Worth a look only with a
  structural reason.

Not every game's resolver reads the datastore's own Cardmarket id. One
Piece, Lorcana and Riftbound read it first. Pokemon, Yu-Gi-Oh and Flesh and
Blood never read it, and Magic uses MTGJSON's map. So an `own-id` row in
those games is a cross-check, not the path production took.

## 4. Classify before fixing

Settle each finding with a second source before writing anything:

- the published dump, which shows what the site shows today;
- TCGplayer's product name and number in `tcgplayer-catalog.json`;
- CardTrader's blueprint version and collector number;
- the guide's `avg1`/`avg7`/`avg30`, where a price resting on one sale sits
  flat.

Agreement between sources is the evidence. An identifier alone is not,
because every vendor here has sent a wrong one.

The shapes met so far, and where each was fixed:

| shape | example | fix |
|---|---|---|
| a finish sold as its own product | One Piece The Best DON!! V.2 is the foil | resolver rule (#851 `foilVersions`) |
| foreign prints on an English shelf | One Piece French Première Édition; Pokemon "BW Promos" | foreign table (#852, #863) |
| rows filed in a catch-all set | YGO DDS-001 in VDP | read the shelf's code (#857) |
| CardTrader's TCGplayer id names a sibling | YGO 25YC Blue-Eyes stamps; One Piece Dash Pack O-Nami | `tcgIDOverrides` (#858, #860) |
| Cardmarket's number names another card | FaB Heartbeat of Candlehold #270 | `fabRenumbers` (#862) |
| MTGJSON links a printing to the wrong product | Chinese alt arts under English Portal; 2X2's crossed #345/#427 | `plausiblePrinting`, `numberedPrinting` (#865, #866) |
| versions only an id tells apart, linked wrong or not at all | Sim Han How's eight WCD Forests all on shh328; PTC ids on two printings | `versionPrintings` ([versions.md](versions.md)) |
| a row exists, only an id can reach it | One Piece tournament promos CardTrader names but cannot link | datastore-gen publishes `cardmarketId` (§6) |
| the vendor sells a row the datastore lacks | FaB TNP019 pitches numbered alike | datastore-gen (§6) |

Leave these alone; each was checked and is correct or undecidable:

- YGO European V.1 prints refused as foreign only where the datastore has
  no European row at that number; datastore-gen's European first-print
  mint adds rows on each of the seven oldest shelves.
- CardTrader linking the product that predates Cardmarket's 2021 split.
- Pokemon Prize Pack series selling one printing several times.
- Magic's ★ foil twins, and Cardmarket's own numbering on FBB and promos.
- Same-name products with no version, label or link to tell them apart.
- A price that rests on a single sale.

## 5. Size, fix, prove

1. **Size the rule before writing it.** Simulate it in Python over the walk
   and count every product it would move. A rule that moves more than the
   evidence covers belongs in a narrower place (AGENTS.md, "blast radius").
   A one-off vendor slip is a table row, keyed by the vendor's own product
   or blueprint id.
2. **Fix on a branch, then walk it.** Run `census.sh <game> branch`, then
   `grade.py <game> --diff master branch`. The diff must name only the
   products the PR claims.
3. **Test the case, and prove the test fails without the fix.** Use the
   datastore's own rows, copied into an inline fixture or read through the
   package's real-datastore helper.
4. **Run all six gates from AGENTS.md**, then open the PR. The body gives
   the live effect (the published dump's numbers), the evidence that picked
   the side, the replay count and what else moved (nothing, or each move
   named).

## 6. Handing work to datastore-gen

A datastore fix goes to whoever owns datastore-gen, as a file they can
check row by row:

- the date and versions measured;
- one line per Cardmarket id with the target row, its evidence and the
  guide trend;
- what was excluded, and why.

Before sending, confirm that the fix will change a landing:

- **Publishing a `cardmarketId` only moves games whose resolver reads it.**
  One Piece does; Pokemon does not.
- **One Piece's `claimByID` and `giveWay` read only the bridge.** A
  published id there can still lose to a wrong CardTrader link, and that
  link needs a `tcgIDOverrides` row too (#860).

## Pitfalls met

- zsh reads an argument starting with `=` as a command. Don't print `=====`
  separators.
- The ssh agent can drop its keys mid-session. Fetch and push over HTTPS
  through `gh auth git-credential`.
- A CardTrader blueprint can list several Cardmarket products. Examples: the
  plain and foil of one DON!!, a Japanese and an English product, or a stale
  product Cardmarket later replaced.
- List-only products (in the product list, not the id map) carry only a
  name. A misspelt duplicate of a real product lives there, like FaB's
  "Heatbeat of Candlehold".
