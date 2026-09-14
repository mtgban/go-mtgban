package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/cardmarket"
	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/coolstuffinc"
	"github.com/mtgban/go-mtgban/gamenerdz"
	"github.com/mtgban/go-mtgban/miniaturemarket"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/starcitygames"
	"github.com/mtgban/go-mtgban/strikezone"
	"github.com/mtgban/go-mtgban/tcgplayer"
	"github.com/mtgban/go-mtgban/vegassingles"
)

// The scrapers below price several games, and a vendor's games differ only in
// the constant naming them to that vendor's own API - never the same constant
// twice, and rarely spelled the way mtgmatcher spells it. Each family keeps
// one table translating the canonical game name bantool is given into that
// constant, and refuses whatever the table does not carry: the "does this
// vendor support this game" check other callers used to answer by which of
// several near-identical options-table entries they enabled now lives here,
// once per family, next to the ids it is translating between.

func tcgplayerCredentials() (publicID, privateID string, err error) {
	publicID = os.Getenv("TCGPLAYER_PUBLIC_KEY")
	privateID = os.Getenv("TCGPLAYER_PRIVATE_KEY")
	if publicID == "" || privateID == "" {
		return "", "", errors.New("missing TCGPLAYER_PUBLIC_KEY or TCGPLAYER_PRIVATE_KEY env vars")
	}
	return publicID, privateID, nil
}

func cardmarketCredentials() (appToken, appSecret string, err error) {
	appToken = os.Getenv("MKM_APP_TOKEN")
	appSecret = os.Getenv("MKM_APP_SECRET")
	if appToken == "" || appSecret == "" {
		return "", "", errors.New("missing MKM_APP_TOKEN or MKM_APP_SECRET env vars")
	}
	return appToken, appSecret, nil
}

func cardtraderToken() (string, error) {
	token := os.Getenv("CARDTRADER_TOKEN_BEARER")
	if token == "" {
		return "", errors.New("missing CARDTRADER_TOKEN_BEARER env var")
	}
	return token, nil
}

func starcitygamesKey() (string, error) {
	key := os.Getenv("SCG_API_KEY")
	if key == "" {
		return "", errors.New("missing SCG_API_KEY env var")
	}
	return key, nil
}

// cardmarketGames translates the canonical game name into Cardmarket's own
// numbering, which a bare game id cannot be told apart from without it.
var cardmarketGames = map[string]int{
	"magic":         cm.GameMagic,
	"fleshandblood": cm.GameFleshAndBlood,
	"lorcana":       cm.GameLorcana,
	"onepiece":      cm.GameOnePiece,
	"pokemon":       cm.GamePokemon,
	"riftbound":     cm.GameRiftbound,
	"yugioh":        cm.GameYuGiOh,
}

// cardmarketIndexBridge names, for the singles side, the CardTrader game
// whose blueprints supply the ids Cardmarket's own catalog cannot: Magic,
// Lorcana and Riftbound need no bridge at all, One Piece names most of its
// shelf from the catalog alone and only widens with the bridge (see
// cardmarketScraper), and the rest cannot be identified without one.
var cardmarketIndexBridge = map[string]int{
	"fleshandblood": cardtrader.GameFleshAndBlood,
	"onepiece":      cardtrader.GameOnePiece,
	"pokemon":       cardtrader.GamePokemon,
	"yugioh":        cardtrader.GameYuGiOh,
}

// cardmarketIndexOptionalBridge names the one game whose bridge is a widener
// rather than a requirement: a CardTrader that will not answer costs One
// Piece the printings a collector number cannot tell apart, and nothing
// else, so the run goes ahead saying so rather than failing and pricing
// nothing.
var cardmarketIndexOptionalBridge = map[string]bool{
	"onepiece": true,
}

// cardmarketSealedBridge is cardmarketIndexBridge's sealed counterpart, and
// covers every non-Magic game rather than the singles side's four: sealed
// sourcing cross-references CardTrader even for the three games Cardmarket
// identifies singles for on its own.
var cardmarketSealedBridge = map[string]int{
	"fleshandblood": cardtrader.GameFleshAndBlood,
	"lorcana":       cardtrader.GameLorcana,
	"onepiece":      cardtrader.GameOnePiece,
	"pokemon":       cardtrader.GamePokemon,
	"riftbound":     cardtrader.GameRiftbound,
	"yugioh":        cardtrader.GameYuGiOh,
}

