// Package cardmarket scrapes Cardmarket, both the price-guide index and
// sealed product, across every game they carry.
package cardmarket

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type responseChan struct {
	ogID   int
	cardID string
	entry  mtgban.InventoryEntry
	// byName marks a price whose printing was named rather than looked
	// up by id, which is a guess however well guarded; see namedLast.
	byName bool
	// tally carries an edition's walked and refused counts in place of a
	// price, one record per edition, so the pool's single collector can
	// count the run without the workers sharing anything. An edition whose
	// catalog never arrived sends one too, with nothing counted and unread
	// set, so the run's total says how much of the catalog it is a total of.
	tally   bool
	walked  int
	refused int
	unread  bool
}

// namedLast holds back the prices whose printing was named until every
// price looked up by an id is in.
//
// The catalogs keep one expansion per old set where the datastore keeps a
// printing per print run, so the set guard puts every run's product on the
// base set's printing, and a run the bridge already priced is a run a name
// can reach too. AddUnique keeps whichever price arrives first, so without
// the wait the winner is whichever expansion the pool happened to walk
// first - and half the time that hands a verified printing over to a guess
// about a different one. Waiting decides it instead: the guess is offered
// only where nothing verified stands.
type namedLast struct {
	add   func(responseChan)
	named []responseChan
	// The run's tally, summed from the editions' records; the collector
	// runs on one goroutine, so plain counts are all this takes.
	walked  int
	refused int
	unread  int
}

// collect takes one result, adding it, holding it back, or counting it
// into the run's tally.
func (n *namedLast) collect(result responseChan) {
	if result.tally {
		n.walked += result.walked
		n.refused += result.refused
		if result.unread {
			n.unread++
		}
		return
	}
	if result.byName {
		n.named = append(n.named, result)
		return
	}
	n.add(result)
}

// flush adds everything held back, in the order it arrived, and reports how
// much that was.
func (n *namedLast) flush() int {
	for i := range n.named {
		n.add(n.named[i])
	}
	return len(n.named)
}

// Index prices singles from Cardmarket's price guide, the low and
// trend numbers rather than any one seller's listing.
type Index struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int
	exchangeRate   float64

	// Optional field to select a single edition to go through
	TargetEdition string

	// TCGBridge maps a Cardmarket product id to the TCGplayer id of the
	// same single, for the keyless catalogs (yugioh, flesh and blood)
	// whose products carry no collector number and no version index, so
	// same-name products are told apart only by an exact id. bantool
	// builds it from cardtrader's blueprints, the one source linking the
	// two marketplaces; the scraper itself stays vendor-pure and receives
	// it as plain data.
	TCGBridge map[int]int

	inventory mtgban.InventoryRecord

	// priceGuide holds one game's published prices, indexed by the product
	// id they belong to: a run asks for one product's prices tens of
	// thousands of times, once per product in the catalog.
	priceGuide map[int]PriceGuide

	// shelved names, for each set of ours, the expansion of this run that
	// sells it; see offShelf. Load fills it once the expansions are known.
	shelved map[string]string

	client *MKMClient
	gameID int
}

var availableIndexNames = []string{
	"MKM Low", "MKM Trend",
}

var name2shorthand = map[string]string{
	"MKM Low":   "MKMLow",
	"MKM Trend": "MKMTrend",
}

func (mkm *Index) printf(format string, a ...any) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMIndex] "+format, a...)
	}
}

