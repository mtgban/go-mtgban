// Package pokemon loads a Pokemon datastore.
//
// The datastore is built from the TCGplayer catalog dump for category 3,
// annotated with the tcgdex card database. Identity is the catalog's: every
// English single product is one card, and the same collector number recurs
// as several products told apart by the qualifiers TCGplayer decorates their
// names with.
//
// Two axes cross on a Pokemon product and both are priced separately, which
// is why each entry is one printing of one product rather than one card with
// flags: the print run (1st Edition, Unlimited) and the foil treatment
// (Holofoil, Reverse Holofoil). A card exists as Normal and Holofoil, or as
// 1st Edition Holofoil and Unlimited Holofoil, and TCGplayer prices each
// crossing on its own sku. Sibling entries share their product's
// tcgPlayerId, and their ids differ by the finish suffix the builder derives
// from the printing name alone.
//
// Unlike Yu-Gi-Oh, where the print runs are the whole finish vocabulary and
// foilness never gates anything, here the treatment really is foilness: a
// storefront's foil flag has to reach the Holofoil printing and not the
// Normal one. So the flag-driven defaults resolve to the plain printings of
// each foilness, and the specific names re-key onto the exact crossing.
package pokemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The printings the catalog prices, as the game's rules spell them. The
// plain ones are the shared vocabulary's (Normal is nonfoil, Holofoil is the
// foil every other treatment is measured against); the rest are this game's
// own, and the crossings name both axes so neither is lost.
const (
	finishHolofoil        = "holofoil"
	finishReverseHolofoil = "reverseholofoil"
	finish1stEdition      = "1stedition"
	finishUnlimited       = "unlimited"
	finish1stEditionHolo  = "1steditionholofoil"
	finishUnlimitedHolo   = "unlimitedholofoil"
)

// setTypePromo is what the builder types a set that hands its cards out.
const setTypePromo = "promo"

// Datastore is the cmd/pokemon output: sets keyed by code, one card entry
// per priced printing, and the sealed products.
type Datastore struct {
	Game   string                  `json:"game"`
	Sets   map[string]DatastoreSet `json:"sets"`
	Cards  []DatastoreCard         `json:"cards"`
	Sealed []DatastoreSealed       `json:"sealed"`
}

// DatastoreSet is one set as the catalog groups it.
type DatastoreSet struct {
	Name         string `json:"name"`
	ReleaseDate  string `json:"releaseDate"`
	Abbreviation string `json:"abbreviation,omitempty"`
	// BaseSetSize is how many cards the set was printed with, the total its
	// cards print beside their number. It is absent on the pooled sets,
	// whose cards keep the total of wherever they were first printed and so
	// agree on none.
	BaseSetSize int `json:"baseSetSize,omitempty"`
	// Type is "promo" on the sets that hand their cards out rather than
	// sell them in packs, and empty on every other.
	Type string `json:"type,omitempty"`
	// Symbol is the URL of the mark this set's cards print, as tcgdex
	// serves it. Absent on the sets tcgdex holds no symbol for, which are
	// the McDonald's collections and most of the promo drawers.
	Symbol string `json:"symbol,omitempty"`
}

// DatastoreCard is one printing of one product: a card as the catalog sells
// it, in one crossing of the print run and the foil treatment.
type DatastoreCard struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Number  string `json:"number"`
	SetCode string `json:"setCode"`
	Rarity  string `json:"rarity"`
	Type    string `json:"type"`

	// Total is the set size the card is printed with, the denominator of
	// the "082/167" on the card face. Older datastores spell it into
	// Number instead; either way the loader keeps the card's own part and
	// the total apart.
	Total string `json:"total,omitempty"`

	// Finish is the TCGplayer printing this entry prices, one crossing of
	// the print run and the foil treatment. Entries sharing everything but
	// the finish are the same product sold several ways.
	Finish string `json:"finish"`

	// Variant is the qualifier residue the builder distills from the
	// product name: empty for most printings, "Full Art", "Staff" or a
	// World Championship player's name for the others.
	Variant string `json:"variant,omitempty"`

	// PromoTypes is the same residue as the labels it is made of, which
	// the joined Variant cannot be read back into: "Full Art Staff" is two
	// labels, and a query naming either has to reach the printing.
	PromoTypes []string `json:"promoTypes,omitempty"`

	// TcgdexID is the tcgdex identifier, annotated where the builder could
	// align the two sources.
	TcgdexID string `json:"tcgdexId,omitempty"`

	Image         string `json:"image"`
	ExternalLinks struct {
		TcgPlayerID int `json:"tcgPlayerId"`

		// The tcgdex identifier, in the place every other identifier
		// lives. The datastore writes it here and flat on the entry both,
		// and the flat field above is what this falls back to for a
		// datastore built before it moved.
		TcgdexID string `json:"tcgdexId,omitempty"`
	} `json:"externalLinks"`
}

