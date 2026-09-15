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

func init() {
	mtgban.Register("coolstuffinc", singlesGames, newScraper)
	mtgban.Register("coolstuffinc_sealed", sealedGames, newScraperSealed)
}

func newScraper(b *mtgmatcher.Backend, _ mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper, err := NewScraper(b)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = opts.LogCallback
	scraper.TargetEdition = opts.TargetEdition
	scraper.Partner = opts.Affiliate
	if opts.MaxConcurrency != 0 {
		scraper.MaxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}

func newScraperSealed(b *mtgmatcher.Backend, _ mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper, err := NewScraperSealed(b)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = opts.LogCallback
	scraper.Partner = opts.Affiliate
	if opts.MaxConcurrency != 0 {
		scraper.MaxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}
