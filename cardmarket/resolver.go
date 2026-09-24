package cardmarket

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// resolver is the Cardmarket-id -> mtgban-uuid resolution Index and Market
// share: naming the printing(s) a product's fields (name, number, expansion)
// belong to, id-map first and then name/number matching, per game. Neither
// scraper differs in how hard that problem is - each game's shelving quirks,
// twins, foreign catalogs and print runs are the same whichever scraper asks
// - they differ only in what they do with an answer once they have one:
// Index reads it off the published price guide, Market queries the
// product's own live listings.
//
// Embedded anonymously, not held by reference: a scraper's own field access
// (mkm.tcgBridge = ...) and its own calls (mkm.resolveProduct(...)) both
// promote through unchanged, so this extraction changed no call site in
// Index - only where the fields and methods are defined.
type resolver struct {
	// TCGBridge maps a Cardmarket product id to the TCGplayer id of the
	// same single, for the catalogs whose same-name products are told
	// apart only by an exact id: Yu-Gi-Oh and Flesh and Blood carry no
	// collector number, Pokemon sells one card on shelf after shelf, and
	// One Piece files the printings a number cannot tell apart under a
	// version index of its own. bantool builds it from cardtrader's
	// blueprints, the one source linking the two marketplaces; the scraper
	// itself stays vendor-pure and receives it as plain data.
	tcgBridge map[int]int

	// TargetEdition optionally restricts resolution to a single edition.
	targetEdition string

	// Catalog is the published id-map catalog a scraper may resolve from
	// before falling back to name/number matching. bantool loads it from
	// MTGJSON_MKMID_PATH; nil is a resolver with no id map at all, which
	// checkCatalog refuses to walk when the game needs one to walk safely.
	catalog *cm.Catalog

	// numbers indexes a set's collector numbers by the card they name,
	// built on first use; see yugiohNumberTaken.
	numbers   map[string]map[string]string
	numbersMu sync.Mutex

	// fabDeckSets indexes a Flesh and Blood set's collector-number prefix
	// onto every set that opens numbers on it, built on first use; see
	// fabShelves and fabDeckSetIndex.
	fabDeckSets   map[string][]*mtgmatcher.Set
	fabDeckSetsMu sync.Mutex

	// shelved names, for each set of ours, the expansion of this run that
	// sells it; see offShelf. A scraper's Load fills it once the
	// expansions are known, via shelvedSets.
	shelved map[string]string

	// printf logs through the owning scraper's own prefix. Left nil, calls
	// through it are no-ops - a resolver built for a test rather than a
	// live scraper needs none.
	printf func(format string, a ...any)

	gameID cm.Game

	// backend is the datastore Index and Market match against, set by
	// their constructors.
	backend *mtgmatcher.Backend
}

func (r *resolver) logf(format string, a ...any) {
	if r.printf != nil {
		r.printf(format, a...)
	}
}

// errNoPrinting marks a product no route named a printing of ours for. It
// is what the id-and-name route answers with instead of nothing at all, so a
// refusal is counted and said out loud rather than passing for a success.
var errNoPrinting = errors.New("named no printing of ours")

// errTwin marks a product named and numbered like another of its expansion,
// or like a printing already priced through a different shelf, either way
// leaving the catalog silent on which variant of the card it is.
var errTwin = errors.New("twin of another product")

// errForeign marks a product of a catalog the datastore does not carry.
var errForeign = errors.New("of a catalog we do not carry")

