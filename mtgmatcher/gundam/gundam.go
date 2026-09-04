// Package gundam loads a Gundam Card Game datastore.
//
// The datastore is built by github.com/mtgban/datastore-gen's cmd/gundam
// from the TCGplayer catalog dump for category 86, annotated with the
// gcg-api mirror of Bandai's own card list. Identity is the catalog's:
// every entry is one priced printing of an English single product, so
// every uuid is priced by construction. Almost every product sells in a
// single finish, and the one sold both plain and holofoil carries an entry
// per finish, the holofoil one's id suffixed "_holo". Alternate arts and
// event printings share their base card's collector number and are told
// apart by the variant label the builder distills from the product name.
package gundam

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

// finishSuffix is the id suffix the builder hangs off the holofoil entry of
// a product sold both ways. It is the one suffix the game has, so folding a
// product's entries back together is a single TrimSuffix.
const finishSuffix = "_holo"

// Datastore is the cmd/gundam output: sets keyed by code, one card entry
// per priced finish, and the sealed products.
type Datastore struct {
	Game string `json:"game"`
	Sets map[string]struct {
		Name        string `json:"name"`
		ReleaseDate string `json:"releaseDate"`
		// Type is "promo" on the sets that hand their cards out rather than
		// sell them in packs, and empty on every other.
		Type string `json:"type,omitempty"`
	} `json:"sets"`
	Cards  []DatastoreCard   `json:"cards"`
	Sealed []DatastoreSealed `json:"sealed"`
}

// DatastoreCard is one printing as the datastore publishes it.
type DatastoreCard struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Number  string `json:"number"`
	SetCode string `json:"setCode"`
	Rarity  string `json:"rarity"`
	Color   string `json:"color"`
	Type    string `json:"type"`

	// Variant is the label distilled from the product name's qualifiers:
	// empty for the base printing, "SP", "Link Rare", the name of the event
	// a promo was handed out at, or the mobile suit's form for the cards
	// that print one ("Waverider Mode").
	Variant string `json:"variant,omitempty"`

	// Finish is the TCGplayer printing this entry prices, "Normal" or
	// "Holofoil". Entries sharing everything but the finish are the same
	// product sold both ways.
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

// Load reads a Gundam datastore from r and returns a Backend for it, or an
// error when r holds something else, so LoadDatastore's auto-detection can
// move on to the next registered game. The datastore names its game at the
// root, and every card carries the identity fields the backend is built
// from.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload Datastore
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Game != "gundam" || len(payload.Sets) == 0 || len(payload.Cards) == 0 {
		return nil, errors.New("not a Gundam datastore")
	}
	for _, card := range payload.Cards {
		if card.ID == "" || card.Name == "" || card.Number == "" || card.Finish == "" {
			return nil, errors.New("not a Gundam datastore")
		}
	}
	return payload.newBackend(), nil
}

// setIsPromotional reports whether a set hands out promotional printings.
// The datastore types the sets it knows to be promotional; where it says
// nothing the name carries it, every promotional set of this game naming
// itself so ("Gundam Promotional Cards", "Promotional Resource Tokens").
func setIsPromotional(set *mtgmatcher.Set) bool {
	return set.Type == "promo" || strings.Contains(strings.ToLower(set.Name), "promotional")
}

// qualifiedName spells a printing the way TCGplayer names the product, the
// card name followed by the qualifier that tells it from its siblings
// ("Gundam (SP)"). A Gundam card is named after a mobile suit or a pilot,
// so a bare name reaches every printing of it and the qualifier is the only
// thing that says which; the catalog spells it out, so it is worth being
// able to search for. Empty for a base printing, which the bare name
// already describes.
func qualifiedName(card *DatastoreCard) string {
	if card.Variant == "" {
		return ""
	}
	return card.Name + " (" + card.Variant + ")"
}

