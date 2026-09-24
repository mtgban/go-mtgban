# Harness templates

Copies of the harnesses that proved the 2026-09 loader work, generalised.
Each Go template goes into the named package as `zz_*_test.go`. Copy it into
both checkouts, run it, then delete it; a `trap` does the deleting in the
shell snippets. Datastores come from `.env`
(`set -a; . ~/src/go-mtgban/.env; set +a`).

## A. Audit: published keys against decoded tags

Every key a game's published file carries, at every level, that no `json`
tag in its loader package names. Map keys (set codes) are folded to `*`.
Run it from the repository root:

```python
import collections, glob, json, os, re
env_file = os.environ.get('ENV_FILE', os.path.expanduser('~/src/go-mtgban/.env'))
env = dict(re.findall(r'^([A-Z_]+)="?([^"\n]*)"?$', open(env_file).read(), re.M))
def walk(x, path, out):
    if isinstance(x, dict):
        for k, v in x.items():
            p = re.sub(r'^\.sets\.[^.\[]+', '.sets.*', f"{path}.{k}")
            out[p][0] += 1
            out[p][1] += v not in (None, '', [], {}, 0, False)
            walk(v, p, out)
    elif isinstance(x, list):
        for v in x: walk(v, path + '[]', out)
for g in ['lorcana', 'riftbound', 'onepiece', 'yugioh', 'fleshandblood', 'pokemon', 'gundam', 'palworld']:
    data = json.load(open(env[g.upper() + '_PATH']))['data']
    if g == 'riftbound':
        data = [b for b in data['pageProps']['page']['blades'] if b['type'] == 'riftboundCardGallery'][0]
    out = collections.defaultdict(lambda: [0, 0])
    walk(data, '', out)
    tags = set(re.findall(r'json:"([^",]+)', ''.join(open(f).read() for f in glob.glob(f'mtgmatcher/{g}/*.go') if not f.endswith('_test.go'))))
    print(f'== {g}')
    for p, (n, filled) in sorted(out.items()):
        if not p.endswith('.*') and p.rsplit('.', 1)[-1].rstrip('[]') not in tags:
            print(f'   {p}  present={n} filled={filled}')
```

A tag shared by two structs hides a gap: Lorcana's file-level `metadata.language` made the
per-card `language` look decoded. Read the loader's structs for any key the
answer turns on.

The `Card` fields each loader's literal sets, side by side:

```python
import re
games = ['lorcana', 'riftbound', 'onepiece', 'yugioh', 'fleshandblood', 'pokemon', 'gundam', 'palworld']
def keys(src):
    i = src.index('mtgmatcher.Card{') + len('mtgmatcher.Card{'); depth, j = 1, i
    while depth:
        depth += {'{': 1, '}': -1}.get(src[j], 0); j += 1
    found, d = set(), 0
    for line in src[i:j].split('\n'):
        m = re.match(r'\s*([A-Z]\w*):', line)
        if d == 0 and m: found.add(m.group(1))
        d += line.count('{') - line.count('}')
    return found
sets = {g: keys(open(f'mtgmatcher/{g}/{g}.go').read()) for g in games}
for k in sorted(set().union(*sets.values())):
    print(f'{k:20}', ' '.join(f'{("x" if k in sets[g] else "."):5}' for g in games))
```

## B. Backend equivalence

`dump.sh CHECKOUT OUT` writes `zz_dump_test.go` into every package holding
`{"data": …}` fixtures (`gen_dump.py`), dumps each fixture's `Backend`,
hashes each published datastore's, and deletes the test files again:

```sh
git worktree add -q --detach <master-dir> origin/master
docs/agents/loader-cleanup/dump.sh <master-dir> <dumps>/master
docs/agents/loader-cleanup/dump.sh "$PWD"       <dumps>/branch
diff -rq <dumps>/master <dumps>/branch
docs/agents/loader-cleanup/jdiff.py <dumps>/master/<pkg>.<fixture>.json <dumps>/branch/<pkg>.<fixture>.json
```

