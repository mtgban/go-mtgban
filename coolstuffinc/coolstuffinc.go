// Package coolstuffinc scrapes Cool Stuff Inc, for singles and sealed
// product.
package coolstuffinc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
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
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	// If set to true scrape will include all entries without a nonfoil NM price
	// but will be almost twice as slow
	IncludeOOS bool

	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	TargetEdition string

	DisableRetail  bool
	DisableBuylist bool

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *http.Client
	game   string
}

// pokemonNonHolo matches the bracket a Pokemon name states a plain printing
// with. The storefront sells a card printed in both finishes as two products
// telling them apart by that bracket alone - the note is empty and the foil
// flag is off on both - and read as the holo, a $2.00 Team Aqua's Kyogre was
// served as the $80.00 one's price.
//
// The rarity is what says whether the plain printing was ever made. A holo
// rare is sold holo and nothing else, so a bracket asking for its plain
// printing asks for one that does not exist and the row is refused. A plain
// rare is the opposite: the catalog holding no nonfoil for it is the catalog
// missing a printing rather than the storefront inventing one, and refusing
// those would drop 25 real listings to catch nothing.
var pokemonNonHolo = regexp.MustCompile(`\(Non-?\s?Holo\)`)

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
func onePieceRenamedTreatment(id, name string) string {
	co, err := mtgmatcher.GetUUID(id)
	if err != nil || len(co.PromoTypes) > 0 {
		return ""
	}
	set, err := mtgmatcher.GetSet(co.SetCode)
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
		if slug == "" || labels[slug] || !slices.Contains(mtgmatcher.AllPromoTypes(), slug) {
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

// NewScraper returns a singles scraper for one game.
func NewScraper(game string) *Coolstuffinc {
	csi := Coolstuffinc{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	csi.client = client.StandardClient()
	csi.MaxConcurrency = defaultConcurrency
	csi.game = game
	return &csi
}

type responseChan struct {
	cardID   string
	invEntry *mtgban.InventoryEntry
	relaxed  bool
}

func (csi *Coolstuffinc) printf(format string, a ...any) {
	if csi.LogCallback != nil {
		csi.LogCallback("[CSI] "+format, a...)
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
	conditions := strings.TrimLeft(fullRow, qtyStr+"+ ")
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
	skipOOS := !csi.IncludeOOS
	switch itemName {
	case "Alpha", "Beta", "Unlimited Edition":
		skipOOS = false
	}
	result, err := Search(ctx, csi.game, itemName, skipOOS, rarities)
	if err != nil {
		return err
	}

	// result.PageId may be empty if the results have only one page
	for page := 1; ; page++ {
		data := result.Data

		if page > 1 {
			link := "https://www.coolstuffinc.com/sq/" + result.PageID + "?page=" + fmt.Sprint(page)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
			if err != nil {
				return err
			}
			resp, err := csi.client.Do(req)
			if err != nil {
				return err
			}
			data, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}

		doc.Find(`div[class="row product-search-row main-container"]`).Each(func(i int, s *goquery.Selection) {
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

				if strings.Contains(conditions, "BGS") ||
					strings.Contains(conditions, "Non-Foil") ||
					strings.Contains(conditions, "Unique") {
					conditions = "Near Mint"
					graded = true
				}

				// Sometimes etched cards have a Near Mint and Near Mint Foil condition
				// for the same card
				if strings.Contains(cardName, "Foil-etched") {
					relaxed = true
				}

				switch conditions {
				case "Near Mint", "Foil Near Mint":
					conditions = "NM"
				case "Played", "Foil Played":
					conditions = "MP"
				default:
					csi.printf("Unsupported '%s' condition for %s", conditions, cardName)
					return
				}
				if strings.Contains(cardName, "Signed by") {
					conditions = "HP"
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
				if csi.Partner != "" {
					link += "?utm_referrer=" + csi.Partner
				}

				var theCard *mtgmatcher.InputCard
				switch csi.game {
				case GameMagic:
					c, err := preprocess(cardName, edition, notes, imgURL)
					if err != nil {
						return
					}
					// preprocess() might return something that derived foil status
					// from one of the fields (cardName in particular)
					c.Foil = c.Foil || isFoil
					theCard = c
				case GameYuGiOh:
					if unknownPrinting(cardName, edition) {
						return
					}
					theCard = &mtgmatcher.InputCard{Name: catalogColor(catalogSpelling(cardName)), Edition: printRunEdition(edition, notes), Variation: strings.TrimSpace(notes + " " + catalogRarity(rarity)), Foil: isFoil}
				case GamePokemon:
					theCard = &mtgmatcher.InputCard{Name: cardName, Edition: edition, Variation: catalogTreatment(notes), Foil: isFoil}
				case GameOnePiece:
					theCard = &mtgmatcher.InputCard{Name: cardName, Edition: edition, Variation: eventNamed(notes), Foil: isFoil}
				case GameGundam:
					name, variation := gundamCard(cardName, "")
					theCard = &mtgmatcher.InputCard{Name: name, Edition: gundamShelf(edition), Variation: strings.TrimSpace(variation + " " + notes), Foil: isFoil}
				case GameLorcana, GameRiftbound, GamePalworld:
					theCard = &mtgmatcher.InputCard{Name: cardName, Edition: edition, Variation: notes, Foil: isFoil}
				default:
					csi.printf("unsupported game")
					return
				}

				cardID, err := mtgmatcher.Match(theCard)
				if errors.Is(err, mtgmatcher.ErrUnsupported) {
					return
				} else if err != nil {
					switch {
					// Ignore expected misses
					case mtgmatcher.IsBasicLand(theCard.Name),
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
								card, _ := mtgmatcher.GetUUID(probe)
								csi.printf("- %s", card)
							}
						}
					}
					return
				}

				// Magic-only finish sanity check: skip cards that do not have the
				// requested finish.
				if csi.game == GameMagic {
					if strings.Contains(cardName, "Foil-etched") {
						co, err := mtgmatcher.GetUUID(cardID)
						if err != nil || !co.Etched {
							return
						}
					}
					if isFoil {
						co, err := mtgmatcher.GetUUID(cardID)
						if err != nil || (!co.Etched && !co.Foil) {
							return
						}
					}
				}

				out := responseChan{
					cardID: cardID,
					invEntry: &mtgban.InventoryEntry{
						Conditions: conditions,
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

		next, _ := doc.Find(`span[id="nextLink"]`).Find("a").Attr("href")
		if next == "" {
			break
		}
	}

	return nil
}

func (csi *Coolstuffinc) scrape(ctx context.Context) error {
	link := csiInventoryURL + csi.game
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

	if csi.TargetEdition != "" {
		filtered := itemNames[:0]
		for _, item := range itemNames {
			if item == csi.TargetEdition {
				filtered = append(filtered, item)
			}
		}
		itemNames = filtered
	}

	mtgban.WorkerPool(ctx, csi.MaxConcurrency, itemNames,
		func(ctx context.Context, itemName string, results chan<- responseChan) error {
			csi.printf("Processing %s", itemName)
			return csi.processSearch(ctx, results, itemName, rarities)
		},
		func(record responseChan) {
			var err error
			if record.relaxed {
				err = csi.inventory.AddRelaxed(record.cardID, record.invEntry)
			} else {
				err = csi.inventory.Add(record.cardID, record.invEntry)
			}
			if err != nil {
				csi.printf("%s", err.Error())
			}
		},
		csi.printf,
	)

	csi.printf("This operation took %v", time.Since(start))

	csi.inventoryDate = time.Now()

	return nil
}

func (csi *Coolstuffinc) parseBL(ctx context.Context) error {
	edition2id, err := LoadBuylistEditions(ctx, csi.game)
	if err != nil {
		return err
	}
	csi.printf("Loaded %d editions", len(edition2id))

	products, err := GetBuylist(ctx, csi.game)
	if err != nil {
		return err
	}
	csi.printf("Found %d products", len(products))

	for _, product := range products {
		if product.RarityName == "Box" {
			continue
		}

		// Filter by set if needed
		if csi.TargetEdition != "" && product.ItemSet != csi.TargetEdition {
			continue
		}

		// Build link early to help debug
		u, _ := url.Parse(csiBuylistLink)
		v := url.Values{}
		v.Set("s", csi.game)
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
		switch csi.game {
		case GameMagic:
			c, err := PreprocessBuylist(product)
			if err != nil {
				continue
			}
			theCard = c
		// The note names the printing for these games - a Riftbound promo's
		// finish and prize track, a Yu-Gi-Oh rarity - where One Piece spends
		// it describing the artwork and Lorcana's changes no answer at all.
		case GamePokemon:
			variation := catalogTreatment(buylistVariation(product))
			theCard = &mtgmatcher.InputCard{Name: product.Name, Edition: pokemonPromoShelf(product, variation), Variation: variation, Foil: product.IsFoil == 1}
		case GameRiftbound:
			variation := buylistVariation(product)
			theCard = &mtgmatcher.InputCard{Name: product.Name, Edition: riftboundShelf(product, variation), Variation: variation, Foil: product.IsFoil == 1}
		// The rarity arrives in a field of its own here, where the sell
		// listing spends the note on it, so a row whose note says nothing
		// still names the tier that tells its printing from its siblings.
		case GameYuGiOh:
			if unknownPrinting(product.Name, product.ItemSet) {
				continue
			}
			theCard = &mtgmatcher.InputCard{Name: catalogColor(catalogSpelling(product.Name)), Edition: printRunEdition(product.ItemSet, product.Notes), Variation: strings.TrimSpace(buylistVariation(product) + " " + catalogRarity(product.RarityName)), Foil: product.IsFoil == 1}
		case GameOnePiece:
			theCard = &mtgmatcher.InputCard{Name: product.Name, Edition: onePieceShelf(product.ItemSet, product.Name), Variation: eventNamed(strings.TrimSpace(product.Number + " " + nameQualifiers(product.Name))), Foil: product.IsFoil == 1}
		// Gundam prints the same card at the same number in three sets, so
		// the shelf has to narrow and the storefront's own code prefix stops
		// it naming one; the wording it hangs behind the name is what tells
		// the parallel runs apart.
		case GameGundam:
			name, variation := gundamCard(product.Name, product.Number)
			theCard = &mtgmatcher.InputCard{Name: name, Edition: gundamShelf(product.ItemSet), Variation: variation, Foil: product.IsFoil == 1}
		// Palworld numbers a parallel apart from the card it parallels, the
		// rarity riding in the number's own tail, so the plain reading names
		// one printing and nothing has to be read out of the wording.
		case GameLorcana, GamePalworld:
			theCard = &mtgmatcher.InputCard{Name: product.Name, Edition: product.ItemSet, Variation: product.Number, Foil: product.IsFoil == 1}
		default:
			return errors.New("unsupported game")
		}

		cardID, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			csi.printf("error: %v", err)
			csi.printf("original: %q", product)
			csi.printf("preprocessed: %q", theCard)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				for _, probe := range alias.Probe() {
					co, _ := mtgmatcher.GetUUID(probe)
					csi.printf("- %s", co)
				}
			}
			continue
		}

		if csi.game == GamePokemon && pokemonNonHolo.MatchString(product.Name) {
			co, cerr := mtgmatcher.GetUUID(cardID)
			if cerr == nil && !co.HasFinish(mtgmatcher.FinishNonfoil) &&
				strings.Contains(co.Rarity, "Holo") {
				continue
			}
		}

		if csi.game == GameOnePiece {
			if renamed := onePieceRenamedTreatment(cardID, product.Name); renamed != "" {
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
			if err != nil {
				csi.printf("%s", err.Error())
				continue
			}
		}
	}

	csi.buylistDate = time.Now()

	return nil
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (csi *Coolstuffinc) SetConfig(opt mtgban.ScraperOptions) {
	csi.DisableRetail = opt.DisableRetail
	csi.DisableBuylist = opt.DisableBuylist
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (csi *Coolstuffinc) Load(ctx context.Context) error {
	var errs []error

	if !csi.DisableRetail {
		err := csi.scrape(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		}
	}

	if !csi.DisableBuylist {
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
	if csi.Info().Game != mtgban.GameMagic {
		return availableMarketNames[:1]
	}
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
	switch csi.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	case GameRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameOnePiece:
		info.Game = mtgban.GameOnePiece
	case GamePokemon:
		info.Game = mtgban.GamePokemon
	case GameYuGiOh:
		info.Game = mtgban.GameYuGiOh
	case GameGundam:
		info.Game = mtgban.GameGundam
	case GamePalworld:
		info.Game = mtgban.GamePalworld
	}
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
)

// catalogTreatment answers the catalog's wording for a treatment the
// storefront spells its own way, and leaves everything else as it stands.
func catalogTreatment(variation string) string {
	return csiTreatments.Replace(variation)
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
// listings this can reach name a printing no promo shelf holds - the Pokemon
// Rumble cards, the holo promos that are their set's own foil - and those keep
// the set they arrived on.
func pokemonPromoShelf(product CSIPriceEntry, variation string) string {
	if product.RarityName != "Promo" ||
		strings.Contains(strings.ToLower(product.ItemSet), "promo") {
		return product.ItemSet
	}
	probe := &mtgmatcher.InputCard{
		Name:      product.Name,
		Edition:   "Promo",
		Variation: variation,
		Foil:      product.IsFoil == 1,
	}
	_, err := mtgmatcher.Match(probe)
	if err != nil {
		return product.ItemSet
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
// both priced as the Organized Play printing they share a number with.
//
// The note only decides where the set it names holds that printing. Vendetta
// issued no b-lettered rune of its own, so its six listings stay on the promo
// shelf, which is where the printing they mean actually is.
func riftboundShelf(product CSIPriceEntry, variation string) string {
	if product.ItemSet != "Promo" {
		return product.ItemSet
	}
	match := riftboundNotePrefix.FindStringSubmatch(product.Notes)
	if match == nil {
		return product.ItemSet
	}
	set, err := mtgmatcher.GetSet(match[1])
	if err != nil {
		return product.ItemSet
	}
	probe := &mtgmatcher.InputCard{
		Name:      product.Name,
		Edition:   set.Name,
		Variation: variation,
		Foil:      product.IsFoil == 1,
	}
	_, err = mtgmatcher.Match(probe)
	if err != nil {
		return product.ItemSet
	}
	return set.Name
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
// The pairs are spelled out rather than found by nearest match. A catalog
// tells its numbered siblings apart by a single character - "Armed Dragon
// LV3" against LV5, "Harpie Lady 1" against 2 - and 255 pairs of names
// inside one set are a single edit apart for that reason, so a reader that
// corrects by distance can answer a card the datastore is merely missing
// with its neighbour. A table cannot.
var csiSpellings = map[string]string{
	"Belial - Marqis of Darkness":   "Belial - Marquis of Darkness",
	"Compulsory Evactuation Device": "Compulsory Evacuation Device",
	"Fearl Imp":                     "Feral Imp",
	"Homumculus the Alchemic Being": "Homunculus the Alchemic Being",
	"Miracle Jurrassic Egg":         "Miracle Jurassic Egg",
	"Perfect Synch - A-Un":          "Perfect Sync - A-Un",
	"Rush Recklessely":              "Rush Recklessly",
	"Sealing Ceremony of Mokuten":   "Sealing Ceremony of Mokuton",
	"Sealing Cermony of Raiton":     "Sealing Ceremony of Raiton",
}

// catalogSpelling spells a Yu-Gi-Oh name the way the catalog does, where this
// storefront has typed it wrong.
func catalogSpelling(name string) string {
	if spelled, found := csiSpellings[name]; found {
		return spelled
	}
	return name
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