// noPrinting answers the error a product no route named a printing for
// refuses with - which for a Pokemon basic energy is no error at all.
//
// Every set prints the nine basic energies and reprints them unchanged for
// years, so the catalogs shelve them where no printing of ours is separable:
// by the year they were printed, or on a trainer kit's own shelf, one product
// standing for a card a dozen sets carry. Naming those is the same noise
// Magic's tokens are, and resolveMagic already passes over them for the same
// reason. They are 57 of the 177 lines a Pokemon run still reports.
func (r *resolver) noPrinting(product *cm.Product) error {
	if r.gameID == cm.GamePokemon && pokemonBasicEnergy(pokemonName(product.Name)) {
		return nil
	}
	return errNoPrinting
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
func (r *resolver) offShelf(product *cm.Product, cardID string) bool {
	co, err := r.backend.GetUUID(cardID)
	if err != nil || len(co.PromoTypes) > 0 {
		return false
	}
	shelf, found := r.shelved[co.SetCode]
	return found && !strings.EqualFold(shelf, product.ExpansionName)
}

// ownedElsewhere reports whether the datastore records cardID's printing
// under a Cardmarket product other than this one.
func (r *resolver) ownedElsewhere(product *cm.Product, cardID string) bool {
	co, err := r.backend.GetUUID(cardID)
	if err != nil {
		return false
	}
	mcmID := co.Identifiers["mcmId"]
	return mcmID != "" && mcmID != fmt.Sprint(product.IDProduct)
}

// matchFab resolves a Flesh and Blood product the bridge does not know,
// from what the catalog says of it. The edition has to name a set of ours
// and the answer has to be in it: Cardmarket carries whole catalogs the
// datastores do not, and Match reaches past the edition when nothing in it
// fits, so without both an unknown set's cards land on whichever set happens
// to hold a number like theirs. Pokemon and Yu-Gi-Oh name their products
// through matchPokemon and matchYugioh, and every other game through the
// matcher alone; see resolveProduct.
func (r *resolver) matchFab(product *cm.Product) string {
	fabRenamed(product)

	r.fabDeckSetsMu.Lock()
	if r.fabDeckSets == nil {
		r.fabDeckSets = fabDeckSetIndex(r.backend)
	}
	deckSets := r.fabDeckSets
	r.fabDeckSetsMu.Unlock()

	shelves := fabShelves(r.backend, product, deckSets)
	if len(shelves) == 0 {
		return ""
	}
	// The treatment parenthetical is the printing's, not the name's:
	// fabFinish reads it off the untouched product name below, and a card
	// whose own name ends in a parenthetical ("Sink Below (Red)") keeps
	// it, so the exact name reaches the matcher whole. The art ahead of
	// the treatment stays on the name for the sets that file it as a
	// printing of its own, and comes off for the sets that do not. The raw
	// name stays as the fallback: the sets spelling a pitch color the other
	// one's way ("Rawhide Rumble" at ARR012, "Rawhide Rumble (Red)" at
	// HVY023) file the stripped name under the wrong set, and only the
	// decorated one still finds them.
	names := []string{versionTail.ReplaceAllString(product.Name, "")}
	_, stripped := fabTreatment(names[0])
	if stripped != names[0] {
		names = []string{stripped}
		if plain := fabDropArt(stripped); plain != stripped {
			names = append(names, plain)
		}
		names = append(names, versionTail.ReplaceAllString(product.Name, ""))
	}
	// A double-sided card is filed under both faces and, in the treatments
	// the fused row was never sold in, under the front face alone: Aether
	// Ashwing // Ash is plain, and the cold foil is Aether Ashwing's.
	if front, _, fused := strings.Cut(names[0], " // "); fused && !strings.Contains(product.Number, "/") {
		names = append(names, front)
	}

	// A game selling one card in several print runs needs one of them
	// named, or the runs alias and nothing resolves. A product naming its
	// treatment names the printing it is, and that printing is asked for
	// first wherever the shelves keep it; the card's own printing stands
	// only when no shelf carries the named one, the card being agreed on
	// and the finish the one disagreement.
	finishes := []string{fabFinish(product.ExpansionName, product.Name), ""}

	// The named finish is asked for on every shelf before the plain
	// printing is settled for on any, so a card the fused row was never
	// sold cold in is found under its front face rather than priced as
	// the plain fused card.
	for _, finish := range finishes {
		for _, sh := range shelves {
			set, edition, numberPrefix, printRun := sh.set, sh.edition, sh.numberPrefix, sh.printRun
			// A promo's programme is the prefix our numbering carries
			// and the expansion Cardmarket sells it under, so the number
			// is only whole once the two are put back together; and any
			// set numbering its cards on letters of its own is asked with
			// them, since a fused card answers to its faces' numbers only
			// when they are whole.
			prefix := numberPrefix
			if prefix == "" {
				prefix = fabSetPrefix(set)
			}
			numbers := fabNumbers(prefix, product.Number)

			// A number is asked of every spelling before a looser number
			// is asked of any: the name alone comes last, after every
			// spelling has been tried at the number, so a wording the set
			// files no printing under cannot slip past the number onto
			// the card's plainest printing.
			for _, number := range numbers {
				for _, name := range names {
					id, err := r.backend.Match(&mtgmatcher.InputCard{
						Name:      name,
						Edition:   edition,
						Variation: number,
						Finish:    finish,
					})
					if err != nil {
						continue
					}
					co, cerr := r.backend.GetUUID(id)
					if cerr != nil || !strings.EqualFold(co.SetCode, set.Code) {
						continue
					}
					// The run has to hold too, for the same reason the
					// set does: a card the datastore keeps in one run
					// only is answered with that run whichever was asked
					// for, and the other run's expansion sells the very
					// same card.
					if printRun != "" && !strings.HasPrefix(co.Finish, mtgmatcher.NormalizeFinish(printRun)) {
						continue
					}
					// A promo's number is only whole with its programme,
					// and a name is not enough on its own: the same card
					// is handed out by several of them, so an answer that
					// did not come back with the number asked for is
					// another programme's.
					if numberPrefix != "" && !sameFabFace(co.Number, numberPrefix+product.Number) {
						continue
					}
					return id
				}
			}
		}
	}
	return ""
}

// resolveMagic answers a product with the printings its two price columns
// belong to. An empty id under a nil error means the product names nothing
// this datastore carries, which is a skip rather than a failure.
func (r *resolver) resolveMagic(product *cm.Product) (string, string, error) {
	// An exact mcmId match ties the product to its printings more
	// reliably than name/number matching, which cannot tell apart
	// products sharing a collector number (e.g. RVR 312 vs 312z,
	// both "312" upstream); preprocess only when no id is known.
	cardID, cardIDFoil := Fallback(r.backend, product)
	if cardID != "" {
		return cardID, cardIDFoil, nil
	}

	// A two-sided token sheet's own product name ("Bird Token (W 1/1) //
	// Spirit Token (W 1/1)") is not one Preprocess/Match below was ever
	// built to read. Cardmarket's own product Number ("T 2/6") is a
	// catalog ordinal, not a collector number, so unlike Cool Stuff Inc's
	// own feed there is no set+number anchor available here - both faces'
	// own names plus the product's edition (magic.MatchTokenPairingByNamesAndEdition,
	// already refusing rather than guessing whenever a name repeats across
	// several same-named tokens in one edition) is the only anchor this
	// vendor's own data gives. Resolved or not, this listing is done here:
	// falling into Preprocess/Match below would only refuse it again, more
	// noisily.
	if strings.Contains(product.Name, "Token") && strings.Contains(product.Name, " // ") {
		if pairID := magic.MatchTokenPairingByNamesAndEdition(r.backend, product.Name, product.ExpansionName, false); pairID != "" {
			cardID, _ = r.backend.MatchID(pairID, false)
		}
		if pairIDFoil := magic.MatchTokenPairingByNamesAndEdition(r.backend, product.Name, product.ExpansionName, true); pairIDFoil != "" {
			cardIDFoil, _ = r.backend.MatchID(pairIDFoil, true)
		}
		return cardID, cardIDFoil, nil
	}

	theCard, err := Preprocess(r.backend, product.Name, product.Number, product.ExpansionName)
	if err != nil {
		_, ok := err.(*PreprocessError)
		if ok {
			return "", "", err
		}
		return "", "", nil
	}

	cardID, err = r.backend.Match(theCard)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return "", "", nil
	} else if err != nil {
		if r.backend.IsToken(theCard.Name) ||
			theCard.Edition == "Pro Tour Collector Set" ||
			strings.HasPrefix(theCard.Edition, "World Championship Decks") {
			return "", "", nil
		}

		r.logf("%v", err)
		r.logf("%q (%d)", theCard, product.IDProduct)
		r.logf("%v | %v | %v ", product.Name, product.ExpansionName, product.Number)

		var alias *mtgmatcher.AliasingError
		if errors.As(err, &alias) {
			probes := alias.Probe()
			for _, probe := range probes {
				card, _ := r.backend.GetUUID(probe)
				r.logf("- %s", card)
			}
		}
		return "", "", err
	}

	co, cerr := r.backend.GetUUID(cardID)
	switch {
	case cerr != nil:
		cardIDFoil, _ = r.backend.MatchID(cardID, true)
	case co.Etched:
		// No separate guide column for etched; keep it off the plain
		// foil sibling MatchID(cardID, true) would otherwise pick.
		cardIDFoil = cardID
	case foilOnlyShelf(product, co.SetCode):
		// The guide duplicates one price into both columns; redirect
		// both ids onto the foil twin actually sold.
		fid, ferr := r.backend.MatchID(cardID, true)
		if ferr == nil && fid != "" {
			cardID = fid
		}
		cardIDFoil = cardID
	default:
		cardIDFoil, _ = r.backend.MatchID(cardID, true)
	}
	return cardID, cardIDFoil, nil
}