- `ZZ_MASK_OVERSIZED=1`, `ZZ_MASK_LANGUAGE=1` and `ZZ_MASK_WATERMARK=1` clear
  that field before encoding. Add a `MASKS` entry for another field.
- `P=<dir>` reads every `<game>.json` from `dir`, to load a local build.
- `ENV_FILE` names another `.env`.
- Variants that tests derive with `strings.Replace` are listed by hand in
  `DERIVED`, and fixtures outside `mtgmatcher/<game>` and cardmarket in
  `BY_HAND`.

## C. Cardmarket catalog replay (`cardmarket/zz_replay_test.go`)

The id map comes from `b2://mtgban-datastore/<game>/cardmarket_catalog.json.xz`,
unpacked to `$ZZ_DIR/<game>.json`. The day's product list is downloaded once
and cached beside it. Write one line per product: the answer after twin
marking, then the raw one.

```go
package cardmarket

import (
	"context"; "encoding/json"; "fmt"; "os"; "path/filepath"; "sort"; "strings"; "testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/internal/datastore"
)

func TestZZReplay(t *testing.T) {
	game, dir, out := os.Getenv("ZZ_GAME"), os.Getenv("ZZ_DIR"), os.Getenv("ZZ_OUT")
	if game == "" { t.Skip() }
	b, err := datastore.Read(game, os.Getenv(strings.ToUpper(game)+"_PATH"))
	if err != nil { t.Fatal(err) }
	f, err := os.Open(filepath.Join(dir, game+".json")); if err != nil { t.Fatal(err) }
	catalog, err := cm.LoadCatalog(f); f.Close(); if err != nil { t.Fatal(err) }
	mkm, err := NewScraperIndex(b); if err != nil { t.Fatal(err) }
	mkm.catalog = catalog

	var list []cm.ProductList
	cache := filepath.Join(dir, game+"-list.json")
	if data, err := os.ReadFile(cache); err == nil {
		if err := json.Unmarshal(data, &list); err != nil { t.Fatal(err) }
	} else {
		if list, err = cm.DownloadProductListSingles(context.Background(), mkm.gameID); err != nil { t.Fatal(err) }
		data, _ := json.Marshal(list)
		if err := os.WriteFile(cache, data, 0o600); err != nil { t.Fatal(err) }
	}
	products := map[int]cm.CatalogProduct{}
	for id, p := range catalog.Data.Products { products[id] = p }
	for _, e := range list {
		if _, ok := products[e.IDProduct]; !ok {
			products[e.IDProduct] = cm.CatalogProduct{ExpansionID: e.ExpansionID, Name: e.Name}
		}
	}
	byExp := map[int][]int{}
	for id, p := range products { byExp[p.ExpansionID] = append(byExp[p.ExpansionID], id) }
	var items []cm.Expansion
	for id := range byExp {
		e := catalog.Data.Expansions[id]
		exp := cm.Expansion{IDExpansion: id, Name: e.Name, SetCode: e.Code}
		if exp.Name == "" { exp.Name = fmt.Sprintf("expansion %d", id) }
		if !foreignExpansion(mkm.gameID, exp) { items = append(items, exp) }
	}
	sort.Slice(items, func(i, j int) bool { return items[i].IDExpansion < items[j].IDExpansion })
	if mkm.gameID == cm.GameOnePiece { mkm.shelved = shelvedSets(mkm.backend, items) }

	var lines []string
	for _, exp := range items {
		ids := byExp[exp.IDExpansion]; sort.Ints(ids)
		results := make([]resolved, 0, len(ids))
		for _, id := range ids { results = append(results, mkm.resolveMapped(id, products[id], exp)) }
		raw := make([]string, len(results))
		for i, r := range results { raw[i] = fmt.Sprintf("%s\t%s\t%v", r.cardID, r.cardIDFoil, r.err) }
		if same := sameProduct(mkm.gameID); same != nil { twinsAmong(results, same, faceOf(mkm.backend, mkm.gameID)) }
		for i, r := range results {
			p := products[r.product.IDProduct]
			lines = append(lines, fmt.Sprintf("%d\t%s\t%s\t%s\t%s\t%s\t%s\t%v\traw:\t%s",
				r.product.IDProduct, exp.Name, p.Name, p.Number, p.Rarity, r.cardID, r.cardIDFoil, r.err, raw[i]))
		}
	}
	if err := os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil { t.Fatal(err) }
}
```

