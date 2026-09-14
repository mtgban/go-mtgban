# Detectors 2 and 3 — finding matches that are wrong but not loud

A refusal announces itself. These two do not: the run prices the card, at some
other card's price. Both produce **candidates**, and a candidate is worth
nothing until the vendor's feed is replayed through master and one side is
shown to be wrong.

---

# Detector 2 — cross-vendor spread on the published site

## What the three pages are

Read the direction before reading the numbers; getting it backwards inverts
every conclusion.

| Page | `source=` is | each table is | comparison |
|---|---|---|---|
| `/arbit` | the **seller** (retail) | a **vendor** (buylist) | retail vs buylist |
| `/reverse` | the **vendor** (buylist) | a **seller** (retail) | buylist vs retail |
| `/global` | a retail market | another retail market | **retail vs retail** |

`/global` is not a buylist comparison at all — do not read its two price
columns as bid and ask.

## Capturing

Needs `MTGBAN_SIG` from `~/src/go-mtgban/.env` (source it, never print it).

**Check the key has not expired before you fetch anything.** An expired
signature does not fail: every page answers **HTTP 200** with a normal-sized
Unauthorized body, so a whole capture looks like it worked and parses to zero
rows. The 2026-09-10 run lost detector 2 that way and only found out at the
parse.

The key is a base64 query string carrying its own `Expires` as a unix
timestamp, so this is answerable offline, without spending a request:

```bash
set -a; . ~/src/go-mtgban/.env; set +a
python3 -c "
import os, base64, urllib.parse, datetime
s = urllib.parse.unquote(os.environ['MTGBAN_SIG'])
s += '=' * (-len(s) % 4)
exp = int(urllib.parse.parse_qs(base64.b64decode(s).decode())['Expires'][0])
t = datetime.datetime.fromtimestamp(exp, datetime.timezone.utc)
now = datetime.datetime.now(datetime.timezone.utc)
print(t.isoformat(), 'valid' if t > now else 'EXPIRED', str(t - now).split('.')[0])
"
```

Read `Expires` and nothing else. The same blob carries `Signature`, `UserEmail`
and `UserName`, so never print the decode whole. If it has expired, stop and
ask Vittorio for a fresh key rather than fetching 121 pages of Unauthorized —
and belt-and-braces, grep one fetched page for `Unauthorized` before trusting
the batch.

The host is the game name, except **Magic, which is `www` — `magic.mtgban.com`
and `www.mtgban.com` are the same host**.

```bash
set -a; . ~/src/go-mtgban/.env; set +a
# hosts: www (== magic), lorcana, onepiece, pokemon, yugioh, fleshandblood, riftbound
curl -sL -m 240 --get \
  --data-urlencode "sig=$MTGBAN_SIG" \
  --data-urlencode "source=$src" \
  --data-urlencode "sort=spread" \
  "https://$host.mtgban.com/$page" -o "raw/$game-$page-$src.html"
```

Enumerate `source=` values from the page's own index rather than guessing, skip
any output that already exists and is non-empty, check the body for
`Too Many Requests` before trusting it, and sleep ~2s between fetches.

Working scripts: `~/src/claude-scratchpad/spread-audit5/{fetch.sh,parse.py,rank.py}`
and `~/src/claude-scratchpad/reverse-audit/{fetch.sh,parse.py,classify.py}`.
The parser reads column positions out of each table's `<thead>` — `/reverse`
carries a Trade Price column `/arbit` does not, and `/global` names its price
columns after the two markets — so do not hard-code indices.

## Ranking — three corrections, all learned by getting it wrong

1. **Rank by dollar difference, not by spread percentage.** MKM Low quotes
   $0.01 for cards with no European supply, so the head of a spread-sorted
   ranking is entirely index noise.
2. **De-duplicate parties.** The same vendor appears long on `/arbit`
   ("Game Nerdz") and short on `/reverse` ("GN"), so every row is counted
   twice. Key on `(cardid, cond, bid, ask)`.
3. **Never compare across conditions.** Taking the minimum ask and maximum bid
   per card over all conditions reads a near-mint bid against a damaged ask as
   an enormous arbitrage. Key on `(cardid, cond)`.

On the 2026-09-05 Pokemon run, 2 and 3 together took a headline figure from
**$29,444 to $9,502**. The inflated rows sort to the top, so they are exactly
what gets investigated first.

## The discriminator that actually finds bugs