func cardmarketScraper(game string) (mtgban.Scraper, error) {
	gameID, ok := cardmarketGames[game]
	if !ok {
		return nil, fmt.Errorf("cardmarket does not support %q", game)
	}

	scraper := cardmarket.NewScraperIndex(gameID)
	err := loadCardmarketCatalog(scraper)
	if err != nil {
		return nil, err
	}

	if bridgedGame, needsBridge := cardmarketIndexBridge[game]; needsBridge {
		bridge, err := cardtraderBridge(bridgedGame)
		if err != nil {
			if !cardmarketIndexOptionalBridge[game] {
				return nil, err
			}
			log.Printf("bridge unavailable, naming what the catalog can on its own: %v", err)
		} else {
			scraper.TCGBridge = bridge
		}
	}

	scraper.LogCallback = GlobalLogCallback
	scraper.Affiliate = os.Getenv("MKM_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// loadCardmarketCatalog hands an index scraper the published catalog it prices
// from, whatever the game and whichever constructor built it: MTGJSON
// publishes Magic's, go-cardmarket's mkmcatalog builds the rest.
func loadCardmarketCatalog(scraper *cardmarket.Index) error {
	path := os.Getenv("MTGJSON_MKMID_PATH")
	if path == "" {
		return errors.New("missing MTGJSON_MKMID_PATH env var")
	}
	reader, err := openPath(path, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
	if err != nil {
		return err
	}
	defer reader.Close()
	catalog, err := cm.LoadCatalog(reader)
	if err != nil {
		return err
	}
	scraper.Catalog = catalog
	return nil
}

func cardmarketSealedScraper(game string) (mtgban.Scraper, error) {
	gameID, ok := cardmarketGames[game]
	if !ok {
		return nil, fmt.Errorf("cardmarket does not support %q", game)
	}

	appToken, appSecret, err := cardmarketCredentials()
	if err != nil {
		return nil, err
	}
	scraper, err := cardmarket.NewScraperSealed(gameID, appToken, appSecret)
	if err != nil {
		return nil, err
	}

	if bridgedGame, needsBridge := cardmarketSealedBridge[game]; needsBridge {
		scraper.TCGBridge, err = cardtraderBridge(bridgedGame)
		if err != nil {
			return nil, err
		}
	}

	scraper.LogCallback = GlobalLogCallback
	scraper.Affiliate = os.Getenv("MKM_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// cardtraderGames translates the canonical game name into CardTrader's own
// numbering.
var cardtraderGames = map[string]int{
	"magic":         cardtrader.GameMagic,
	"fleshandblood": cardtrader.GameFleshAndBlood,
	"gundam":        cardtrader.GameGundam,
	"lorcana":       cardtrader.GameLorcana,
	"onepiece":      cardtrader.GameOnePiece,
	"pokemon":       cardtrader.GamePokemon,
	"riftbound":     cardtrader.GameRiftbound,
	"yugioh":        cardtrader.GameYuGiOh,
}

func cardtraderMarketScraper(game string) (mtgban.Scraper, error) {
	gameID, ok := cardtraderGames[game]
	if !ok {
		return nil, fmt.Errorf("cardtrader does not support %q", game)
	}
	token, err := cardtraderToken()
	if err != nil {
		return nil, err
	}
	scraper, err := cardtrader.NewScraperMarket(gameID, token)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = GlobalLogCallback
	scraper.ShareCode = os.Getenv("CT_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

func cardtraderSealedScraper(game string) (mtgban.Scraper, error) {
	gameID, ok := cardtraderGames[game]
	if !ok {
		return nil, fmt.Errorf("cardtrader does not support %q", game)
	}
	token, err := cardtraderToken()
	if err != nil {
		return nil, err
	}
	scraper, err := cardtrader.NewScraperSealed(gameID, token)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = GlobalLogCallback
	scraper.ShareCode = os.Getenv("CT_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// coolstuffincGames translates the canonical game name into CoolStuffInc's
// own tag, which names games this store does not carry (dbs, swu) beside
// the ones it does.
var coolstuffincGames = map[string]string{
	"magic":     coolstuffinc.GameMagic,
	"gundam":    coolstuffinc.GameGundam,
	"lorcana":   coolstuffinc.GameLorcana,
	"onepiece":  coolstuffinc.GameOnePiece,
	"palworld":  coolstuffinc.GamePalworld,
	"pokemon":   coolstuffinc.GamePokemon,
	"riftbound": coolstuffinc.GameRiftbound,
	"yugioh":    coolstuffinc.GameYuGiOh,
}

func coolstuffincScraper(game string) (mtgban.Scraper, error) {
	tag, ok := coolstuffincGames[game]
	if !ok {
		return nil, fmt.Errorf("coolstuffinc does not support %q", game)
	}
	scraper := coolstuffinc.NewScraper(tag)
	scraper.LogCallback = GlobalLogCallback
	scraper.Partner = os.Getenv("CSI_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// coolstuffincSealedGames is coolstuffincGames minus the two games sealed is
// not scheduled for: Gundam and Palworld have never had a coolstuffinc_sealed
// entry, and widening it to the singles side's full set would enable a run
// nothing has verified prices sealed product correctly.
var coolstuffincSealedGames = map[string]string{
	"magic":     coolstuffinc.GameMagic,
	"lorcana":   coolstuffinc.GameLorcana,
	"onepiece":  coolstuffinc.GameOnePiece,
	"pokemon":   coolstuffinc.GamePokemon,
	"riftbound": coolstuffinc.GameRiftbound,
	"yugioh":    coolstuffinc.GameYuGiOh,
}

func coolstuffincSealedScraper(game string) (mtgban.Scraper, error) {
	tag, ok := coolstuffincSealedGames[game]
	if !ok {
		return nil, fmt.Errorf("coolstuffinc does not support %q", game)
	}
	scraper := coolstuffinc.NewScraperSealed(tag)
	scraper.LogCallback = GlobalLogCallback
	scraper.Partner = os.Getenv("CSI_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// gamenerdzGames translates the canonical game name into the display string
// GameNerdz spells its own catalog with.
var gamenerdzGames = map[string]string{
	"magic":         gamenerdz.GameMagic,
	"fleshandblood": gamenerdz.GameFleshAndBlood,
	"lorcana":       gamenerdz.GameLorcana,
	"onepiece":      gamenerdz.GameOnePiece,
	"pokemon":       gamenerdz.GamePokemon,
}

func gamenerdzScraper(game string) (mtgban.Scraper, error) {
	tag, ok := gamenerdzGames[game]
	if !ok {
		return nil, fmt.Errorf("gamenerdz does not support %q", game)
	}
	scraper := gamenerdz.NewScraper(tag)
	scraper.LogCallback = GlobalLogCallback
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// miniaturemarketGames translates the canonical game name into Miniature
// Market's own tag, which happens to already be spelled the same way.
var miniaturemarketGames = map[string]string{
	"magic":         miniaturemarket.GameMagic,
	"fleshandblood": miniaturemarket.GameFleshAndBlood,
	"gundam":        miniaturemarket.GameGundam,
	"lorcana":       miniaturemarket.GameLorcana,
	"onepiece":      miniaturemarket.GameOnePiece,
	"riftbound":     miniaturemarket.GameRiftbound,
}

func miniaturemarketSealedScraper(game string) (mtgban.Scraper, error) {
	tag, ok := miniaturemarketGames[game]
	if !ok {
		return nil, fmt.Errorf("miniaturemarket does not support %q", game)
	}
	scraper := miniaturemarket.NewScraperSealed(tag)
	scraper.LogCallback = GlobalLogCallback
	scraper.Affiliate = os.Getenv("MM_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// starcitygamesGames translates the canonical game name into Star City
// Games' own numbering.
var starcitygamesGames = map[string]int{
	"magic":         starcitygames.GameMagic,
	"fleshandblood": starcitygames.GameFleshAndBlood,
	"lorcana":       starcitygames.GameLorcana,
	"riftbound":     starcitygames.GameRiftbound,
}

func starcitygamesScraper(game string) (mtgban.Scraper, error) {
	gameID, ok := starcitygamesGames[game]
	if !ok {
		return nil, fmt.Errorf("starcitygames does not support %q", game)
	}
	apiKey, err := starcitygamesKey()
	if err != nil {
		return nil, err
	}
	scraper := starcitygames.NewScraper(gameID, apiKey)
	scraper.LogCallback = GlobalLogCallback
	scraper.Affiliate = os.Getenv("SCG_PARTNER")
	return scraper, nil
}

func starcitygamesSealedScraper(game string) (mtgban.Scraper, error) {
	gameID, ok := starcitygamesGames[game]
	if !ok {
		return nil, fmt.Errorf("starcitygames does not support %q", game)
	}
	apiKey, err := starcitygamesKey()
	if err != nil {
		return nil, err
	}
	scraper := starcitygames.NewScraperSealed(gameID, apiKey)
	scraper.LogCallback = GlobalLogCallback
	scraper.Affiliate = os.Getenv("SCG_PARTNER")
	return scraper, nil
}

// strikezoneGames translates the canonical game name into Strike Zone's own
// underscored tag.
// strikezone.GameYuGiOh exists but is not in this table: no
// strikezone_yugioh entry has ever been scheduled, and adding a game here
// makes it runnable immediately - that is a decision for its own change,
// not a side effect of this one.
var strikezoneGames = map[string]string{
	"magic":         strikezone.GameMagic,
	"fleshandblood": strikezone.GameFleshAndBlood,
	"lorcana":       strikezone.GameLorcana,
	"pokemon":       strikezone.GamePokemon,
}

func strikezoneScraper(game string) (mtgban.Scraper, error) {
	tag, ok := strikezoneGames[game]
	if !ok {
		return nil, fmt.Errorf("strikezone does not support %q", game)
	}
	scraper := strikezone.NewScraper(tag)
	scraper.LogCallback = GlobalLogCallback
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// tcgIndexScraper builds either of the two clients TCGplayer needs: Magic is
// identified by SKU and reads its own dedicated endpoint, everything else is
// read by category through tcgplayer.SupportedGames, which is what answers
// whether a game is supported at all.
func tcgIndexScraper(game string) (mtgban.Scraper, error) {
	tag, ok := tcgGameTag(game)
	if !ok {
		return nil, fmt.Errorf("tcgplayer does not support %q", game)
	}

	publicID, privateID, err := tcgplayerCredentials()
	if err != nil {
		return nil, err
	}

	if game == "magic" {
		scraper, err := tcgplayer.NewScraperIndex(publicID, privateID)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("TCG_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}

	scraper, err := tcgplayer.NewScraperGameIndex(tag, publicID, privateID)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = GlobalLogCallback
	scraper.Affiliate = os.Getenv("TCG_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// mtgbanGameTags translates the canonical game name into the mtgban.GameX
// constants tcgplayer.SupportedGames and TCGSYPList are keyed by. Magic is
// absent: it takes the empty string in that vocabulary, which tcgGameTag
// answers directly rather than by a table lookup.
var mtgbanGameTags = map[string]string{
	"fleshandblood": mtgban.GameFleshAndBlood,
	"gundam":        mtgban.GameGundam,
	"lorcana":       mtgban.GameLorcana,
	"onepiece":      mtgban.GameOnePiece,
	"palworld":      mtgban.GamePalworld,
	"pokemon":       mtgban.GamePokemon,
	"riftbound":     mtgban.GameRiftbound,
	"yugioh":        mtgban.GameYuGiOh,
}

// tcgGameTag reports whether TCGplayer supports the canonical game name, and
// the mtgban.GameX tag it answers to - mtgban.GameMagic (the empty string)
// for Magic, which is not in mtgbanGameTags because every caller branches on
// it before reaching a table built for the rest.
func tcgGameTag(game string) (tag string, ok bool) {
	if game == "magic" {
		return mtgban.GameMagic, true
	}
	tag, ok = mtgbanGameTags[game]
	return tag, ok
}

func tcgMarketScraper(game string) (mtgban.Scraper, error) {
	tag, ok := tcgGameTag(game)
	if !ok {
		return nil, fmt.Errorf("tcgplayer does not support %q", game)
	}

	publicID, privateID, err := tcgplayerCredentials()
	if err != nil {
		return nil, err
	}

	tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")

	if game == "magic" {
		if tcgSKUPath == "" {
			return nil, errors.New("missing MTGJSON_TCGSKU_PATH env var")
		}
		scraper, err := tcgplayer.NewScraperMarket(publicID, privateID)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("TCG_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}

		start := time.Now()
		skuReader, err := openPath(tcgSKUPath, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
		if err != nil {
			return nil, err
		}
		defer skuReader.Close()
		skus, err := tcgplayer.LoadTCGSKUs(skuReader)
		if err != nil {
			return nil, err
		}
		scraper.SKUsData = skus
		log.Println("loading skus took:", time.Since(start))

		return scraper, nil
	}

	scraper, err := tcgplayer.NewScraperGame(tag, publicID, privateID)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = GlobalLogCallback
	scraper.Affiliate = os.Getenv("TCG_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

func tcgSealedScraper(game string) (mtgban.Scraper, error) {
	tag, ok := tcgGameTag(game)
	if !ok {
		return nil, fmt.Errorf("tcgplayer does not support %q", game)
	}

	publicID, privateID, err := tcgplayerCredentials()
	if err != nil {
		return nil, err
	}

	if game == "magic" {
		tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
		if tcgSKUPath == "" {
			return nil, errors.New("missing MTGJSON_TCGSKU_PATH env var")
		}
		scraper, err := tcgplayer.NewScraperSealed(publicID, privateID)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("TCG_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}

		start := time.Now()
		skuReader, err := openPath(tcgSKUPath, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
		if err != nil {
			return nil, err
		}
		defer skuReader.Close()
		skus, err := tcgplayer.LoadTCGSKUs(skuReader)
		if err != nil {
			return nil, err
		}
		scraper.SKUsData = skus
		log.Println("loading skus took:", time.Since(start))

		return scraper, nil
	}

	scraper, err := tcgplayer.NewScraperGameSealed(tag, publicID, privateID)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = GlobalLogCallback
	scraper.Affiliate = os.Getenv("TCG_PARTNER")
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}

// tcgSYPScraper reads Store Your Products, which is served per category and
// resolved against the catalog rather than an exported sku file, so every
// game the list covers reads the same way. Which games that is is narrower
// than tcgGameTag answers for and lives in tcgplayer.SYPGames, so the check
// stays there: tcgGameTag only translates the name here, and NewScraperSYP
// is what refuses one SYP does not cover.
func tcgSYPScraper(game string) (mtgban.Scraper, error) {
	tag, ok := tcgGameTag(game)
	if !ok {
		return nil, fmt.Errorf("tcgplayer does not support %q", game)
	}

	// NewScraperSYP's own SYPGames check runs on tag alone and does not
	// need auth to be real, so it is called before auth is required to be
	// non-empty: a game SYP does not cover is refused by name before a
	// missing credential ever gets the chance to say something vaguer.
	auth := os.Getenv("TCGPLAYER_AUTH")
	scraper, err := tcgplayer.NewScraperSYP(tag, auth)
	if err != nil {
		return nil, err
	}
	if auth == "" {
		return nil, errors.New("missing TCGPLAYER_AUTH env var")
	}
	catalogPath := os.Getenv("TCGPLAYER_CATALOG_PATH")
	if catalogPath == "" {
		return nil, errors.New("missing TCGPLAYER_CATALOG_PATH env var")
	}

	scraper.LogCallback = GlobalLogCallback
	scraper.Affiliate = os.Getenv("TCG_PARTNER")

	// The list names skus and the datastore names products: the catalog
	// dump published beside each game's datastore is the step between,
	// and is the same file the datastore generators are built from.
	reader, err := openPath(catalogPath, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	catalog, err := tcgplayer.LoadSYPCatalog(reader)
	if err != nil {
		return nil, err
	}
	scraper.Catalog = catalog

	return scraper, nil
}

// vegassinglesGames translates the canonical game name into Vegas Singles'
// own display string.
var vegassinglesGames = map[string]string{
	"magic":     vegassingles.GameMagic,
	"gundam":    vegassingles.GameGundam,
	"onepiece":  vegassingles.GameOnePiece,
	"pokemon":   vegassingles.GamePokemon,
	"riftbound": vegassingles.GameRiftbound,
}

func vegassinglesScraper(game string) (mtgban.Scraper, error) {
	tag, ok := vegassinglesGames[game]
	if !ok {
		return nil, fmt.Errorf("vegassingles does not support %q", game)
	}
	scraper := vegassingles.NewScraper(tag)
	scraper.LogCallback = GlobalLogCallback
	if MaxConcurrency != 0 {
		scraper.MaxConcurrency = MaxConcurrency
	}
	return scraper, nil
}
