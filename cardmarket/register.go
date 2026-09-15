package cardmarket

import (
	"errors"
	"fmt"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Secret names the environment variables bantool has always read these
// credentials from.
const (
	// SecretAppToken and SecretAppSecret authenticate the Market and Sealed
	// scrapers, which query Cardmarket's own API rather than reading a
	// published download the way Index does.
	SecretAppToken  = "MKM_APP_TOKEN"
	SecretAppSecret = "MKM_APP_SECRET"

	// SecretBanKey authenticates the mtgban price snapshot Market's Load
	// reads to pre-filter which cards are worth a live call. It is
	// optional: only cardmarket_market reads it, into BanPriceKey.
	SecretBanKey = "BAN_API_KEY"
)

// Resource names the typed inputs the constructors below read out of
// mtgban.Options.
const (
	// ResourceCatalog names the published id-map catalog Index and Market
	// resolve from before falling back to name/number matching.
	ResourceCatalog = "cardmarket.catalog"

	// ResourceBridge names the Cardmarket-id to TCGplayer-id bridge; see
	// BridgeUse.
	ResourceBridge = "cardmarket.bridge"
)

// WithCatalog hands Index or Market the published id-map catalog it
// resolves from before falling back to name/number matching.
func WithCatalog(catalog *cm.Catalog) mtgban.Option {
	return mtgban.WithResource(ResourceCatalog, catalog)
}

// WithBridge hands a Cardmarket scraper the Cardmarket-id to TCGplayer-id
// bridge cardtrader's blueprints supply - the one source linking the two
// marketplaces. See BridgeUse for which games need one.
func WithBridge(bridge map[int]int) mtgban.Option {
	return mtgban.WithResource(ResourceBridge, bridge)
}

// BridgeUse says how much a game's Index and Market scraper leans on the
// TCGplayer bridge, for bantool to decide whether the bridge - a cardtrader
// crawl - is worth building, and whether a failure to build it is fatal.
type BridgeUse int

const (
	// BridgeUnused means the catalog names every printing of this game on
	// its own; a bridge given anyway changes nothing.
	BridgeUnused BridgeUse = iota
	// BridgeHelps means the catalog names most of the shelf by itself; the
	// bridge settles what a collector number cannot, and its absence only
	// costs those printings.
	BridgeHelps
	// BridgeRequired means same-name products abound and the catalog alone
	// cannot tell them apart; without a bridge the scraper cannot run.
	BridgeRequired
)

// BridgeUseOf reports how much a game's Index and Market scrapers lean on
// the TCGplayer bridge; see resolveProduct. Sealed's own need is not a
// function of the game - see cardmarket_sealed's registration below - so it
// is not read here.
func BridgeUseOf(game mtgban.Game) BridgeUse {
	switch game {
	case mtgban.GamePokemon, mtgban.GameYuGiOh, mtgban.GameFleshAndBlood:
		return BridgeRequired
	case mtgban.GameOnePiece:
		return BridgeHelps
	default:
		return BridgeUnused
	}
}

// cardmarketGames are the games Cardmarket's Index, Market and Sealed
// scrapers price - the same seven mkmGames names.
var cardmarketGames = []mtgban.Game{
	mtgban.GameMagic, mtgban.GameLorcana, mtgban.GameRiftbound,
	mtgban.GameOnePiece, mtgban.GamePokemon, mtgban.GameYuGiOh,
	mtgban.GameFleshAndBlood,
}

func init() {
	mtgban.Register("cardmarket", cardmarketGames, buildIndex)
	mtgban.Register("cardmarket_market", cardmarketGames, buildMarket)
	mtgban.Register("cardmarket_sealed", cardmarketGames, buildSealed)
}

// buildIndex is cardmarket's Constructor. It needs no credential: Index
// prices from a published catalog and the public price guide.
func buildIndex(b *mtgmatcher.Backend, _ mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	game, err := mtgban.GameOf(b)
	if err != nil {
		return nil, err
	}
	scraper, err := NewScraperIndex(b)
	if err != nil {
		return nil, err
	}

	catalog, err := mtgban.Resource[*cm.Catalog](opts, ResourceCatalog)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, errors.New("cardmarket needs WithCatalog")
	}
	scraper.Catalog = catalog

	bridge, err := mtgban.Resource[map[int]int](opts, ResourceBridge)
	if err != nil {
		return nil, err
	}
	if len(bridge) == 0 && BridgeUseOf(game) == BridgeRequired {
		return nil, fmt.Errorf("cardmarket needs WithBridge for %s", game)
	}
	scraper.TCGBridge = bridge

	scraper.LogCallback = opts.LogCallback
	scraper.Affiliate = opts.Affiliate
	if opts.MaxConcurrency != 0 {
		scraper.MaxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}

// buildMarket is cardmarket_market's Constructor.
func buildMarket(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	game, err := mtgban.GameOf(b)
	if err != nil {
		return nil, err
	}
	appToken, err := auth.Secret(SecretAppToken)
	if err != nil {
		return nil, err
	}
	appSecret, err := auth.Secret(SecretAppSecret)
	if err != nil {
		return nil, err
	}
	scraper, err := NewScraperMarket(b, appToken, appSecret)
	if err != nil {
		return nil, err
	}

	catalog, err := mtgban.Resource[*cm.Catalog](opts, ResourceCatalog)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, errors.New("cardmarket_market needs WithCatalog")
	}
	scraper.Catalog = catalog

	bridge, err := mtgban.Resource[map[int]int](opts, ResourceBridge)
	if err != nil {
		return nil, err
	}
	if len(bridge) == 0 && BridgeUseOf(game) == BridgeRequired {
		return nil, fmt.Errorf("cardmarket_market needs WithBridge for %s", game)
	}
	scraper.TCGBridge = bridge

	scraper.LogCallback = opts.LogCallback
	scraper.Affiliate = opts.Affiliate
	scraper.BanPriceKey, err = mtgban.OptionalSecret(auth, SecretBanKey)
	if err != nil {
		return nil, err
	}
	return scraper, nil
}

// buildSealed is cardmarket_sealed's Constructor. Unlike Index and Market,
// it requires the bridge for every game rather than following BridgeUseOf:
// whether Sealed.Load ends up needing it depends on what the datastore's
// own sealed products carry, which the constructor cannot see in advance.
func buildSealed(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	appToken, err := auth.Secret(SecretAppToken)
	if err != nil {
		return nil, err
	}
	appSecret, err := auth.Secret(SecretAppSecret)
	if err != nil {
		return nil, err
	}
	scraper, err := NewScraperSealed(b, appToken, appSecret)
	if err != nil {
		return nil, err
	}

	bridge, err := mtgban.Resource[map[int]int](opts, ResourceBridge)
	if err != nil {
		return nil, err
	}
	if len(bridge) == 0 {
		return nil, errors.New("cardmarket_sealed needs WithBridge")
	}
	scraper.TCGBridge = bridge

	scraper.LogCallback = opts.LogCallback
	scraper.Affiliate = opts.Affiliate
	if opts.MaxConcurrency != 0 {
		scraper.MaxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}
