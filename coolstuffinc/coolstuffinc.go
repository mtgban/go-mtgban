// Package coolstuffinc scrapes Cool Stuff Inc, for singles and sealed
// product.
package coolstuffinc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

const (
	defaultConcurrency = 8

	csiInventoryURL = "https://www.coolstuffinc.com/sq/?s="
)

// The games this scraper covers, as the storefront names them.
const (
	GameMagic             = "mtg"
	GameLorcana           = "lorcana"
	GameRiftbound         = "riftbound"
	GameYuGiOh            = "yugioh"
	GameDragonBallSuper   = "dbs"
	GameOnePiece          = "optcg"
	GameStarWarsUnlimited = "swu"
	GamePokemon           = "pokemon"
	GameGundam            = "gundam"
	GamePalworld          = "palworld"
)

// csiGames is what the two constructors are built through: it names the
// shelf a game is searched under, and a game named nowhere here is not one
// Cool Stuff Inc is read for. The shelves themselves stay public - Search,
// GetBuylist and LoadBuylistEditions each ask for one by name.
var csiGames = map[mtgban.Game]string{
	mtgban.GameMagic:     GameMagic,
	mtgban.GameLorcana:   GameLorcana,
	mtgban.GameRiftbound: GameRiftbound,
	mtgban.GameYuGiOh:    GameYuGiOh,
	mtgban.GameOnePiece:  GameOnePiece,
	mtgban.GamePokemon:   GamePokemon,
	mtgban.GameGundam:    GameGundam,
	mtgban.GamePalworld:  GamePalworld,
}

var deductions = []float64{1, 1, 0.75}

var availableMarketNames = []string{
	"Cool Stuff Inc", "Cool Stuff Inc (unique)",
}

var name2shorthand = map[string]string{
	"Cool Stuff Inc":          "CSI",
	"Cool Stuff Inc (unique)": "CSIUnique",
}

// Coolstuffinc prices Cool Stuff Inc's singles, both what they sell and what
// they buy.
type Coolstuffinc struct {
	logCallback mtgban.LogCallbackFunc
	partner     string

	// If set to true scrape will include all entries without a nonfoil NM price
	// but will be almost twice as slow
	includeOOS bool

	inventoryDate  time.Time
	buylistDate    time.Time
	maxConcurrency int

	targetEdition string

	disableRetail  bool
	disableBuylist bool

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client  *http.Client
	game    mtgban.Game
	shelf   string
	backend *mtgmatcher.Backend
}

// pokemonNonHolo matches the bracket a Pokemon name states a plain printing
// with. The storefront sells a card printed in both finishes as two products
// telling them apart by that bracket alone - the note is empty and the foil
// flag is off on both - and read as the holo, a $2.00 Team Aqua's Kyogre was
// served as the $80.00 one's price. The bracket's own case varies
// ("(NON-HOLO)" on Black & White prints), so the match has to as well.
//
// The rarity is what says whether the plain printing was ever made. A holo
// rare is sold holo and nothing else, so a bracket asking for its plain
// printing asks for one that does not exist and the row is refused. A plain
// rare is the opposite: the catalog holding no nonfoil for it is the catalog
// missing a printing rather than the storefront inventing one, and refusing
// those would drop 25 real listings to catch nothing.
var pokemonNonHolo = regexp.MustCompile(`(?i)\(Non-?\s?Holo\)`)

// pokemonNonHoloDeckExclusive answers whether the catalog's Deck Exclusives
// shelf carries the plain printing a "(Non-Holo)" bracket asks for, taken
// only when the probe lands on PR-1840's own nonfoil at the listing's own
// number.
func pokemonNonHoloDeckExclusive(b *mtgmatcher.Backend, name, numbered string, foil bool) bool {
	num := mtgmatcher.ExtractNumber(numbered)
	if num == "" {
		return false
	}
	id, err := b.Match(&mtgmatcher.InputCard{Name: name + " - " + numbered, Edition: "Deck Exclusives", Foil: foil})
	if err != nil {
		return false
	}
	co, err := b.GetUUID(id)
	return err == nil && co.SetCode == "PR-1840" && co.Finish == mtgmatcher.FinishNonfoil &&
		strings.TrimLeft(co.Number, "0") == num
}

// nameParenthetical matches a qualifier a buylist name carries in brackets,
// like "(Parallel)" or "(Alternate Art)".
var nameParenthetical = regexp.MustCompile(`\(([^)]+)\)`)

// nameQualifiers answers the wording a One Piece buylist name carries behind
// the card's own, which is where that feed spends the qualifier telling one
// printing from another - the note beside it describes the artwork instead
// ("Hand Blocking Sun", "Leg Gun"), and the matcher reads neither the name
// nor a description. A number-shaped bracket is left behind: the feed repeats
// the collector number there and it says nothing the number field has not.
func nameQualifiers(name string) string {
	var words []string
	for _, match := range nameParenthetical.FindAllStringSubmatch(name, -1) {
		qualifier := strings.TrimSpace(match[1])
		if qualifier == "" || buylistNumberWord.MatchString(qualifier) {
			continue
		}
		words = append(words, qualifier)
	}
	return strings.Join(words, " ")
}

// buylistNumberWord matches the number-shaped words of a buylist note.
var buylistNumberWord = regexp.MustCompile(`(?i)^[A-Z]{0,4}\d+[a-z]?(?:/[A-Z]{0,4}\d+)?[,.]?$`)

var buylistReprintNote = regexp.MustCompile(`(?i)\breprints?\b`)

// jpArtWordingRe matches "Japanese Art"/"Artwork"/"Art Style", which
// mtgmatcher's language filter otherwise reads as a request for a
// Japanese-language card. Leaves "Japanese Letters in Art" alone.
var jpArtWordingRe = regexp.MustCompile(`(?i)Japanese Art(?:work|\s+Style)?`)

// jpArtWording rewrites the wording jpArtWordingRe matches to "JP Art", the
// catalog's own name for the treatment.
func jpArtWording(s string) string {
	return jpArtWordingRe.ReplaceAllString(s, "JP Art")
}

// onePieceEvents spells a One Piece event the way the catalog names it, for
// the names this storefront gives it instead. They are its own: the catalog
// sells the card in "BANDAI Card Games Fest 25-26" and it goes up here as
// the Afro Luffy promo, after the art rather than the pack. A nickname
// names one product and nothing else, which is why they are listed one at a
// time rather than read for.
var onePieceEvents = map[string]string{
	"afro luffy promo":   "BANDAI Card Games Fest 25-26",
	"l.a. dodgers promo": "Dodgers x One Piece",
	// The playmat and the participation pack name the product the card came
	// in, where the catalog names the event that handed it out.
	"bcgf playmat promo":                             "Official Playmat -Bandai Card Games Fest 24-25 Edition-",
	"offline regional participation pack 2024 vol.2": "Offline Regional 2024 Vol. 2 Participant",
}

// onePieceStarterDeck matches the starter deck a name states in brackets.
var onePieceStarterDeck = regexp.MustCompile(`\(Starter Deck (\d+)\)`)

// onePieceSpellings spells the One Piece names this storefront writes its
// own way: the Heroines Edition event card lost a word.
var onePieceSpellings = map[string]string{
	"But If We See Each Other Again...Will You Call Me Your Shipmate?!!": "But If We Ever See Each Other Again... Will You Call Me Your Shipmate?!!",
}

// onePieceSpelling spells a One Piece name the way the catalog does.
func onePieceSpelling(name string) string {
	if spelled, found := onePieceSpellings[name]; found {
		return spelled
	}
	return name
}

// onePieceShelf answers the set a One Piece listing belongs to, which is the
// shelf it arrived on except where that shelf says only "Promo".
//
// A starter deck card reprinted as a promo is filed here under the promo
// shelf with the deck named in brackets, and the promo shelf holds a printing
// of its own at the same number: the P-041 Luffy is both the plain promo and
// the Starter Deck 18 card. The two met, and a $0.50 deck card was priced as
// the $60.00 promo standing beside it.
//
// The bracket only decides where the shelf has nothing to say. Every other
// listing naming a deck arrives on a real set already - a Backlight on ST11,
// a Kuzan on OP12 - and there the shelf is what the catalog agrees with,
// while the bracket names the deck the card was reprinted from.
func onePieceShelf(shelf, name string) string {
	if shelf != "Promo" {
		return shelf
	}
	match := onePieceStarterDeck.FindStringSubmatch(name)
	if match == nil {
		return shelf
	}
	return "Starter Deck " + match[1]
}

// onePieceRenamedTreatment answers the printing a One Piece listing means
// when it names its treatment with a word the catalog does not use for that
// set, and "" wherever the listing is already answered.
//
// This storefront calls the Gear5 starter deck's premium printing "Full Art"
// where the catalog files every alternate printing of that set as "Parallel",
// so the word named no label there and the row settled on the plain card - a
// $15.00 Monkey.D.Luffy priced as the $0.50 one beside it.
//
// The guard is what keeps it from touching a real Full Art. The word must
// name a label the catalog uses somewhere, so a typo reaches nothing; the
// card's own set must hold no printing of it, which is false for all 75 real
// Full Art printings, since their sets are the ones that use the name; and
// the set must wear a single premium label throughout, so the one printing
// the storefront can mean is the one the number carries.
func onePieceRenamedTreatment(b *mtgmatcher.Backend, id, name string) string {
	co, err := b.GetUUID(id)
	if err != nil || len(co.PromoTypes) > 0 {
		return ""
	}
	set, err := b.GetSet(co.SetCode)
	if err != nil {
		return ""
	}

	labels := map[string]bool{}
	var alternate string
	for _, card := range set.Cards {
		for _, promoType := range card.PromoTypes {
			labels[promoType] = true
		}
		if card.Number == co.Number && len(card.PromoTypes) > 0 {
			if alternate != "" && alternate != card.UUID {
				return ""
			}
			alternate = card.UUID
		}
	}
	if len(labels) != 1 || alternate == "" {
		return ""
	}

	for _, match := range nameParenthetical.FindAllStringSubmatch(name, -1) {
		slug := mtgmatcher.PromoTypeSlug(strings.TrimSpace(match[1]))
		if slug == "" || labels[slug] || !slices.Contains(b.AllPromoTypes, slug) {
			continue
		}
		return alternate
	}
	return ""
}

