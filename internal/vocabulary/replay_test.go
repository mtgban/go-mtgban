package vocabulary

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestReplayCatalogNames matches every product name a catalog holds and
// writes what each one answered with.
//
// It asserts nothing on its own. It is a before-and-after: run it on two
// checkouts and diff the two files, and every listing whose answer moved is
// named, with what it used to answer beside what it answers now. Every real
// defect found while moving the loaders onto the published facts came out of
// this and none out of the suite - a fold that sent "CS 25-26 Celebration
// Pack" to the wrong printing of Charlotte Pudding, a label built per-token
// that lost 41 names of 7,099, a mark taken from another printing's number.
// The goldens covered none of them, because a golden holds the cases someone
// thought of.
//
//	REPLAY_CATALOG=output/catalogs/onepiece.json \
//	ONEPIECE_PATH=output/onepiece.json \
//	REPLAY_OUT=/tmp/before.txt go test ./internal/vocabulary/ -run Replay
func TestReplayCatalogNames(t *testing.T) {
	catalog := os.Getenv("REPLAY_CATALOG")
	if catalog == "" {
		t.Skip("set REPLAY_CATALOG to a tcgdumper catalog dump, beside the game's own path variable")
	}
	var played int
	for _, game := range GameNames() {
		path := PathOf(game)
		if path == "" {
			continue
		}
		b, err := datastore.Read(game, path)
		if err != nil {
			t.Fatal(err)
		}
		lines, err := replay(b, catalog)
		if err != nil {
			t.Fatal(err)
		}
		played++
		t.Logf("%s: %d names replayed", game, len(lines))
		if out := os.Getenv("REPLAY_OUT"); out != "" {
			if err := os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if played == 0 {
		t.Skip("no game datastore is named in this run")
	}
}

// parens are the qualifiers a catalog writes behind a product's name, which
// is where a storefront's own wording comes from.
var parens = regexp.MustCompile(`\(([^()]*)\)`)

// replay matches every card product in a catalog the way a listing carrying
// that product's name would arrive: the name with its qualifiers taken off,
// the qualifiers as the wording, the number the catalog files it under, and
// the group as the edition.
func replay(b *mtgmatcher.Backend, path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var catalog struct {
		Groups []struct {
			GroupID int    `json:"groupId"`
			Name    string `json:"name"`
		} `json:"groups"`
		Products []struct {
			Name         string `json:"name"`
			GroupID      int    `json:"groupId"`
			ProductType  string `json:"productType"`
			ExtendedData []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"extendedData"`
		} `json:"products"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, err
	}
	editions := map[int]string{}
	for _, group := range catalog.Groups {
		editions[group.GroupID] = group.Name
	}
	// REPLAY_SHELF_TOTALS replays the number the way a storefront that
	// composes number/size from its shelf writes it: the card's own
	// numerator over the total most of the group's cards print, or over the
	// group's card count where none prints one. Right wherever the shelf is
	// the printing's own set, wrong wherever the set pools cards from
	// elsewhere - Unown Z/28 sold as Z/115 - which is what a matcher has to
	// survive, and 4,442 printings in 58 sets of the Pokemon datastore are
	// numbered against a total their shelf does not share.
	shelfTotals := os.Getenv("REPLAY_SHELF_TOTALS") != ""
	shelfFigure := map[int]string{}
	if shelfTotals {
		totals := map[int]map[string]int{}
		count := map[int]int{}
		for _, product := range catalog.Products {
			if product.ProductType != "Cards" {
				continue
			}
			count[product.GroupID]++
			for _, extended := range product.ExtendedData {
				if _, total, found := strings.Cut(extended.Value, "/"); extended.Name == "Number" && found && total != "" {
					if totals[product.GroupID] == nil {
						totals[product.GroupID] = map[string]int{}
					}
					totals[product.GroupID][strings.TrimLeft(total, "0")]++
				}
			}
		}
		for group, n := range count {
			figure, most := fmt.Sprint(n), 0
			for total, seen := range totals[group] {
				if seen > most || seen == most && total < figure {
					figure, most = total, seen
				}
			}
			shelfFigure[group] = figure
		}
	}
	lines := make([]string, 0, len(catalog.Products))
	for _, product := range catalog.Products {
		if product.ProductType != "Cards" {
			continue
		}
		var wording []string
		// The number first, where the catalog files one: a listing carries
		// it, and leaving it out makes two printings of a name ambiguous
		// that no storefront is ambiguous about.
		for _, extended := range product.ExtendedData {
			if extended.Name == "Number" && extended.Value != "" {
				number := extended.Value
				if numerator, _, found := strings.Cut(number, "/"); shelfTotals && found {
					number = numerator + "/" + shelfFigure[product.GroupID]
				}
				wording = append(wording, number)
			}
		}
		for _, qualifier := range parens.FindAllStringSubmatch(product.Name, -1) {
			wording = append(wording, qualifier[1])
		}
		in := mtgmatcher.InputCard{
			Name:      strings.TrimSpace(parens.ReplaceAllString(product.Name, "")),
			Variation: strings.Join(wording, " "),
			Edition:   editions[product.GroupID],
		}
		answer, err := b.Match(&in)
		if err != nil {
			answer = "ERR:" + err.Error()
		}
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s", product.Name, in.Edition, answer))
	}
	sort.Strings(lines)
	return lines, nil
}
