// Command lorcanadatastore builds the Lorcana datastore file consumed by
// go-mtgban's mtgmatcher/lorcana loader: it takes the LorcanaJSON allCards
// payload and merges in what our TCGplayer catalog dump for category 71
// knows and it does not.
//
// Unlike Riftbound, where the official gallery says nothing at all about
// commerce, LorcanaJSON already carries a TCGplayer product id for 99.6% of
// its cards, so the merge is deliberately narrow (see
// todo/lorcana-datastore.md for the measurements behind that):
//
//   - it fills the product id on cards that have none, when exactly one
//     unclaimed catalog product matches by name and collector number;
//   - it records the extra product ids TCGplayer uses for a card's foil,
//     which it sells as a separate product, so a feed keyed on those ids
//     resolves to the card instead of being dropped.
//
// Card identity is left entirely to LorcanaJSON. Its integer card ids are
// the matcher's uuids and are quoted directly in chart URLs, and its foil
// sub-type names ("Silver", "RainbowPillars", …) are what
// mtgmatcher/lorcana's selectFinish resolves storefront wording against;
// TCGplayer knows only Normal/Holofoil/Cold Foil and can reproduce neither.
//
// The output is the LorcanaJSON payload itself with the extra data merged
// in, so the loader reads it unchanged and a stock LorcanaJSON reader still
// parses it - and it is round-tripped through that very loader before being
// written, so a broken upstream payload can never be published.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// lorcanaCategory is Lorcana's TCGplayer category, the one the catalog dump
// is expected to carry.
const lorcanaCategory = 71