// DatastoreSealed is a sealed product, which carries no printing of its own
// and hangs off the set that issued it.
type DatastoreSealed struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SetCode       string `json:"setCode"`
	Image         string `json:"image"`
	ExternalLinks struct {
		TcgPlayerID int `json:"tcgPlayerId"`
	} `json:"externalLinks"`
}

// Load reads a Pokemon datastore from r and returns a Backend for it, or an
// error when r holds something else (so LoadDatastore's auto-detection can
// move on to the next registered game). The datastore names its game at the
// root, and every card carries the identity fields the backend is built
// from. The collector number is not among them: the catalog sells cards it
// numbers with nothing at all.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload Datastore
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Game != "pokemon" || len(payload.Sets) == 0 || len(payload.Cards) == 0 {
		return nil, errors.New("not a Pokemon datastore")
	}
	for _, card := range payload.Cards {
		if card.ID == "" || card.Name == "" || card.Finish == "" {
			return nil, errors.New("not a Pokemon datastore")
		}
	}
	return payload.newBackend(), nil
}

// promoTypesOf reads a printing's labels, preferring the list the builder
// distills them into. A datastore built before that list was recorded
// carries only the joined spelling, which stays one label rather than being
// split on spaces: plenty of labels are several words long ("Cosmos Holo",
// "Pokemon Center Exclusive"), and splitting would leave pieces that name
// nothing.
func promoTypesOf(card *DatastoreCard) []string {
	if len(card.PromoTypes) > 0 {
		return card.PromoTypes
	}
	if card.Variant == "" {
		return nil
	}
	return []string{card.Variant}
}

// describingPromoTypes keeps the labels worth declaring as tags: the ones a
// printing's own finish does not already say. The catalog qualifies a name
// with the treatment it is sold in often enough ("(Cosmos Holo)") that
// declaring those would fill the tag list with finish names. The card keeps
// the full list either way, which the matcher still reads to tell sibling
// printings apart; only the declaration is filtered, the same terms
// Riftbound carries its number-restating labels on.
func describingPromoTypes(card *DatastoreCard) []string {
	sold := canonicalFinish(card.Finish)
	var out []string
	for _, promoType := range promoTypesOf(card) {
		if label := canonicalFinish(promoType); label != "" && strings.Contains(sold, label) {
			continue
		}
		out = append(out, promoType)
	}
	return out
}

// promoTypeSlugs is promoTypesOf as the tokens a query can carry, which is
// what a card stores: a search splits its words apart before a filter sees
// them, so a tag only survives the trip as one.
func promoTypeSlugs(card *DatastoreCard) []string {
	labels := promoTypesOf(card)
	if len(labels) == 0 {
		return nil
	}
	out := make([]string, 0, len(labels))
	for _, label := range labels {
		out = append(out, mtgmatcher.PromoTypeSlug(label))
	}
	return out
}

// qualifiedName spells a printing the way TCGplayer names the product, the
// card name followed by the qualifier that tells it from the siblings
// sharing its number ("Pikachu (Cosmos Holo)"). Empty for a printing the
// catalog qualifies with nothing, which the bare name already describes.
//
// Empty too for a spelling the datastore already carries as a name of its
// own: some rows keep the qualifier inside the name where others split it
// into the variant, and indexing the split row's spelling would pour its
// printings into the bucket the whole name answers with.
func qualifiedName(card *DatastoreCard, printingsByName map[string][]string) string {
	if card.Variant == "" {
		return ""
	}
	qualified := card.Name + " (" + card.Variant + ")"
	if printingsByName[mtgmatcher.Normalize(qualified)] != nil {
		return ""
	}
	return qualified
}

// upperCodes spells every set code the way the rest of the library expects
// to find one. Set codes are upper case everywhere - MTGJSON writes Magic's
// that way and the other catalogs follow - and GetSet upper-cases what it is
// asked for before the lookup, so a lower-cased key is one nothing can
// reach.
func (payload *Datastore) upperCodes() {
	sets := make(map[string]DatastoreSet, len(payload.Sets))
	for code, set := range payload.Sets {
		sets[strings.ToUpper(code)] = set
	}
	payload.Sets = sets
	for i := range payload.Cards {
		payload.Cards[i].SetCode = strings.ToUpper(payload.Cards[i].SetCode)
	}
	for i := range payload.Sealed {
		payload.Sealed[i].SetCode = strings.ToUpper(payload.Sealed[i].SetCode)
	}
}

