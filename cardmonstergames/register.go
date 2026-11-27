package cardmonstergames

import (
	"errors"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/tcgplayer"

	api "github.com/mtgban/go-tcgplayer"
)

func init() {
	mtgban.Register("cardmonstergames", []matcher.Game{matcher.GameMagic}, newScraper)
}

func newScraper(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	catalog, err := mtgban.Resource[*api.CatalogDump](opts, tcgplayer.ResourceCatalog)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, errors.New("cardmonstergames needs tcgplayer.WithCatalog")
	}

	scraper := NewScraper(b)
	scraper.catalog = catalog
	scraper.logCallback = opts.LogCallback
	return scraper, nil
}
