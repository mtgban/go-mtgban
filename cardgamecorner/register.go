package cardgamecorner

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// ResourceBridge names the Cardmarket-id to TCGplayer-id bridge; see
// WithBridge.
const ResourceBridge = "cardgamecorner.bridge"

// WithBridge hands the scraper the Cardmarket-id to TCGplayer-id bridge
// cardtrader's blueprints supply. The store sells Cardmarket's catalog and
// names its images after Cardmarket's product ids, which the bridge turns
// into a printing for the games whose datastore publishes few of them.
func WithBridge(bridge map[int]int) mtgban.Option {
	return mtgban.WithResource(ResourceBridge, bridge)
}

func init() {
	mtgban.Register("cardgamecorner",
		[]matcher.Game{
			matcher.GameMagic, matcher.GamePokemon, matcher.GameYuGiOh,
			matcher.GameOnePiece, matcher.GameLorcana, matcher.GameRiftbound,
			matcher.GameFleshAndBlood,
		},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper, err := NewScraper(b)
			if err != nil {
				return nil, err
			}
			scraper.bridge, err = mtgban.Resource[map[int]int](opts, ResourceBridge)
			if err != nil {
				return nil, err
			}
			scraper.logCallback = opts.LogCallback
			if opts.MaxConcurrency != 0 {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