// eventNamed adds the catalog's name for every event the wording gives its
// own name to. The storefront's words stay: they are what the listing says
// about the art, and the catalog's name is only what files it.
func eventNamed(wording string) string {
	lower := strings.ToLower(wording)
	for nickname, event := range onePieceEvents {
		if strings.Contains(lower, nickname) {
			wording += " " + event
		}
	}
	return wording
}

// buylistVariation builds everything the buylist knows about a printing
// beyond its name. The feed spends a free-text note on the qualifier that
// the sell listing spends its variation on - "Love Ball Foil", "Detective
// Pikachu Stamped" - and reading only the number field asked the two sides
// of one product different questions.
//
// The note's own numbers stay behind. They name other products the buyer
// might have meant ("Can be Pikachu 2, 19, 41, or 45"), while the collector
// number arrives in its own field and again in the product name. A note
// that says a printing reprints another stays behind whole, because every
// word after the number is then the name of the set being reprinted rather
// than of this one.
func buylistVariation(product CSIPriceEntry) string {
	if buylistReprintNote.MatchString(product.Notes) {
		return product.Number
	}
	words := []string{product.Number}
	for word := range strings.FieldsSeq(product.Notes) {
		if buylistNumberWord.MatchString(word) {
			continue
		}
		words = append(words, word)
	}
	return strings.TrimSpace(strings.Join(words, " "))
}