// NewScraperIndex returns an index scraper for one game, authenticated with an
// app token and secret.
func NewScraperIndex(gameID int, appToken, appSecret string) (*Index, error) {
	mkm := Index{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.client = NewMKMClient(appToken, appSecret)
	mkm.MaxConcurrency = defaultConcurrency
	mkm.gameID = gameID
	return &mkm, nil
}

// errNoPrinting marks a product no route named a printing of ours for. It
// is what the id-and-name route answers with instead of nothing at all, so a
// refusal is counted and said out loud rather than passing for a success.
var errNoPrinting = errors.New("named no printing of ours")

func (mkm *Index) processEdition(ctx context.Context, channel chan<- responseChan, expansion MKMExpansion) error {
	products, err := mkm.client.MKMProductsInExpansion(ctx, expansion.IDExpansion)
	if err != nil {
		// The pool logs this and moves on, and the run's tally used to
		// close over the expansions that did answer as though they were
		// all of them: a run several expansions timed out of read as a
		// smaller catalog rather than a partial one. Say the expansion is
		// missing from the count instead of leaving it out silently.
		channel <- responseChan{tally: true, unread: true}
		return err
	}

	var refused []string
	for _, product := range products {
		err := mkm.processProduct(channel, &product)
		switch {
		case errors.Is(err, errNoPrinting):
			refused = append(refused, fmt.Sprintf("%d %q (%s) in %s",
				product.IDProduct, product.Name, product.Number, product.ExpansionName))
		case err != nil:
			mkm.printf("product id %d returned %s", product.IDProduct, err)
		}
	}

	mkm.reportRefused(expansion.Name, len(products), refused)
	channel <- responseChan{tally: true, walked: len(products), refused: len(refused)}
	return nil
}

// reportRefused says what an expansion refused and counts it into the run's
// tally. Every refusal is named, one line each, except in an expansion
// nothing resolved in: that is a catalog we do not carry at all - Cardmarket
// sells whole Japanese programs the datastores have no set for - and the
// count is the whole story, where naming each of its products would be tens
// of thousands of lines saying it again.
func (mkm *Index) reportRefused(expansion string, total int, refused []string) {
	if len(refused) == 0 {
		return
	}
	mkm.printf("%s: %d of %d products named no printing of ours", expansion, len(refused), total)
	if len(refused) == total {
		return
	}
	for _, product := range refused {
		mkm.printf("no printing for %s", product)
	}
}

// versionTail matches the parenthetical Cardmarket tells same-name products
// apart with, which names its own version index and the rarity beside it.
var versionTail = regexp.MustCompile(` \(V\.\d+.*\)$`)

// rarityTail captures the rarity out of that same parenthetical, which is
// the only place Cardmarket writes it.
var rarityTail = regexp.MustCompile(` \(V\.\d+ - ([^)]+)\)$`)

// numberTail matches the digits a collector number ends on, which is the part
// two catalogs numbering the same card agree about.
var numberTail = regexp.MustCompile(`\d+[A-Za-z]?$`)

// nameCode matches the collector number Cardmarket writes at the end of a One
// Piece product name, beside the one it writes in the number field.
var nameCode = regexp.MustCompile(`\(([A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*)\)$`)

// onePieceNumber picks the collector number a One Piece product is asked
// with, out of the two Cardmarket writes for it.
//
// The number field is sometimes another card's ("Sanji (OP01-013)" filed
// under ST13-016, "Corrida Coliseum (OP04-096)" under OP04-092) and
// sometimes a typo of the name's own code with a digit doubled ("ST13-0151",
// "P-0611"). The code inside the name is the card's, so it answers for the
// product when the two disagree.
//
// Only on a shelf naming a set of ours, though. Cardmarket sells its promos
// in buckets no set of ours answers for - "Judge Promos", "Winner Cards",
// "Premium Bandai Products" - and with nothing to hold it to a set the
// matcher reaches past the edition and lands on the ordinary booster
// printing of that number, which is a mass-printed card wearing a promo's
// price. A refusal says less but claims nothing.
func onePieceNumber(name, number, expansion string) string {
	fields := nameCode.FindStringSubmatch(name)
	if fields == nil || strings.EqualFold(fields[1], number) {
		return number
	}
	if _, err := mtgmatcher.GetSetByName(expansion); err != nil {
		return number
	}
	return fields[1]
}

// shelvedSets names, for each set of ours, the one expansion of this catalog
// that sells it. A set no expansion names is absent, and so is an expansion
// naming no set.
func shelvedSets(list []MKMExpansion) map[string]string {
	shelved := make(map[string]string, len(list))
	for _, exp := range list {
		set, err := mtgmatcher.GetSetByName(exp.Name)
		if err != nil {
			continue
		}
		shelved[set.Code] = exp.Name
	}
	return shelved
}

// offShelf reports whether a product answered with a printing that another
// expansion of this catalog sells as a product of its own.
//
// Cardmarket files a card on every shelf that ever handed it out, and most
// of those shelves are no set of ours - "Judge Promos", "Unnumbered Promos",
// "Special Tournament Promos", a starter deck whose reprints the datastore
// does not carry. With nothing holding the listing to an edition the matcher
// reaches past it and lands on the ordinary printing of that number, which
// is the very printing the shelf named after that set already prices as a
// product of its own. Two products on one printing means one of them
// publishes a price for a card it does not sell, and which one wins is
// whichever expansion the pool happened to walk first.
//
// Only an unlabelled answer reads that way. The datastore labels every
// printing that is not the ordinary one - the alternate arts, the event
// copies, the deck reprints - and those are what the promo shelves really do
// sell, so a labelled answer is this product's own card however far from its
// shelf the datastore files it. The set has to be one this catalog sells
// elsewhere, too: where no other expansion names it nothing else is pricing
// it, and refusing would drop the only price there is.
func (mkm *Index) offShelf(product *MKMProduct, cardID string) bool {
	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil || len(co.PromoTypes) > 0 {
		return false
	}
	shelf, found := mkm.shelved[co.SetCode]
	return found && !strings.EqualFold(shelf, product.ExpansionName)
}

// numberPrefix returns the letters a collector number opens on, the set code
// aside: "EN005" yields "EN", "DCR-005" yields "", "SGX1-END19" yields "END".
func numberPrefix(number string) string {
	if _, tail, dashed := strings.Cut(number, "-"); dashed {
		number = tail
	}
	return strings.ToUpper(number[:len(number)-len(strings.TrimLeft(number, letters))])
}

// letters spells the alphabet a collector number's prefix is drawn from.
const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// otherPrintRun reports whether a product's collector number names a print
// run other than the one the answer's number belongs to.
//
// Cardmarket sells the North American print of an old Yu-Gi-Oh set ("005"),
// the European ("EN005") and the Asian ("A005") as three products of one
// expansion, and files the special-edition promos beside them ("SP1"), where
// the datastore carries the single row the set is numbered by ("DCR-005").
// The matcher reads none of those prefixed numbers - its own numbering opens
// on digits or on a set code - so it drops them and answers the name alone,
// which puts every run on that one row. The prefix has to be the answer's
// own, the region infix the datastore writes and Cardmarket omits aside
// ("D19" is "SGX1-END19", "EN005" is not "DCR-005").
func otherPrintRun(number, full string) bool {
	prefix := numberPrefix(number)
	return prefix != "" && !strings.HasSuffix(numberPrefix(full), prefix)
}

// productFinish names the printing a product is, for the catalogs that sell
// each printing as its own product rather than as a column beside the card.
func productFinish(gameID int, product *MKMProduct) string {
	if gameID == GameFleshAndBlood {
		return fabFinish(product.ExpansionName, product.Name)
	}
	return ""
}

// foreignShelves are the tails Cardmarket appends to a set's name when it
// shelves that set's non-English printings apart from the English ones. They
// are separate catalogs of the same cards, and the datastore carries only the
// English ones, so a price from one of these shelves would land on a printing
// it is not.
var foreignShelves = []string{"(Japanese)", "(Korean)", "(PMT)"}

// foreignShelf reports whether an expansion name wears one of those tails.
func foreignShelf(name string) bool {
	for _, tail := range foreignShelves {
		if strings.HasSuffix(name, tail) {
			return true
		}
	}
	return false
}

// yugiohRunIndex captures the index Cardmarket appends to a Yu-Gi-Oh product
// name when one card is sold as several products.
var yugiohRunIndex = regexp.MustCompile(` \(V\.(\d+) - `)

// yugiohFirstAtIndexOne names the sets whose index counts the other way
// round. Cardmarket synthesizes the index per set and what it counts differs
// from one to the next - a rarity here, a print run there - so no reading of
// it is right everywhere. The default below is the one the catalog bears out
// most often; a set whose prices say its runs are swapped belongs here, and
// the entry is all it takes to correct it.
var yugiohFirstAtIndexOne = map[string]bool{}

// yugiohRun names the print run a product's index stands for, or nothing
// when it carries no index.
//
// A set printed twice sells both runs under one name, and nothing else the
// catalog says tells them apart: the collector number is the same, the
// rarity is the same, and the shelf is the same. Only the index is left, and
// it is read here rather than trusted - the first edition is the scarcer run
// and the dearer one, which is how a set that reads the wrong way round is
// found and added above.
//
// Measured over the run's collisions, the higher index is the dearer product
// 924 times against 388, so it is the first edition by default.
func yugiohRun(product *MKMProduct) string {
	fields := yugiohRunIndex.FindStringSubmatch(product.Name)
	if fields == nil {
		return ""
	}
	index, err := strconv.Atoi(fields[1])
	if err != nil || index < 1 {
		return ""
	}
	first := index > 1
	if yugiohFirstAtIndexOne[product.ExpansionName] {
		first = index == 1
	}
	if first {
		return "1st Edition"
	}
	return "Unlimited"
}

// matchProduct resolves a product the bridge does not know, from what the
// catalog says of it. The edition has to name a set of ours and the answer
// has to be in it: Cardmarket carries whole Japanese catalogs the datastores
// do not, and Match reaches past the edition when nothing in it fits, so
// without both an unknown set's cards land on whichever set happens to hold
// a number like theirs.
func (mkm *Index) matchProduct(product *MKMProduct) string {
	edition := product.ExpansionName
	// A non-English catalog wearing an English set's name passes that gate,
	// so it needs one of its own.
	if mkm.gameID == GamePokemon && pokemonForeignDenied(edition, product.Number) {
		return ""
	}
	var printRun, numberPrefix string
	if mkm.gameID == GameFleshAndBlood {
		// Cardmarket sells each print run as its own expansion
		// ("Monarch - First"), a name no set of ours carries: the run
		// belongs to the printing, where productFinish puts it, reading
		// the same suffix off the untouched expansion name.
		printRun, edition = fabPrintRun(edition)
	}
	set, err := mtgmatcher.GetSetByName(edition)
	if err != nil && mkm.gameID == GameFleshAndBlood {
		// What Cardmarket calls the expansion is not always what we call
		// the set. Translating is the fallback rather than the first move,
		// so an expansion whose name we already know keeps answering for
		// itself and only the ones nothing answers for are rewritten.
		edition, numberPrefix = fabEdition(edition)
		set, err = mtgmatcher.GetSetByName(edition)
	}
	if err != nil {
		return ""
	}
	names := []string{versionTail.ReplaceAllString(product.Name, "")}
	if mkm.gameID == GameFleshAndBlood {
		// The treatment parenthetical is the printing's, not the name's:
		// fabFinish reads it off the untouched product name below, and a
		// card whose own name ends in a parenthetical ("Sink Below (Red)")
		// keeps it, so the exact name reaches the matcher whole. The raw
		// name stays as the fallback: the sets spelling a pitch color the
		// other one's way ("Rawhide Rumble" at ARR012, "Rawhide Rumble
		// (Red)" at HVY023) file the stripped name under the wrong set,
		// and only the decorated one still finds them.
		_, stripped := fabTreatment(names[0])
		if stripped != names[0] {
			names = []string{stripped, names[0]}
		}
	}

	// A Yu-Gi-Oh card is named by name + number + rarity, and a set prints
	// one number in several rarities ("Ultra Rare" beside "Ultimate Rare"):
	// with the tail only deleted the two are indistinguishable and both go
	// unpriced. The rarity does not belong in the name, so it rides beside
	// the number instead, which is where the matcher reads it from. The
	// other catalogs write their own vocabulary in that tail - Riftbound
	// spells treatments there - so this stays the one game's.
	var rarity string
	if mkm.gameID == GameYuGiOh {
		if fields := rarityTail.FindStringSubmatch(product.Name); fields != nil {
			rarity = fields[1]
		}
	}

	numbers := []string{product.Number}
	// A promo's programme is the prefix our numbering carries and the
	// expansion Cardmarket sells it under, so the number is only whole once
	// the two are put back together.
	if numberPrefix != "" {
		numbers = []string{numberPrefix + product.Number, product.Number}
	}
	// The oldest Yu-Gi-Oh sets are numbered by their original Asian print
	// ("A015") where the datastore numbers them by set ("LOB-015"); the
	// digits are what the two agree on.
	if tail := numberTail.FindString(product.Number); tail != "" && set.Code != "" {
		numbers = append(numbers, set.Code+"-"+tail)
	}

	// A game selling one card in several print runs needs one of them
	// named, or the runs alias and nothing resolves. The later run is what
	// the id route lands on, so it is what the fallback asks for first.
	finishes := []string{""}
	switch mkm.gameID {
	case GameYuGiOh:
		// The later run is what the id route lands on, so it is what the
		// fallback asks for after the index has had its say.
		finishes = []string{yugiohRun(product), "Unlimited", ""}
	case GameFleshAndBlood:
		finishes = []string{productFinish(mkm.gameID, product), ""}
	}

	for _, name := range names {
		for _, number := range numbers {
			variation := strings.TrimSpace(number + " " + rarity)
			for _, finish := range finishes {
				id, err := mtgmatcher.Match(&mtgmatcher.InputCard{
					Name:      name,
					Edition:   edition,
					Variation: variation,
					Finish:    finish,
				})
				if err != nil {
					continue
				}
				co, cerr := mtgmatcher.GetUUID(id)
				if cerr != nil || !strings.EqualFold(co.SetCode, set.Code) {
					continue
				}
				// The run has to hold too, for the same reason the set
				// does: a card the datastore keeps in one run only is
				// answered with that run whichever was asked for, and the
				// other run's expansion sells the very same card.
				if printRun != "" && !strings.HasPrefix(co.Finish, mtgmatcher.NormalizeFinish(printRun)) {
					continue
				}
				if mkm.gameID == GameYuGiOh && otherPrintRun(product.Number, co.Number) {
					continue
				}
				// A promo's number is only whole with its programme, and a
				// name is not enough on its own: the same card is handed
				// out by several of them, so an answer that did not come
				// back with the number asked for is another programme's.
				if numberPrefix != "" && !sameFabNumber(co.Number, numberPrefix+product.Number) {
					continue
				}
				return id
			}
		}
	}
	return ""
}

func (mkm *Index) processProduct(channel chan<- responseChan, product *MKMProduct) error {
	var cardID string
	var cardIDFoil string
	var byName bool
	var err error

	switch mkm.gameID {
	case GameMagic:
		// An exact mcmId match ties the product to its printings more
		// reliably than name/number matching, which cannot tell apart
		// products sharing a collector number (e.g. RVR 312 vs 312z,
		// both "312" upstream); preprocess only when no id is known.
		cardID, cardIDFoil = Fallback(product)
		if cardID != "" {
			break
		}

		theCard, err := Preprocess(product.Name, product.Number, product.ExpansionName)
		if err != nil {
			_, ok := err.(*PreprocessError)
			if ok {
				return err
			}
			return nil
		}

		cardID, err = mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return nil
		} else if err != nil {
			if mtgmatcher.IsToken(theCard.Name) ||
				theCard.Edition == "Pro Tour Collector Set" ||
				strings.HasPrefix(theCard.Edition, "World Championship Decks") {
				return nil
			}

			mkm.printf("%v", err)
			mkm.printf("%q", theCard)
			mkm.printf("%v | %v | %v ", product.Name, product.ExpansionName, product.Number)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					mkm.printf("- %s", card)
				}
			}
			return err
		}

		cardIDFoil, _ = mtgmatcher.MatchID(cardID, true)
	case GameLorcana, GameRiftbound, GameOnePiece:
		// One Piece sells one card under several printings that share a
		// collector number - the alternate arts a V-index stands in for,
		// and the promo shelves that reprint a booster card at its own
		// number - and the catalog says which only by an index whose
		// order is its own. The bridge says it outright: cardtrader links
		// the product to a TCGplayer id, and the id names one printing.
		//
		// It answers first, and what it does not know the catalog still
		// names below. The bridge speaks through cardtrader's blueprints
		// and so knows only part of the shelf.
		if mkm.gameID == GameOnePiece {
			if tcgID, found := mkm.TCGBridge[product.IDProduct]; found {
				if id, idErr := mtgmatcher.MatchID(fmt.Sprint(tcgID), false); idErr == nil {
					cardID = id
					cardIDFoil, _ = mtgmatcher.MatchID(cardID, true)
					if mkm.offShelf(product, cardID) {
						return errNoPrinting
					}
					break
				}
			}
		}

		fields := strings.SplitN(product.Name, " (V.", 2)
		cardName := fields[0]
		number := product.Number
		if mkm.gameID == GameOnePiece {
			number = onePieceNumber(cardName, product.Number, product.ExpansionName)
		}
		// The V-index cardmarket synthesizes for same-number siblings is
		// how One Piece tells a base art from its variants (V.1 the base,
		// the rest its alternates), and says nothing for the finish-driven
		// games; hand it to the matcher's own rules either way. The foil
		// probes are inert for One Piece too - both flags resolve to the
		// same printing.
		if len(fields) > 1 {
			number = strings.TrimSpace(number + " V." + strings.TrimSuffix(fields[1], ")"))
		}

		cardID, err = mtgmatcher.Match(&mtgmatcher.InputCard{Name: cardName, Edition: product.ExpansionName, Variation: number, Foil: false})
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return nil
		} else if err != nil && !errors.Is(err, mtgmatcher.ErrCardWrongVariant) {
			mkm.printf("%v", err)
			mkm.printf("%+v", product)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				mkm.printf("%s got ids: %s", cardName, probes)
				for _, probe := range probes {
					co, _ := mtgmatcher.GetUUID(probe)
					mkm.printf("%s: %s", probe, co)
				}
			}
			return err
		}
		// A wrong-variant miss above may just mean the card has no nonfoil
		// printing (Match validates the finish); adopt the foil id then.
		var errFoil error
		cardIDFoil, errFoil = mtgmatcher.Match(&mtgmatcher.InputCard{Name: cardName, Edition: product.ExpansionName, Variation: number, Foil: true})
		if cardID == "" {
			cardID = cardIDFoil
		}

		if cardID == "" {
			// Neither finish matched, so the miss was genuine; the foil
			// probe's error may carry the more informative verdict
			if errFoil != nil {
				err = errFoil
			}
			mkm.printf("%v", err)
			mkm.printf("%+v", product)
			return err
		}

		// One Piece is the catalog that files one card onto shelf after
		// shelf; see offShelf.
		if mkm.gameID == GameOnePiece && mkm.offShelf(product, cardID) {
			return errNoPrinting
		}
	case GameYuGiOh, GameFleshAndBlood, GamePokemon:
		// Same-name products abound in these catalogs - and Yu-Gi-Oh and
		// Flesh and Blood carry no collector number to tell them apart,
		// though Pokemon does - so a product resolves through the
		// TCGplayer id the cardtrader bridge knows it by first, and only
		// falls back on what the catalog says of it.
		if tcgID, found := mkm.TCGBridge[product.IDProduct]; found {
			cardID, _ = mtgmatcher.MatchID(fmt.Sprint(tcgID), false)
			// The flag lands on the product's default printing, where the
			// catalog says which printing this product actually is:
			// Cardmarket sells each Flesh and Blood treatment as its own
			// product and each print run as its own expansion.
			if finish := productFinish(mkm.gameID, product); finish != "" {
				if id, ferr := mtgmatcher.MatchIDFinish(fmt.Sprint(tcgID), finish); ferr == nil {
					cardID = id
				}
			}
		}
		// The bridge speaks through cardtrader's blueprints and knows only
		// part of the catalog - half of Yu-Gi-Oh's, a third of Flesh and
		// Blood's - and what it leaves out is ordinary cards. They can be
		// named without it.
		if cardID == "" {
			cardID = mkm.matchProduct(product)
			byName = cardID != ""
		}
		if cardID == "" {
			return errNoPrinting
		}
		cardIDFoil = cardID
		if mkm.gameID == GameYuGiOh {
			// Yu-Gi-Oh's second column is the first edition's, which is a
			// print run rather than a foil, so the flag cannot name it -
			// both flags answer with the unlimited printing and the column
			// was dropped for having nowhere to attach. Naming the run
			// reaches it, and errors into an empty id for the products
			// sold in no first edition, which the guard below drops.
			cardIDFoil, _ = mtgmatcher.MatchIDFinish(cardID, "1st Edition")
		}
		if mkm.gameID == GamePokemon {
			// Pokemon's second column is the reverse holo's, which the flag
			// cannot name either: a holo rare's own printing is already a
			// foil one, so both flags answer it and the reverse beside it
			// is never reached.
			cardIDFoil, _ = mtgmatcher.MatchIDFinish(cardID, "Reverse Holofoil")
		}
	default:
		return errors.New("unsupported game")
	}

	// Look for the price presence
	guide, found := mkm.priceGuide[product.IDProduct]
	if !found {
		return fmt.Errorf("IdProduct %d not found in PriceGuide", product.IDProduct)
	}

	// Sorted as availableIndexNames
	prices := []float64{guide.LowPrice, guide.TrendPrice}
	foilLow, foilTrend := guide.SecondPrinting(mkm.gameID)
	foilprices := []float64{foilLow, foilTrend}

	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil {
		return err
	}

	// A catalog that gives each treatment its own product prices one
	// printing per product, and the product's own columns are that
	// printing's whatever its foilness - there is no second column for
	// them to be in. Every other catalog keeps the foil beside the plain
	// card and splits the two across the columns.
	perTreatment := mkm.gameID == GameFleshAndBlood || mkm.gameID == GameOnePiece

	// If card is not foil, add prices from the prices array, then check
	// if there is a foil printing, and add prices from the foilprices array.
	// If a card is foil-only or is etched, then we just use foilprices data.
	if perTreatment || (!co.Foil && !co.Etched) {
		link := BuildURL(product.IDProduct, mkm.gameID, mkm.Affiliate, false)

		quantity := product.CountArticles - product.CountFoils
		if perTreatment {
			quantity = product.CountArticles
		}

		for i := range availableIndexNames {
			if prices[i] == 0 {
				continue
			}

			out := responseChan{
				ogID:   product.IDProduct,
				cardID: cardID,
				byName: byName,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i] * mkm.exchangeRate,
					Quantity:   quantity,
					URL:        link,
					SellerName: availableIndexNames[i],
					OriginalID: fmt.Sprint(product.IDProduct),
				},
			}

			channel <- out
		}

		if !perTreatment && (foilprices[0] != 0 || foilprices[1] != 0) {
			link := BuildURL(product.IDProduct, mkm.gameID, mkm.Affiliate, true)

			// An empty foil id means the card has no foil printing (Match
			// errored on the foil probe), so residual foil prices in the
			// guide have nothing to attach to
			if cardIDFoil != "" && cardID != cardIDFoil {
				for i := range availableIndexNames {
					if foilprices[i] == 0 {
						continue
					}
					out := responseChan{
						ogID:   product.IDProduct,
						cardID: cardIDFoil,
						byName: byName,
						entry: mtgban.InventoryEntry{
							Conditions: "NM",
							Price:      foilprices[i] * mkm.exchangeRate,
							Quantity:   product.CountFoils,
							URL:        link,
							SellerName: availableIndexNames[i],
							OriginalID: fmt.Sprint(product.IDProduct),
						},
					}

					channel <- out
				}
			}
		}
	} else {
		link := BuildURL(product.IDProduct, mkm.gameID, mkm.Affiliate, true)

		for i := range availableIndexNames {
			if foilprices[i] == 0 || product.CountFoils == 0 {
				continue
			}
			out := responseChan{
				ogID:   product.IDProduct,
				cardID: cardID,
				byName: byName,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      foilprices[i] * mkm.exchangeRate,
					Quantity:   product.CountFoils,
					URL:        link,
					SellerName: availableIndexNames[i],
					OriginalID: fmt.Sprint(product.IDProduct),
				},
			}

			channel <- out
		}
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mkm *Index) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}
	mkm.exchangeRate = rate

	priceGuide, err := GetPriceGuide(ctx, mkm.gameID)
	if err != nil {
		return err
	}
	mkm.priceGuide = make(map[int]PriceGuide, len(priceGuide))
	for _, entry := range priceGuide {
		mkm.priceGuide[entry.IDProduct] = entry
	}

	mkm.printf("Obtained today's price guide with %d prices", len(priceGuide))

	list, err := mkm.client.Expansions(ctx, mkm.gameID)
	if err != nil {
		return err
	}
	list = FilterAndSortExpansions(list)

	// The non-English programs are whole separate catalogs (OP01-JP beside
	// OP01, "Metal Raiders (Korean)" beside Metal Raiders) whose prices must
	// not land on the English printings the datastore carries. Yu-Gi-Oh
	// shelves them the same way, and one more besides: the PMT tail marks
	// the European multi-language print of a set, which is a catalog of its
	// own for the same reason.
	switch mkm.gameID {
	case GameOnePiece, GameYuGiOh:
		kept := list[:0]
		for _, exp := range list {
			if strings.HasSuffix(exp.SetCode, "-JP") || foreignShelf(exp.Name) {
				continue
			}
			kept = append(kept, exp)
		}
		list = kept
		if mkm.gameID == GameOnePiece {
			mkm.shelved = shelvedSets(list)
		}
	}

	mkm.printf("Parsing %d expansion ids", len(list))

	// Pre-filter items if a target edition is set
	items := list
	if mkm.TargetEdition != "" {
		items = nil
		for _, exp := range list {
			if exp.Name == mkm.TargetEdition {
				items = append(items, exp)
			}
		}
	}

	walked, refused, unread := mkm.collectPrices(ctx, items,
		func(ctx context.Context, exp MKMExpansion, channel chan<- responseChan) error {
			mkm.printf("Processing %s (%d)", exp.Name, exp.IDExpansion)
			err := mkm.processEdition(ctx, channel, exp)
			if err != nil {
				return fmt.Errorf("expansion %s (id %d) returned %s", exp.Name, exp.IDExpansion, err.Error())
			}
			return nil
		})

	mkm.printf("Walked %d products, %d of which named no printing of ours", walked, refused)
	if unread > 0 {
		mkm.printf("%d of %d expansions never answered, and none of their products is in that count", unread, len(items))
	}
	mkm.printf("Total number of requests: %d", mkm.client.RequestNo())
	mkm.inventoryDate = time.Now()
	return nil
}