// foilOnlyShelf reports whether product's shelf sells only the foil shown,
// at a number an ordinary nonfoil printing also carries. setCode holds the
// Holiday Release rule to the two sets whose "(V.2)" is the silverfoil of
// its "(V.1)": LTC's box topper and LTR's scroll showcase. Modern Horizons 3
// Extras' "(V.3)" onwards are sold in both finishes or at a number of their
// own. Closed on purpose: widening it would misprice a shelf that
// legitimately sells both finishes.
func foilOnlyShelf(product *cm.Product, setCode string) bool {
	switch product.ExpansionName {
	case "Commander: Magic: The Gathering - FINAL FANTASY: Collector's Edition",
		"Commander: Marvel Super Heroes: Collector's Edition",
		"Commander: Teenage Mutant Ninja Turtles: Extras":
		return true
	case "Commander: Modern Horizons 3: Extras":
		return strings.HasSuffix(product.Name, "(V.2)") || !strings.Contains(product.Name, "(V.")
	case "The Lord of the Rings: Tales of Middle-earth Holiday Release":
		return (setCode == "LTC" || setCode == "LTR") && strings.HasSuffix(product.Name, "(V.2)")
	}
	return false
}

// resolveProduct answers a product with the printings its two price columns
// belong to, whatever the game. An empty id under a nil error means the
// product names nothing this datastore carries, which is a skip rather than
// a failure.
func (r *resolver) resolveProduct(product *cm.Product) (string, string, bool, error) {
	var cardID string
	var cardIDFoil string
	var byName bool
	var err error

	switch r.gameID {
	case cm.GameMagic:
		cardID, cardIDFoil, err = r.resolveMagic(product)
		if err != nil || cardID == "" {
			return "", "", false, err
		}
	case cm.GameLorcana, cm.GameRiftbound, cm.GameOnePiece:
		if r.gameID == cm.GameLorcana && lorcanaFaces[product.IDProduct] {
			return "", "", false, nil
		}
		// A product the datastore records by id is that printing: the
		// pre-errata One Piece cards no TCGplayer product sells, or a
		// Lorcana card Cardmarket names its own way.
		uuid := r.backend.ConvertID(mtgmatcher.IDSpaceCardmarket, fmt.Sprint(product.IDProduct))
		if uuid != "" {
			cardID = uuid
			cardIDFoil, _ = r.backend.MatchID(cardID, true)
			break
		}
		// The bridge answers next, naming one printing where the
		// catalog's own V-index or wording cannot. One Piece takes it
		// outright; Riftbound and Lorcana only where the name still agrees.
		if tcgID, found := r.tcgBridge[product.IDProduct]; found {
			id, idErr := r.backend.MatchID(fmt.Sprint(tcgID), false)
			if idErr == nil && r.gameID != cm.GameOnePiece && !bridgeNamesCard(r.backend, product, id) {
				idErr = errNoPrinting
			}
			if idErr == nil {
				cardID = id
				cardIDFoil, _ = r.backend.MatchID(cardID, true)
				// Lorcana files a foil TCGplayer sells apart under the plain
				// card's row, and a product linked to it sells the foil alone.
				co, coErr := r.backend.GetUUID(cardID)
				if r.gameID == cm.GameLorcana && coErr == nil && cardIDFoil != "" &&
					co.Identifiers["tcgplayerProductId"] != fmt.Sprint(tcgID) {
					cardID = cardIDFoil
				}
				if r.offShelf(product, cardID) {
					return "", "", false, errNoPrinting
				}
				break
			}
		}

		fields := strings.SplitN(product.Name, " (V.", 2)
		cardName := fields[0]
		number := product.Number
		if r.gameID == cm.GameOnePiece {
			number = onePieceNumber(r.backend, cardName, product.Number, product.ExpansionName)
		}
		// The V-index cardmarket synthesizes for same-number siblings is
		// how One Piece tells a base art from its variants (V.1 the base,
		// the rest its alternates); hand it to the matcher's own rules
		// either way. Lorcana reads it too, but only as a last-resort
		// tiebreak among same-named Special printings a bare number cannot
		// separate - a promo Ravensburger reprints wave to wave, each wave
		// its own locally-numbered pool - so it is otherwise inert. The
		// foil probes are inert for One Piece too - both flags resolve to
		// the same printing.
		if len(fields) > 1 {
			number = strings.TrimSpace(number + " V." + strings.TrimSuffix(fields[1], ")"))
		}
		// Only the rarity tells an oversized card from the set's own at its
		// number; a card with no oversized printing is then unsupported.
		if product.Rarity == "Oversized" {
			number = strings.TrimSpace(number + " Oversized")
		}

		cardID, err = r.backend.Match(&mtgmatcher.InputCard{Name: cardName, Edition: product.ExpansionName, Variation: number, Foil: false})
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return "", "", false, nil
		} else if err != nil && !errors.Is(err, mtgmatcher.ErrCardWrongVariant) {
			// One Piece's misses are judged beside the product's
			// siblings first - a version of a card another version
			// already priced is a twin, not a report - so the walk says
			// what is left to say; see twinsAmong.
			if r.gameID == cm.GameOnePiece {
				return "", "", false, err
			}
			r.logf("%v", err)
			r.logf("%+v", product)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				r.logf("%s got ids: %s", cardName, probes)
				for _, probe := range probes {
					co, _ := r.backend.GetUUID(probe)
					r.logf("%s: %s", probe, co)
				}
			}
			return "", "", false, err
		}
		// A wrong-variant miss above may just mean the card has no nonfoil
		// printing (Match validates the finish); adopt the foil id then.
		var errFoil error
		cardIDFoil, errFoil = r.backend.Match(&mtgmatcher.InputCard{Name: cardName, Edition: product.ExpansionName, Variation: number, Foil: true})
		if cardID == "" {
			cardID = cardIDFoil
		}

		if cardID == "" {
			// Neither finish matched, so the miss was genuine; the foil
			// probe's error may carry the more informative verdict
			if errFoil != nil {
				err = errFoil
			}
			if r.gameID != cm.GameOnePiece {
				r.logf("%v", err)
				r.logf("%+v", product)
			}
			return "", "", false, err
		}

		// A starter deck's exclusive foil is a V.N at its card's own number,
		// filed by the datastore as that card's Holofoil.
		if r.gameID == cm.GameLorcana && len(fields) > 1 && strings.TrimSuffix(fields[1], ")") != "1" {
			holo, holoErr := r.backend.MatchIDFinish(cardID, "Holofoil")
			if holoErr == nil && holo != cardID && holo != cardIDFoil {
				cardID, cardIDFoil = holo, holo
			}
		}

		// One Piece is the catalog that files one card onto shelf after
		// shelf; see offShelf.
		if r.gameID == cm.GameOnePiece && r.offShelf(product, cardID) {
			return "", "", false, errNoPrinting
		}
		// A pre-errata printing the datastore records under a sibling
		// product is that sibling's, whatever the name reaches.
		if r.gameID == cm.GameOnePiece && r.ownedElsewhere(product, cardID) {
			return "", "", false, errNoPrinting
		}
	case cm.GameYuGiOh, cm.GameFleshAndBlood, cm.GamePokemon:
		// Same-name products abound in these catalogs - and Yu-Gi-Oh and
		// Flesh and Blood carry no collector number to tell them apart,
		// though Pokemon does - so a product resolves through the
		// TCGplayer id the cardtrader bridge knows it by first, and only
		// falls back on what the catalog says of it.
		if r.gameID == cm.GamePokemon && pokemonCodeCard(product.Name) {
			return "", "", false, nil
		}
		if r.gameID == cm.GameYuGiOh {
			product, err = yugiohPrintNumber(product)
			if err != nil {
				return "", "", false, err
			}
		}
		// The id names the card, and the product's own wording names the
		// printing: Cardmarket sells each Flesh and Blood treatment as its
		// own product and each print run as its own expansion, and the
		// flag alone lands on the id's default printing. The printing the
		// wording names is asked for first, of the id and then of the
		// name, since the id's card may carry it under another row; only
		// when nothing carries it does the id's own printing stand, the
		// card being agreed on and the finish the one disagreement.
		var loose string
		if tcgID, found := r.tcgBridge[product.IDProduct]; found {
			cardID, _ = r.backend.MatchID(fmt.Sprint(tcgID), false)
			if finish := productFinish(r.gameID, product); finish != "" && cardID != "" {
				loose = cardID
				cardID, _ = r.backend.MatchIDFinish(fmt.Sprint(tcgID), finish)
			}
		}
		// CardTrader's tcg id can name a sibling rarity, another card,
		// another printing's number or another set; distrust it.
		if r.gameID == cm.GameYuGiOh && cardID != "" {
			if r.yugiohOtherCard(product, cardID) || r.yugiohOtherNumber(product, cardID) {
				cardID = ""
			} else {
				id := r.yugiohWorded(product, cardID)
				if id == "" {
					id = r.yugiohShelfSet(product, cardID)
				}
				if id != "" {
					cardID, byName = id, true
				}
			}
		}
		// The bridge speaks through cardtrader's blueprints and knows only
		// part of the catalog - half of Yu-Gi-Oh's, a third of Flesh and
		// Blood's - and what it leaves out is ordinary cards. They can be
		// named without it.
		if cardID == "" {
			switch r.gameID {
			case cm.GamePokemon:
				cardID, err = r.matchPokemon(product)
			case cm.GameYuGiOh:
				cardID, err = r.matchYugioh(product)
			}
			if err != nil {
				if errors.Is(err, errNoPrinting) {
					err = r.noPrinting(product)
				}
				return "", "", false, err
			}
			byName = cardID != ""
		}
		if cardID == "" && r.gameID == cm.GameFleshAndBlood {
			cardID = r.matchFab(product)
			byName = cardID != ""
		}
		if cardID == "" {
			cardID = loose
		}
		if cardID == "" {
			return "", "", false, r.noPrinting(product)
		}
		cardIDFoil = cardID
		if r.gameID == cm.GameYuGiOh {
			// Yu-Gi-Oh's second printing is the first edition, a print run
			// rather than a foil, so the flag cannot name it. Naming the
			// run reaches it, and errors into an empty id for the products
			// sold in no first edition, which the guard below drops.
			cardIDFoil, _ = r.backend.MatchIDFinish(cardID, "1st Edition")
		}
		if r.gameID == cm.GamePokemon {
			// Pokemon's second column is the reverse holo's, which the flag
			// cannot name either: a holo rare's own printing is already a
			// foil one, so both flags answer it and the reverse beside it
			// is never reached. This is Index's own use of cardIDFoil -
			// resolveProduct is shared between the two scrapers (see the
			// package doc above) - and it stays exactly as it was: Market's
			// own path (queryPokemonPrintings, pokemonFinishPlan) resolves
			// cardID's whole finish set fresh from the matcher instead of
			// through this pair, and simply never reads cardIDFoil for this
			// game, so leaving it filled here costs Market nothing while
			// Index still depends on it.
			cardIDFoil, _ = r.backend.MatchIDFinish(cardID, pokemonReverseHolo)
		}
	default:
		return "", "", false, errors.New("unsupported game")
	}

	return cardID, cardIDFoil, byName, nil
}