// NewScraper returns a singles scraper for the datastore's game.
func NewScraper(b *mtgmatcher.Backend) (*Coolstuffinc, error) {
	game, err := mtgban.GameOf(b)
	if err != nil {
		return nil, err
	}
	shelf, ok := csiGames[game]
	if !ok {
		return nil, fmt.Errorf("unsupported game %q", game)
	}
	csi := Coolstuffinc{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	csi.client = newCSIHTTPClient()
	csi.maxConcurrency = defaultConcurrency
	csi.game = game
	csi.shelf = shelf
	csi.backend = b
	return &csi, nil
}

type responseChan struct {
	cardID   string
	invEntry *mtgban.InventoryEntry
	relaxed  bool
}

func (csi *Coolstuffinc) printf(format string, a ...any) {
	if csi.logCallback != nil {
		csi.logCallback("[CSI] "+format, a...)
	}
}

// saleTail is what the price column writes in front of a discounted price.
// The space is a non-breaking one, as the page spells it.
const saleTail = "Was\u00a0"

// offerCondition reads the condition out of one offer row's text. The row
// spells the quantity, the condition, whatever promotions it is running and
// the price, and only the condition is wanted.
//
// The promotions are tails on the condition, and a row may run several at
// once: the bundle flag sits in the condition column and the sale's "Was" in
// the price one, so cutting a single tail left the other row's flag glued to
// the condition and the whole offer was dropped as unparseable. Cutting
// until nothing more comes off reads them in whatever order the page lays
// them out, and a row running none is unchanged by the first pass.
func offerCondition(fullRow, qtyStr, bundleStr string) string {
	// The count is a prefix, not a set of characters to eat: trimming it as
	// a cutset ate the leading digit of a condition that opens with one,
	// which is how "1st Edition" reached the log as "st Edition".
	conditions := strings.TrimPrefix(fullRow, qtyStr)
	conditions = strings.TrimPrefix(conditions, "+")
	conditions = strings.TrimLeft(conditions, " ")
	conditions = strings.Split(conditions, "$")[0]
	for {
		trimmed := strings.TrimSuffix(conditions, bundleStr)
		trimmed = strings.TrimSuffix(trimmed, saleTail)
		if trimmed == conditions {
			return conditions
		}
		conditions = trimmed
	}
}

// firstEditionShelf reads the print run a Pokemon shelf names in its own
// title. The storefront files the first-edition run of a set as a shelf
// beside the set - "1st Edition Fossil" next to "Fossil" - where the catalog
// files the run as a finish of the set itself, so the shelf name has to
// become one. Left as it was, every listing on those shelves matched the
// unlimited printing and was published at a fraction of its price, and
// nothing said so: the match succeeded, it just answered with the other run.
func firstEditionShelf(edition string) (string, []string) {
	trimmed := strings.TrimPrefix(edition, "1st Edition ")
	if trimmed == edition {
		return edition, nil
	}
	// Base Set is the one shelf whose run the catalog files as a set of its
	// own rather than a finish of the set beside it: the first-edition and
	// shadowless printings are "Base Set (Shadowless)", and "Base Set" holds
	// only the shadowed unlimited run. Naming the shelf's own set asks for a
	// run that set has one card of, so every other row fell through to the
	// unlimited printing and was published at the first edition's price.
	if set, found := runShelfEditions[trimmed]; found {
		trimmed = set
	}
	return trimmed, conditionRuns["1st Edition"]
}

// runShelfEditions name the set a print-run shelf sells where the catalog
// files the run apart from the shelf's own set. Fossil, Jungle, Team Rocket,
// the Gyms and the Neos all carry theirs as a finish and are left alone.
var runShelfEditions = map[string]string{
	"Base Set": "Base Set (Shadowless)",
}

// conditionRuns are the print runs this storefront names where a condition
// would go. The run is a finish in the game's own vocabulary rather than a
// state of the card, and the spelling has to be exact: asking for the plain
// run of a card printed in holo answers with the holo of the other run - the
// unlimited Lapras where the first-edition one was listed, at a fraction of
// its price - so both spellings are tried and the answer is checked for
// carrying the run before it is taken.
var conditionRuns = map[string][]string{
	"1st Edition": {"1st Edition Holofoil", "1st Edition"},
}

// conditionRun reads the run a row names in its condition column, giving the
// finishes it may be sold as, widest first.
func conditionRun(conditions string) []string {
	for marker, finishes := range conditionRuns {
		if strings.Contains(conditions, marker) {
			return finishes
		}
	}
	return nil
}

// matchRun resolves a listing sold as one of a card's print runs, refusing
// an answer that does not carry it rather than pricing the run as the
// ordinary printing.
func matchRun(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, finishes []string) (string, error) {
	var err error
	for _, finish := range finishes {
		probe := *inCard
		probe.Finish = finish
		var cardID string
		cardID, err = b.Match(&probe)
		if err != nil {
			continue
		}
		co, uerr := b.GetUUID(cardID)
		if uerr != nil {
			continue
		}
		if co.Finish == mtgmatcher.Normalize(finish) {
			return cardID, nil
		}
	}
	if err == nil {
		err = mtgmatcher.ErrUnsupported
	}
	return "", err
}

// skippedConditions name a copy the catalog has no row for, so there is
// nothing to price it as and nothing to report either. Asian names no
// language in particular, and the only Evolving Wilds sold under it - the
// one listing that carries the wording - is a name with no set, number or
// language beside it, against seventy-seven printings that are all English.
var skippedConditions = []string{"Asian"}

// isSkippedCondition reports whether the wording names one of those.
func isSkippedCondition(conditions string) bool {
	for _, skipped := range skippedConditions {
		if strings.Contains(conditions, skipped) {
			return true
		}
	}
	return false
}

// conditionPrintings are the printings this storefront sells as an offer of
// their own, naming them where a condition would go. The run is a real
// printing rather than a state of the card, so the wording moves to the
// variation and lets the matcher pick it, and the offer is priced as the
// stock it is.
var conditionPrintings = map[string]string{
	"PRE-ERRATA": "Pre-Errata",
}

// conditionPrinting reads the printing a row names in its condition column,
// or "" where the wording names a condition.
func conditionPrinting(conditions string) string {
	for marker, variation := range conditionPrintings {
		if strings.Contains(conditions, marker) {
			return variation
		}
	}
	return ""
}

// gradedMarkers are the wordings a row carries when the copy is not being
// sold at a condition tier: a slab named for the service that graded it, the
// one-off the storefront files as unique, or the printing a single copy is
// when it is sold on another printing's product - the shadowless Venusaur
// offered on the unlimited Venusaur's page, at three times what the copy
// beside it costs. None of those is a condition and none of their prices is
// the card's, so such a row is published as its own seller rather than
// beside the ordinary copies.
var gradedMarkers = []string{"BGS", "PSA", "CGC", "TAG", "Non-Foil", "Unique", "Shadowless", "No Set Symbol"}

// isGraded reports whether the condition wording names one of those rather
// than a condition. A wording it does not know is refused by the condition
// parser, which says so, so a new grading service costs a line here.
func isGraded(conditions string) bool {
	for _, marker := range gradedMarkers {
		if strings.Contains(conditions, marker) {
			return true
		}
	}
	return false
}

// bundleRe matches the wording of the bundle promotion, whatever count it
// gives away: "Buy 1 get 3 free!" sells four copies for the listed price.
var bundleRe = regexp.MustCompile(`^Buy 1 get (\d+) free!$`)

// bundledCopies reads how many copies the listed price buys: one, unless the
// row runs the bundle promotion, whose price covers the bought copy and the
// free ones together. The wording carries the count, so a promotion the
// condition parser learned to cut is also the one the price is divided by,
// rather than only the one spelling the exact count the flag used to name.
//
// The price is the only thing the promotion changes. The count beside it is
// the same card-qty column every row on the page carries, promotion or not,
// capped at "20+" the way a stock figure is, so it goes out as the stock it
// reads as rather than divided to match the price.
func bundledCopies(bundleStr string) int {
	match := bundleRe.FindStringSubmatch(bundleStr)
	if match == nil {
		return 1
	}
	free, err := strconv.Atoi(match[1])
	if err != nil {
		return 1
	}
	return 1 + free
}

func (csi *Coolstuffinc) processSearch(ctx context.Context, results chan<- responseChan, itemName string, rarities []string) error {
	skipOOS := !csi.includeOOS
	switch itemName {
	case "Alpha", "Beta", "Unlimited Edition":
		skipOOS = false
	}
	result, err := Search(ctx, csi.shelf, itemName, skipOOS, rarities)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(result.Data))
	if err != nil {
		return fmt.Errorf("page 1: %w", err)
	}

	// An empty next link is the last page, and the only thing that says
	// so: a search whose rows divide evenly into pages ends on a full one
	// and asking past the end answers that page over again.
	next := result.NextLink

	// A shelf that ends on its first page is already read; one that does
	// not is read again from the top at the larger page size.
	if next != "" {
		wide, wideNext := widenSearchPage(ctx, csi.client, next)
		if wide != nil {
			doc = wide
			next = wideNext
		}
	}

	for page := 1; ; page++ {
		doc.Find(searchRowSelector).Each(func(i int, s *goquery.Selection) {
			// The storefront escapes its names twice, so the decode the
			// parser already did leaves the entity still written out:
			// "Fiendish Engine &#937;" is the Omega the datastore spells.
			cardName := html.UnescapeString(s.Find(`span[itemprop="name"]`).Text())

			pid, _ := s.Find(`span[class="rating-display "]`).Attr("data-pid")
			edition := itemName
			notes := s.Find(`div[class="large-8 medium-12 small- 12 product-notes"]`).Text()
			notes = strings.TrimPrefix(notes, "Notes: ")

			// The storefront prints the rarity in the row under the
			// breadcrumb, and for Yu-Gi-Oh it is the only thing that
			// tells apart the printings a set files at one number: the
			// Battle Packs sell the same card as Common and as Mosaic
			// Rare, both numbered alike.
			rarity := strings.TrimSpace(s.Find(`div[class="breadcrumb-trail"]`).Parent().Parent().Next().Text())

			imgURL, _ := s.Find(`a[class="productLink"]`).Find("img").Attr("data-src")
			if imgURL == "" {
				imgURL, _ = s.Find(`a[class="productLink"]`).Find("img").Attr("src")
				if imgURL == "" {
					csi.printf("img not found %s %s", cardName, edition)
				}
			}

			s.Find(`div[itemprop="offers"]`).Each(func(i int, se *goquery.Selection) {
				var relaxed bool
				var graded bool
				fullRow := strings.TrimSpace(se.Text())

				switch {
				case strings.Contains(fullRow, "Out of Stock"),
					strings.Contains(fullRow, "not currently available"):
					return
				}

				qtyStr := se.Find(`span[class="card-qty"]`).Text()
				qtyStr = strings.TrimSpace(strings.TrimSuffix(qtyStr, "+"))
				// If preorder has no quantity,, set max allowed
				if qtyStr == "" && strings.Contains(notes, "Preorder") {
					qtyStr = "20"
				}

				qty, err := strconv.Atoi(qtyStr)
				if err != nil {
					csi.printf("%s", fullRow)
					csi.printf("%s %s %v", cardName, edition, err)
					return
				}

				bundleStr := se.Find(`div[class="b1-gx-free"]`).Text()
				bundleCopies := bundledCopies(bundleStr)

				conditions := offerCondition(fullRow, qtyStr, bundleStr)

				isFoil := strings.HasPrefix(conditions, "Foil")

				if isGraded(conditions) {
					conditions = "Near Mint"
					graded = true
				}

				printing := conditionPrinting(conditions)
				if printing != "" {
					conditions = "Near Mint"
				}

				runFinishes := conditionRun(conditions)
				if runFinishes != nil {
					conditions = "Near Mint"
				}

				if isSkippedCondition(conditions) {
					return
				}

				// Sometimes etched cards have a Near Mint and Near Mint Foil condition
				// for the same card
				if strings.Contains(cardName, "Foil-etched") {
					relaxed = true
				}

				matchCond := strings.TrimPrefix(conditions, "Foil ")

				var grade mtgban.Condition
				switch matchCond {
				case "Played":
					grade = mtgban.MP
				default:
					parsed, err := mtgban.ParseCondition(matchCond)
					if err != nil {
						csi.printf("Unsupported '%s' condition for %s", conditions, cardName)
						return
					}
					grade = parsed
				}
				if strings.Contains(cardName, "Signed by") {
					grade = mtgban.HP
				}

				priceStr := se.Find(`b[itemprop="price"]`).Text()
				price, err := strconv.ParseFloat(priceStr, 64)
				if err != nil {
					csi.printf("%v", err)
					return
				}
				if bundleCopies > 1 {
					price /= float64(bundleCopies)
				}

				if price == 0.0 || qty == 0 {
					return
				}

				link := "https://www.coolstuffinc.com/p/" + pid
				if csi.partner != "" {
					link += "?utm_referrer=" + csi.partner
				}

				var theCard *mtgmatcher.InputCard
				switch csi.game {
				case mtgban.GameMagic:
					c, err := preprocess(csi.backend, cardName, edition, notes, imgURL)
					if err != nil {
						return
					}
					// preprocess() might return something that derived foil status
					// from one of the fields (cardName in particular)
					c.Foil = c.Foil || isFoil
					theCard = c
				case mtgban.GameYuGiOh:
					if unknownPrinting(cardName, edition) {
						return
					}
					theCard = &mtgmatcher.InputCard{Name: catalogColor(catalogSpelling(jpArtWording(cardName))), Edition: printRunEdition(edition, notes), Variation: strings.TrimSpace(jpArtWording(notes) + " " + catalogRarity(rarity)), Foil: isFoil}
				case mtgban.GamePokemon:
					shelf, shelfRun := firstEditionShelf(edition)
					if shelfRun != nil {
						runFinishes = shelfRun
					}
					variation := catalogTreatment(notes)
					shelf = pokemonPromoShelf(csi.backend, cardName, shelf, rarity, isFoil, variation)
					theCard = pokemonListing(csi.backend, cardName, shelf, variation, isFoil)
				case mtgban.GameOnePiece:
					shelf := onePieceShelf(edition, cardName)
					donName, donDescription, isDon := onePieceDonName(cardName)
					if isDon {
						theCard = &mtgmatcher.InputCard{Name: donName, Edition: shelf, Variation: donDescription, Foil: isFoil}
						_, err := csi.backend.Match(theCard)
						if err != nil {
							renamed := onePieceDonRenamed(donDescription)
							if renamed != "" {
								theCard = &mtgmatcher.InputCard{Name: donName, Edition: shelf, Variation: renamed, Foil: isFoil}
							}
						}
						break
					}
					theCard = &mtgmatcher.InputCard{Name: onePieceSpelling(cardName), Edition: shelf, Variation: eventNamed(jpArtWording(notes)), Foil: isFoil}
				case mtgban.GameGundam:
					name, variation := gundamCard(csi.backend, cardName, gundamNumber(notes))
					theCard = &mtgmatcher.InputCard{Name: name, Edition: gundamShelf(edition), Variation: strings.TrimSpace(variation + " " + notes + " " + gundamTier(rarity)), Foil: isFoil}
				// Palworld numbers a parallel apart from the card it
				// parallels, so the note names one printing on its own -
				// once the rarity code this storefront sometimes types onto
				// the number is written back beside it.
				case mtgban.GamePalworld:
					theCard = &mtgmatcher.InputCard{Name: palworldName(cardName), Edition: edition, Variation: palworldNotes(notes), Foil: isFoil}
				case mtgban.GameRiftbound:
					shelf := riftboundShelf(csi.backend, edition, notes, cardName, notes, isFoil)
					theCard = &mtgmatcher.InputCard{Name: cardName, Edition: shelf, Variation: notes, Foil: isFoil}
					_, err := csi.backend.Match(theCard)
					if err != nil {
						fromImage := riftboundImageCard(csi.backend, imgURL, isFoil)
						if fromImage != nil {
							theCard = fromImage
						}
					}
				case mtgban.GameLorcana:
					theCard = &mtgmatcher.InputCard{Name: lorcanaSpelling(cardName), Edition: edition, Variation: lorcanaVariation(notes), Foil: isFoil}
				}

				if printing != "" {
					theCard.Variation = strings.TrimSpace(theCard.Variation + " " + printing)
				}

				var cardID string
				if runFinishes != nil {
					cardID, err = matchRun(csi.backend, theCard, runFinishes)
				} else {
					cardID, err = csi.backend.Match(theCard)
				}
				if errors.Is(err, mtgmatcher.ErrUnsupported) {
					return
				} else if err != nil {
					switch {
					// Ignore expected misses
					case magic.IsBasicLand(theCard.Name),
						notes == "" && strings.Contains(edition, "The List"),
						strings.Contains(notes, "Preorder"):
					default:
						csi.printf("%v", err)
						csi.printf("%v", theCard)
						csi.printf("'%s' '%s' '%s'", cardName, edition, notes)
						csi.printf("- %s", link)

						var alias *mtgmatcher.AliasingError
						if errors.As(err, &alias) {
							for _, probe := range alias.Probe() {
								card, _ := csi.backend.GetUUID(probe)
								csi.printf("- %s", card)
							}
						}
					}
					return
				}

				// Magic-only finish sanity check: skip cards that do not have the
				// requested finish.
				if csi.game == mtgban.GameMagic {
					if strings.Contains(cardName, "Foil-etched") {
						co, err := csi.backend.GetUUID(cardID)
						if err != nil || !co.Etched {
							return
						}
					}
					if isFoil {
						co, err := csi.backend.GetUUID(cardID)
						if err != nil || (!co.Etched && !co.Foil) {
							return
						}
					}
				}

				out := responseChan{
					cardID: cardID,
					invEntry: &mtgban.InventoryEntry{
						Conditions: grade,
						Price:      price,
						Quantity:   qty,
						URL:        link,
						OriginalID: pid,
						SellerName: availableMarketNames[0],
					},
					relaxed: relaxed || graded,
				}

				if graded {
					out.invEntry.SellerName = availableMarketNames[1]
				}
				results <- out
			})
		})

		if next == "" {
			break
		}

		doc, err = fetchSearchPage(ctx, csi.client, next)
		if err != nil {
			return fmt.Errorf("page %d: %w", page+1, err)
		}
		next = searchNextLink(doc)
	}

	return nil
}