func (payload *Datastore) newBackend() *mtgmatcher.Backend {
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

	for _, card := range payload.Cards {
		n := mtgmatcher.Normalize(card.Name)
		if b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.Name
		}
		b.AddName(card.Name)
		// The qualified spelling is searchable but never canonical: it names
		// one printing where the bare name names the suit, and Match reads
		// CanonicalNames to decide whether a name needs its parentheticals
		// split off.
		qualified := qualifiedName(&card)
		if qualified == "" {
			continue
		}
		slug := mtgmatcher.PromoTypeSlug(card.Variant)
		if !slices.Contains(b.AllPromoTypes, slug) {
			b.AllPromoTypes = append(b.AllPromoTypes, slug)
		}
		// First spelling seen wins: the catalog writes a couple of these
		// events two ways, and one token can only read back as one.
		if b.PromoTypeLabels[slug] == "" {
			b.PromoTypeLabels[slug] = card.Variant
		}
		b.AddName(qualified)
	}
	sort.Strings(b.AllPromoTypes)
	sort.Strings(b.AllNames)
	sort.Strings(b.AllCanonicalNames)
	sort.Strings(b.AllLowerNames)

	// Group sibling entries back into their product: a dual-printing
	// product's Normal and Holofoil entries are the same card twice, and the
	// matcher wants it once, with FoilUUIDs naming the uuid each finish
	// prices.
	type product struct {
		normal *DatastoreCard
		foil   *DatastoreCard
	}
	var productOrder []string
	products := map[string]*product{}
	for i := range payload.Cards {
		card := &payload.Cards[i]
		key := strings.TrimSuffix(card.ID, finishSuffix)
		entry, found := products[key]
		if !found {
			entry = &product{}
			products[key] = entry
			productOrder = append(productOrder, key)
		}
		// The catalog's own spelling of the finish goes through the game's
		// vocabulary rather than being compared as written.
		if (Rules{}).CanonicalFinish(card.Finish) == mtgmatcher.FinishFoil {
			entry.foil = card
		} else {
			entry.normal = card
		}
	}

	for _, key := range productOrder {
		entry := products[key]
		card := entry.normal
		if card == nil {
			card = entry.foil
		}
		if b.Sets[card.SetCode] == nil {
			continue
		}

		var promoTypes []string
		if card.Variant != "" {
			promoTypes = []string{mtgmatcher.PromoTypeSlug(card.Variant)}
		}

		// Only the finishes a product is actually sold in are registered:
		// output() folds a storefront's unreliable foil flag onto the sold
		// finish when there is one, and routes it to the right sibling when
		// there are two.
		var finishes []string
		foilUUIDs := map[string]string{}
		if entry.normal != nil {
			finishes = append(finishes, mtgmatcher.FinishNonfoil)
			foilUUIDs[mtgmatcher.FinishNonfoil] = entry.normal.ID
		}
		if entry.foil != nil {
			finishes = append(finishes, mtgmatcher.FinishFoil)
			foilUUIDs[mtgmatcher.FinishFoil] = entry.foil.ID
		}

		convertedCard := mtgmatcher.Card{
			UUID:     card.ID,
			Name:     card.Name,
			SetCode:  card.SetCode,
			Finishes: finishes,
			Number:   card.Number,
			Images: map[string]string{
				"full":      card.Image,
				"thumbnail": card.Image,
			},
			Language:   "English",
			Colors:     splitColors(card.Color),
			Rarity:     card.Rarity,
			Types:      cardTypes(card.Type),
			PromoTypes: promoTypes,
			IsPromo:    setIsPromotional(b.Sets[card.SetCode]),
			Printings:  printingsByName[mtgmatcher.Normalize(card.Name)],

			// The collector number carries no decoration of its own, so the
			// plain number a search matches is the number itself.
			OriginalNumber: card.Number,
		}
		convertedCard.FoilUUIDs = foilUUIDs

		if card.ExternalLinks.TcgPlayerID != 0 {
			pid := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": pid,
			}
			// The product id names the product, not one of its finishes, so
			// it points at the plain entry where that exists and at the
			// holofoil one when the card is only sold that way. MatchID
			// re-resolves the finish from the caller's own flag either way.
			b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer][pid] = card.ID
		}

		b.Sets[card.SetCode].Cards = append(b.Sets[card.SetCode].Cards, convertedCard)

		for _, finish := range finishes {
			co := mtgmatcher.CardObject{
				Card:    convertedCard,
				Edition: b.Sets[card.SetCode].Name,
				Foil:    finish == mtgmatcher.FinishFoil,
			}
			// co is fresh on every iteration, so the stored pointer is not
			// aliased by the other finish.
			co.UUID = foilUUIDs[finish]
			co.Finish = finish
			b.UUIDs[co.UUID] = &co
			b.AllUUIDs = append(b.AllUUIDs, co.UUID)
			b.Hashes[mtgmatcher.Normalize(card.Name)] = append(b.Hashes[mtgmatcher.Normalize(card.Name)], co.UUID)
			if qualified := qualifiedName(card); qualified != "" {
				qn := mtgmatcher.Normalize(qualified)
				b.Hashes[qn] = append(b.Hashes[qn], co.UUID)
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
			if len(card.Colors) > 1 && !slices.Contains(colors, "multicolor") {
				colors = append(colors, "multicolor")
			}
		}
		sort.Slice(rarities, func(i, j int) bool {
			return gundamRarityMap[rarities[i]] > gundamRarityMap[rarities[j]]
		})
		b.Sets[code].Rarities = rarities
		sort.Strings(colors)
		b.Sets[code].Colors = colors
	}

	// Sealed products live in the sealed namespace throughout; AddSealed is
	// what files them there.
	for _, product := range payload.Sealed {
		b.AddSealed(product.ID, product.Name, product.SetCode, product.Image, product.ExternalLinks.TcgPlayerID)
	}
	b.SortSealed()

	b.SetRules(Rules{})

	return &b
}

// gundamRarityMap orders the rarities the catalog spells for this game, so a
// set lists them commonest first. The "+" suffixes mark the parallel runs of
// a rarity, which sit above the plain one and below the rarity over it.
var gundamRarityMap = map[string]int{
	"Common":      1,
	"C+":          2,
	"C++":         3,
	"Uncommon":    4,
	"U+":          5,
	"Rare":        6,
	"R+":          7,
	"Legend Rare": 8,
	"LR+":         9,
	"LR++":        10,
	"Promo":       11,
}

// cardTypes is the card's type, as the one-element list a Card carries. The
// catalog writes a couple of them in capitals ("EX BASE" beside "EX Base"),
// which are the same type shouted, so they fold onto the spelling the rest
// of the game uses.
func cardTypes(cardType string) []string {
	if cardType == "" {
		return nil
	}
	if folded, found := typeSpellings[strings.ToLower(cardType)]; found {
		return []string{folded}
	}
	return []string{cardType}
}

// typeSpellings maps a type's lowercased spelling onto the one the datastore
// writes most, keyed lowercased so a new shouting spelling folds too.
var typeSpellings = map[string]string{
	"unit":        "Unit",
	"pilot":       "Pilot",
	"command":     "Command",
	"base":        "Base",
	"resource":    "Resource",
	"ex base":     "EX Base",
	"ex resource": "EX Resource",
}

// splitColors turns the catalog's color value into its components
// ("Blue;Green" and "Blue/Green" both appear in the wild).
func splitColors(color string) []string {
	if color == "" {
		return nil
	}
	fields := strings.FieldsFunc(color, func(r rune) bool {
		return r == ';' || r == '/'
	})
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}
