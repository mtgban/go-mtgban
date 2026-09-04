// Package palworld loads a Palworld OFFICIAL CARD GAME datastore.
//
// The datastore is built by github.com/mtgban/datastore-gen's cmd/palworld
// from the TCGplayer catalog dump for category 91, annotated with
// palworldtcg.gg's card list. Identity is the catalog's: every entry is one
// priced printing of an English single product, so every uuid is priced by
// construction. A product sells in one finish, plain or foil, and the foil
// entry's id is suffixed "_foil".
//
// Unlike the other Bandai-shaped games, a parallel printing here is filed
// under a collector number of its own: the rarity's code is the number's
// tail, so "ETD01-001" and "ETD01-001TSR" are the plain card and its Trial
// Deck Super Deck Rare, and no two printings share a number. The card list
// also numbers a card with an "E" the set code does not carry - the set is
// BP01 and its cards are numbered EBP01-nnn - so a number's prefix never
// names the set it belongs to.
package palworld

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

// finishSuffix is the id suffix the builder hangs off the foil entry of a
// product. It is the one suffix the game has, so folding a product's
// entries back together is a single TrimSuffix.
const finishSuffix = "_foil"

// Datastore is the cmd/palworld output: sets keyed by code, one card entry
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
	ID   string `json:"id"`
	Name string `json:"name"`

	// Number is the collector number, the rarity's code included where the
	// printing carries one ("ETD01-001TSR"). A single printing carries none
	// at all - the catalog sells a "Soul" the card list does not number -
	// so this is not a field a card is required to have.
	Number  string `json:"number,omitempty"`
	SetCode string `json:"setCode"`
	Rarity  string `json:"rarity"`
	Color   string `json:"color"`
	Type    string `json:"type"`

	// Variant is the label distilled from the product name's qualifiers,
	// empty for all but a handful: the rarity codes the catalog writes in
	// parentheses are an echo of the number's own tail, and the builder
	// drops them rather than filing the same fact twice.
	Variant string `json:"variant,omitempty"`

	// Finish is the TCGplayer printing this entry prices, "Normal" or
	// "Foil".
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

// Load reads a Palworld datastore from r and returns a Backend for it, or
// an error when r holds something else, so LoadDatastore's auto-detection
// can move on to the next registered game. The datastore names its game at
// the root, and every card carries the identity fields the backend is built
// from - the collector number excepted, which one real printing lacks.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload Datastore
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Game != "palworld" || len(payload.Sets) == 0 || len(payload.Cards) == 0 {
		return nil, errors.New("not a Palworld datastore")
	}
	for _, card := range payload.Cards {
		if card.ID == "" || card.Name == "" || card.Finish == "" {
			return nil, errors.New("not a Palworld datastore")
		}
	}
	return payload.newBackend(), nil
}

// setIsPromotional reports whether a set hands out promotional printings.
// The datastore types the sets it knows to be promotional; where it says
// nothing the name carries it, this game's promo set naming itself so.
func setIsPromotional(set *mtgmatcher.Set) bool {
	return set.Type == "promo" || strings.Contains(strings.ToLower(set.Name), "promo")
}

// qualifiedName spells a printing the way TCGplayer names the product, the
// card name followed by the qualifier that tells it from its siblings.
// Empty for almost every card here: this game separates its printings by
// number rather than by label, so only the handful carrying a real variant
// gain a second spelling.
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
		qualified := qualifiedName(&card)
		if qualified == "" {
			continue
		}
		slug := mtgmatcher.PromoTypeSlug(card.Variant)
		if !slices.Contains(b.AllPromoTypes, slug) {
			b.AllPromoTypes = append(b.AllPromoTypes, slug)
		}
		if b.PromoTypeLabels[slug] == "" {
			b.PromoTypeLabels[slug] = card.Variant
		}
		b.AddName(qualified)
	}
	sort.Strings(b.AllPromoTypes)
	sort.Strings(b.AllNames)
	sort.Strings(b.AllCanonicalNames)
	sort.Strings(b.AllLowerNames)

	// Group sibling entries back into their product. No product of this game
	// is sold both ways today, but the datastore is shaped to carry one, so
	// the pairing is done rather than assumed away.
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

			// The rarity code the number ends in is part of the number the
			// card is sold under, not a decoration laid over a plainer one,
			// so the plain number a search matches is the number itself.
			OriginalNumber: card.Number,
		}
		convertedCard.FoilUUIDs = foilUUIDs

		if card.ExternalLinks.TcgPlayerID != 0 {
			pid := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": pid,
			}
			b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer][pid] = card.ID
		}

		b.Sets[card.SetCode].Cards = append(b.Sets[card.SetCode].Cards, convertedCard)

		for _, finish := range finishes {
			co := mtgmatcher.CardObject{
				Card:    convertedCard,
				Edition: b.Sets[card.SetCode].Name,
				Foil:    finish == mtgmatcher.FinishFoil,
			}
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
			return palworldRarityMap[rarities[i]] > palworldRarityMap[rarities[j]]
		})
		b.Sets[code].Rarities = rarities
		sort.Strings(colors)
		b.Sets[code].Colors = colors
	}

	for _, product := range payload.Sealed {
		b.AddSealed(product.ID, product.Name, product.SetCode, product.Image, product.ExternalLinks.TcgPlayerID)
	}
	b.SortSealed()

	b.SetRules(Rules{})

	return &b
}

// palworldRarityMap orders the rarities the catalog spells for this game, so
// a set lists them commonest first. The trial-deck rarities run beside the
// booster ones rather than under them: a deck's cards are its own run.
var palworldRarityMap = map[string]int{
	"Trial Deck":                 1,
	"Common":                     2,
	"Uncommon":                   3,
	"Trial Deck Rare":            4,
	"Rare":                       5,
	"Double Rare":                6,
	"Super Rare":                 7,
	"Trial Deck Super Deck Rare": 8,
	"Over Super Rare":            9,
	"Super Parallel":             10,
	"Trial Deck Super Parallel":  11,
	"Super Special Parallel":     12,
	"Super Special Soul":         13,
	"Promo":                      14,
}

// cardTypes is the card's type, as the one-element list a Card carries.
func cardTypes(cardType string) []string {
	if cardType == "" {
		return nil
	}
	return []string{cardType}
}

// splitColors turns the catalog's color value into its components.
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