func (csi *Coolstuffinc) scrape(ctx context.Context) error {
	link := csiInventoryURL + csi.shelf
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := csi.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	var itemNames []string
	var rarities []string
	doc.Find(`fieldset`).Each(func(i int, s *goquery.Selection) {
		title := s.Find(`h2[class="mb10"] b`).Text()
		switch title {
		case "Item Set":
			s.Find(`div[class="toggleTable"]`).Find("li").Each(func(j int, se *goquery.Selection) {
				itemName, _ := se.Find(`input[type="checkbox"]`).Attr("value")
				switch {
				case strings.Contains(itemName, "Bulk"),
					strings.Contains(itemName, "Random Lots"),
					strings.Contains(itemName, "Relic Token"),
					itemName == "Magic":
					return
				}

				itemNames = append(itemNames, itemName)
			})
		case "Rarity":
			// Reading the tiers off the page the editions already come
			// from spares the search a list of its own, which is how
			// every game but Magic came to be asked for Magic's tiers
			rarities = singlesRarities(s)
		}
	})
	// Sort for predictable results
	sort.Strings(itemNames)

	csi.printf("Found %d items over %d rarity tiers", len(itemNames), len(rarities))

	start := time.Now()

	if csi.targetEdition != "" {
		filtered := itemNames[:0]
		for _, item := range itemNames {
			if item == csi.targetEdition {
				filtered = append(filtered, item)
			}
		}
		itemNames = filtered
	}

	// The search answers a product on every shelf it is filed on, and a
	// Heroines Edition card sits on two, so the same offer arrives twice
	// with one url, one condition and one quantity. The second is the
	// first again, not a second seller, and adding it would only be
	// refused as the duplicate it is.
	seen := map[string]bool{}
	mtgban.WorkerPool(ctx, csi.maxConcurrency, itemNames,
		func(ctx context.Context, itemName string, results chan<- responseChan) error {
			csi.printf("Processing %s", itemName)
			return csi.processSearch(ctx, results, itemName, rarities)
		},
		func(record responseChan) {
			if offerSeen(seen, record) {
				return
			}
			var err error
			if record.relaxed {
				err = csi.inventory.AddRelaxed(record.cardID, record.invEntry)
			} else {
				err = csi.inventory.Add(record.cardID, record.invEntry)
			}
			// The search lists a card once per shelf it sits on, so the
			// same listing arrives twice; the second is the same entry.
			if err != nil && !strings.Contains(err.Error(), "same url, and qty") {
				csi.printf("%s", err.Error())
			}
		},
		csi.printf,
	)

	csi.printf("This operation took %v", time.Since(start))

	csi.inventoryDate = time.Now()

	return nil
}

// offerSeen reports whether this very offer was already collected - the
// same printing at the same condition from the same seller behind one url -
// and records it otherwise. The url and the condition alone do not tell
// offers apart: a product row lists its foil, its graded copies and its
// first edition as further offers of the same url, and every one of them
// is filed at NM.
func offerSeen(seen map[string]bool, record responseChan) bool {
	entry := record.invEntry
	key := strings.Join([]string{entry.URL, record.cardID, string(entry.Conditions), entry.SellerName}, "\x00")
	if seen[key] {
		return true
	}
	seen[key] = true
	return false
}

func (csi *Coolstuffinc) parseBL(ctx context.Context) error {
	edition2id, err := LoadBuylistEditions(ctx, csi.shelf)
	if err != nil {
		return err
	}
	csi.printf("Loaded %d editions", len(edition2id))

	products, err := GetBuylist(ctx, csi.shelf)
	if err != nil {
		return err
	}
	csi.printf("Found %d products", len(products))

	// Some Magic PIDs get a placeholder isFoil=1 row at a flat price beside
	// their real nonfoil row; nonfoilPID feeds magicPhantomFoilTwin below.
	nonfoilPID := map[string]bool{}
	if csi.game == mtgban.GameMagic {
		for _, product := range products {
			if product.IsFoil == 0 {
				nonfoilPID[product.PID] = true
			}
		}
	}

	for _, product := range products {
		if product.RarityName == "Box" {
			continue
		}

		// Filter by set if needed
		if csi.targetEdition != "" && product.ItemSet != csi.targetEdition {
			continue
		}

		// Build link early to help debug
		u, _ := url.Parse(csiBuylistLink)
		v := url.Values{}
		v.Set("s", csi.shelf)
		v.Set("a", "1")
		v.Set("name", product.Name)
		v.Set("f[]", fmt.Sprint(product.IsFoil))

		id, found := edition2id[product.ItemSet]
		if found {
			v.Set("is[]", id)
		}
		u.RawQuery = v.Encode()
		link := u.String()

		var theCard *mtgmatcher.InputCard
		var runFinishes []string
		switch csi.game {
		case mtgban.GameMagic:
			c, err := PreprocessBuylist(csi.backend, product)
			if err != nil {
				continue
			}
			theCard = c
		// The note names the printing for these games - a Riftbound promo's
		// finish and prize track, a Yu-Gi-Oh rarity - where One Piece spends
		// it describing the artwork and Lorcana's changes no answer at all.
		case mtgban.GamePokemon:
			theCard, runFinishes = pokemonBuylistCard(csi.backend, product)
		case mtgban.GameRiftbound:
			variation := buylistVariation(product)
			shelf := riftboundShelf(csi.backend, product.ItemSet, product.Notes, product.Name, variation, product.IsFoil == 1)
			theCard = &mtgmatcher.InputCard{Name: product.Name, Edition: shelf, Variation: variation, Foil: product.IsFoil == 1}
			_, err := csi.backend.Match(theCard)
			if err != nil {
				fromSKU := riftboundSKUCard(csi.backend, product.Image, product.IsFoil == 1)
				if fromSKU != nil {
					theCard = fromSKU
				}
			}
		// The rarity arrives in a field of its own here, where the sell
		// listing spends the note on it, so a row whose note says nothing
		// still names the tier that tells its printing from its siblings.
		case mtgban.GameYuGiOh:
			if unknownPrinting(product.Name, product.ItemSet) {
				continue
			}
			theCard = &mtgmatcher.InputCard{Name: catalogColor(catalogSpelling(jpArtWording(product.Name))), Edition: printRunEdition(product.ItemSet, product.Notes), Variation: strings.TrimSpace(jpArtWording(buylistVariation(product)) + " " + catalogRarity(product.RarityName)), Foil: product.IsFoil == 1}
		case mtgban.GameOnePiece:
			donName, donDescription, isDon := onePieceDonName(product.Name)
			if isDon {
				donShelf := onePieceShelf(product.ItemSet, product.Name)
				donFoil := product.IsFoil == 1
				theCard = &mtgmatcher.InputCard{Name: donName, Edition: donShelf, Variation: donDescription, Foil: donFoil}
				_, err := csi.backend.Match(theCard)
				if err != nil {
					renamed := onePieceDonRenamed(donDescription)
					if renamed != "" {
						theCard = &mtgmatcher.InputCard{Name: donName, Edition: donShelf, Variation: renamed, Foil: donFoil}
					}
				}
				break
			}
			theCard = &mtgmatcher.InputCard{Name: onePieceSpelling(jpArtWording(product.Name)), Edition: onePieceShelf(product.ItemSet, product.Name), Variation: eventNamed(strings.TrimSpace(product.Number + " " + nameQualifiers(jpArtWording(product.Name)))), Foil: product.IsFoil == 1}
		// Gundam prints the same card at the same number in three sets, so
		// the shelf has to narrow and the storefront's own code prefix stops
		// it naming one; the wording it hangs behind the name is what tells
		// the parallel runs apart.
		case mtgban.GameGundam:
			name, variation := gundamCard(csi.backend, product.Name, product.Number)
			theCard = &mtgmatcher.InputCard{Name: name, Edition: gundamShelf(product.ItemSet), Variation: strings.TrimSpace(variation + " " + gundamTier(product.RarityName)), Foil: product.IsFoil == 1}
		// Palworld numbers a parallel apart from the card it parallels, the
		// rarity riding in the number's own tail, so the plain reading names
		// one printing and nothing has to be read out of the wording. This
		// feed writes its numbers clean today, but it is the same hands
		// typing the sell listings that glue the rarity onto one.
		case mtgban.GamePalworld:
			theCard = &mtgmatcher.InputCard{Name: palworldName(product.Name), Edition: product.ItemSet, Variation: palworldNotes(product.Number), Foil: product.IsFoil == 1}
		// Notes starts with the same number Number carries and spends the
		// rest on the qualifier; Number alone misses a Starter Deck Exclusive.
		case mtgban.GameLorcana:
			theCard = &mtgmatcher.InputCard{Name: lorcanaSpelling(product.Name), Edition: product.ItemSet, Variation: lorcanaVariation(product.Notes), Foil: product.IsFoil == 1}
		}

		var cardID string
		var err error
		if runFinishes != nil {
			cardID, err = matchRun(csi.backend, theCard, runFinishes)
		} else {
			cardID, err = csi.backend.Match(theCard)
		}
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			csi.printf("error: %v", err)
			csi.printf("original: %q", product)
			csi.printf("preprocessed: %q", theCard)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				for _, probe := range alias.Probe() {
					co, _ := csi.backend.GetUUID(probe)
					csi.printf("- %s", co)
				}
			}
			continue
		}

		if csi.game == mtgban.GameMagic {
			co, cerr := csi.backend.GetUUID(cardID)
			if cerr == nil && magicPhantomFoilTwin(co, product.IsFoil, nonfoilPID[product.PID]) {
				continue
			}
		}

		if csi.game == mtgban.GamePokemon && pokemonNonHolo.MatchString(product.Name) {
			co, cerr := csi.backend.GetUUID(cardID)
			if cerr == nil && !co.HasFinish(mtgmatcher.FinishNonfoil) &&
				strings.Contains(co.Rarity, "Holo") {
				continue
			}
		}

		if csi.game == mtgban.GameOnePiece {
			if renamed := onePieceRenamedTreatment(csi.backend, cardID, product.Name); renamed != "" {
				cardID = renamed
			}
		}

		buyPrice, err := mtgmatcher.ParsePrice(product.Price)
		if err != nil {
			csi.printf("%s error: %s", product.Name, err.Error())
			continue
		}

		var priceRatio, sellPrice float64

		invCards := csi.inventory[cardID]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = buyPrice / sellPrice * 100
		}

		for i, deduction := range deductions {
			buyEntry := mtgban.BuylistEntry{
				Conditions: mtgban.DefaultGradeTags[i],
				BuyPrice:   buyPrice * deduction,
				PriceRatio: priceRatio,
				URL:        link,
				CustomFields: map[string]string{
					"originalProduct": fmt.Sprintf("%q", product),
				},
			}

			err := csi.buylist.Add(cardID, &buyEntry)
			if errors.Is(err, mtgban.ErrDuplicateEntry) {
				// The buylist names a token once per deck it came in,
				// every one at one price for the one printing.
				continue
			}
			if err != nil {
				csi.printf("%s", err.Error())
				continue
			}
		}
	}

	csi.buylistDate = time.Now()

	return nil
}