`ZZ_GAME=<game> ZZ_DIR=<dir> ZZ_OUT=<file> go test -count=1 -run '^TestZZReplay$' ./cardmarket/`
on each side, then `diff`. Gundam and Palworld have no Cardmarket index.

## D. Star City Games catalog replay (`starcitygames/zz_scg_replay_test.go`)

The first run downloads the whole catalog export to `$ZZ_CATALOG` (about
115 MB, needs `SCG_API_KEY`), and both sides then read that file.

```go
package starcitygames

import (
	"context"; "fmt"; "io"; "os"; "strings"; "testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
)

func TestZZSCGReplay(t *testing.T) {
	game, cache, out := os.Getenv("ZZ_GAME"), os.Getenv("ZZ_CATALOG"), os.Getenv("ZZ_OUT")
	want := map[string]int{"fleshandblood": GameFleshAndBlood, "lorcana": GameLorcana, "riftbound": GameRiftbound}[game]
	if want == 0 { t.Skip() }
	if _, err := os.Stat(cache); err != nil {
		body, err := NewSCGClient(os.Getenv("SCG_API_KEY")).DownloadCatalog(context.Background())
		if err != nil { t.Fatal(err) }
		f, err := os.Create(cache); if err != nil { t.Fatal(err) }
		_, err = io.Copy(f, body); body.Close(); f.Close()
		if err != nil { t.Fatal(err) }
	}
	b, err := datastore.Read(game, os.Getenv(strings.ToUpper(game)+"_PATH"))
	if err != nil { t.Fatal(err) }
	f, err := os.Open(cache); if err != nil { t.Fatal(err) }
	defer f.Close()
	var lines []string
	err = decodeCatalog(f, func(p CatalogProduct) error {
		if gameFromCatalog(p.Game) != want || p.ProductType != ProductTypeSingles { return nil }
		id, err := resolveProduct(b, want, p)
		if err != nil { id = "ERR:" + err.Error() }
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s", p.SKU, p.Name, p.Set, p.CollectorNumber, p.Finish, p.Language, id))
		return nil
	})
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil { t.Fatal(err) }
}
```

## E. TCGplayer names, and builder changes

The catalog replay needs only an unpacked
`b2://mtgban-datastore/<game>/tcgplayer-catalog.json.xz`. Set that game's
path and no other:

```sh
unset LORCANA_PATH RIFTBOUND_PATH ONEPIECE_PATH YUGIOH_PATH FLESHANDBLOOD_PATH POKEMON_PATH GUNDAM_PATH PALWORLD_PATH
<GAME>_PATH=<datastore> REPLAY_CATALOG=<catalog.json> REPLAY_OUT=<side>.txt \
  go test -count=1 -run '^TestReplayCatalogNames$' ./internal/vocabulary/
```

A builder change is built twice from the same saved inputs, as datastore-gen's
AGENTS.md "How to verify a change" says. Save the upstream once:
- Riftbound: `curl -sSL` the playriftbound.com gallery page for its
  `buildId`, then the data URL, passed with `-gallery`.
- Lorcana: `lorcanajson.org/files/current/en/allCards.json`, passed with
  `-lorcana`.

Then load both builds here with `P` (section B) and replay them (C to E)
before calling the change safe for today's loader.