// collectPrices runs worker over every expansion and files what it produces
// into the inventory, prices whose printing was named last. It is where the
// wait namedLast describes is actually taken: the pool hands its results to
// the collector rather than to the inventory, so a named price cannot win a
// printing merely by being walked first.
func (mkm *Index) collectPrices(ctx context.Context, items []MKMExpansion, worker func(context.Context, MKMExpansion, chan<- responseChan) error) (walked, refused, unread int) {
	// The bridge is keyed by the Cardmarket id and valued by the TCGplayer
	// one, and a cardtrader blueprint names every Cardmarket product it
	// sells as, so nothing stops two products from resolving to one
	// printing. An index wants a single price per name per uuid, and a
	// second one is worth the log line the callback already prints rather
	// than a second row no consumer can choose between.
	add := mkm.inventory.AddStrict
	switch mkm.gameID {
	case GameYuGiOh, GameFleshAndBlood, GamePokemon:
		add = mkm.inventory.AddUnique
	}

	addOne := func(result responseChan) {
		err := add(result.cardID, &result.entry)
		if err != nil {
			card, cerr := mtgmatcher.GetUUID(result.cardID)
			if cerr != nil {
				mkm.printf("%d - %s: %s", result.ogID, cerr.Error(), result.cardID)
				return
			}
			// Skip too many errors
			if mtgmatcher.IsToken(card.Name) ||
				card.Edition == "Pro Tour Collector Set" ||
				strings.HasPrefix(card.Edition, "World Championship Decks") {
				return
			}
			mkm.printf("%d - %s", result.ogID, err.Error())
		}
	}

	collector := namedLast{add: addOne}

	mtgban.WorkerPool(ctx, mkm.MaxConcurrency, items, worker, collector.collect, mkm.printf)

	mkm.printf("Adding %d prices whose printing was named", collector.flush())
	return collector.walked, collector.refused, collector.unread
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mkm *Index) Inventory() mtgban.InventoryRecord {
	return mkm.inventory
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (mkm *Index) MarketNames() []string {
	return availableIndexNames
}

// InfoForScraper describes one of the sub-scrapers named above.
func (mkm *Index) InfoForScraper(name string) mtgban.ScraperInfo {
	info := mkm.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (mkm *Index) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Market Index"
	info.Shorthand = "MKMIndex"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.MetadataOnly = true
	info.Family = "MKM"
	switch mkm.gameID {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	case GameRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameOnePiece:
		info.Game = mtgban.GameOnePiece
	case GameYuGiOh:
		info.Game = mtgban.GameYuGiOh
	case GameFleshAndBlood:
		info.Game = mtgban.GameFleshAndBlood
	case GamePokemon:
		info.Game = mtgban.GamePokemon
	}
	return
}