// pokemonReverseHolo is the printing Pokemon's guide prices in its second
// pair of columns, as TCGplayer names it.
const pokemonReverseHolo = "Reverse Holofoil"

// resolveUUIDs answers a product from the printings its map entry lists,
// splitting them by finish the way the guide's columns are split. Ids the
// datastore does not carry are passed over - a double-faced card lists its
// back face too, and the index knows only fronts. Within a finish the
// printing whose number agrees with the product's wins, the way Fallback
// already prefers it; a pick between printings the number cannot settle is
// said out loud. Both ids empty means the entry decided nothing.
func (r *resolver) resolveUUIDs(product *cm.Product, uuids []string) (string, string) {
	var plain, foil []string
	var plainMatched, foilMatched bool
	for _, uuid := range uuids {
		co, err := r.backend.GetUUID(uuid)
		if err != nil {
			continue
		}
		// See plausiblePrinting: a Magic WCD or Oversized product's map
		// entry can carry an id mtgjson has wrongly linked, the same
		// drift Fallback's own mcmId route guards against.
		if r.gameID == cm.GameMagic && !plausiblePrinting(r.backend, product.ExpansionName, uuid) {
			continue
		}
		sameNumber := strings.EqualFold(co.PlainNumber, product.Number)
		if co.Foil || co.Etched {
			if sameNumber && !foilMatched {
				foil = append([]string{uuid}, foil...)
				foilMatched = true
			} else {
				foil = append(foil, uuid)
			}
		} else {
			if sameNumber && !plainMatched {
				plain = append([]string{uuid}, plain...)
				plainMatched = true
			} else {
				plain = append(plain, uuid)
			}
		}
	}

	if len(plain) > 1 && !plainMatched {
		r.logf("id %d %q lists %d plain printings and the number settles none; keeping %s",
			product.IDProduct, product.Name, len(plain), plain[0])
	}
	if len(foil) > 1 && !foilMatched {
		r.logf("id %d %q lists %d foil printings and the number settles none; keeping %s",
			product.IDProduct, product.Name, len(foil), foil[0])
	}

	var cardID, cardIDFoil string
	switch {
	case len(plain) > 0:
		cardID = plain[0]
		if len(foil) > 0 {
			cardIDFoil = foil[0]
		} else {
			// The entry lists no foil printing, but the datastore may
			// still carry one, the way resolveProduct probes for it.
			cardIDFoil, _ = r.backend.MatchID(cardID, true)
		}
	case len(foil) > 0:
		// A foil-only product prices through its own columns; both ids
		// point to it, the way Fallback answers a single printing.
		cardID = foil[0]
		cardIDFoil = foil[0]
	}

	// The map names the printing, not the finish sold; redirect a
	// foil-only shelf's product the way resolveMagic does.
	co, err := r.backend.GetUUID(cardID)
	if err == nil && !co.Etched && foilOnlyShelf(product, co.SetCode) {
		fid, ferr := r.backend.MatchID(cardID, true)
		if ferr == nil && fid != "" {
			cardID = fid
		}
		cardIDFoil = cardID
	}
	return cardID, cardIDFoil
}

