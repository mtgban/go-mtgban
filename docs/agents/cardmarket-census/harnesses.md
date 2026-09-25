# Census harnesses

`census.sh` writes each template below into `cardmarket/` under its heading's
name, runs it, and deletes it on exit. Edit a template here, not a copy: the
script reads this file.

These templates walk the catalog the way `Index.walkCatalog` does. That means
the bridge where bantool passes one, One Piece's `claimByID` and `giveWay`,
Flesh and Blood's `disownBridged`, and `twinsAmong`. Section C of
`../loader-cleanup/harnesses.md` does less, which is enough for diffing two
loaders but not for grading products: without the claim pass it counts One
Piece twins as landings.

### `cardmarket/zz_census_fetch_test.go`

The product list, the price guide, and CardTrader's blueprints for one game,
with the bridge built exactly as bantool's `cardtraderBridge` builds it
(`TCGplayerProductID()`, so `tcgIDOverrides` apply).

```go
package cardmarket

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/mtgban"
)

func TestZZCensusFetch(t *testing.T) {
	game, dir := os.Getenv("ZZ_GAME"), os.Getenv("ZZ_DIR")
	if game == "" {
		t.Skip()
	}
	var g mtgban.Game
	for _, x := range mtgban.AllGames {
		if strings.EqualFold(string(x), game) {
			g = x
		}
	}
	ctx := context.Background()
	write := func(name string, v any) {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(dir, game+"-"+name+".json"), raw, 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}
	list, err := cm.DownloadProductListSingles(ctx, mkmGames[g])
	if err != nil {
		t.Fatal(err)
	}
	write("list", list)
	guide, err := cm.DownloadPriceGuide(ctx, mkmGames[g])
	if err != nil {
		t.Fatal(err)
	}
	write("guide", guide)
	client := cardtrader.NewCTAuthClient(os.Getenv("CARDTRADER_TOKEN_BEARER"))
	bps, _, err := cardtrader.BlueprintsForGame(ctx, client, g, "", t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	write("blueprints", bps)
	bridge := map[int]int{}
	for _, bp := range bps {
		tcgID := bp.TCGplayerProductID()
		if tcgID == 0 {
			continue
		}
		for _, mkmID := range bp.CardMarketIDs {
			bridge[mkmID] = tcgID
		}
	}
	write("bridge", bridge)
	t.Logf("%d list, %d guide, %d blueprints, %d bridged", len(list), len(guide), len(bps), len(bridge))
}
```

### `cardmarket/zz_census_walk_test.go`

One row per product: its shelf, the catalog's name, number and rarity, the
verdict, the printings it lands on, whether the wording answered, and the
candidates of an aliasing error.