func (payload *Datastore) newBackend() *mtgmatcher.Backend {
	payload.upperCodes()

	var b mtgmatcher.Backend

	b.UUIDs = map[string]*mtgmatcher.CardObject{}
	b.Hashes = map[string][]string{}
	b.PromoTypeLabels = map[string]string{}
	b.CanonicalNames = map[string]string{}
	b.ExternalIdentifiers = map[string]map[string]string{mtgmatcher.IDSpaceTCGplayer: {}}
	b.SetSealedUUIDs = map[string][]string{}

	b.Sets = map[string]*mtgmatcher.Set{}
	for code, set := range payload.Sets {
		b.AllSets = append(b.AllSets, code)
		releaseDateTime, _ := time.Parse("2006-01-02", set.ReleaseDate)
		b.Sets[code] = &mtgmatcher.Set{
			Name:            set.Name,
			Code:            code,
			Type:            set.Type,
			ReleaseDate:     set.ReleaseDate,
			ReleaseDateTime: releaseDateTime,
			BaseSetSize:     set.BaseSetSize,
			Symbol:          set.Symbol,
		}
	}
	sort.Strings(b.AllSets)
	b.IndexSets()

	printingsByName := map[string][]string{}
	for _, card := range payload.Cards {
		n := mtgmatcher.Normalize(card.Name)
		if !slices.Contains(printingsByName[n], card.SetCode) {
			printingsByName[n] = append(printingsByName[n], card.SetCode)
		}
	}

	// The name indexes dedupe through maps rather than slices.Contains:
	// this datastore carries forty thousand entries. Each list holds
	// distinct values of its own kind, and two spellings can normalize or
	// lowercase to one string, so each is deduped on what it actually
	// holds: searchFunc adds a matching entry's whole hash bucket, and a
	// key stored twice returns that bucket twice.
	seenPromoType := map[string]bool{}
	for i := range payload.Cards {
		card := &payload.Cards[i]
		n := mtgmatcher.Normalize(card.Name)
		if b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.Name
		}
		for _, promoType := range describingPromoTypes(card) {
			slug := mtgmatcher.PromoTypeSlug(promoType)
			if !seenPromoType[slug] {
				seenPromoType[slug] = true
				b.AllPromoTypes = append(b.AllPromoTypes, slug)
			}
			// The builder folds a qualifier to lower case on the way in,
			// so the words are title-cased back and the game's own
			// suffixes looked up. First spelling seen wins: the catalog
			// writes a few of these two ways, and one token can only read
			// back as one.
			if b.PromoTypeLabels[slug] == "" {
				b.PromoTypeLabels[slug] = promoTypeLabel(promoType)
			}
		}
		// Searchable but never canonical: the qualified spelling names one
		// printing where the bare name names the card, and Match reads
		// CanonicalNames to decide whether a name keeps its parentheticals.
		if qualified := qualifiedName(card, printingsByName); qualified != "" {
			b.AddName(qualified)
		}
		b.AddName(card.Name)
	}
	sort.Strings(b.AllPromoTypes)
	sort.Strings(b.AllNames)
	sort.Strings(b.AllCanonicalNames)
	sort.Strings(b.AllLowerNames)

	// Group sibling entries back into their product: a product priced in
	// several printings is the same card several times, and the matcher
	// wants it once, with FoilUUIDs naming the uuid each printing prices.
	// The builder stamps every entry with its product's tcgPlayerId; an
	// entry left without one falls back to its id with the suffix stripped.
	var productOrder []string
	products := map[string][]*DatastoreCard{}
	for i := range payload.Cards {
		card := &payload.Cards[i]
		key := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
		if card.ExternalLinks.TcgPlayerID == 0 {
			key = trimFinishSuffix(card.ID)
		}
		if _, found := products[key]; !found {
			productOrder = append(productOrder, key)
		}
		products[key] = append(products[key], card)
	}

	for _, key := range productOrder {
		group := products[key]
		// The printing both flag values resolve to. A plain printing is
		// what a storefront means when it says nothing, so the nonfoil
		// flag prefers Normal and the foil flag prefers Holofoil, each
		// falling back through the print runs that carry its foilness.
		card := pickPrinting(group, mtgmatcher.FinishNonfoil, finishUnlimited, finish1stEdition)
		if b.Sets[card.SetCode] == nil {
			continue
		}

		var types []string
		if card.Type != "" {
			types = []string{card.Type}
		}

		convertedCard := mtgmatcher.Card{
			UUID:    card.ID,
			Name:    card.Name,
			SetCode: card.SetCode,
			// Only the foilness classes the product is actually sold in
			// are registered: a card printed holo-only must not answer a
			// nonfoil query with its holo, and output() folds a
			// storefront's unreliable flag onto the class it does sell.
			Finishes: soldFinishes(group),
			Number:   ownNumber(card),
			Images: map[string]string{
				"full":      card.Image,
				"thumbnail": card.Image,
			},
			Language:   "English",
			Rarity:     card.Rarity,
			Types:      types,
			PromoTypes: promoTypeSlugs(card),
			IsPromo:    payload.Sets[card.SetCode].Type == setTypePromo,
			Printings:  printingsByName[mtgmatcher.Normalize(card.Name)],

			OriginalNumber: Rules{}.PlainNumber(ownNumber(card)),
			SetTotal:       setTotal(card),
		}

		// Register the uuid each printing prices under the name the game's
		// rules give it, beside the flag-driven defaults, so an input
		// naming a treatment reaches the exact crossing it names.
		foilUUIDs := map[string]string{}
		if plain := pickFinish(group, mtgmatcher.FinishNonfoil, finishUnlimited, finish1stEdition); plain != nil {
			foilUUIDs[mtgmatcher.FinishNonfoil] = plain.ID
		}
		if foil := pickFinish(group, finishHolofoil, finishUnlimitedHolo, finish1stEditionHolo, finishReverseHolofoil); foil != nil {
			foilUUIDs[mtgmatcher.FinishFoil] = foil.ID
		}
		for _, entry := range group {
			foilUUIDs[canonicalFinish(entry.Finish)] = entry.ID
		}
		convertedCard.FoilUUIDs = foilUUIDs

		if card.ExternalLinks.TcgPlayerID != 0 {
			pid := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": pid,
			}
			if id := card.ExternalLinks.TcgdexID; id != "" {
				convertedCard.Identifiers["tcgdexId"] = id
			} else if card.TcgdexID != "" {
				convertedCard.Identifiers["tcgdexId"] = card.TcgdexID
			}
			// The product id names the product, not one of its printings,
			// so it points at the same default entry the flags resolve to.
			b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer][pid] = card.ID
		}

		b.Sets[card.SetCode].Cards = append(b.Sets[card.SetCode].Cards, convertedCard)

		var qualified string
		if name := qualifiedName(card, printingsByName); name != "" {
			qualified = mtgmatcher.Normalize(name)
		}
		for _, entry := range group {
			co := mtgmatcher.CardObject{
				Card:    convertedCard,
				Edition: b.Sets[card.SetCode].Name,
			}
			// co is fresh on every iteration, so the stored pointer is not
			// aliased by the sibling printings
			co.UUID = entry.ID
			co.Finish = canonicalFinish(entry.Finish)
			// Every holo treatment is a foil, and the flag is what a caller
			// with no finish vocabulary reads: leaving it false filed every
			// holo printing in the game as a plain one.
			co.Foil = isFoilFinish(co.Finish)
			b.UUIDs[entry.ID] = &co
			b.AllUUIDs = append(b.AllUUIDs, entry.ID)
			b.Hashes[mtgmatcher.Normalize(card.Name)] = append(b.Hashes[mtgmatcher.Normalize(card.Name)], entry.ID)
			if qualified != "" {
				b.Hashes[qualified] = append(b.Hashes[qualified], entry.ID)
			}
		}
	}

	for code := range b.Sets {
		var rarities []string
		for _, card := range b.Sets[code].Cards {
			if !slices.Contains(rarities, card.Rarity) {
				rarities = append(rarities, card.Rarity)
			}
		}
		sort.Strings(rarities)
		b.Sets[code].Rarities = rarities
	}

	// Sealed products live in the sealed namespace throughout: uuids in
	// AllSealedUUIDs, names in the sealed name index, and the product id as
	// an identifier for BuildSealedProductMap rather than the external
	// identifier index, mirroring how Magic keeps sealed out of MatchID's
	// reach.
	// Sealed products live in the sealed namespace throughout; AddSealed
	// is what files them there.
	for _, product := range payload.Sealed {
		b.AddSealed(product.ID, product.Name, product.SetCode, product.Image, product.ExternalLinks.TcgPlayerID)
	}
	b.SortSealed()

	b.SetRules(NewRules(&b))

	return &b
}