// magicPhantomFoilTwin reports whether a Magic buylist row is CSI's flat-
// priced isFoil=1 placeholder for a printing that has no real foil finish.
func magicPhantomFoilTwin(co *mtgmatcher.CardObject, isFoil int, hasNonfoilRow bool) bool {
	return isFoil == 1 && hasNonfoilRow && !co.Foil && !co.Etched
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (csi *Coolstuffinc) SetConfig(opt mtgban.ScraperOptions) {
	csi.disableRetail = opt.DisableRetail
	csi.disableBuylist = opt.DisableBuylist
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (csi *Coolstuffinc) Load(ctx context.Context) error {
	var errs []error

	if !csi.disableRetail {
		err := csi.scrape(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		}
	}

	if !csi.disableBuylist {
		err := csi.parseBL(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

// Inventory returns what Load collected. See mtgban.Seller.
func (csi *Coolstuffinc) Inventory() mtgban.InventoryRecord {
	return csi.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (csi *Coolstuffinc) Buylist() mtgban.BuylistRecord {
	return csi.buylist
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (csi *Coolstuffinc) MarketNames() []string {
	return availableMarketNames
}

// InfoForScraper describes one of the sub-scrapers named above.
func (csi *Coolstuffinc) InfoForScraper(name string) mtgban.ScraperInfo {
	info := csi.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (csi *Coolstuffinc) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cool Stuff Inc"
	info.Shorthand = "CSI"
	info.InventoryTimestamp = &csi.inventoryDate
	info.BuylistTimestamp = &csi.buylistDate
	info.CreditMultiplier = 1.25
	info.Game = csi.game
	return
}

// singlesRarities answers the rarity tiers a singles search should ask for,
// read from the Rarity fieldset of the storefront's advanced search.
func singlesRarities(fieldset *goquery.Selection) []string {
	var rarities []string
	fieldset.Find("li").Each(func(_ int, s *goquery.Selection) {
		rarity, found := s.Find(`input[type="checkbox"]`).Attr("value")
		if !found || rarity == "" {
			return
		}
		if sealedTiers[strings.ToLower(strings.TrimSpace(s.Text()))] {
			return
		}

		rarities = append(rarities, rarity)
	})
	return rarities
}

// sealedTiers names the rarity tiers the storefront files sealed products
// under, spelled as the search page prints them. They are the tiers a
// singles search leaves out; a tier missing from here only lets sealed
// through to be refused later, where a card tier missing from the search
// loses the card outright.
var sealedTiers = map[string]bool{
	"box":  true,
	"pack": true,
}

// csiTreatments spells the patterned holos the catalog's way. Cool Stuff Inc
// calls them foils - "Master Ball Foil" - where the catalog files the pattern
// as what it is, and the difference is not cosmetic: "foil" is a finish the
// matcher already reads, so asking for a Master Ball Alomomola by that name
// answers with the set's reverse holo rather than missing.
var csiTreatments = strings.NewReplacer(
	"Master Ball Foil", "Master Ball Pattern",
	"Poke Ball Foil", "Poke Ball Pattern",
	"Shatterfoil", "Cracked Ice Holo",
	"Cosmo Holo", "Cosmos Holo",
)

// catalogTreatment answers the catalog's wording for a treatment the
// storefront spells its own way, and leaves everything else as it stands.
func catalogTreatment(variation string) string {
	return csiTreatments.Replace(variation)
}

// numberedListing reads a listing name apart from the collector number this
// storefront glues onto it, answering the name alone and what it took, or the
// name whole and nothing where the storefront named the card by itself.
//
// Only a tail opening on a number is taken, since the storefront also hangs
// plain wording off a dash ("Ancient Mew - Movie Promo") and the catalog
// spells some cards with one of its own. What is taken is set aside rather
// than thrown away: the printing is written in parentheses behind the number
// ("16/111 (Reverse Foil)") and neither side's foil column says so, which the
// matcher reads off the name it is handed.
func numberedListing(name string) (string, string) {
	head, tail, found := strings.Cut(name, " - ")
	if !found {
		return name, ""
	}
	number, _, _ := strings.Cut(tail, " ")
	if !buylistNumberWord.MatchString(number) {
		return name, ""
	}
	return head, tail
}

// pokemonListing reads a Pokemon listing the way the catalog names it. The
// Classic Collection reprints are sold under Celebrations with the
// collection in the note; the special energies are named "Special Metal
// Energy" where the catalog names the energy and labels it special; the
// Base Set Professor Oak is spelled "Imposter" the way Base Set 2 prints
// it, though Base Set printed "Impostor"; Nidoran is named without the sex
// the catalog names it by, which the number settles; the Elite Four
// cards of the Platinum sets are named "Alakazam 4" for the catalog's
// "Alakazam E4"; and the metal cards are named for the metal, "Metal Mew ex"
// for the catalog's Mew ex labelled a metal card.
func pokemonListing(b *mtgmatcher.Backend, name, edition, variation string, foil bool) *mtgmatcher.InputCard {
	name, numbered := numberedListing(name)
	name = strings.TrimSpace(nonStampedName.ReplaceAllString(name, ""))
	card := &mtgmatcher.InputCard{Name: name, Edition: edition, Variation: variation, Foil: foil}
	m := basicEnergyName.FindStringSubmatch(name)
	if m != nil {
		bracket := strings.Trim(m[2], "()")
		fixed := pokemonBasicEnergy(b, m[1], bracket, edition, numbered, variation, foil)
		if fixed != nil {
			return fixed
		}
	}
	nonHolo := pokemonNonHolo.MatchString(name) || pokemonNonHolo.MatchString(numbered)
	if nonHolo && !strings.Contains(strings.ToLower(edition), "promo") {
		strippedName := strings.TrimSpace(pokemonNonHolo.ReplaceAllString(name, ""))
		strippedNumbered := strings.TrimSpace(pokemonNonHolo.ReplaceAllString(numbered, ""))
		if pokemonNonHoloDeckExclusive(b, strippedName, strippedNumbered, foil) {
			card.Name = strippedName
			card.Edition = "Deck Exclusives"
			numbered = strippedNumbered
		}
	}
	if edition == "Celebrations" && (strings.Contains(variation, "Classic Collection") || strings.Contains(numbered, "Classic Collection")) {
		card.Edition = "Celebrations: Classic Collection"
		if m := classicNumber.FindStringSubmatch(variation); m != nil {
			card.Variation = m[1]
		}
	}
	// A reprint note ("25th Anniversary Stamp Base Set Reprint") reaches
	// retail's variation whole; clear it before "Stamp" reads as a demand.
	reprint := buylistReprintNote.MatchString(variation)
	if reprint {
		card.Variation = ""
	}
	if name == "Vivillon" {
		numbered = pokemonVivillonColors.Replace(numbered)
	}
	// A "(Non-Holo)" listing is never a holo pull, so it is never this
	// redirect's business even when its note also carries one of
	// pokemonDeckHoloNotes' markers.
	redirected := ""
	if !nonHolo {
		redirected = pokemonDeckHoloRedirect(b, name, numbered, variation)
	}
	if redirected != "" {
		card.Edition = redirected
		card.Variation = "Cracked Ice Holo"
		card.Foil = true
	}
	m = goldStar.FindStringSubmatch(name)
	if m != nil {
		card.Name = m[1] + " Star"
	}
	m = unownListing.FindStringSubmatch(name)
	if m != nil {
		card.Name = "Unown"
		card.Variation = m[2] + "/" + m[3]
	}
	if m := specialEnergy.FindStringSubmatch(name); m != nil {
		card.Name = m[1]
		card.Variation = strings.TrimSpace("Special " + card.Variation)
	}
	// Only where the whole name is no card and the rest is one: "Metal
	// Energy" is a card, and so is none of "Energy (Secret Rare)".
	m = metalCardName.FindStringSubmatch(card.Name)
	if m != nil {
		_, whole := b.SearchEquals(card.Name)
		_, rest := b.SearchEquals(m[1])
		if whole != nil && rest == nil {
			card.Name = m[1]
			if !mtgmatcher.SlugDescribes(card.Variation, "metalcard") {
				card.Variation = strings.TrimSpace(card.Variation + " Metal Card")
			}
		}
	}
	if name == "Imposter Professor Oak" && strings.HasPrefix(edition, "Base Set") && !strings.HasPrefix(edition, "Base Set 2") {
		card.Name = "Impostor Professor Oak"
	}
	if m := eliteFour.FindStringSubmatch(name); m != nil && strings.HasPrefix(edition, "Platinum") {
		card.Name = m[1] + " E4"
	}
	// An energy or promo sold under a main set with the energy or promo
	// set's own number ("MEE007" on Ascended Heroes) is that set's.
	if m := prefixedNumber.FindStringSubmatch(variation); m != nil {
		if set, found := pokemonNumberSets[m[1]]; found {
			card.Edition = set
			// The catalog numbers the older promo sets with the
			// programme on ("SM190") and the newer ones without.
			if m[1] != "SM" && m[1] != "SWSH" {
				card.Variation = strings.Replace(variation, m[0], m[2], 1)
			}
		}
	}
	if respelled, found := pokemonRespellings[name]; found {
		card.Name = respelled
	}
	// The notes say a printing is the plain one ("Non-Stamped Version"),
	// name the illustrator, or name the stamp a promo carries; the plain
	// words come off, and a stamp names a promo shelf the catalog files
	// apart from the set the listing arrived on.
	card.Variation = strings.TrimSpace(plainWords.ReplaceAllString(card.Variation, " "))
	if !reprint && stamped.MatchString(card.Variation) {
		card.Edition = "Promo"
	}
	// The Team Galactic inventions are named by their invention alone
	// where the catalog names the invention with its number.
	if _, err := b.SearchEquals(card.Name); err != nil {
		if invention := galacticInvention(b, card.Name); invention != "" {
			card.Name = invention
		}
	}
	// The catalog letters the type in a special energy's name ("Bubbly W
	// Energy") where the storefront spells it out, but not in every one
	// ("Magnetic Metal Energy"), so the spelled name is kept where the
	// catalog knows it.
	if m := typedEnergy.FindStringSubmatch(card.Name); m != nil {
		if _, err := b.SearchEquals(card.Name); err != nil {
			card.Name = m[1] + " " + energyLetters[m[2]] + " Energy"
		}
	}
	if name == "Nidoran" || name == "Nidoran?" {
		for _, sex := range []string{"Nidoran M", "Nidoran F"} {
			probe := *card
			probe.Name = sex
			if numbered != "" {
				probe.Name += " - " + numbered
			}
			if _, err := b.Match(&probe); err == nil {
				card.Name = sex
				break
			}
		}
	}
	if numbered != "" {
		card.Name += " - " + numbered
	}
	return card
}

// pokemonBuylistCard reads a buylist row the way a sell listing is read, the
// print run included. The run rides in the shelf's title on both sides of the
// storefront, and only the sell listings were reading it: a buy row arrived
// with its shelf spelled whole, matched the set of that name, and was
// published against the unlimited printing at the first edition's price.
func pokemonBuylistCard(b *mtgmatcher.Backend, product CSIPriceEntry) (*mtgmatcher.InputCard, []string) {
	// CSI's "0" placeholder for an unnumbered year energy otherwise reads
	// as a number word of its own at the head of the variation.
	if product.Number == "0" {
		product.Number = ""
	}
	variation := catalogTreatment(buylistVariation(product))
	shelf, run := firstEditionShelf(product.ItemSet)
	shelf = pokemonPromoShelf(b, product.Name, shelf, product.RarityName, product.IsFoil == 1, variation)
	return pokemonListing(b, product.Name, shelf, variation, product.IsFoil == 1), run
}

// pokemonNumberSets are the sets a number's prefix names outright.
var pokemonNumberSets = map[string]string{
	"MEE":  "MEE: Mega Evolution Energies",
	"SVE":  "SVE: Scarlet & Violet Energies",
	"MEP":  "ME: Mega Evolution Promo",
	"SVP":  "SV: Scarlet & Violet Promo Cards",
	"SWSH": "SWSH: Sword & Shield Promo Cards",
	"SM":   "SM Promos",
}

// pokemonRespellings pairs the names this storefront misspells with the
// catalog's own.
var pokemonRespellings = map[string]string{
	"Galatic HQ":                   "Galactic HQ",
	"Sprigattito":                  "Sprigatito",
	"Unit Energy GFW":              "Unit Energy GRW",
	"Delta Species Rainbow Energy": "Delta Rainbow Energy",
}

var (
	plainWords = regexp.MustCompile(`(?i)\bNon-?Stamped(?: Version)?\b|\bIllus\. [^,]+,`)
	stamped    = regexp.MustCompile(`(?i)\bStamp(?:ed)?\b`)

	// nonStampedName matches the same bracket as plainWords where it sits
	// in the name's own head instead - "Psyduck (Non-Stamped) - SM199".
	nonStampedName = regexp.MustCompile(`(?i)\s*\(Non-?Stamped\)`)
)

// galacticInvention names the Team Galactic invention the catalog files
// under its number, or nothing when no invention ends in the name given.
func galacticInvention(b *mtgmatcher.Backend, name string) string {
	found := ""
	for _, candidate := range b.Names(mtgmatcher.NameFormCanonical, false) {
		if strings.HasPrefix(candidate, "Team Galactic's Invention") && strings.HasSuffix(candidate, " "+name) {
			if found != "" {
				return ""
			}
			found = candidate
		}
	}
	return found
}

var energyLetters = map[string]string{
	"Grass": "G", "Fire": "R", "Water": "W", "Lightning": "L", "Psychic": "P",
	"Fighting": "F", "Darkness": "D", "Metal": "M", "Fairy": "Y", "Dragon": "N",
	"Colorless": "C",
}

var (
	typedEnergy    = regexp.MustCompile(`^(\w+) (Grass|Fire|Water|Lightning|Psychic|Fighting|Darkness|Metal|Fairy|Dragon|Colorless) Energy$`)
	prefixedNumber = regexp.MustCompile(`\b(MEE|SVE|MEP|SVP|SWSH|SM)(\d{3})\b`)
	classicNumber  = regexp.MustCompile(`Classic Collection (\d+)`)
	specialEnergy  = regexp.MustCompile(`^Special ((?:Metal|Darkness) Energy)$`)
	eliteFour      = regexp.MustCompile(`^(.+) 4( LV\.X)?$`)

	// goldStar matches a Gold Star the way this storefront names it, "Mew *
	// (Star)" for the catalog's "Mew Star" - $1,800 of buylist refusals on
	// Ex Dragon Frontiers' Mew alone.
	goldStar = regexp.MustCompile(`^(.+) \* \(Star\)$`)

	// unownListing matches an EX Unseen Forces Unown the way this
	// storefront names it, the letter written into both the name and its
	// own index ("Unown A - A/28"), where the catalog names every one of
	// them "Unown" and numbers it by the letter alone.
	unownListing = regexp.MustCompile(`^Unown ([A-Z!?]) - ([A-Z!?])/(\d+)$`)

	// metalCardName matches an Ultra-Premium Collection metal card the way
	// this storefront names it, for the metal and the set it copies ("Metal
	// Mew ex", "Metal Base Set Charizard"), where the catalog names the card.
	metalCardName = regexp.MustCompile(`^Metal (?:Base Set )?(.+)$`)
)

// pokemonVivillonColors spells the two Vivillon colours this storefront
// shortens where the catalog's own promo label is longer than the colour
// alone - "(Pink)" for "(Meadow Pink)", the only thing that tells the XY
// 17/146 Pink Pattern apart from the identically-numbered Orange one.
var pokemonVivillonColors = strings.NewReplacer(
	"(Pink)", "(Meadow Pink)",
	"(Orange)", "(High Plains Orange)",
)

// pokemonDeckHoloNotes names the edition a print-run note belongs to, and
// the set code the redirect has to land on to be trusted.
var pokemonDeckHoloNotes = []struct {
	marker, edition, wantSet string
}{
	{"Theme Deck", "Deck Exclusives", "PR-1840"},
	{"EX Battle Stadium", "EX Battle Stadium", "BST"},
	{"Prism Holo", "Miscellaneous Cards & Products", "MCAP"},
}

// pokemonDeckHoloRedirect answers the edition for a pokemonDeckHoloNotes
// marker whose probe - asking for the "Cracked Ice Holo" label so a
// cracked-ice twin outranks the plain printing of the same number - lands
// on that marker's own set code, or "" otherwise.
func pokemonDeckHoloRedirect(b *mtgmatcher.Backend, name, numbered, notes string) string {
	for _, r := range pokemonDeckHoloNotes {
		if !strings.Contains(notes, r.marker) && !strings.Contains(numbered, r.marker) {
			continue
		}
		tail := strings.TrimSpace(strings.Replace(numbered, r.marker, "", 1))
		tail = strings.TrimSpace(strings.TrimSuffix(tail, "-"))
		probeName := name
		if tail != "" {
			probeName += " - " + tail
		}
		probe := &mtgmatcher.InputCard{Name: probeName, Edition: r.edition, Variation: "Cracked Ice Holo", Foil: true}
		id, err := b.Match(probe)
		if err != nil {
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil || co.SetCode != r.wantSet {
			continue
		}
		return r.edition
	}
	return ""
}

// basicEnergyName matches this storefront's "<Type> Energy" and a treatment
// bracket it carries of its own ("(Cosmo Holo)"), apart from the number
// tail's. pokemonBasicEnergy gates the rename on the listing's own number.
var basicEnergyName = regexp.MustCompile(`^(Grass|Fire|Water|Lightning|Psychic|Fighting|Darkness|Metal|Fairy) Energy(?:\s+(\(.+\)))?$`)

// pokemonBasicEnergy answers a bare "<Type> Energy" listing the way the
// catalog spells its Scarlet & Violet / Mega Evolution era printings, read
// off the SVE/MEE-shaped product number glued onto the tail.
func pokemonBasicEnergy(b *mtgmatcher.Backend, energyType, bracket, edition, numbered, variation string, foil bool) *mtgmatcher.InputCard {
	name := "Basic " + energyType + " Energy"
	bracket = catalogTreatment(bracket)

	build := func(fromVariation bool, numberedTail string) *mtgmatcher.InputCard {
		card := &mtgmatcher.InputCard{Name: name, Edition: edition, Variation: bracket, Foil: foil}
		if fromVariation {
			card.Variation = strings.TrimSpace(bracket + " " + numberedTail)
		} else if numberedTail != "" {
			card.Name += " - " + numberedTail
		}
		return card
	}

	tail, fromVariation := csiTreatments.Replace(numbered), false
	pm := prefixedNumber.FindStringSubmatch(tail)
	if pm == nil {
		tail, fromVariation = csiTreatments.Replace(variation), true
		pm = prefixedNumber.FindStringSubmatch(tail)
	}
	if pm == nil {
		// A plain rename needs the tail's own number to say which
		// printing is meant, on the shelf's own set: the same gate
		// the on-shelf and SVE/MEE probes below both apply.
		num := mtgmatcher.ExtractNumber(numbered)
		if num == "" {
			return nil
		}
		shelf, err := b.GetSetByName(edition)
		if err != nil {
			return nil
		}
		plain := build(false, numbered)
		plain.Variation = strings.TrimSpace(bracket + " " + variation)
		id, err := b.Match(plain)
		if err != nil {
			return nil
		}
		co, err := b.GetUUID(id)
		if err != nil || co.SetCode != shelf.Code || strings.TrimLeft(co.Number, "0") != num {
			return nil
		}
		return plain
	}

	shelf, shelfErr := b.GetSetByName(edition)

	bare := strings.TrimLeft(pm[2], "0")
	if bare == "" {
		bare = "0"
	}
	onShelf := build(fromVariation, strings.TrimSpace(strings.Replace(tail, pm[0], bare, 1)))
	if shelfErr == nil {
		id, err := b.Match(onShelf)
		if err == nil {
			co, err := b.GetUUID(id)
			if err == nil && co.SetCode == shelf.Code {
				return onShelf
			}
		}
	}

	set, found := pokemonNumberSets[pm[1]]
	if !found {
		return nil
	}
	target, err := b.GetSetByName(set)
	if err != nil {
		return nil
	}
	redirected := build(fromVariation, strings.TrimSpace(strings.Replace(tail, pm[0], pm[2], 1)))
	redirected.Edition = set
	id, err := b.Match(redirected)
	if err != nil {
		return nil
	}
	co, err := b.GetUUID(id)
	if err != nil || co.SetCode != target.Code {
		return nil
	}
	return redirected
}

// pokemonPromoShelf answers the shelf a Pokemon listing belongs to, which is
// the one it arrived on unless the catalog files the card as a promo.
//
// A promo carrying a main set's number is sold here under that set, with only
// the rarity field saying otherwise: the Pokemon Day 2025 Eevee sits on SV
// Prismatic Evolutions at 074/131, where that set's own Eevee already stands.
// The two met there and a $2.50 promo was priced as the card it was stamped
// from. The catalog keeps those on a promo shelf instead.
//
// The promo shelf only decides where it answers at all. Twenty of the fifty
// buylist listings this can reach name a printing no promo shelf holds - the
// Pokemon Rumble cards, the holo promos that are their set's own foil - and
// those keep the set they arrived on. The same shape reproduces on retail -
// the identical "Eevee - 074/131 (Pokemon Day 2025)" listing sits on the same
// shelf, with the same breadcrumb rarity "Promo" - so both sides call this
// with their own name for the rarity field: RarityName on the buylist, the
// scraped breadcrumb text on retail.
func pokemonPromoShelf(b *mtgmatcher.Backend, name, itemSet, rarityName string, foil bool, variation string) string {
	if rarityName != "Promo" ||
		strings.Contains(strings.ToLower(itemSet), "promo") {
		return itemSet
	}
	probe := &mtgmatcher.InputCard{
		Name:      name,
		Edition:   "Promo",
		Variation: variation,
		Foil:      foil,
	}
	_, err := b.Match(probe)
	if err != nil {
		return itemSet
	}
	return "Promo"
}

// riftboundNotePrefix matches the set code a Riftbound note opens with.
var riftboundNotePrefix = regexp.MustCompile(`^([A-Z]{2,4})-`)

// riftboundShelf answers the set a Riftbound listing belongs to, which is the
// shelf it arrived on except where that shelf says only "Promo".
//
// The Nexus Night runes are sold under the promo shelf with the set that
// issued them written at the head of the note - "UNL-R05b", "SFD-R05b" - and
// the promo shelf holds a printing of its own at that number. All of them met
// there, so a $5.00 Unleashed Chaos Rune and an $11.00 Spiritforged one were
// both priced as the Organized Play printing they share a number with. The
// same three listings, word for word, sell on the retail search too.
//
// The note only decides where the set it names holds that printing. Vendetta
// issued no b-lettered rune of its own, so its six listings stay on the promo
// shelf, which is where the printing they mean actually is.
func riftboundShelf(b *mtgmatcher.Backend, itemSet, notes, name, variation string, foil bool) string {
	if itemSet != "Promo" {
		return itemSet
	}
	match := riftboundNotePrefix.FindStringSubmatch(notes)
	if match == nil {
		return itemSet
	}
	set, err := b.GetSet(match[1])
	if err != nil {
		return itemSet
	}
	probe := &mtgmatcher.InputCard{
		Name:      name,
		Edition:   set.Name,
		Variation: variation,
		Foil:      foil,
	}
	_, err = b.Match(probe)
	if err != nil {
		return itemSet
	}
	return set.Name
}

// onePieceDonHead matches the wrapping the storefront hangs in front of
// a DON!! card's description, numbered or not.
var onePieceDonHead = regexp.MustCompile(`(?i)^DON!!\s*(?:\([0-9]+\))?\s*-?\s*`)

// onePieceDonCharacters spells the four characters this storefront
// names in full where the catalog names them by the name the promo type
// carries. They are not misspellings and not this storefront's
// invention - both names are the character's - so a listing saying
// "Edward Newgate" is describing the card the catalog files under
// "whitebeard", and neither wording reaches the other on its own.
//
// Read as a second attempt, never as a correction. Some promo types
// carry the full name themselves - the first anniversary's DON!! is
// "monkeydluffy1st" - so rewriting every listing would lose the cards
// whose catalog wording the storefront already matched. Only the four
// the storefront actually differs on are here, and a character the
// catalog spells differently again stays refused: every DON!! in a set
// shares a name and a number, so a guess buys another card.
var onePieceDonCharacters = strings.NewReplacer(
	"Edward Newgate", "Whitebeard",
	"Charlotte Linlin", "Big Mom",
	"Monkey.D.Luffy", "Luffy",
	"Portgas.D.Ace", "Ace",
)

// onePieceDonName answers a DON!! listing the way the catalog files it,
// and reports whether the listing is one at all.
//
// The catalog names all 238 of the game's DON!! cards "DON!! Card" and
// tells them apart by promo type alone - the character, the artwork,
// the border - which the matcher already reads out of a listing's own
// wording. The storefront writes those same words into the product name
// and publishes no collector number for a DON!! at all, so the name
// reaches nothing and the wording never gets as far as the rules that
// would have read it. Hand over the name the catalog uses and leave the
// description where a description belongs.
func onePieceDonName(name string) (string, string, bool) {
	if !strings.HasPrefix(strings.ToUpper(name), "DON!!") {
		return "", "", false
	}
	return "DON!! Card", onePieceDonHead.ReplaceAllString(name, ""), true
}

// onePieceDonRenamed answers the same description with the characters
// the catalog names differently swapped in, and an empty string when it
// names none of them.
func onePieceDonRenamed(description string) string {
	renamed := onePieceDonCharacters.Replace(description)
	if renamed == description {
		return ""
	}
	return renamed
}

// riftboundImageStem reads the file name off a product image, whatever
// the storefront pictures in, and riftboundImageTCG the TCGplayer
// product id the oldest Origins products are pictured by.
var (
	riftboundImageStem = regexp.MustCompile(`(?i)/([^/]+)\.(?:jpe?g|png|webp|avif)$`)
	riftboundImageTCG  = regexp.MustCompile(`[0-9]{5,}`)

	riftboundImageTail = regexp.MustCompile(`(?i)(?:ovr|alt)?[_.]*(?:v[0-9]+)?$`)

	// riftboundImagePrintIndex strips a tally a sku tacks onto its own
	// number - "148_2" for Anivia. Anchored so it never eats a number
	// that is only ever "_<digits>" itself.
	riftboundImagePrintIndex = regexp.MustCompile(`^([0-9]+)_[0-9]+$`)

	// riftboundImageSig matches the tail the storefront hangs on a
	// signature printing's sku, which the catalog numbers with a star
	// instead ("SFD224SIG" for 224*). It is read rather than dropped:
	// the same set files an overnumbered printing of the same card at
	// that number without a star, so dropping it answers the wrong
	// printing at the wrong price rather than nothing.
	riftboundImageSig = regexp.MustCompile(`(?i)SIG$`)

	// The image pads its numbers to three digits where the catalog does
	// not ("VEN021" for 21, "UNLT01" for T1). PlainNumber reduces the
	// codes the catalog publishes and leaves a bare padded number as it
	// stands, so the padding comes off here.
	riftboundImagePad = regexp.MustCompile(`^([A-Za-z]*)0+([0-9])`)
)

// riftboundImageCard answers the card a listing's product image names.
//
// The storefront writes the champion's subtitle into the product name -
// "Akali - Deadly Weapon" - where the catalog files every one of a
// champion's cards under the champion alone. Vendetta's 021 and 038 are
// both just "Akali", so the listing's own words are the only thing
// telling them apart and the catalog's words are not, and the name
// reaches nothing. The image is the storefront's own sku and it carries
// the set code and the collector number, which says which printing the
// listing is when its wording cannot.
//
// It is read only where the wording was refused. Where both have an
// opinion the wording is the better one: a handful of images are a
// sibling printing's, reused, and the notes carry the exact number those
// listings are sold at.
func riftboundImageCard(b *mtgmatcher.Backend, imgURL string, foil bool) *mtgmatcher.InputCard {
	match := riftboundImageStem.FindStringSubmatch(imgURL)
	if match == nil {
		return nil
	}
	return riftboundSKUCard(b, match[1], foil)
}

// riftboundSKUCard answers the card one of the storefront's skus names.
// The sale listings carry it inside an image url and the buylist feed
// hands it over bare, in an Image field of its own.
func riftboundSKUCard(b *mtgmatcher.Backend, stem string, foil bool) *mtgmatcher.InputCard {
	if stem == "" {
		return nil
	}

	// A run of five digits or more is a TCGplayer product id rather than
	// a collector number, which runs to three. The catalog carries those
	// ids, so one that answers a card is the card; one that answers
	// nothing was never an id and the sku is read instead.
	for _, id := range riftboundImageTCG.FindAllString(stem, -1) {
		uuid := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, id)
		if uuid == "" {
			continue
		}
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		card := riftboundCardAt(b, co.SetCode, co.Number, foil)
		if card != nil {
			return card
		}
	}

	// The set code sits somewhere in the name, behind a letter or two of
	// the storefront's own, and the number is what follows it. The codes
	// come from the datastore rather than a list here, so a set added
	// upstream is read without a change. Longest first: a short code is
	// a substring of longer ones.
	codes := slices.Clone(b.AllSets)
	slices.SortFunc(codes, func(a, b string) int {
		return len(b) - len(a)
	})

	upper := strings.ToUpper(stem)
	for _, code := range codes {
		index := strings.Index(upper, strings.ToUpper(code))
		if index < 0 {
			continue
		}
		number := upper[index+len(code):]
		var signature string
		if riftboundImageSig.MatchString(number) {
			number = riftboundImageSig.ReplaceAllString(number, "")
			signature = "*"
		}
		number = riftboundImageTail.ReplaceAllString(number, "")
		m := riftboundImagePrintIndex.FindStringSubmatch(number)
		if m != nil {
			number = m[1]
		}
		number = strings.Trim(number, "-_.")
		if number == "" {
			continue
		}
		number += signature
		card := riftboundCardAt(b, code, number, foil)
		if card != nil {
			return card
		}
	}
	return nil
}

// riftboundCardAt answers the one card a set files at a number, and
// nothing where the number names several cards or none. Several is not a
// miss to guess at: the number is being read to tell printings apart, so
// a number that does not is no answer.
func riftboundCardAt(b *mtgmatcher.Backend, code, number string, foil bool) *mtgmatcher.InputCard {
	set, err := b.GetSet(code)
	if err != nil {
		return nil
	}
	plain := riftboundImagePad.ReplaceAllString(b.PlainNumber(number), "${1}${2}")
	var name string
	for _, card := range set.Cards {
		if !strings.EqualFold(b.PlainNumber(card.Number), plain) {
			continue
		}
		if name != "" && name != card.Name {
			return nil
		}
		name = card.Name
	}
	if name == "" {
		return nil
	}
	return &mtgmatcher.InputCard{
		Name:      name,
		Edition:   set.Name,
		Variation: number,
		Foil:      foil,
	}
}

// csiRarities spells the storefront's Yu-Gi-Oh rarity names the way the
// catalog does. Only the foil tiers disagree: the storefront drops the
// "Rare" the catalog keeps, writes Starfoil as two words, and drops the
// possessive s from Collector's. Everything else - Common, Rare, Mosaic
// Rare - it already spells alike, so a name absent from this table passes
// through as it stands.
// csiUnknownPrintings names the listings this storefront sells under a
// printing the catalog does not carry, keyed by the name and the shelf it
// sits on.
//
// Both are a rarity the set prints and the card at that number is not.
// Gladiator Beast Octavius exists in Gladiator's Assault as a Secret Rare
// and nothing else, and the Exodia of Limited Pack World Championship 2025
// is one of that set's two Emblazoned rarities and never the plain Secret
// Rare - EN000 is the only one of its 21 numbers without the plain tiers,
// which is what says the card is Emblazoned-only rather than half-published.
// Each is priced at a quarter against the printing it lands on, $13.00 and
// $500.00, which is the price of a card that is not that card.
//
// They are listed one at a time because no rule separates them from a
// decorated rarity a storefront spells shorter: "Secret Rare" for the
// catalog's "Prismatic Secret Rare" is the same shape and is correct.
// Refusing a rarity the card does not carry drops 55 Yu-Gi-Oh listings to
// catch these two, and around 35 of those are right.
var csiUnknownPrintings = map[string]bool{
	"Exodia the Forbidden One (Secret Rare)|Limited Pack World Championship 2025": true,
	"Gladiator Beast Octavius (Super Rare)|Gladiators Assault":                    true,
}

// unknownPrinting reports a listing that names a printing the catalog does
// not carry, which is a listing worth dropping rather than matching: the
// nearest printing to it is a different card at a different price.
func unknownPrinting(name, edition string) bool {
	return csiUnknownPrintings[name+"|"+edition]
}

// csiColors spells a Duelist League colour the way the catalog files it, for
// the ones this storefront names differently. A league prints one number in
// several colours and nothing else tells them apart, so the word is the whole
// identification.
//
// The spelling has to be swapped rather than added to. "Light Blue" says the
// word blue, which names the blue printing outright, so a listing carrying
// both words names two printings and ties where it used to answer one - and
// the storefront's own word says nothing else worth keeping.
var csiColors = strings.NewReplacer("(Light Blue)", "(Silver)")

// catalogColor spells the colour a Yu-Gi-Oh listing names the way the catalog
// files it.
func catalogColor(name string) string {
	return csiColors.Replace(name)
}

// csiSpellings corrects the Yu-Gi-Oh names this storefront misspells. Each
// key is a name no set in the game has and each value is the card it means,
// one slip of the fingers away: a doubled letter, a dropped one, a swapped
// pair. Left as typed they look up nothing at all and the listing goes
// unpriced.
//
// One pair is not a slip: the storefront writes the Greek letter Konami
// prints on the card where the catalog writes the word for it. The catalog
// is what has to be asked, so it is corrected here alongside the typos
// rather than argued with.
//
// The pairs are spelled out rather than found by nearest match. A catalog
// tells its numbered siblings apart by a single character - "Armed Dragon
// LV3" against LV5, "Harpie Lady 1" against 2 - and 255 pairs of names
// inside one set are a single edit apart for that reason, so a reader that
// corrects by distance can answer a card the datastore is merely missing
// with its neighbour. A table cannot.
var csiSpellings = map[string]string{
	"Belial - Marqis of Darkness":   "Belial - Marquis of Darkness",
	"Compulsory Evactuation Device": "Compulsory Evacuation Device",
	"Doube-Edged Sword Technique":   "Double-Edged Sword Technique",
	"Fearl Imp":                     "Feral Imp",
	"Fiendish Engine Ω":             "Fiendish Engine Omega",
	"Homumculus the Alchemic Being": "Homunculus the Alchemic Being",
	"Miracle Jurrassic Egg":         "Miracle Jurassic Egg",
	"Perfect Synch - A-Un":          "Perfect Sync - A-Un",
	"Rush Recklessely":              "Rush Recklessly",
	"Sealing Ceremony of Mokuten":   "Sealing Ceremony of Mokuton",
	"Sealing Cermony of Raiton":     "Sealing Ceremony of Raiton",
}

// catalogSpelling spells a Yu-Gi-Oh name the way the catalog does, where this
// storefront has typed it wrong.
//
// A set that files one card under several printings gets the printing hung
// behind the name in brackets, and the name in front of it is typed the same
// wrong way as on the listings that carry no bracket. The correction reaches
// the head of the line for that reason, leaving the bracket to the matcher,
// which reads it into the variation either way. No name in the catalog begins
// with any key in the table, so a head that matches one is the typo and never
// the opening of a longer card name.
func catalogSpelling(name string) string {
	if spelled, found := csiSpellings[name]; found {
		return spelled
	}
	head, printing, bracketed := strings.Cut(name, " (")
	if !bracketed {
		return name
	}
	spelled, found := csiSpellings[head]
	if !found {
		return name
	}
	return spelled + " (" + printing
}

var csiRarities = map[string]string{
	"Star Foil":      "Starfoil Rare",
	"Shatterfoil":    "Shatterfoil Rare",
	"Collector Rare": "Collector's Rare",
}

// csiPrintRuns names both runs a Yu-Gi-Oh edition was printed in, keyed by
// the edition the storefront publishes. It sells both under that one name
// and tells them apart the only way anybody can, by the copyright date
// printed on the card, which it writes into the note. Nothing downstream can
// pick between them - the later run reissues the original's numbers - which
// is why mtgmatcher's Yu-Gi-Oh edition aliases leave these sets out on
// purpose, and why naming only one of the two would file the other under it.
//
// The earlier run is spelled out even where the storefront's own name
// already reaches it, so the table states the whole answer rather than
// leaning on a name that happens to match. Two of them do not match: the
// catalog keeps a "The" the storefront drops, and numbers Retro Pack without
// the 1 it sells the pack as. Both were settled by the buylist's own
// collector numbers, which write the earlier run plain ("LOB-001") and the
// later one with the language ("LOB-EN015").
var csiPrintRuns = map[string][2]string{
	"Dark Crisis":                      {"Dark Crisis", "Dark Crisis (25th Anniversary Edition)"},
	"Invasion of Chaos":                {"Invasion of Chaos", "Invasion of Chaos (25th Anniversary Edition)"},
	"Legend of Blue Eyes White Dragon": {"The Legend of Blue Eyes White Dragon", "Legend of Blue Eyes White Dragon (25th Anniversary Edition)"},
	"Light of Destruction":             {"Light of Destruction", "Light of Destruction (2020 Date Reprint)"},
	"Metal Raiders":                    {"Metal Raiders", "Metal Raiders (25th Anniversary Edition)"},
	"Pharaohs Servant":                 {"Pharaoh's Servant", "Pharaoh's Servant (25th Anniversary Edition)"},
	"Retro Pack 1":                     {"Retro Pack", "Retro Pack (2020 Date Reprint)"},
	"Retro Pack 2":                     {"Retro Pack 2", "Retro Pack 2 (2020 Date Reprint)"},
	"Spell Ruler":                      {"Spell Ruler", "Spell Ruler (25th Anniversary Edition)"},
}

// printRunEdition answers the edition the note's copyright date names. An
// edition sold in one run, or a note that says no date, is left as it is.
func printRunEdition(edition, notes string) string {
	runs, found := csiPrintRuns[edition]
	if !found || !strings.Contains(notes, "Copyright") {
		return edition
	}
	if strings.Contains(notes, "2020") {
		return runs[1]
	}
	return runs[0]
}

// catalogRarity answers the catalog's name for a rarity the storefront
// prints, so the rarity tier can read it.
func catalogRarity(rarity string) string {
	if spelled, found := csiRarities[rarity]; found {
		return spelled
	}
	return rarity
}
