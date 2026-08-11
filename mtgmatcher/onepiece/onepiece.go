// Package onepiece loads a One Piece Card Game datastore.
//
// The datastore is built by github.com/mtgban/datastore-gen's cmd/onepiece
// from the TCGplayer catalog dump for category 68, annotated with
// punk-records' mirror of the official Bandai card list. Identity is the
// catalog's: every English single product is one printing, so every uuid
// is priced by construction. Alternate arts, parallels and event printings
// share their base card's collector number and are told apart by the
// variant label the builder distills from the product name.
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
// per printing, and the sealed products.
type Datastore struct {
	Sets map[string]struct {
		Name        string `json:"name"`
		ReleaseDate string `json:"releaseDate"`
	} `json:"sets"`
	Cards  []DatastoreCard   `json:"cards"`
	Sealed []DatastoreSealed `json:"sealed"`
}

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

	// BandaiID is the official card list's _pN printing id, annotated where
	// the builder could align the two sources unambiguously.
	BandaiID string `json:"bandaiId,omitempty"`

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

// Load reads a One Piece datastore from r and returns a Backend for it, or
// an error when r holds something else (so LoadDatastore's auto-detection
// can move on to the next registered game). The card ids are strings and
// every card carries a collector number, which no other game's datastore
// shape satisfies.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload Datastore
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Sets) == 0 || len(payload.Cards) == 0 {
		return nil, errors.New("not a One Piece datastore")
	}
	for _, card := range payload.Cards {
		if card.ID == "" || card.Name == "" || card.Number == "" {
			return nil, errors.New("not a One Piece datastore")
		}
	}
	return payload.newBackend(), nil
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

	for _, card := range payload.Cards {
		if n := mtgmatcher.Normalize(card.Name); b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.Name
		}
		if slices.Contains(b.AllCanonicalNames, card.Name) {
			continue
		}
		b.AllNames = append(b.AllNames, mtgmatcher.Normalize(card.Name))
		b.AllCanonicalNames = append(b.AllCanonicalNames, card.Name)
		b.AllLowerNames = append(b.AllLowerNames, strings.ToLower(card.Name))
	}
	sort.Strings(b.AllNames)
	sort.Strings(b.AllCanonicalNames)
	sort.Strings(b.AllLowerNames)

	for _, card := range payload.Cards {
		if b.Sets[card.SetCode] == nil {
			continue
		}

		var promoTypes []string
		if card.Variant != "" {
			promoTypes = []string{card.Variant}
		}

		convertedCard := mtgmatcher.Card{
			UUID:    card.ID,
			Name:    card.Name,
			SetCode: card.SetCode,
			// One product is one printing regardless of foil stamping, so
			// both flag values resolve to the same uuid and a storefront's
			// unreliable foil flag can never strand a match.
			Finishes: []string{mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil},
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
			Printings:  printingsByName[mtgmatcher.Normalize(card.Name)],

			OriginalNumber: card.Number,
		}
		convertedCard.FoilUUIDs = map[string]string{
			mtgmatcher.FinishNonfoil: card.ID,
			mtgmatcher.FinishFoil:    card.ID,
		}

		if card.ExternalLinks.TcgPlayerID != 0 {
			pid := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": pid,
			}
			if card.BandaiID != "" {
				convertedCard.Identifiers["bandaiId"] = card.BandaiID
			}
			b.ExternalIdentifiers[pid] = card.ID
		}

		b.Sets[card.SetCode].Cards = append(b.Sets[card.SetCode].Cards, convertedCard)

		co := mtgmatcher.CardObject{
			Card:    convertedCard,
			Edition: b.Sets[card.SetCode].Name,
		}
		b.UUIDs[card.ID] = &co
		b.AllUUIDs = append(b.AllUUIDs, card.ID)
		b.Hashes[mtgmatcher.Normalize(card.Name)] = append(b.Hashes[mtgmatcher.Normalize(card.Name)], card.ID)
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