// soldFinishes reports the foilness classes a product is actually sold in,
// so the flag-driven lookup cannot hand back a printing the product does not
// have. A card sold only as Holofoil is foil and nothing else.
func soldFinishes(group []*DatastoreCard) []string {
	var out []string
	for _, entry := range group {
		finish := mtgmatcher.FinishNonfoil
		if isFoilFinish(canonicalFinish(entry.Finish)) {
			finish = mtgmatcher.FinishFoil
		}
		if !slices.Contains(out, finish) {
			out = append(out, finish)
		}
	}
	sort.Strings(out)
	return out
}

// isFoilFinish reports whether a canonical finish name is one of the foil
// treatments. Every crossing names its treatment, so the test is the same
// for the plain holo and for the ones that also name a print run.
func isFoilFinish(finish string) bool {
	return strings.Contains(finish, "holofoil") || finish == mtgmatcher.FinishFoil
}

// findPrinting returns the group's first entry of the first finish present,
// in the given preference order, or nil when the group has none of them.
func pickFinish(group []*DatastoreCard, finishes ...string) *DatastoreCard {
	for _, finish := range finishes {
		for _, entry := range group {
			if canonicalFinish(entry.Finish) == finish {
				return entry
			}
		}
	}
	return nil
}

// pickPrinting is findPrinting with the group's first entry as a fallback,
// for the card the matcher reads the shared fields off.
func pickPrinting(group []*DatastoreCard, finishes ...string) *DatastoreCard {
	if entry := pickFinish(group, finishes...); entry != nil {
		return entry
	}
	return group[0]
}