// resolveMapped answers one product of the id map. The map answers first;
// what it left unmapped is answered from what the catalog says of it, by
// resolveProduct, so a product the file does not know yet is matched rather
// than lost.
func (r *resolver) resolveMapped(id int, mapped cm.CatalogProduct, expansion cm.Expansion) resolved {
	product := &cm.Product{
		IDProduct:     id,
		Name:          mapped.Name,
		Number:        mapped.Number,
		Rarity:        mapped.Rarity,
		ExpansionName: expansion.Name,
		ExpansionCode: expansion.SetCode,
	}
	product.Expansion.IDExpansion = expansion.IDExpansion

	cardID, cardIDFoil := r.resolveUUIDs(product, mapped.UUIDs)
	if cardID != "" {
		return resolved{product: product, cardID: cardID, cardIDFoil: cardIDFoil}
	}
	cardID, cardIDFoil, byName, err := r.resolveProduct(product)
	return resolved{product: product, cardID: cardID, cardIDFoil: cardIDFoil, byName: byName, err: err}
}

// refusalName is the card a refused product names, as the report files it:
// a Pokemon product carries its attacks and energy symbols in brackets the
// card's name does not, and every other game's product name is the card's.
func (r *resolver) refusalName(name string) string {
	if r.gameID == cm.GamePokemon {
		return pokemonName(name)
	}
	return name
}

