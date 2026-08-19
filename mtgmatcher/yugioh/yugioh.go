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
	} `json:"sets"`
	Cards  []DatastoreCard   `json:"cards"`
	Sealed []DatastoreSealed `json:"sealed"`
}

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

	// Finish is the TCGplayer printing this entry prices, "1st Edition",
	// "Unlimited" or "Limited". Entries sharing everything but the finish
	// are the same product sold in several print runs.
	Finish string `json:"finish"`

	Image         string `json:"image"`
	ExternalLinks struct {
		TcgPlayerID int `json:"tcgPlayerId"`
	} `json:"externalLinks"`
}

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
// an error when r holds something else (so LoadDatastore's auto-detection
// can move on to the next registered game). The datastore names its game at
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

// addName files a spelling in each search list that does not hold it yet.
func addName(b *mtgmatcher.Backend, name string, seenNormalized, seenLower, seenCanonical map[string]bool) {
	if n := mtgmatcher.Normalize(name); !seenNormalized[n] {
		seenNormalized[n] = true
		b.AllNames = append(b.AllNames, n)
	}
	if lower := strings.ToLower(name); !seenLower[lower] {
		seenLower[lower] = true
		b.AllLowerNames = append(b.AllLowerNames, lower)
	}
	if !seenCanonical[name] {
		seenCanonical[name] = true
		b.AllCanonicalNames = append(b.AllCanonicalNames, name)
	}
}

func (payload *Datastore) newBackend() *mtgmatcher.Backend {
	var b mtgmatcher.Backend

	b.UUIDs = map[string]*mtgmatcher.CardObject{}
	b.Hashes = map[string][]string{}
	b.CanonicalNames = map[string]string{}
	b.ExternalIdentifiers = map[string]string{}
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
	seenNormalized := map[string]bool{}
	seenLower := map[string]bool{}
	seenCanonical := map[string]bool{}
	for _, card := range payload.Cards {
		n := mtgmatcher.Normalize(card.Name)
		if b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.Name
		}
		if card.Variant != "" && !slices.Contains(b.AllPromoTypes, card.Variant) {
			b.AllPromoTypes = append(b.AllPromoTypes, card.Variant)
		}
		// Searchable but never canonical: the qualified spelling names one
		// printing where the bare name names the card, and Match reads
		// CanonicalNames to decide whether a name keeps its parentheticals.
		if qualified := qualifiedName(&card, printingsByName); qualified != "" {
			addName(&b, qualified, seenNormalized, seenLower, seenCanonical)
		}
		addName(&b, card.Name, seenNormalized, seenLower, seenCanonical)
	}
	sort.Strings(b.AllPromoTypes)
	sort.Strings(b.AllNames)
	sort.Strings(b.AllCanonicalNames)
	sort.Strings(b.AllLowerNames)

	// Group sibling entries back into their product: a product priced in
	// several print runs is the same card several times, and the matcher
	// wants it once, with FoilUUIDs naming the uuid each run prices. The
	// builder stamps every entry with its product's tcgPlayerId; an entry
	// left without one falls back to its id with the run suffix stripped.
	var productOrder []string
	products := map[string][]*DatastoreCard{}
	for i := range payload.Cards {
		card := &payload.Cards[i]
		key := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
		if card.ExternalLinks.TcgPlayerID == 0 {
			key = trimRunSuffix(card.ID)
		}
		if _, found := products[key]; !found {
			productOrder = append(productOrder, key)
		}
		products[key] = append(products[key], card)
	}

	for _, key := range productOrder {
		group := products[key]
		// The run both flag values resolve to: a run is not foilness, so a
		// vendor's foil flag must neither strand a match nor select a run.
		// Unlimited is the widest run, 1st Edition and Limited follow.
		card := pickRun(group, finishUnlimited, finish1stEdition, finishLimited)
		if b.Sets[card.SetCode] == nil {
			continue
		}

		var promoTypes []string
		if card.Variant != "" {
			promoTypes = []string{card.Variant}
		}

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
			Language:   "English",
			Colors:     colors,
			Rarity:     card.Rarity,
			Types:      []string{card.Type},
			PromoTypes: promoTypes,
			Printings:  printingsByName[mtgmatcher.Normalize(card.Name)],

			OriginalNumber: card.Number,
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
			b.ExternalIdentifiers[pid] = card.ID
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

	// Sealed products live in the sealed namespace throughout: uuids in
	// AllSealedUUIDs, names in the sealed name index, and the product id
	// as an identifier for BuildSealedProductMap rather than the external
	// identifier index, mirroring how Magic keeps sealed out of MatchId's
	// reach.
	for _, product := range payload.Sealed {
		if b.Sets[product.SetCode] == nil {
			continue
		}

		card := mtgmatcher.Card{
			UUID:    product.ID,
			Name:    product.Name,
			SetCode: product.SetCode,
			Rarity:  "product",
			Images: map[string]string{
				"full":      product.Image,
				"thumbnail": product.Image,
			},
			Language: "English",
		}
		if product.ExternalLinks.TcgPlayerID != 0 {
			card.Identifiers = map[string]string{
				"tcgplayerProductId": fmt.Sprint(product.ExternalLinks.TcgPlayerID),
			}
		}

		b.Sets[product.SetCode].SealedProduct = append(b.Sets[product.SetCode].SealedProduct, mtgmatcher.SealedProduct{
			UUID:        product.ID,
			Name:        product.Name,
			SetCode:     product.SetCode,
			Identifiers: card.Identifiers,
		})

		if _, found := b.UUIDs[product.ID]; found {
			continue
		}
		n := mtgmatcher.Normalize(product.Name)
		if !slices.Contains(b.AllSealed, n) {
			b.AllSealed = append(b.AllSealed, n)
			b.AllCanonicalSealed = append(b.AllCanonicalSealed, product.Name)
			b.AllLowerSealed = append(b.AllLowerSealed, strings.ToLower(product.Name))
		}
		b.Hashes[n] = append(b.Hashes[n], product.ID)

		b.UUIDs[product.ID] = &mtgmatcher.CardObject{
			Card:    card,
			Edition: b.Sets[product.SetCode].Name,
			Sealed:  true,
		}
		b.AllSealedUUIDs = append(b.AllSealedUUIDs, product.ID)
		b.SetSealedUUIDs[product.SetCode] = append(b.SetSealedUUIDs[product.SetCode], product.ID)
	}
	sort.Strings(b.AllSealedUUIDs)
	for code := range b.SetSealedUUIDs {
		sort.Strings(b.SetSealedUUIDs[code])
	}
	sort.Strings(b.AllSealed)
	sort.Strings(b.AllCanonicalSealed)
	sort.Strings(b.AllLowerSealed)

	b.SetRules(Rules{})

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

// trimRunSuffix strips the print-run tail the builder suffixes ids with,
// the grouping fallback for an entry without a tcgPlayerId.
func trimRunSuffix(id string) string {
	for _, suffix := range []string{"_1e", "_unl", "_lim"} {
		if strings.HasSuffix(id, suffix) {
			return strings.TrimSuffix(id, suffix)
		}
	}
	return id
}