```go
package cardmarket

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

func TestZZCensusWalk(t *testing.T) {
	game, dir, out := os.Getenv("ZZ_GAME"), os.Getenv("ZZ_DIR"), os.Getenv("ZZ_OUT")
	if game == "" {
		t.Skip()
	}
	b, err := datastore.Read(game, filepath.Join(dir, game+"-datastore.json"))
	if err != nil {
		t.Fatal(err)
	}
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatal(err)
	}
	cf, err := os.Open(filepath.Join(dir, game+"-catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	mkm.catalog, err = cm.LoadCatalog(cf)
	cf.Close()
	if err != nil {
		t.Fatal(err)
	}
	// bantool passes the bridge only where the game uses one; Magic walks
	// without it.
	g, err := mtgban.GameOf(b)
	if err != nil {
		t.Fatal(err)
	}
	if BridgeUseOf(g) != BridgeUnused {
		raw, err := os.ReadFile(filepath.Join(dir, game+"-bridge.json"))
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal(raw, &mkm.tcgBridge)
		if err != nil {
			t.Fatal(err)
		}
	}
	var list []cm.ProductList
	raw, err := os.ReadFile(filepath.Join(dir, game+"-list.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(raw, &list)
	if err != nil {
		t.Fatal(err)
	}

	products := map[int]cm.CatalogProduct{}
	for id, p := range mkm.catalog.Data.Products {
		products[id] = p
	}
	for _, e := range list {
		if _, ok := products[e.IDProduct]; !ok {
			products[e.IDProduct] = cm.CatalogProduct{ExpansionID: e.ExpansionID, Name: e.Name}
		}
	}
	byExpansion := map[int][]int{}
	for id, p := range products {
		byExpansion[p.ExpansionID] = append(byExpansion[p.ExpansionID], id)
	}
	var items []cm.Expansion
	for id := range byExpansion {
		e := mkm.catalog.Data.Expansions[id]
		name := e.Name
		if name == "" {
			name = fmt.Sprintf("expansion %d", id)
		}
		exp := cm.Expansion{IDExpansion: id, Name: name, SetCode: e.Code}
		if !foreignExpansion(mkm.gameID, exp) {
			items = append(items, exp)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].IDExpansion < items[j].IDExpansion })
	if mkm.gameID == cm.GameOnePiece {
		mkm.shelved = shelvedSets(mkm.backend, items)
		mkm.claimByID(byExpansion, products, items)
	}

	lines := []string{"product_id\texp_id\texp_name\tname\tnumber\trarity\tverdict\terr\tcard_id\tcard_id_foil\tby_name\tcands"}
	for _, exp := range items {
		ids := byExpansion[exp.IDExpansion]
		sort.Ints(ids)
		results := make([]resolved, 0, len(ids))
		cands := map[int]string{}
		for _, id := range ids {
			res := mkm.resolveMapped(id, products[id], exp)
			var alias *mtgmatcher.AliasingError
			if errors.As(res.err, &alias) {
				var c []string
				for _, probe := range alias.Probe() {
					co, _ := mkm.backend.GetUUID(probe)
					c = append(c, probe+"="+co.SetCode+"|"+co.Number+"|"+strings.Join(co.PromoTypes, ","))
				}
				cands[id] = strings.Join(c, " ; ")
			}
			results = append(results, res)
		}
		if mkm.gameID == cm.GameFleshAndBlood {
			mkm.disownBridged(results)
		}
		if same := sameProduct(mkm.gameID); same != nil {
			twinsAmong(results, same, faceOf(mkm.backend, mkm.gameID))
		}
		if mkm.gameID == cm.GameOnePiece {
			mkm.giveWay(results)
		}
		for _, r := range results {
			p := products[r.product.IDProduct]
			verdict := "landed"
			switch {
			case errors.Is(r.err, errTwin):
				verdict = "twin"
			case errors.Is(r.err, errForeign):
				verdict = "foreign"
			case errors.Is(r.err, errNoPrinting):
				verdict = "refused"
			case r.err != nil:
				verdict = "error"
			case r.cardID == "":
				verdict = "skipped"
			}
			errText := ""
			if r.err != nil {
				errText = r.err.Error()
			}
			lines = append(lines, fmt.Sprintf("%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%v\t%s",
				r.product.IDProduct, exp.IDExpansion, exp.Name, p.Name, p.Number, p.Rarity, verdict,
				errText, r.cardID, r.cardIDFoil, r.byName, cands[r.product.IDProduct]))
		}
	}
	err = os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: %d products in %d expansions", game, len(lines)-1, len(items))
}
```

### `cardmarket/zz_census_backend_test.go`

Every uuid with the fields grading reads, and the id indexes (`ext`).

```go
package cardmarket

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

func TestZZCensusBackend(t *testing.T) {
	game, dir, out := os.Getenv("ZZ_GAME"), os.Getenv("ZZ_DIR"), os.Getenv("ZZ_OUT")
	if game == "" {
		t.Skip()
	}
	b, err := datastore.Read(game, filepath.Join(dir, game+"-datastore.json"))
	if err != nil {
		t.Fatal(err)
	}
	type rec struct {
		UUID, Name, Set, Number, Plain, Finish, Rarity string
		Foil                                          bool
		Finishes, PromoTypes                          []string
		Identifiers                                   map[string]string
	}
	var recs []rec
	for _, u := range b.GetUUIDs() {
		co, _ := b.GetUUID(u)
		recs = append(recs, rec{u, co.Name, co.SetCode, co.Number, co.PlainNumber, co.Finish, co.Rarity, co.Foil, co.Finishes, co.PromoTypes, co.Identifiers})
	}
	sets := map[string]string{}
	for code, s := range b.Sets {
		sets[code] = s.Name
	}
	data, err := json.Marshal(map[string]any{"uuids": recs, "ext": b.ExternalIdentifiers, "sets": sets})
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(out, data, 0o644)
	if err != nil {
		t.Fatal(err)
	}
}
```

## One-off probes

A question the walk cannot answer - what `matchFab` does with a corrected
number, which rows a label reaches - is a small `zz_probe_test.go` beside
the templates. Load the census datastore with
`datastore.Read(game, "$D/data/<game>-datastore.json")`, build
`NewScraperIndex(b)`, and call the resolver method under question on a
hand-built `cm.Product` carrying the catalog's own name, number, rarity,
expansion name and code. Delete it before committing.
