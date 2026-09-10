// Package yugioh loads a Yu-Gi-Oh! datastore.
//
// The datastore is built from the TCGplayer catalog dump for category 2,
// annotated with the YGOPRODeck card database. Identity is the catalog's:
// every English single product is one card, and the same collector number
// recurs as several products told apart by rarity (the Rarity Collection
// sets print one number in half a dozen rarities). The print runs a product
// sold in (1st Edition, Unlimited, Limited) are priced separately, so each
// entry is one run of one product, its id suffixed _1e, _unl or _lim, and
// sibling entries share their product's tcgPlayerId. The run is data about
// the same printing, never foilness, exactly as the foil stamping is for
// One Piece.
package yugioh

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The print runs the catalog prices, as the game's rules spell them.
const (
	finish1stEdition = "1stedition"
	finishUnlimited  = "unlimited"
	finishLimited    = "limited"
)

// Datastore is the builder output: sets keyed by code, one card entry per
// priced print run, and the sealed products.
type Datastore struct {
	Game string `json:"game"`
	Sets map[string]struct {
		Name        string `json:"name"`
		ReleaseDate string `json:"releaseDate"`
		// Type is "promo" on the sets that hand their cards out rather
		// than sell them in packs, and empty on every other. Yu-Gi-Oh
		// rarities name the foil treatment and never the promotion, so
		// this is the only thing that says so.
		Type string `json:"type,omitempty"`
	} `json:"sets"`
	Cards  []DatastoreCard   `json:"cards"`
	Sealed []DatastoreSealed `json:"sealed"`
}

// DatastoreCard is one printing as the datastore publishes it.
type DatastoreCard struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Number    string `json:"number"`
	SetCode   string `json:"setCode"`
	Rarity    string `json:"rarity"`
	Attribute string `json:"attribute"`
	Type      string `json:"type"`

	// Variant is the name-qualifier residue the builder distills from the
	// product name: empty for most printings, "Alternate Art", a color
	// ("Red") or an event label for the others.
	Variant string `json:"variant,omitempty"`

	// PromoTypes is the same residue as the labels it is made of, which
	// the joined Variant cannot be read back into: "OTS Stamp Blue" is two
	// tags, and a query naming either has to reach the printing.
	PromoTypes []string `json:"promoTypes,omitempty"`

	// OriginalReleaseDate is when this printing released, published only
	// where its set's date does not cover it - the Back to Duel field
	// centres are handed out monthly at events and filed under a set the
	// catalog dates to 2006. Empty means the set dates the printing, which
	// is what CardReleaseDate falls back to.
	OriginalReleaseDate string `json:"originalReleaseDate,omitempty"`

	// Language is what the printing is printed in, where that is not
	// English: the advent calendars are sold in German and their cards are
	// named in it, "Junk Synchron - \"Gerumpelsynchronisierer\"" at
	// AC11-DE001. Empty means English, which is what every other entry is.
	Language string `json:"language,omitempty"`

	// Watermark is the mark saying which printing of a number this is: the
	// ink it was made in ("blue", one of six Duelist League foils), the
	// version ("version1", one of four Blue-Eyes at LCKC-EN001), or the
	// letter its artwork is filed under ("a", one of three Dark Magician
	// Girls at RA03-EN123). For those printings it is the only thing
	// telling one from its siblings, and no printing wears two.
	//
	// It is not the card's own LIGHT or DARK, which is Attribute, and it
	// is not a promotion, which is why it is not among the promo types.
	Watermark string `json:"watermark,omitempty"`

	// Finish is the TCGplayer printing this entry prices, "1st Edition",
	// "Unlimited" or "Limited". Entries sharing everything but the finish
	// are the same product sold in several print runs.
	Finish string `json:"finish"`

	Image         string `json:"image"`
	ExternalLinks struct {
		TcgPlayerID int `json:"tcgPlayerId"`
	} `json:"externalLinks"`
}

// DatastoreSealed is one sealed product as the datastore publishes it.
type DatastoreSealed struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SetCode       string `json:"setCode"`
	ReleaseDate   string `json:"releaseDate"`
	Image         string `json:"image"`
	ExternalLinks struct {
		TcgPlayerID int `json:"tcgPlayerId"`
	} `json:"externalLinks"`
}

// Load reads a Yu-Gi-Oh! datastore from r and returns a Backend for it, or
// an error when r holds something else. The datastore names its game at
// the root, and every card carries the identity fields the backend is built
// from. The collector number is not among them: the game never numbered the
// Yugi's Legendary Decks reprints, and a card the catalog sells under no
// number still has to be sold.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload Datastore
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Game != "yugioh" || len(payload.Sets) == 0 || len(payload.Cards) == 0 {
		return nil, errors.New("not a Yu-Gi-Oh datastore")
	}
	for _, card := range payload.Cards {
		if card.ID == "" || card.Name == "" || card.Finish == "" {
			return nil, errors.New("not a Yu-Gi-Oh datastore")
		}
	}
	return payload.newBackend(), nil
}