// checkCatalog reports whether the id map can be walked. For the games that
// shelve whole foreign catalogs, the map says which shelves those are only
// through the expansion codes: a map written before it carried them cannot
// be walked safely, and the run refuses rather than price the foreign
// printings onto the English ones.
func (r *resolver) checkCatalog() error {
	if r.catalog == nil {
		return errors.New("no id map to price from")
	}
	switch r.gameID {
	case cm.GameOnePiece, cm.GameYuGiOh, cm.GameFleshAndBlood:
	default:
		return nil
	}
	for _, expansion := range r.catalog.Data.Expansions {
		if expansion.Code != "" {
			return nil
		}
	}
	return errors.New("the id map carries no expansion codes to tell the foreign shelves by")
}

// matchPokemon names a Pokemon product's printing from what the catalog
// says of it, held to the sets its expansion may hold. An expansion naming
// no set of ours is a catalog we do not carry, and a miss in one is said so
// rather than reported product by product.
func (r *resolver) matchPokemon(product *cm.Product) (string, error) {
	if pokemonForeignDenied(product.ExpansionName, product.Number) {
		return "", errForeign
	}
	name := pokemonName(product.Name)
	type candidate struct {
		edition, number string
		prefixed        bool
	}
	var candidates []candidate
	editions, prefix := pokemonEditions(r.backend, product.ExpansionName)
	number := product.Number
	if prefix != "" && number != "" {
		number = prefix + number
	}
	for _, edition := range editions {
		candidates = append(candidates, candidate{edition, number, prefix != ""})
	}
	if pokemonLettered.MatchString(number) {
		for _, edition := range pokemonLetteredSets {
			candidates = append(candidates, candidate{edition, number, false})
		}
	}
	if m := pokemonPromoNumber.FindStringSubmatch(product.Number); m != nil {
		if shelf, found := pokemonPromoCodes[m[1]]; found {
			promo := pokemonExpansions[shelf]
			candidates = append(candidates, candidate{promo.sets[0], promo.prefix + m[2], promo.prefix != ""})
		}
	}
	carried := false
	for _, c := range candidates {
		set, err := r.backend.GetSetByName(c.edition)
		if err != nil {
			continue
		}
		carried = true
		id, err := r.backend.Match(&mtgmatcher.InputCard{Name: name, Edition: c.edition, Variation: c.number})
		if err != nil {
			continue
		}
		co, err := r.backend.GetUUID(id)
		if err != nil || !strings.EqualFold(co.SetCode, set.Code) {
			continue
		}
		if c.prefixed && !strings.EqualFold(co.Number, c.number) {
			continue
		}
		return id, nil
	}
	if !carried || pokemonForeign(product.ExpansionName) {
		return "", errForeign
	}
	return "", errNoPrinting
}

