package coolstuffinc

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// singlesGames are the games "coolstuffinc" registers for.
var singlesGames = []mtgban.Game{
	mtgban.GameMagic, mtgban.GameLorcana, mtgban.GameRiftbound, mtgban.GameOnePiece,
	mtgban.GamePokemon, mtgban.GameYuGiOh, mtgban.GameGundam, mtgban.GamePalworld,
}

// sealedGames are the games "coolstuffinc_sealed" registers for: every
// singles game but Gundam and Palworld, which this storefront sells no
// sealed product of.
var sealedGames = []mtgban.Game{
	mtgban.GameMagic, mtgban.GameLorcana, mtgban.GameRiftbound, mtgban.GameOnePiece,
	mtgban.GamePokemon, mtgban.GameYuGiOh,
}

// ResourceIncludeOOS names the optional out-of-stock listing input.
const ResourceIncludeOOS = "coolstuffinc.include_oos"

// WithIncludeOOS includes listings without a nonfoil NM price.
func WithIncludeOOS() mtgban.Option {
	return mtgban.WithResource(ResourceIncludeOOS, true)
}

func init() {
	mtgban.Register("coolstuffinc", singlesGames, newScraper)
	mtgban.Register("coolstuffinc_sealed", sealedGames, newScraperSealed)
}

func newScraper(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper, err := NewScraper(b)
	if err != nil {
		return nil, err
	}
	scraper.logCallback = opts.LogCallback
	scraper.targetEdition = opts.TargetEdition
	scraper.partner = opts.Affiliate
	includeOOS, err := mtgban.Resource[bool](opts, ResourceIncludeOOS)
	if err != nil {
		return nil, err
	}
	scraper.includeOOS = includeOOS
	if opts.MaxConcurrency != 0 {
		scraper.maxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}

func newScraperSealed(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper, err := NewScraperSealed(b)
	if err != nil {
		return nil, err
	}
	scraper.logCallback = opts.LogCallback
	scraper.partner = opts.Affiliate
	if opts.MaxConcurrency != 0 {
		scraper.maxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}