type tcgProduct struct {
	ProductID    int    `json:"productId"`
	Name         string `json:"name"`
	GroupID      int    `json:"groupId"`
	ExtendedData []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"extendedData"`
	// Skus enumerate every printing/condition/language a product is sold in.
	// Only the printing matters here, and it is the whole reason the catalog
	// dump is read instead of a price feed: a printing exists whether or not
	// anyone happens to be selling it today.
	Skus []struct {
		PrintingID int `json:"printingId"`
	} `json:"skus"`
}

func (p tcgProduct) extended(name string) string {
	for _, e := range p.ExtendedData {
		if e.Name == name {
			return e.Value
		}
	}
	return ""
}

// tcgCatalog is the dump tcgdumper (github.com/mtgban/go-tcgplayer) writes
// for a category, published next to the datastore it describes.
type tcgCatalog struct {
	Category struct {
		CategoryID int `json:"categoryId"`
	} `json:"category"`
	Printings []struct {
		PrintingID int    `json:"printingId"`
		Name       string `json:"name"`
	} `json:"printings"`
	Groups []struct {
		GroupID int    `json:"groupId"`
		Name    string `json:"name"`
	} `json:"groups"`
	Products []tcgProduct `json:"products"`
}

// foilOnly reports, for each product, whether every printing it is sold in
// is a foil one. TCGplayer's category 71 has exactly three printings -
// Normal, Holofoil and Cold Foil - and a printing it does not list for a
// product is one that does not exist.
func (c *tcgCatalog) foilOnly() map[int]bool {
	isFoil := map[int]bool{}
	for _, p := range c.Printings {
		isFoil[p.PrintingID] = p.Name != "Normal"
	}

	out := map[int]bool{}
	for _, product := range c.Products {
		var foil, nonfoil bool
		for _, sku := range product.Skus {
			if isFoil[sku.PrintingID] {
				foil = true
			} else {
				nonfoil = true
			}
		}
		out[product.ProductID] = foil && !nonfoil
	}
	return out
}

// number reduces a collector number to the loader's canonical form: what
// precedes any "/total" tail, without leading zeros. An all-zero number stays
// "0", because a genuine 0-numbered promo exists.
func number(code string) string {
	code = strings.Split(code, "/")[0]
	trimmed := strings.TrimLeft(code, "0")
	if trimmed == "" && code != "" {
		return "0"
	}
	return trimmed
}

// normalizeName reduces a name to what two spellings of the same card share:
// TCGplayer drops diacritics ("Te Ka" for "Te Kā") and appends storefront
// decoration in parentheses, neither of which is part of the card's identity.
func normalizeName(name string) string {
	if idx := strings.IndexByte(name, '('); idx >= 0 {
		name = name[:idx]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// fetch reads a local path, or an http(s) URL when one is given. The
// LorcanaJSON download location is deliberately not hardcoded: CI already
// holds it in vars.DATASTORE_LORCANA and passes it in, so there is one place
// to change if upstream moves.
func fetch(location string) ([]byte, error) {
	if !strings.HasPrefix(location, "http://") && !strings.HasPrefix(location, "https://") {
		return os.ReadFile(location)
	}
	resp, err := http.Get(location)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", location, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// card is the handful of fields the merge reads out of a generically decoded
// LorcanaJSON card, so everything else survives the round trip untouched.
type card struct {
	raw      map[string]any
	links    map[string]any
	fullName string
	setCode  string
	number   string
	tcgID    int
}

func decodeCard(item any) (card, bool) {
	raw, ok := item.(map[string]any)
	if !ok {
		return card{}, false
	}
	name, _ := raw["fullName"].(string)
	setCode, _ := raw["setCode"].(string)
	num, _ := raw["number"].(float64)
	if name == "" || setCode == "" {
		return card{}, false
	}

	// externalLinks is present on every card in practice; create it rather
	// than skip the card, so a card missing it can still be given an id.
	links, ok := raw["externalLinks"].(map[string]any)
	if !ok {
		links = map[string]any{}
		raw["externalLinks"] = links
	}
	id, _ := links["tcgPlayerId"].(float64)

	return card{
		raw:      raw,
		links:    links,
		fullName: name,
		setCode:  setCode,
		number:   strconv.Itoa(int(num)),
		tcgID:    int(id),
	}, true
}

func main() {
	output := flag.String("o", "", "output file (default stdout)")
	minCards := flag.Int("min-cards", 3000, "refuse to emit a datastore with fewer cards")
	catalogPath := flag.String("tcg-catalog", "", "tcgdumper catalog dump for category 71 (required)")
	source := flag.String("lorcana", "", "LorcanaJSON allCards file, path or URL (required)")
	flag.Parse()

	if *catalogPath == "" {
		log.Fatalln("-tcg-catalog is required: the dump carries the product ids")
	}
	if *source == "" {
		log.Fatalln("-lorcana is required: the LorcanaJSON allCards file this enriches")
	}

	catalogData, err := os.ReadFile(*catalogPath)
	if err != nil {
		log.Fatalln("tcg catalog:", err)
	}
	var catalog tcgCatalog
	if err := json.Unmarshal(catalogData, &catalog); err != nil {
		log.Fatalln("tcg catalog:", err)
	}
	if catalog.Category.CategoryID != lorcanaCategory {
		log.Fatalf("tcg catalog: category %d, want %d (wrong game's dump)",
			catalog.Category.CategoryID, lorcanaCategory)
	}
	foilOnly := catalog.foilOnly()
	log.Printf("catalog: %d groups, %d products", len(catalog.Groups), len(catalog.Products))

	payload, err := fetch(*source)
	if err != nil {
		log.Fatalln("lorcana source:", err)
	}
	// Decode generically so everything the loader does not care about
	// survives the round trip untouched.
	var doc map[string]any
	if err := json.Unmarshal(payload, &doc); err != nil {
		log.Fatalln("lorcana source:", err)
	}
	items, _ := doc["cards"].([]any)
	if len(items) == 0 {
		log.Fatalln("lorcana source: no cards")
	}

	var cards []card
	claimed := map[int]bool{}
	for _, item := range items {
		c, ok := decodeCard(item)
		if !ok {
			continue
		}
		cards = append(cards, c)
		if c.tcgID != 0 {
			claimed[c.tcgID] = true
		}
	}
	log.Printf("lorcana: %d cards, %d already carrying a product id", len(cards), len(claimed))

	// Index the products no card claims, by normalized name and collector
	// number. Both lookups below key on that pair rather than on the group,
	// because TCGplayer files promotional printings in their own groups
	// (DLPC, D23, D100) while LorcanaJSON files them under the set they
	// belong to, so the group never lines up for exactly the cards that need
	// the most help.
	unclaimed := map[string][]tcgProduct{}
	for _, product := range catalog.Products {
		if claimed[product.ProductID] {
			continue
		}
		num := product.extended("Number")
		if num == "" {
			// Not a single: sealed, puzzle inserts, accessories. Nothing to
			// identify it by, and the matcher has no concept for it yet.
			continue
		}
		key := normalizeName(product.Name) + "|" + number(num)
		unclaimed[key] = append(unclaimed[key], product)
	}
	// Stable order, so unchanged data keeps producing byte-identical output.
	for key := range unclaimed {
		products := unclaimed[key]
		sort.Slice(products, func(i, j int) bool {
			return products[i].ProductID < products[j].ProductID
		})
	}

	var filled, extras int
	for _, c := range cards {
		key := normalizeName(c.fullName) + "|" + c.number
		candidates := unclaimed[key]

		if c.tcgID == 0 {
			// Only an unambiguous match may stand in for an id upstream did
			// not publish: several candidates means we cannot tell which
			// printing is the card's, and a wrong id silently reroutes a
			// card's whole price history.
			if len(candidates) != 1 {
				continue
			}
			c.links["tcgPlayerId"] = candidates[0].ProductID
			filled++
			continue
		}

		// TCGplayer sometimes sells a card's foil as its own product, leaving
		// the claimed product foilless; those extra ids resolve to this same
		// printing. The name must match exactly once the decoration is
		// stripped AND the product must be foil-only, which excludes the
		// oversized, errata and region-exclusive listings that share a name
		// and number but are a different object whose prices must not land
		// here.
		var ids []int
		for _, product := range candidates {
			if !foilOnly[product.ProductID] {
				continue
			}
			if !strings.HasSuffix(product.Name, "(Foil)") {
				continue
			}
			ids = append(ids, product.ProductID)
		}
		if len(ids) > 0 {
			c.links["tcgPlayerExtraIds"] = ids
			extras += len(ids)
		}
	}
	log.Printf("merged: %d product ids filled in, %d extra product ids recorded", filled, extras)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		log.Fatalln(err)
	}

	// Round-trip through the real loader before publishing anything: an
	// upstream format change or a truncated download must fail here, not in
	// every consumer.
	backend, err := lorcana.Load(bytes.NewReader(buf.Bytes()))
	if err != nil {
		log.Fatalln("validation:", err)
	}
	// An id filed under the empty string would make GetUUID("") answer with a
	// real card; the loader guards against it, and this makes sure.
	if _, err := backend.GetUUID(""); err == nil {
		log.Fatalln("validation: the empty uuid resolves to a card")
	}
	log.Printf("validated: %d sets, %d cards, %d uuids, %d tcgplayer ids",
		len(backend.Sets), len(cards), len(backend.GetUUIDs()), len(backend.ExternalIdentifiers))
	if len(cards) < *minCards {
		log.Fatalf("only %d cards (minimum %d); refusing to publish", len(cards), *minCards)
	}

	out := os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			log.Fatalln(err)
		}
		defer f.Close()
		out = f
	}
	if _, err := out.Write(buf.Bytes()); err != nil {
		log.Fatalln(err)
	}
}