// matchYugioh names a Yu-Gi-Oh product's printing from what the catalog
// says of it, held to the sets its expansion may hold. A number written with
// a region prefix names a print run of its own ("EN000" is the European
// print, "A000" the Asian), carried only where the catalog has a set for it;
// an expansion naming no set of ours, or a run we have no set for, is a
// catalog we do not carry rather than a product that named nothing.
func (r *resolver) matchYugioh(product *cm.Product) (string, error) {
	name := versionTail.ReplaceAllString(product.Name, "")
	var rarity string
	if fields := rarityTail.FindStringSubmatch(product.Name); fields != nil {
		rarity = fields[1]
	}
	// The oversized printing is not in the shelf's set; only the name and
	// the tag reach it. See yugiohOversized.
	if strings.EqualFold(rarity, "Oversized") {
		if id, err := yugiohOversized(r.backend, name); err == nil {
			return id, nil
		}
	}
	region := ""
	if product.Number != "" {
		region = numberPrefix(product.Number)
	}
	tail := numberTail.FindString(product.Number)

	carried := false
	for _, edition := range yugiohEditions(product.ExpansionName) {
		set, err := r.backend.GetSetByName(edition)
		if err != nil {
			continue
		}
		carried = true
		sets := []*mtgmatcher.Set{set}
		// The European print is a set of its own where the catalog has
		// one, numbered with the region the product writes.
		if region == "EN" {
			worldwide, err := r.backend.GetSet(set.Code + "-EN")
			if err == nil {
				sets = []*mtgmatcher.Set{worldwide}
			}
		}
		for _, set := range sets {
			numbers := []string{product.Number}
			if tail != "" {
				base := strings.TrimSuffix(set.Code, "-EN")
				if region == "" {
					// The catalog writes the region into most sets'
					// numbers ("MP18-EN065") and Cardmarket leaves it
					// out; both spellings are asked.
					numbers = append(numbers, base+"-"+tail, base+"-EN"+tail)
				} else {
					numbers = append(numbers, base+"-"+region+tail)
				}
			}
			// A number the set gives to another card is the storefront's
			// mistake rather than the card's: the oldest sets are sold
			// once more under a numbering shifted by a card or two, and
			// the name is what still says which card it is. Only a
			// number so contradicted is set aside; a number the set
			// merely lacks stays a refusal.
			if tail != "" && r.yugiohNumberTaken(set.Code, numbers[1:], name) {
				numbers = append(numbers, "")
			}
			for _, number := range numbers {
				variation := strings.TrimSpace(number + " " + rarity)
				// A shelf whose printings differ only by a mark the
				// storefront does not name takes it from the version
				// index; see yugiohVersionVariants on what that order is
				// worth. A version the table does not cover is left
				// alone, and aliases as it did before.
				if labels := yugiohVersionVariants[strings.TrimSuffix(set.Code, "-EN")]; labels != nil {
					if index := cm.ProductVersion(product); index >= 1 && index <= len(labels) {
						variation = strings.TrimSpace(variation + " " + labels[index-1])
					}
				}
				// The run is a flag on each Cardmarket listing, not a
				// product of its own, so the card's default run answers.
				id, err := r.backend.Match(&mtgmatcher.InputCard{
					Name:      name,
					Edition:   set.Name,
					Variation: variation,
				})
				if err != nil {
					continue
				}
				co, err := r.backend.GetUUID(id)
				if err != nil || !strings.EqualFold(co.SetCode, set.Code) {
					continue
				}
				if number != "" && otherPrintRun(product.Number, co.Number) {
					continue
				}
				return id, nil
			}
		}
	}
	if carried && region != "" && tail != "" {
		id := r.yugiohInfixed(product, name, rarity, region, tail)
		if id != "" {
			return id, nil
		}
	}
	if !carried || region != "" {
		return "", errForeign
	}
	if strings.HasPrefix(product.ExpansionName, "OTS Tournament Pack") && r.yugiohPastEnd(product.ExpansionName, tail) {
		return "", errForeign
	}
	if r.yugiohEuropean(name, tail) {
		return "", errTwin
	}
	return "", errNoPrinting
}

// yugiohPlainSuffix matches the plain digits a set's own collector number
// ends on, after any "EN" region infix, for comparing against a tail
// Cardmarket writes with no infix of its own.
var yugiohPlainSuffix = regexp.MustCompile(`-(?:EN)?(\d+)$`)

// yugiohPastEnd reports whether an OTS Tournament Pack's number tail runs
// past the last one any set carrying its expansion prints - the Portuguese
// packs' extra numbers. Scoped to OTS so a number a set merely lacks stays a
// refusal elsewhere (Gouki Re-Match in TestMatchYugiohShelves).
func (r *resolver) yugiohPastEnd(expansion, tail string) bool {
	n, err := strconv.Atoi(tail)
	if err != nil {
		return false
	}
	numbers := r.yugiohNumbers()
	var sets int
	for _, edition := range yugiohEditions(expansion) {
		set, err := r.backend.GetSetByName(edition)
		if err != nil {
			continue
		}
		sets++
		for number := range numbers[set.Code] {
			m := yugiohPlainSuffix.FindStringSubmatch(number)
			if m == nil {
				continue
			}
			last, _ := strconv.Atoi(m[1])
			if last >= n {
				return false
			}
		}
	}
	return sets > 0
}

// yugiohInfixed answers the printing numbered like the product, with the
// "EN" region infix the datastore writes and Cardmarket leaves out -
// "RDS-ENSE1" for a product numbered "SE1".
func (r *resolver) yugiohInfixed(product *cm.Product, name, rarity, region, tail string) string {
	var bases []string
	base, _, dashed := strings.Cut(product.Number, "-")
	if dashed {
		bases = append(bases, base)
	} else {
		editions := append([]string{}, yugiohEditions(product.ExpansionName)...)
		for _, edition := range append(editions, product.ExpansionCode) {
			set, err := r.backend.GetSetByName(edition)
			if err == nil {
				bases = append(bases, strings.TrimSuffix(set.Code, "-EN"))
			}
		}
	}
	for _, base := range bases {
		number := base + "-EN" + region + tail
		id, err := r.backend.Match(&mtgmatcher.InputCard{
			Name:      name,
			Variation: strings.TrimSpace(number + " " + rarity),
		})
		if err != nil {
			continue
		}
		co, err := r.backend.GetUUID(id)
		if err == nil && strings.EqualFold(co.Number, number) {
			return id
		}
	}
	return ""
}

