package coolstuffinc

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// singlesGames are the games "coolstuffinc" registers for.
var singlesGames = []matcher.Game{
	matcher.GameMagic, matcher.GameLorcana, matcher.GameRiftbound, matcher.GameOnePiece,
	matcher.GamePokemon, matcher.GameYuGiOh, matcher.GameGundam, matcher.GamePalworld,
}

// sealedGames are the games "coolstuffinc_sealed" registers for: every
// singles game but Gundam and Palworld, which this storefront sells no
// sealed product of.
var sealedGames = []matcher.Game{
	matcher.GameMagic, matcher.GameLorcana, matcher.GameRiftbound, matcher.GameOnePiece,
	matcher.GamePokemon, matcher.GameYuGiOh,
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

func newScraper(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
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

func newScraperSealed(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
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
