# Harness templates

Copies of the harnesses that worked, generalised. Drop into the scraper's
package as `zz_*_test.go`, run, then delete the copy — the source of truth
lives in `~/src/claude-scratchpad/ci-sweep/<tag>/`.

Datastore env vars: `ALLPRINTINGS5_PATH`, `FLESHANDBLOOD_PATH`, `YUGIOH_PATH`,
`POKEMON_PATH`, `ONEPIECE_PATH`, `LORCANA_PATH`, `RIFTBOUND_PATH`,
`GUNDAM_PATH`, `PALWORLD_PATH`. Source them with
`set -a; . ~/src/go-mtgban/.env; set +a` and never echo their neighbours.

## A. Catalog-style replay (Star City, Cool Stuff, Strike Zone …)

The incident's `head` carries the whole listing, so a regex over it rebuilds the
product. Reconstruct the vendor's own struct and call the scraper's resolve
function — not `Match` — so the scraper's preprocessing is under test too.

```go
package starcitygames

import (
	"bufio"; "encoding/json"; "errors"; "fmt"; "os"; "regexp"; "strings"; "testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

var zzHead = regexp.MustCompile(`(unknown variant|unknown card name|aliasing detected) for "(.*?)" \[(.*?) #(.*?)\] sku=(\S+)`)

func TestZZReplay(t *testing.T) {
	path := os.Getenv("ZZ_INCIDENTS")
	if path == "" { t.Skip("ZZ_INCIDENTS not set") }
	b, err := datastore.Read("fleshandblood", os.Getenv("FLESHANDBLOOD_PATH"))
	if err != nil { t.Fatal(err) }

	f, err := os.Open(path); if err != nil { t.Fatal(err) }
	defer f.Close()
	out, _ := os.Create(os.Getenv("ZZ_OUT")); defer out.Close()

	seen := map[string]bool{}
	var n, landed, skipped int
	sc := bufio.NewScanner(f); sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var rec struct{ Head string `json:"head"` }
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil { t.Fatal(err) }
		m := zzHead.FindStringSubmatch(rec.Head)
		if m == nil || seen[m[5]] { continue }   // dedupe on the sku/id
		seen[m[5]] = true; n++

		p := CatalogProduct{Name: m[2], Set: m[3], SKU: m[5], /* … */}
		id, err := resolveProductID(b, GameFleshAndBlood, p)   // the PRODUCTION path

		verdict, where := "landed", ""
		switch {
		case errors.Is(err, mtgmatcher.ErrUnsupported):
			verdict, where, skipped = "skipped", "unsupported", skipped+1
		case err != nil:
			verdict, where = "refused", err.Error()
		default:
			landed++
			co, _ := b.GetUUID(id)
			where = co.Name + "|" + co.SetCode + "|" + co.Number + "|" + strings.Join(co.PromoTypes, "+") + "|" + id
		}
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", m[2], m[3], m[5], verdict, where)
	}
	t.Logf("replayed %d distinct: %d land, %d skipped, %d refused", n, landed, skipped, n-landed-skipped)
}
```

Run and clean up in one go:

```bash
cp $D/<tag>/zz_replay_test.go <pkg>/ && \
  ZZ_INCIDENTS=$D/incidents/<game>--<scraper>.jsonl ZZ_OUT=$D/<tag>/replay-1.tsv \
  go test -count=1 -run TestZZReplay ./<pkg>/ -v 2>&1 | grep 'zz_replay\|FAIL'
rm <pkg>/zz_replay_test.go
```

## B. CardTrader replay (blueprint-driven)

CardTrader's incident carries the raw blueprint in `ctx[1]`. Parse it back into
a `Blueprint`, then run the four game-specific readers the listing loop uses.
A listing's finish is not in the log, so sweep the plausible finishes and count
a line as landed if **any** of them lands.

```go
var zzBP = regexp.MustCompile(`&\{ID:(\d+) Name:(.*?) Version:(.*?) GameID:\d+ CategoryID:(\d+) ExpansionID:(\d+) ScryfallID:\S* TCGplayerID:(\d+) CardMarketIDs:\[(.*?)\] Expansion:\{Name:(.*?) Code:(.*?)\} Properties:\{Number:(.*?) Language:(.*?)\}`)

