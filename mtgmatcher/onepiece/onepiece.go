// Package onepiece loads a One Piece Card Game datastore.
//
// The datastore is built by github.com/mtgban/datastore-gen's cmd/onepiece
// from the TCGplayer catalog dump for category 68, annotated with
// punk-records' mirror of the official Bandai card list. Identity is the
// catalog's: every entry is one priced printing of an English single
// product, so every uuid is priced by construction. Most products sell in
// a single finish, but the few sold both plain and foil carry one entry
// per finish, the foil one's id suffixed "_foil". Alternate arts,
// parallels and event printings share their base card's collector number
// and are told apart by the variant label the builder distills from the
// product name.
package onepiece

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

// Datastore is the cmd/onepiece output: sets keyed by code, one card entry
// per priced finish, and the sealed products.
type Datastore struct {
	Game string `json:"game"`
	Sets map[string]struct {
		Name        string `json:"name"`
		ReleaseDate string `json:"releaseDate"`
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
	// empty for the base printing, "Alternate Art", "Parallel", "Manga",
	// "SP" or an event name for the others.
	Variant string `json:"variant,omitempty"`

	// Finish is the TCGplayer printing this entry prices, "Normal" or
	// "Foil". Entries sharing everything but the finish are the same
	// product sold both ways.
	Finish string `json:"finish"`

	// BandaiID is the official card list's _pN printing id, annotated where
	// the builder could align the two sources unambiguously.
	BandaiID string `json:"bandaiId,omitempty"`

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

// Load reads a One Piece datastore from r and returns a Backend for it, or
// an error when r holds something else (so LoadDatastore's auto-detection
// can move on to the next registered game). The datastore names its game at
// the root, and every card carries the identity fields the backend is built
// from.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload Datastore
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Game != "onepiece" || len(payload.Sets) == 0 || len(payload.Cards) == 0 {
		return nil, errors.New("not a One Piece datastore")
	}
	for _, card := range payload.Cards {
		if card.ID == "" || card.Name == "" || card.Number == "" || card.Finish == "" {
			return nil, errors.New("not a One Piece datastore")
		}
	}
	return payload.newBackend(), nil
}

// setIsPromotional reports whether a set hands out promotional printings.
// The datastore types no set, so the name carries it: the promo set names
// itself "One Piece Promotion Cards", and the pre-release sets hand out
// stamped copies ahead of a release, which are promos by every other name.
// Matching on the name rather than the code keeps the Premium Booster sets
// out, whose codes begin "PRB" but which are an ordinary product.
func setIsPromotional(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "promotion cards") || strings.Contains(lower, "pre-release")
}

// qualifiedName spells a printing the way TCGplayer names the product, the
// character name followed by the qualifier that tells it from its siblings
// ("Nami (Premium Card Collection -Best Selection Vol. 6-)"). Every One Piece
// card is named after a character, so the bare name reaches a hundred
// printings and the qualifier is the only thing that says which one; the
// catalog spells it out, so it is worth being able to search for. Empty for
// a base printing, which the bare name already describes.
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

	// Each list holds distinct values of its own kind. Two spellings can
	// normalize to one string - "Teemo, Scout" and "Teemo - Scout" both
	// become "teemocout" - and AllNames holds the normalized form, so
	// appending once per distinct spelling put one entry in twice.
	// searchFunc adds a matching entry's whole hash bucket, so every card
	// of that name came back from a search once per spelling.
	// AllNames holds the normalized name and AllLowerNames the lowercased
	// one, and either can fold two spellings into one string - the epithets
	// differ in punctuation and in case. Appending once per distinct
	// spelling put that entry in twice, and searchFunc adds a matching
	// entry's whole hash bucket, so a search returned every printing of
	// such a name once per spelling.
	for _, card := range payload.Cards {
		n := mtgmatcher.Normalize(card.Name)
		if b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.Name
		}
		b.AddName(card.Name)
		// The qualified spelling is searchable but never canonical: it names
		// one printing where the bare name names the character, and Match
		// reads CanonicalNames to decide whether a name needs its
		// parentheticals split off. Leaving it out of that map keeps the
		// matcher reading them exactly as it does today.
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
	// product's Normal and Foil entries are the same card twice, and the
	// matcher wants it once, with FoilUUIDs naming the uuid each finish
	// prices. The foil sibling's id is the bare id plus "_foil".
	type product struct {
		normal *DatastoreCard
		foil   *DatastoreCard
	}
	var productOrder []string
	products := map[string]*product{}
	for i := range payload.Cards {
		card := &payload.Cards[i]
		key := strings.TrimSuffix(card.ID, "_foil")
		entry, found := products[key]
		if !found {
			entry = &product{}
			products[key] = entry
			productOrder = append(productOrder, key)
		}
		// The catalog's own spelling of the finish goes through the game's
		// vocabulary rather than being compared as written
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
			Types:      []string{card.Type},
			PromoTypes: promoTypes,
			IsPromo:    setIsPromotional(b.Sets[card.SetCode].Name),
			Printings:  printingsByName[mtgmatcher.Normalize(card.Name)],

			OriginalNumber: Rules{}.PlainNumber(card.Number),
		}
		convertedCard.FoilUUIDs = foilUUIDs

		if card.ExternalLinks.TcgPlayerID != 0 {
			pid := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": pid,
			}
			if card.BandaiID != "" {
				convertedCard.Identifiers["bandaiId"] = card.BandaiID
			}
			// The product id names the product, not one of its finishes, so
			// it points at the plain entry where that exists and at the foil
			// one when the card is only sold foil. MatchID re-resolves the
			// finish from the caller's own flag either way.
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
			// aliased by the other finish
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
			return onepieceRarityMap[rarities[i]] > onepieceRarityMap[rarities[j]]
		})
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

	b.SetRules(Rules{})

	return &b
}

var onepieceRarityMap = map[string]int{
	"C":   1,
	"UC":  2,
	"R":   3,
	"SR":  4,
	"L":   5,
	"SEC": 6,
	"SP":  7,
	"TR":  8,
	"PR":  9,
}

// splitColors turns the catalog's color value into its components
// ("Red;Green" and "Red/Green" both appear in the wild).
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