// trimFinishSuffix strips the printing tail the builder suffixes ids with,
// the grouping fallback for an entry without a tcgPlayerId. The longer
// suffixes are tried first, so "_1eholo" is not read as "_holo".
func trimFinishSuffix(id string) string {
	for _, suffix := range []string{"_1eholo", "_unlholo", "_reverse", "_holo", "_1e", "_unl"} {
		if strings.HasSuffix(id, suffix) {
			return strings.TrimSuffix(id, suffix)
		}
	}
	return id
}

// ownNumber is the card's part of the collector number alone: the "082" of
// the "082/167" printed on the card, the set total being the set's fact
// rather than the card's. An older datastore writes the whole face into
// Number, so the split happens here rather than trusting either shape.
func ownNumber(card *DatastoreCard) string {
	number, _, _ := strings.Cut(card.Number, "/")
	return number
}

// plainNumber is the collector number as a person writes it, which for this
// game means without the zeros the catalog pads an ordinal out to three
// digits with: card 1 is written "1", not "001". It is what OriginalNumber
// carries, the field a plain-number search matches, so "cn:1" reaches the
// card that "cns:001" does. Number keeps the padding, being the number
// exactly as written.
//
// Only a leading zero is dropped, which is the whole of the rule: a number
// starting with one is a bare ordinal and nothing else, all 140 of them, so
// the shape needs no describing. The padding inside a code is untouched
// because a code does not open with it - "SWSH020" and "TG01" keep theirs,
// and reducing them would invent a spelling that is neither what the card
// prints nor a plainer way of writing it.
//
// Pokemon is the only game this applies to. Every other game here numbers a
// card with a code - "OP04-047", "YS13-ENV08", "WTR018" - and none of them
// carries a bare ordinal at all.
//
// A number of nothing but zeros would trim away to nothing, and an empty
// OriginalNumber is a card no plain-number search can reach, so it keeps
// what it had. No such number exists today; the guard is what makes the
// trim safe to read.
func plainNumber(number string) string {
	if plain := strings.TrimLeft(number, "0"); plain != "" {
		return plain
	}
	return number
}

// setTotal is the set size the card's face prints beside its number, the
// "167" of "082/167", which is what tells a reprint from its original:
// Cascoon is 44/130 in Diamond & Pearl and 44/127 in Platinum. An older
// datastore spells the total into Number, a newer one keeps it in Total,
// and a card whose face prints no total at all - which is most promos -
// answers with nothing, the absence being as much a fact as the number.
func setTotal(card *DatastoreCard) string {
	if _, total, found := strings.Cut(card.Number, "/"); found {
		return total
	}
	return card.Total
}