**One store, one condition, one card id, two prices far apart.** A shop does
not list the same product at $1.50 and $1,000 in NM, so when it appears to, two
printings have been folded onto one id. On the 2026-09-01 `/reverse` audit
(45 vendor pages, 130,399 rows) 17 of the top 400 showed this and **every one
was a real collapse** — that is a far better hit rate than any cross-vendor
ranking. Look for the same-store split before forming a hypothesis.

## Before believing a candidate

- Legitimate discrepancies are common — a genuine EU-vs-US gap, a shop short of
  stock, a hand-priced premium tier. Expect many false positives.
- `tcgdirectnet` and `syp` are wrong at the source. Exclude them.
- Magic is 99% golden: a Magic spread is go-mtgban's bug, not the datastore's.
- Verify by replaying that vendor's feed through master. The page shows the
  **last published run**, so it cannot tell you what is broken *now* — a fix
  that already merged still shows the old price until a new GitHub run
  publishes. Schedule a run if you need the site to confirm; do not grade by
  reloading.
- Re-rank often: as fixes land, the ordering shifts underneath you.

---

# Detector 3 — one scraper, buying and selling the same card

## Where it comes from

bantool already reports it — no capture needed. `mtgban.SuspectPricings`
(`mtgban/suspect.go`) is called by `reportSuspectPricings`
(`cmd/bantool/main.go`) and logs:

```
[SCRAPER] 12 cards are bought at 90% or more of their asking price
[SCRAPER] - 96% buy $10.00 ask $10.40 <buyURL> <retailURL>
```

Grep a run log for `bought at`, or compute the same thing offline from the
published dumps (`b2://mtgban-dumps`) — it is only a per-condition join of a
scraper's own inventory against its own buylist.

## Reading the ratio

A shop buys to resell, so buy sits well under ask: across a Yu-Gi-Oh run the
median pairing is **25%** and the 99th percentile **86%**. The shipped
threshold is 90 (`SuspectRatioThreshold`), but treat **anything over ~70% as
worth opening** — above that the two prices are usually describing different
cards: a textured foil bought at the plain card's id, a Secret Rare bought at
the id its common shares. Above 100% the shop would be paying more than it
charges, which no shop does.

The mechanism is that **retail and buylist are separate code paths in the same
scraper file** and derive the id differently. Hareruya has `preprocess(title)`
and `Preprocess(product Product)` with byte-identical return blocks — a fix to
one does nothing for the other. Check both sides before declaring it fixed.

## The benign mode, and the three checks

The big false positive is **a shop that hand-prices its premium tier**. The
tell is a bimodal ratio distribution: vegassingles riftbound priced ordinary
cards at a mechanical 60% of ask (median, n=263) while its "signature" cards
sat at 88.9% (n=38, 17 over threshold). Split the population by whatever names
the premium tier before concluding anything.

Then, in order:

1. **Is one uuid fed by more than one product?** That is the collision the
   report exists to find. Compare `original_id` across the inventory and
   buylist entries — the JSON key is snake_case and the Go field is
   `OriginalID`; reading the Go name off decoded JSON silently makes every set
   look like one element, which once produced a wrong "zero collisions".
2. **Does the vendor also sell the other printing under an explicit name?** If
   it lists both a `-signature-` and a plain Extended Art, the plain one
   colliding onto the premium printing is a real bug.
3. **Does the datastore hold both sides?** A nonfoil listing with only a foil
   printing has nowhere to go — that is a gap, not a bug.

## The stronger sibling: the one-store-two-prices census

Do not wait for the ratio report. Census a single vendor's feed directly for
**two entries on one card id at one grade**. `mtgban/base.go`'s `add` sorts the
higher price first, so whenever two of a shop's products fold onto one id the
dearer one silently prices the other's card (`AddUnique` documents exactly
this; almost no scraper calls it).

It needs one vendor and no cross-vendor join, and every hit is either a matcher
bug or a printing the catalog cannot hold. Hareruya 2026-09-02: 39 colliding
cards over 23,996 NM rows found **12 matcher bugs**, all shipped in PR #364;
collisions went 39 → 4. Strike Zone has the identical structure and has still
never been run.

**Classify before fixing.** For each collision, dump *every* printing the
catalog holds of that name. Most collisions are not bugs — Hareruya's 33
non-bugs were booster provenance (the card's `sourceProducts` names the
Collector/Draft/Set Booster packs, so the catalog knows the difference but has
no second id to price), per-store championship stamps, MPS partial/full gloss,
and phantom finishes.

**A skip needs a preference, not a blanket refusal.** Dropping every duplicate
wording cost 34 of 35 Secret Lair-deck listings and 2 of 23 booster ones their
only listing. Keep the duplicate while nothing else holds the id and let it give
way when its counterpart arrives, in either order — pages are read concurrently.
