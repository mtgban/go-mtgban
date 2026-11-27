// Package stompinggrounds scrapes the singles of Stomping Grounds, which sit
// on a Shopify storefront of their own.
package stompinggrounds

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// StompingGrounds prices the singles Stomping Grounds has in stock. The
// storefront publishes no stock counts. Every product's photo is named after
// the printing's TCGplayer or Scryfall id, which is what it is matched by.
type StompingGrounds struct {
	logCallback mtgban.LogCallbackFunc

	backend *matcher.Backend
	shelf   sgSinglesShelf
	client  *sgClient

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

// sgSinglesShelf is how the storefront files one game's singles: the Shopify
// product type they carry, whether a title opens with the card's name as the
// datastore spells it, which other games' titles do not ("Sink Below (Red)",
// "Latias (Prism Star)"), and whether a finish is asked for with MatchID's
// flags. Those reach Magic's foils numbered apart: the store sells "Shifting
// Sky [Eighth Edition]" plain and foil under the photo of 8ED 100★.
type sgSinglesShelf struct {
	productType string
	namesCard   bool
	finishFlags bool
}

// sgSingles is what NewScraper is built through.
var sgSingles = map[matcher.Game]sgSinglesShelf{
	matcher.GameFleshAndBlood: {productType: "Flesh And Blood Single"},
	matcher.GameLorcana:       {productType: "Lorcana Single"},
	matcher.GameMagic:         {productType: "MTG Single", namesCard: true, finishFlags: true},
	matcher.GameOnePiece:      {productType: "One Piece Single"},
	matcher.GamePokemon:       {productType: "Pokemon Single"},
	matcher.GameRiftbound:     {productType: "Riftbound: League of Legends TCG"},
}

// NewScraper returns a singles scraper for the datastore's game.
func NewScraper(b *matcher.Backend) (*StompingGrounds, error) {
	shelf, found := sgSingles[b.Game]
	if !found {
		return nil, fmt.Errorf("unsupported game %q", b.Game)
	}
	sg := StompingGrounds{
		backend:   b,
		shelf:     shelf,
		client:    newClient(),
		inventory: mtgban.InventoryRecord{},
	}
	return &sg, nil
}

func (sg *StompingGrounds) printf(format string, a ...any) {
	if sg.logCallback != nil {
		sg.logCallback("[SG] "+format, a...)
	}
}

// photoID reads the id a product's photo is named by, "722944.jpg" for
// TCGplayer's or "<scryfall id>_<suffix>.png" for Scryfall's.
var photoID = regexp.MustCompile(`^(\d+|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})[_.]`)

// productID returns the id of the printing a product sells, or "" where its
// photo names none.
func productID(product sfProduct) string {
	if product.FeaturedImage == nil {
		return ""
	}
	link, err := url.Parse(product.FeaturedImage.URL)
	if err != nil {
		return ""
	}
	m := photoID.FindStringSubmatch(path.Base(link.Path))
	if m == nil {
		return ""
	}
	return m[1]
}

// grades are the conditions a variant's title opens with.
var grades = []string{"Near Mint", "Lightly Played", "Moderately Played", "Heavily Played", "Damaged"}

// splitVariant reads a variant's title, a grade and, where a product is sold
// in several printings, the printing: "Lightly Played Holofoil".
func splitVariant(title string) (grade, printing string, ok bool) {
	for _, name := range grades {
		rest, found := strings.CutPrefix(title, name)
		if found && (rest == "" || rest[0] == ' ') {
			return name, strings.TrimSpace(rest), true
		}
	}
	return "", "", false
}

// printings are the names a variant or a title gives a printing, TCGplayer's,
// longest first so that "Rainbow Foil" is read before "Foil".
var printings = func() []string {
	var names []string
	for _, finish := range matcher.Finishes {
		names = append(names, finish.TCGplayer)
	}
	slices.SortStableFunc(names, func(a, b string) int {
		return len(b) - len(a)
	})
	return names
}()

// storePrintings are the store's own names for a printing, which Flesh and
// Blood's catalog spells with the run's "Edition".
var storePrintings = map[string]string{
	"Unlimited Normal":       "Unlimited Edition Normal",
	"Unlimited Rainbow Foil": "Unlimited Edition Rainbow Foil",
}

// etched is a Magic title naming an etched foil, "(Foil Etched)" or
// "- Modern Horizons 2 Etched Foil".
var etched = regexp.MustCompile(`\([^()]*\bEtched\b[^()]*\)| Etched Foil$`)

// finishOf names the printing one variant sells, or reports a variant that
// names something else after its grade, a language or a signature. A
// product sold in one printing names it in its title instead, at the end,
// "Origins Foil", or opening the set, "Crobat 076  - Holofoil ME Mega
// Evolution Promo", or not at all where it is the plain one.
func finishOf(title, printing string) (string, bool) {
	printing = cmp.Or(storePrintings[printing], printing)
	if printing != "" && !slices.Contains(printings, printing) {
		return "", false
	}
	finish := printing
	if finish == "" {
		for _, name := range printings {
			if strings.HasSuffix(title, " "+name) || strings.Contains(title, " - "+name+" ") {
				finish = name
				break
			}
		}
	}
	if etched.MatchString(title) {
		finish = "Etched"
	}
	return cmp.Or(finish, "Normal"), true
}

// matchFinish resolves the printing an id names in a finish, through the
// flags where the shelf asks with them. Elsewhere the store names a print
// run's plain printing the way Flesh and Blood does, "1st Edition Normal",
// where Pokemon's catalog names it by the run alone.
func (sg *StompingGrounds) matchFinish(id, finish string) (string, error) {
	if sg.shelf.finishFlags {
		foil := finish != "Normal"
		cardID, err := sg.backend.MatchID(id, foil, finish == "Etched")
		if err != nil || !foil {
			return cardID, err
		}
		// The flags fall back to a plain foil where no etched one is known,
		// and to a plain printing where no foil is
		co, err := sg.backend.GetUUID(cardID)
		if err == nil && (finish == "Etched" && !co.Etched || !co.Foil && !co.Etched) {
			return "", matcher.ErrCardWrongFinish
		}
		return cardID, err
	}
	cardID, err := sg.backend.MatchIDFinish(id, finish)
	run, found := strings.CutSuffix(finish, " Normal")
	if errors.Is(err, matcher.ErrCardUnnamedFinish) && found {
		return sg.backend.MatchIDFinish(id, run)
	}
	return cardID, err
}

// namesAnother reports whether a title names a card other than the one its
// photo's id landed on, a photo the store reused. A name the datastore does
// not know, a misspelling or a crossover's own title, is left to the id, and
// so is one the landed name extends: "Glimpse, the Unthinkable Playtest", a
// token's pairing.
func (sg *StompingGrounds) namesAnother(title, cardID string) bool {
	name, _, _ := strings.Cut(title, " (")
	name, _, _ = strings.Cut(name, " [")
	name = trailingNumber.ReplaceAllString(name, "")
	uuids, err := sg.backend.SearchEquals(name)
	if err != nil {
		return false
	}
	landed, err := sg.backend.GetUUID(cardID)
	if err != nil || strings.HasPrefix(matcher.Normalize(landed.Name), matcher.Normalize(name)) {
		return false
	}
	for _, uuid := range uuids {
		co, err := sg.backend.GetUUID(uuid)
		if err == nil && co.Name == landed.Name {
			return false
		}
	}
	return true
}

// titleShape reads an older product's title, the card's name, its notes and
// the set's name: "Sanctum Weaver (Extended Art) [Modern Horizons 2]".
var titleShape = regexp.MustCompile(`^(.+?)((?: \([^()]+\))*) \[([^\[\]]+)\]$`)

// skuShape reads the collector number off an older product's SKU,
// "MH2-462-EN-NF-1", or off the front of a two-faced card's,
// "VOW-384//384-EN-NF-1".
var skuShape = regexp.MustCompile(`^[0-9A-Z]+-([0-9A-Za-z]+)(?://[0-9A-Za-z]+)?-EN-(?:NF|FO)-\d+$`)

// matchTitle resolves a product whose photo names no printing the datastore
// carries by its title, numbered by its SKU, and refuses a printing numbered
// otherwise.
func (sg *StompingGrounds) matchTitle(product sfProduct, finish string) (string, error) {
	title := titleShape.FindStringSubmatch(product.Title)
	if title == nil || len(product.Variants.Nodes) == 0 {
		return "", matcher.ErrUnsupported
	}
	sku := skuShape.FindStringSubmatch(product.Variants.Nodes[0].SKU)
	if sku == nil {
		return "", matcher.ErrUnsupported
	}
	in := matcher.InputCard{
		Name:      title[1],
		Edition:   title[3],
		Variation: sku[1],
		Foil:      finish != "Normal",
	}
	if finish == "Etched" {
		in.Finish = matcher.FinishEtched
	}
	cardID, err := sg.backend.Match(&in)
	if err != nil {
		return "", err
	}
	co, err := sg.backend.GetUUID(cardID)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(strings.TrimLeft(co.Number, "0"), strings.TrimLeft(sku[1], "0")) {
		return "", fmt.Errorf("%s landed on %s %s", sku[1], co.SetCode, co.Number)
	}
	return cardID, nil
}

// trailingNumber is a collector number a title gives before its notes,
// "The Destined Black Mage 0447 (Borderless)".
var trailingNumber = regexp.MustCompile(` \d+$`)

// gidNumber is the number closing a Storefront API id,
// "gid://shopify/ProductVariant/45941347942686".
func gidNumber(gid string) string {
	return path.Base(gid)
}

// processProduct files every variant of a product in stock.
func (sg *StompingGrounds) processProduct(product sfProduct) {
	id := productID(product)
	link := singlesURL + "/products/" + product.Handle
	resolved := map[string]string{}
	for _, variant := range product.Variants.Nodes {
		if !variant.AvailableForSale {
			continue
		}
		grade, printing, ok := splitVariant(variant.Title)
		if !ok {
			continue
		}
		finish, ok := finishOf(product.Title, printing)
		if !ok {
			continue
		}
		price, err := strconv.ParseFloat(variant.Price.Amount, 64)
		if err != nil || price <= 0 {
			continue
		}
		cond, err := mtgban.ParseCondition(grade)
		if err != nil {
			sg.printf("unsupported %s condition", grade)
			continue
		}

		cardID, found := resolved[finish]
		if !found {
			cardID, err = sg.matchFinish(id, finish)
			if errors.Is(err, matcher.ErrCardUnknownID) && sg.shelf.namesCard {
				cardID, err = sg.matchTitle(product, finish)
			}
			if err == nil && sg.shelf.namesCard && sg.namesAnother(product.Title, cardID) {
				cardID, err = "", errors.New("photo of another card")
			}
			if err != nil && !errors.Is(err, matcher.ErrCardUnknownID) && !errors.Is(err, matcher.ErrUnsupported) {
				sg.printf("%q %s (%s): %v", product.Title, finish, id, err)
			}
			resolved[finish] = cardID
		}
		if cardID == "" {
			continue
		}

		variantID := gidNumber(variant.ID)
		err = sg.inventory.Add(cardID, &mtgban.InventoryEntry{
			Conditions: cond,
			Price:      price,
			Quantity:   1,
			URL:        link + "?variant=" + variantID,
			OriginalID: gidNumber(product.ID),
			InstanceID: variantID,
		})
		if err != nil {
			sg.printf("%s", err.Error())
		}
	}
}

// walk hands every product of the game's singles in stock to fn, oldest
// first, and returns how many there were; the storefront keeps the ones
// sold out too. Shopify walks no search past searchLimit products, so a walk
// that gets far into one searches again from the creation time it reached,
// skipping the products it already saw.
func (sg *StompingGrounds) walk(ctx context.Context, fn func(sfProduct)) (int, error) {
	seen := map[string]bool{}
	var since string
	for {
		query := "product_type:'" + sg.shelf.productType + "' AND available_for_sale:true"
		if since != "" {
			query += " AND created_at:>='" + since + "'"
		}
		start := since
		var after string
		for walked := 0; walked < searchWindow; {
			page, err := sg.client.search(ctx, query, after)
			if err != nil {
				return len(seen), err
			}
			for _, product := range page.Nodes {
				since = product.CreatedAt
				if seen[product.ID] {
					continue
				}
				seen[product.ID] = true
				fn(product)
			}
			if !page.PageInfo.HasNextPage {
				return len(seen), nil
			}
			walked += len(page.Nodes)
			after = page.PageInfo.EndCursor
		}
		if since == start {
			return len(seen), fmt.Errorf("over %d products created at %s", searchWindow, since)
		}
	}
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (sg *StompingGrounds) Load(ctx context.Context) error {
	n, err := sg.walk(ctx, sg.processProduct)
	if err == nil && n == 0 {
		err = errors.New("no " + sg.shelf.productType + " on the storefront")
	}
	if err != nil {
		return fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, err)
	}
	sg.printf("Priced %d printings of %d products", len(sg.inventory), n)
	sg.inventoryDate = time.Now()
	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (sg *StompingGrounds) Inventory() mtgban.InventoryRecord {
	return sg.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (sg *StompingGrounds) Info() (info mtgban.ScraperInfo) {
	info.Name = "Stomping Grounds"
	info.Shorthand = "SG"
	info.InventoryTimestamp = &sg.inventoryDate
	info.NoQuantityInventory = true
	info.Game = sg.backend.Game
	return
}
