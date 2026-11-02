package tcgplayer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/mtgban/go-tcgplayer"
)

// LoadCatalog reads a category's catalog dump, as go-tcgplayer's tcgdumper
// writes it each night: every product with its skus, the groups, and the
// category's printing, condition and language tables. The game datastores
// are built from the same dump, so it lists every product they can name.
//
// The products are read one at a time and keep only what the scrapers read
// of them: id, name, group, category, type, skus, and the collector number
// of their extended data. Magic's dump is 530 MiB of JSON, most of it the
// skus; the card text, urls and images beside them are dropped as read.
func LoadCatalog(reader io.Reader) (*tcgplayer.CatalogDump, error) {
	dec := json.NewDecoder(reader)
	err := catalogDelim(dec, '{')
	if err != nil {
		return nil, err
	}
	var dump tcgplayer.CatalogDump
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch token {
		case "products":
			err = loadCatalogProducts(dec, &dump)
		case "category":
			err = dec.Decode(&dump.Category)
		case "conditions":
			err = dec.Decode(&dump.Conditions)
		case "languages":
			err = dec.Decode(&dump.Languages)
		case "printings":
			err = dec.Decode(&dump.Printings)
		case "rarities":
			err = dec.Decode(&dump.Rarities)
		case "groups":
			err = dec.Decode(&dump.Groups)
		default:
			var skip json.RawMessage
			err = dec.Decode(&skip)
		}
		if err != nil {
			return nil, err
		}
	}
	err = catalogDelim(dec, '}')
	if err != nil {
		return nil, err
	}
	if len(dump.Products) == 0 {
		return nil, errors.New("empty catalog dump")
	}
	return &dump, nil
}

// loadCatalogProducts reads the products array an element at a time.
func loadCatalogProducts(dec *json.Decoder, dump *tcgplayer.CatalogDump) error {
	err := catalogDelim(dec, '[')
	if err != nil {
		return err
	}
	for dec.More() {
		var product tcgplayer.Product
		err := dec.Decode(&product)
		if err != nil {
			return err
		}
		kept := tcgplayer.Product{
			ProductID:   product.ProductID,
			Name:        product.Name,
			GroupID:     product.GroupID,
			CategoryID:  product.CategoryID,
			ProductType: product.ProductType,
			Skus:        product.Skus,
		}
		number := product.Extended("Number")
		if number != "" {
			kept.ExtendedData = []tcgplayer.ExtendedField{{Name: "Number", Value: number}}
		}
		dump.Products = append(dump.Products, kept)
	}
	return catalogDelim(dec, ']')
}

// catalogDelim reads the next token, which must be delim.
func catalogDelim(dec *json.Decoder, delim json.Delim) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	if token != delim {
		return fmt.Errorf("catalog dump: got %v, want %v", token, delim)
	}
	return nil
}

// catalogProducts are the products the dump files under any of types, once
// each: the dumper lists a product once for every type it is filed under.
// A dump with none, as one written without the types would be, is refused
// rather than priced as an empty inventory.
func catalogProducts(dump *tcgplayer.CatalogDump, types []string) ([]tcgplayer.Product, error) {
	seen := map[int]bool{}
	var out []tcgplayer.Product
	for _, product := range dump.Products {
		if seen[product.ProductID] || !slices.Contains(types, product.ProductType) {
			continue
		}
		seen[product.ProductID] = true
		out = append(out, product)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("catalog dump lists no %q products for category %d", types, dump.Category.CategoryID)
	}
	return out, nil
}

// checkCatalog fails a dump of some other category than the scraper's.
func checkCatalog(dump *tcgplayer.CatalogDump, category int) error {
	if dump == nil {
		return errors.New("no catalog dump")
	}
	if dump.Category.CategoryID != category {
		return fmt.Errorf("catalog dump is of category %d (%s), not %d", dump.Category.CategoryID, dump.Category.Name, category)
	}
	return nil
}
