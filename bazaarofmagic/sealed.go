package bazaarofmagic

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// Sealed prices Bazaar of Magic's sealed Magic product. The store publishes
// no identifier the datastore carries, so each product is found by the
// words of its name.
type Sealed struct {
	logCallback mtgban.LogCallbackFunc

	backend *matcher.Backend
	client  *http.Client

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

// NewScraperSealed returns a sealed scraper matching against b.
func NewScraperSealed(b *matcher.Backend) *Sealed {
	bom := Sealed{backend: b}
	bom.inventory = mtgban.InventoryRecord{}
	bom.client = newClient()
	return &bom
}

func (bom *Sealed) printf(format string, a ...any) {
	if bom.logCallback != nil {
		bom.logCallback("[BoMSealed] "+format, a...)
	}
}

// sealedPrefixRe matches the brand the newer listings open with.
var sealedPrefixRe = regexp.MustCompile(`^magic:? the gathering\s*-?\s*`)

// sealedCountRe matches a parenthesised count, which says what a box holds
// rather than which box it is: "(36 boosters)", "(incl. 3 play boosters)".
var sealedCountRe = regexp.MustCompile(`\((\d+ (boosters?|cards?|packs?)|incl\.?)[^)]*\)`)

var sealedWordRe = regexp.MustCompile(`[a-z0-9]+`)

// sealedNoise are words one side writes and the other leaves out.
var sealedNoise = map[string]bool{
	"the": true, "of": true, "and": true, "a": true, "an": true,
	"magic": true, "gathering": true, "mtg": true, "x": true,
	"deck": true, "decks": true, "pack": true, "packs": true,
	"series": true, "drop": true, "rainbow": true,
	"universes": true, "beyond": true,
}

// boosterKinds are the words that say which booster a pack is; a listing
// naming none means the set's draft booster.
var boosterKinds = map[string]bool{
	"draft": true, "collector": true, "set": true, "play": true,
	"theme": true, "jumpstart": true, "prerelease": true,
	"commander": true, "planeswalker": true, "intro": true, "event": true,
	"challenger": true, "brawl": true, "welcome": true, "starter": true,
	"scene": true, "beginner": true, "bundle": true, "fat": true,
}

// sealedWords reduces a product name to the words that carry it, in order.
// A count keeps its language, since "(12 boosters - Japanese)" names a
// different product.
func sealedWords(name string) []string {
	name = strings.ToLower(name)
	name = sealedPrefixRe.ReplaceAllString(name, "")
	name = sealedCountRe.ReplaceAllStringFunc(name, func(s string) string {
		if strings.Contains(s, "japanese") {
			return " japanese "
		}
		return " "
	})
	name = strings.ReplaceAll(name, "'", "")

	var out []string
	var prev string
	for _, w := range sealedWordRe.FindAllString(name, -1) {
		skip := sealedNoise[w] ||
			((w == "kit" || w == "kits") && prev == "prerelease") ||
			(w == "edition" && prev == "gift")
		prev = w
		if skip {
			continue
		}
		switch w {
		case "boosterbox":
			out = append(out, "booster", "box")
		case "boosters":
			out = append(out, "booster")
		default:
			out = append(out, w)
		}
	}
	return out
}

func sorted(words []string) []string {
	out := slices.Clone(words)
	slices.Sort(out)
	return out
}

func distinct(words []string) []string {
	return slices.Compact(sorted(words))
}

// sealedIndex finds a sealed product by the words of its name.
type sealedIndex struct {
	ordered  map[string][]string
	words    map[string][]string
	distinct map[string][]string
}

func newSealedIndex(b *matcher.Backend) *sealedIndex {
	index := &sealedIndex{
		ordered:  map[string][]string{},
		words:    map[string][]string{},
		distinct: map[string][]string{},
	}
	for _, uuid := range b.GetSealedUUIDs() {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		words := sealedWords(co.Name)
		key := strings.Join(words, " ")
		index.ordered[key] = append(index.ordered[key], uuid)
		key = strings.Join(sorted(words), " ")
		index.words[key] = append(index.words[key], uuid)
		key = strings.Join(distinct(words), " ")
		index.distinct[key] = append(index.distinct[key], uuid)
	}
	return index
}

// find returns the one product saying the same words as name in the same
// order, then in any order, then counted once, or the draft booster where
// name names a booster of no kind.
func (index *sealedIndex) find(name string) (string, bool) {
	words := sealedWords(name)
	uuids := index.ordered[strings.Join(words, " ")]
	if len(uuids) == 0 {
		uuids = index.words[strings.Join(sorted(words), " ")]
	}
	if len(uuids) == 0 {
		uuids = index.distinct[strings.Join(distinct(words), " ")]
	}
	if len(uuids) == 0 && slices.Contains(words, "booster") && !slices.ContainsFunc(words, func(w string) bool { return boosterKinds[w] }) {
		draft := sorted(append(slices.Clone(words), "draft"))
		uuids = index.words[strings.Join(draft, " ")]
	}
	if len(uuids) != 1 {
		return "", false
	}
	return uuids[0], true
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (bom *Sealed) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}

	rows, err := GetListings(ctx, bom.client, sealedURL, SealedQuery())
	if err != nil {
		return fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, err)
	}
	bom.printf("Found %d in-stock sealed products", len(rows))

	index := newSealedIndex(bom.backend)
	var found int
	for _, row := range rows {
		if row.Price <= 0 {
			continue
		}
		uuid, ok := index.find(row.Title)
		if !ok {
			continue
		}
		found++
		err := bom.inventory.Add(uuid, &mtgban.InventoryEntry{
			Conditions: mtgban.NM,
			Price:      row.Price * rate,
			URL:        row.URL,
			OriginalID: row.ID,
		})
		if err != nil {
			bom.printf("%v", err)
		}
	}
	bom.printf("Matched %d of them", found)

	bom.inventoryDate = time.Now()
	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (bom *Sealed) Inventory() mtgban.InventoryRecord {
	return bom.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (bom *Sealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Bazaar of Magic"
	info.Shorthand = "BoMSealed"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &bom.inventoryDate
	info.SealedMode = true
	info.NoQuantityInventory = true
	info.Game = bom.backend.Game
	return
}