// setTypePromo is what the builder types a set that hands its cards out.
const setTypePromo = "promo"

// promoTypesOf reads a printing's labels, preferring the list the builder
// distills them into. A datastore built before that list was recorded
// carries only the joined spelling, which stays one tag rather than being
// split on spaces: several labels are two words long ("Duel Terminal"), and
// splitting would declare halves of them that name nothing.
//
// Whether to fall back is asked of the datastore and not of the card. A
// datastore that labels anything labels everything it meant to, so a card
// without a list has no labels rather than an unread one - and reading its
// variant back would put on exactly what the builder took off. The colours
// are why that matters here: an ink is published as the ink it is, and a
// printing wearing nothing else has no promo types at all, which is what
// lets the wording reach it through the ink instead of through a tag that
// says the same thing twice.
func promoTypesOf(card *DatastoreCard, labelled bool) []string {
	if len(card.PromoTypes) > 0 {
		return card.PromoTypes
	}
	if labelled || card.Variant == "" {
		return nil
	}
	return []string{card.Variant}
}

// promoTypeSlugs is promoTypesOf as the tokens a query can carry, which is
// what a card stores: a search splits its words apart before a filter sees
// them, so a tag only survives the trip as one.
func promoTypeSlugs(card *DatastoreCard, labelled bool) []string {
	labels := promoTypesOf(card, labelled)
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
// sharing its number ("Dark Magician (Purple)"). The rarity stays out of the
// spelling: it is a field of its own that the card already carries, and four
// Duelist League printings of Dark Magician share the number DL11-EN001 and
// the rarity Rare, so on the very printings that need telling apart it is
// the color and nothing else that does it. Empty for a printing the catalog
// qualifies with nothing, which the bare name already describes.
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

func (payload *Datastore) newBackend() *mtgmatcher.Backend {
	var b mtgmatcher.Backend

	// Whether this datastore labels its printings at all, asked once: see
	// promoTypesOf.
	var labelled bool
	for _, card := range payload.Cards {
		if len(card.PromoTypes) > 0 {
			labelled = true
			break
		}
	}

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
			ReleaseDate:     set.ReleaseDate,
			ReleaseDateTime: releaseDateTime,
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

	// The name indexes dedupe through maps rather than the template's
	// slices.Contains: this datastore is an order of magnitude larger than
	// its siblings.
	// Each list holds distinct values of its own kind, and two spellings can
	// normalize or lowercase to one string, so each is deduped on what it
	// actually holds: searchFunc adds a matching entry's whole hash bucket,
	// and a key stored twice returns that bucket twice.
	for _, card := range payload.Cards {
		n := mtgmatcher.Normalize(card.Name)
		if b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.Name
		}
		for _, promoType := range promoTypesOf(&card, labelled) {
			slug := mtgmatcher.PromoTypeSlug(promoType)
			if !slices.Contains(b.AllPromoTypes, slug) {
				b.AllPromoTypes = append(b.AllPromoTypes, slug)
			}
			// The builder folds a qualifier to lower case on the way in,
			// so the words are title-cased back and the acronyms looked
			// up. First spelling seen wins: the catalog writes a few of
			// these two ways, and one token can only read back as one.
			if b.PromoTypeLabels[slug] == "" {
				b.PromoTypeLabels[slug] = promoTypeLabel(slug)
			}
		}
		// Searchable but never canonical: the qualified spelling names one
		// printing where the bare name names the card, and Match reads
		// CanonicalNames to decide whether a name keeps its parentheticals.
		if qualified := qualifiedName(&card, printingsByName); qualified != "" {
			b.AddName(qualified)
		}
		b.AddName(card.Name)
	}
	sort.Strings(b.AllPromoTypes)
	sort.Strings(b.AllNames)
	sort.Strings(b.AllCanonicalNames)
	sort.Strings(b.AllLowerNames)

	// Group sibling entries back into their product: a product priced in
	// several print runs is the same card several times, and the matcher
	// wants it once, with FoilUUIDs naming the uuid each run prices. Which
	// entries are one product is read off the identifiers they publish,
	// never off the shape of their ids.
	var productOrder []string
	products := map[string][]*DatastoreCard{}
	for i := range payload.Cards {
		card := &payload.Cards[i]
		key := card.productKey()
		if _, found := products[key]; !found {
			productOrder = append(productOrder, key)
		}
		products[key] = append(products[key], card)
	}

	// The name-qualifiers the catalog sells a printing under, by uuid, for
	// the rules to read; see Rules.
	qualifiers := map[string]string{}
	for _, key := range productOrder {
		group := products[key]
		// The run both flag values resolve to: a run is not foilness, so a
		// vendor's foil flag must neither strand a match nor select a run.
		// Unlimited is the widest run, 1st Edition and Limited follow.
		card := pickRun(group, finishUnlimited, finish1stEdition, finishLimited)
		if b.Sets[card.SetCode] == nil {
			continue
		}

		promoTypes := promoTypeSlugs(card, labelled)

		var colors []string
		if card.Attribute != "" {
			colors = []string{card.Attribute}
		}

		convertedCard := mtgmatcher.Card{
			UUID:    card.ID,
			Name:    card.Name,
			SetCode: card.SetCode,
			// One product is one printing regardless of print run, so both
			// flag values resolve to the default run's uuid; only the run
			// wording selectFinish reads can re-key onto a sibling.
			Finishes: []string{mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil},
			Number:   card.Number,
			Images: map[string]string{
				"full":      card.Image,
				"thumbnail": card.Image,
			},
			// English unless the printing says otherwise, which only the
			// German advent calendars do. A language is not a promotion and
			// is not among the promo types; the matcher already refuses a
			// printing whose language a listing did not ask for.
			Language: cmp.Or(card.Language, "English"),
			Colors:   colors,
			Rarity:   card.Rarity,

			OriginalReleaseDate: card.OriginalReleaseDate,

			// The one field on a Card for a mark a printing wears that is
			// neither a promotion nor a property of the card - Magic tells
			// the Guild Kits apart by it the same way. Colors is the card's
			// Attribute and stays that: a printing's ink beside a monster's
			// DARK would be two vocabularies in one field, and the site
			// filters colours through it.
			Watermark: card.Watermark,

			Types:      []string{card.Type},
			PromoTypes: promoTypes,
			IsPromo:    payload.Sets[card.SetCode].Type == setTypePromo,
			Printings:  printingsByName[mtgmatcher.Normalize(card.Name)],

			PlainNumber: Rules{}.PlainNumber(card.Number),
		}
		// Register the uuid each run prices under the name the game's rules
		// give it, beside the flag-driven defaults, so an input naming a run
		// reaches every sibling.
		foilUUIDs := map[string]string{
			mtgmatcher.FinishNonfoil: card.ID,
			mtgmatcher.FinishFoil:    card.ID,
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
			// The product id names the product, not one of its runs, so it
			// points at the same default entry the flags resolve to.
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
			// aliased by the sibling runs
			co.UUID = entry.ID
			co.Finish = canonicalFinish(entry.Finish)
			if card.Variant != "" {
				qualifiers[entry.ID] = card.Variant
			}
			b.UUIDs[entry.ID] = &co
			b.AllUUIDs = append(b.AllUUIDs, entry.ID)
			b.Hashes[mtgmatcher.Normalize(card.Name)] = append(b.Hashes[mtgmatcher.Normalize(card.Name)], entry.ID)
			if qualified != "" {
				b.Hashes[qualified] = append(b.Hashes[qualified], entry.ID)
			}
		}
	}

	for code := range b.Sets {
		var rarities, colors []string
		for _, card := range b.Sets[code].Cards {
			if !slices.Contains(rarities, card.Rarity) {
				rarities = append(rarities, card.Rarity)
			}
			for _, color := range card.Colors {
				if !slices.Contains(colors, color) {
					colors = append(colors, color)
				}
			}
		}
		sort.Strings(rarities)
		b.Sets[code].Rarities = rarities
		sort.Strings(colors)
		b.Sets[code].Colors = colors
	}

	// Sealed products live in the sealed namespace throughout; AddSealed
	// is what files them there.
	for _, product := range payload.Sealed {
		b.AddSealed(product.ID, product.Name, product.SetCode, product.Image, product.ExternalLinks.TcgPlayerID)
	}
	b.SortSealed()

	b.SetRules(Rules{qualifiers: qualifiers})

	return &b
}

// pickRun returns the group's first entry of the first print run present,
// in the given preference order, falling back to the group's first entry.
func pickRun(group []*DatastoreCard, finishes ...string) *DatastoreCard {
	for _, finish := range finishes {
		for _, entry := range group {
			if canonicalFinish(entry.Finish) == finish {
				return entry
			}
		}
	}
	return group[0]
}

// productKey names the product an entry is a printing of, read off what the
// entry publishes: the product id the catalog stamps on every printing it
// sells. An entry the builder mints carries none - it is minted one printing
// at a time, from an upstream record nothing else is minted from - so it is a
// product of one printing and stands for itself.
//
// Nothing here takes an id apart. The tail an id ends in is the builder's to
// spell, and a loader that reads one stops folding the day the spelling
// changes - which is exactly what "_holo" becoming "_holofoil" did here.
func (card *DatastoreCard) productKey() string {
	if card.ExternalLinks.TcgPlayerID != 0 {
		return fmt.Sprint(card.ExternalLinks.TcgPlayerID)
	}
	return card.ID
}