// …per blueprint, per plausible finish:
var product Product
product.Properties.FabFoilNew = treatment      // or PokemonReverse / FirstEdition …
in := mtgmatcher.InputCard{
	Name:      gameName(gameID, &bp),
	Edition:   gameEdition(gameID, &bp),
	Variation: gameVariation(gameID, &bp, bp.Properties.Number),
	Finish:    gameFinish(gameID, &bp, product),
	Foil:      gameFoil(gameID, product),
}
id, err := b.Match(&in)
```

Weight rows by how many log lines each blueprint accounted for — one blueprint
can be 28 lines, and fixing it is worth more than 28 singletons.

## C. Datastore probes

Python over the game JSON (top level: `cards` list, `sets` dict code→{name}):

```python
import json, sys
d = json.load(open(sys.argv[1])); cards = d["cards"]; sets = d["sets"]
def show(title, pred, limit=30):
    rows = [c for c in cards if pred(c)]
    print("==", title, len(rows))
    for c in rows[:limit]:
        print("  %-9s %-14s %-46s %-22s %-9s %s %s" % (
            c.get("setCode"), c.get("number") or "", c["name"][:46],
            c.get("finish"), (c.get("rarity") or "")[:9],
            c.get("promoTypes") or "", c["id"]))
show("by name", lambda c: "Ogerpon" in c["name"])
show("by number", lambda c: (c.get("number") or "").startswith("SVP"))
```

A reusable Go probe (built once from a temp `cmd/zzX` dir in a worktree)
supports `set:CODE`, `name:SUBSTR`, `setname:NAME`, `sets:` — worth building
when you will probe a game more than a handful of times.

Magic is 942MB: stream it or grep it, do not `json.load` it.

## D. Regression diff (per listing, never recall)

Any shared matcher change needs a corpus that already works, walked before and
after, diffed per listing.

```python
# walkdiff.py before.jsonl after.jsonl [limit]
import json, sys, collections
def load(p):
    return {r["id"]: r for r in (json.loads(l) for l in open(p))}
a, b = load(sys.argv[1]), load(sys.argv[2])
gained = lost = moved = 0; ex = collections.defaultdict(list)
for k in a.keys() | b.keys():
    ra, rb = a.get(k, {}), b.get(k, {})
    la = ra.get("landed") if not ra.get("dropped") else ""
    lb = rb.get("landed") if not rb.get("dropped") else ""
    if la == lb: continue
    kind = "gained" if not la else "lost" if not lb else "moved"
    ex[kind].append((k, ra.get("exp_code"), ra.get("name"), ra.get("number"), la, lb))
    gained += kind == "gained"; lost += kind == "lost"; moved += kind == "moved"
print("gained", gained, "lost", lost, "moved", moved)
for kind in ("lost", "moved", "gained"):
    for e in ex[kind][:25]: print(kind, *e, sep="\t")
```

`lost` and `moved` are the numbers that matter. `moved` is only acceptable when
every move is a correction you can name.

## E. Pinning a rule with an inline fixture

Copy the rows **verbatim** out of the real datastore so the test states what the
data actually looks like, then load them as a backend:

```go
const labelFixture = `{
  "game": "fleshandblood",
  "sets": {"ROS": {"name": "Rosetta", "releaseDate": "2025-04-25"}},
  "cards": [
    {"externalLinks": {"fabId": "ROS001"}, "fabId": "ROS001", "finish": "Normal",
     "id": "ros001_561243", "name": "Florian, Rotwood Harbinger", "number": "ROS001",
     "rarity": "Majestic", "setCode": "ROS"}
  ]
}`

b, err := Load(strings.NewReader(labelFixture))
if err != nil { t.Fatal(err) }
// hand b to what is under test: b.Match(...), or the scraper built on it
```

Generate the fixture rather than typing it:

```python
ids = ["ros001_561243", "ros001-mv_561247_cold"]
rows = {c["id"]: c for c in json.load(open(path))["cards"] if c["id"] in ids}
sets = {rows[i]["setCode"]: d["sets"][rows[i]["setCode"]] for i in ids}
```

Include the negative case: the listing that must still refuse.

## F. Golden test data

`mtgmatcher/<game>/testdata/<game>_test_data.json` holds baked
`{description, input, uuid, error}` cases; regenerate with
`go test -run TestXMatch ./mtgmatcher/<game>/ -update-<game>`. It refuses to
flip a case between success and error unless the description carries a
`negative:` prefix — when a flip is genuinely intended, edit that entry by hand
first, and diff the regenerated file case-by-case before committing:

```python
old = {x["description"]: x for x in json.load(open("HEAD-copy.json"))}
new = {x["description"]: x for x in json.load(open("testdata/….json"))}
for d in set(old) & set(new):
    if old[d].get("uuid") != new[d].get("uuid"): print("CHANGED", d)
```

Magic's goldens must stay byte-identical — if a change forces a Magic regen,
the change is wrong.