// yugiohEuropean reports whether the card has a European print at this
// number - "MRL-E129" for tail "129" - that a shelf of ours already prices.
// Cardmarket's "Spell Ruler" catalog carries Magic Ruler's own numbers,
// #104-129, as if they were Spell Ruler's; the row is real, just filed
// under Magic Ruler's shelf instead of the one the product sits on.
func (r *resolver) yugiohEuropean(name, tail string) bool {
	if tail == "" {
		return false
	}
	uuids, err := r.backend.SearchEquals(name)
	if err != nil {
		return false
	}
	for _, uuid := range uuids {
		co, err := r.backend.GetUUID(uuid)
		if err == nil && strings.HasSuffix(strings.ToUpper(co.Number), "-E"+strings.ToUpper(tail)) {
			return true
		}
	}
	return false
}

// reportRefused says what an expansion refused and counts it into the run's
// tally. Every refusal is named, one line each, except in an expansion
// nothing resolved in: that is a catalog we do not carry at all - Cardmarket
// sells whole Japanese programs the datastores have no set for - and the
// count is the whole story, where naming each of its products would be tens
// of thousands of lines saying it again.
//
// A shelf whose every refusal is a foreign one is not worth even that line.
// Cardmarket files the Japanese and other Asian Pokemon catalogs under the
// same game as the English one - 535 of its 774 shelves, 43,603 products -
// and those are programs we do not carry rather than printings we failed to
// find, so a run saying so shelf by shelf is 532 lines that name no work.
// The run's own tally still counts them, and the caller's walk says how
// many in one line at the end.
func (r *resolver) reportRefused(expansion string, total int, refused []string, twins, foreign int) {
	if len(refused) == 0 && twins == 0 {
		return
	}
	count := len(refused) + twins + foreign
	line := fmt.Sprintf("%s: %d of %d products named no printing of ours", expansion, count, total)
	var why []string
	if twins > 0 {
		why = append(why, fmt.Sprintf("%d twins of another product", twins))
	}
	if foreign > 0 {
		why = append(why, fmt.Sprintf("%d of a catalog we do not carry", foreign))
	}
	if len(why) > 0 {
		line += " (" + strings.Join(why, ", ") + ")"
	}
	r.logf("%s", line)
	if len(refused) == total {
		return
	}
	for _, product := range refused {
		r.logf("no printing for %s", product)
	}
}

// disownBridged takes the bridge's answer away from a product it landed on
// a card the same shelf sells, and prices, under another product's name,
// and lets the name answer instead. The bridge speaks through another
// marketplace's links, and a link tied to the neighbouring product lands
// a card on its neighbour's printing: Cardmarket's Herald of Ravages on
// the datastore's Herald of Rebirth, the red Lead with Heart on the
// yellow. A spelling the datastore does not share is not that - no priced
// product of the shelf claims the printing - and the id keeps its say
// over it, the misspelt listing of a card the shelf also sells refused
// included.
func (r *resolver) disownBridged(results []resolved) {
	claimed := map[string]bool{}
	for _, res := range results {
		if res.err != nil || res.cardID == "" {
			continue
		}
		name := fabBaseName(res.product.Name)
		claimed[mtgmatcher.Normalize(name)] = true
		claimed[mtgmatcher.Normalize(unpitched(name))] = true
	}
	for i, res := range results {
		if res.err != nil || res.cardID == "" || res.byName || fabNamesPrinting(r.backend, res.product, res.cardID) {
			continue
		}
		co, err := r.backend.GetUUID(res.cardID)
		if err != nil || !claimed[mtgmatcher.Normalize(fabBaseName(co.Name))] {
			continue
		}
		cardID := r.matchFab(res.product)
		if cardID == "" {
			results[i] = resolved{product: res.product, err: errNoPrinting}
			continue
		}
		results[i] = resolved{product: res.product, cardID: cardID, cardIDFoil: cardID, byName: true}
	}
}

// yugiohNumbers indexes every set's collector numbers by the card each
// names, built once over the whole datastore on first use under a mutex;
// the map itself is never written again, so callers read it lock-free.
func (r *resolver) yugiohNumbers() map[string]map[string]string {
	r.numbersMu.Lock()
	defer r.numbersMu.Unlock()
	if r.numbers == nil {
		r.numbers = map[string]map[string]string{}
		for _, uuid := range r.backend.GetUUIDs() {
			co, err := r.backend.GetUUID(uuid)
			if err != nil {
				continue
			}
			index := r.numbers[co.SetCode]
			if index == nil {
				index = map[string]string{}
				r.numbers[co.SetCode] = index
			}
			index[strings.ToUpper(co.Number)] = mtgmatcher.Normalize(co.Name)
		}
	}
	return r.numbers
}

// yugiohNumberTaken reports whether one of the numbers names a card of the
// set other than the one named.
func (r *resolver) yugiohNumberTaken(setCode string, numbers []string, name string) bool {
	index := r.yugiohNumbers()[setCode]
	for _, number := range numbers {
		holder, held := index[strings.ToUpper(number)]
		if held && holder != mtgmatcher.Normalize(name) {
			return true
		}
	}
	return false
}
