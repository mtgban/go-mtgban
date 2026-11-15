// Package vegassingles scrapes the Vegas Singles buylist.
package vegassingles

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

const (
	defaultConcurrency = 8
)

// The games this scraper covers, as Hippolink names them.
const (
	GameMagic     = "magic"
	GameRiftbound = "riftbound"
	GameOnePiece  = "onepiece"
	GamePokemon   = "pokemon"
	GameGundam    = "gundam"
)

// vsGames is what NewScraper is built through, and a game named nowhere here
// is not one Vegas Singles is read for.
var vsGames = map[matcher.Game]string{
	matcher.GameMagic:     GameMagic,
	matcher.GameRiftbound: GameRiftbound,
	matcher.GameOnePiece:  GameOnePiece,
	matcher.GamePokemon:   GamePokemon,
	matcher.GameGundam:    GameGundam,
}

// Vegassingles prices what Vegas Singles buys of one game.
type Vegassingles struct {
	logCallback    mtgban.LogCallbackFunc
	maxConcurrency int

	backend *matcher.Backend

	client *VSClient
	game   string

	buylistDate time.Time
	buylist     mtgban.BuylistRecord
}

// NewScraper returns a scraper for the game b was loaded for.
func NewScraper(b *matcher.Backend) (*Vegassingles, error) {
	game, ok := vsGames[b.Game]
	if !ok {
		return nil, fmt.Errorf("unsupported game %q", b.Game)
	}
	vs := Vegassingles{backend: b}
	vs.buylist = mtgban.BuylistRecord{}
	vs.client = NewVSClient()
	vs.game = game
	vs.maxConcurrency = defaultConcurrency
	return &vs, nil
}

func (vs *Vegassingles) printf(format string, a ...any) {
	if vs.logCallback != nil {
		vs.logCallback("[VS] "+format, a...)
	}
}

// resolve names the printing a product is sold as in its finish and in
// language. Magic files its foils and etched foils as printings of their own,
// which the flags reach; every other game names the finish.
func (vs *Vegassingles) resolve(product HLProduct, language string) (string, error) {
	card := matcher.InputCard{
		ID: strconv.FormatInt(product.TCGplayerProductID, 10),
	}
	// TCGplayer calls a sku English on a product that is itself a Japanese
	// printing, so only another language narrows the product
	if language != "English" {
		card.Language = language
	}
	if vs.backend.Game == matcher.GameMagic {
		// The flag, not the finish's spelling, says foil: Hippolink names
		// Magic finishes of its own, such as alternative_foil
		card.Foil = product.IsFoil
		if matcher.FinishSlug(product.Finish) == matcher.FinishEtched {
			card.Variation = "Etched"
		}
	} else {
		card.Finish = product.Finish
	}
	return vs.backend.Match(&card)
}

func (vs *Vegassingles) processProduct(product HLProduct) {
	productID := strconv.FormatInt(product.TCGplayerProductID, 10)

	u, _ := url.Parse("https://buylist.vegas.singles/")
	q := u.Query()
	q.Set("game", vs.game)
	q.Set("q", product.ProductName)
	u.RawQuery = q.Encode()
	link := u.String()

	// Hippolink offers a product's languages as skus of the one product
	cardIDs := map[string]string{}
	for _, offer := range product.Conditions {
		if offer.Cash <= 0 {
			continue
		}

		cardID, found := cardIDs[offer.Language]
		if !found {
			var err error
			cardID, err = vs.resolve(product, offer.Language)
			// The datastore loads few printings in other languages, so a
			// miss there is expected rather than an alarm
			if err != nil && !errors.Is(err, matcher.ErrUnsupported) && offer.Language == "English" {
				vs.printf("%s %q (%s, %s): %v", productID, product.ProductName, product.Finish, offer.Language, err)
			}
			cardIDs[offer.Language] = cardID
		}
		if cardID == "" {
			continue
		}

		cond, err := mtgban.ParseCondition(offer.Condition)
		if err != nil {
			vs.printf("unsupported %s condition", offer.Condition)
			continue
		}

		err = vs.buylist.Add(cardID, &mtgban.BuylistEntry{
			Conditions: cond,
			BuyPrice:   offer.Cash,
			URL:        link,
			OriginalID: productID,
			InstanceID: strconv.FormatInt(offer.TCGplayerSkuID, 10),
		})
		if err != nil {
			vs.printf("%s: %s", productID, err.Error())
		}
	}
}

// scrape reads every page of the game's buylist. The first page says how
// many there are, and failing it fails the run; a later page that will not
// load is logged and skipped.
func (vs *Vegassingles) scrape(ctx context.Context) error {
	first, err := vs.client.GetBuylistPage(ctx, vs.game, 0)
	if err != nil {
		return err
	}
	vs.printf("Buylist holds %d products", first.Total)

	// Step by what the host served rather than what was asked for, so a
	// lower cap on its side cannot skip products
	var offsets []int
	served := len(first.Products)
	for offset := served; served > 0 && offset < first.Total; offset += served {
		offsets = append(offsets, offset)
	}

	products := first.Products
	mtgban.WorkerPool(ctx, vs.maxConcurrency, offsets,
		func(ctx context.Context, offset int, results chan<- []HLProduct) error {
			page, err := vs.client.GetBuylistPage(ctx, vs.game, offset)
			if err != nil {
				return fmt.Errorf("offset %d: %w", offset, err)
			}
			results <- page.Products
			return nil
		},
		func(page []HLProduct) {
			products = append(products, page...)
		},
		vs.printf,
	)

	for _, product := range products {
		vs.processProduct(product)
	}
	vs.buylistDate = time.Now()
	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (vs *Vegassingles) Load(ctx context.Context) error {
	return vs.scrape(ctx)
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (vs *Vegassingles) Buylist() mtgban.BuylistRecord {
	return vs.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (vs *Vegassingles) Info() (info mtgban.ScraperInfo) {
	info.Name = "Vegas Singles"
	info.Shorthand = "VS"
	info.BuylistTimestamp = &vs.buylistDate
	info.Game = vs.backend.Game
	return
}
