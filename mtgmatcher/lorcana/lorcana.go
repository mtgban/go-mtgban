package lorcana

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// LorcanaJSON is the top-level structure of the Lorcana JSON data file.
type LorcanaJSON struct {
	Metadata struct {
		FormatVersion string `json:"formatVersion"`
		GeneratedOn   string `json:"generatedOn"`
		Language      string `json:"language"`
	} `json:"metadata"`
	Sets map[string]struct {
		PrereleaseDate string `json:"prereleaseDate"`
		ReleaseDate    string `json:"releaseDate"`
		HasAllCards    bool   `json:"hasAllCards"`
		Type           string `json:"type"`
		Number         int    `json:"number"`
		Name           string `json:"name"`
	} `json:"sets"`
	Cards []struct {
		Abilities []struct {
			Effect   string `json:"effect"`
			FullText string `json:"fullText"`
			Name     string `json:"name"`
			Type     string `json:"type"`
		} `json:"abilities,omitempty"`
		Artists          []string          `json:"artists"`
		ArtistsText      string            `json:"artistsText"`
		Code             string            `json:"code"`
		Color            string            `json:"color"`
		Colors           []string          `json:"colors"`
		Cost             int               `json:"cost"`
		FlavorText       string            `json:"flavorText,omitempty"`
		FoilTypes        []string          `json:"foilTypes,omitempty"`
		FullIdentifier   string            `json:"fullIdentifier"`
		FullName         string            `json:"fullName"`
		FullText         string            `json:"fullText"`
		FullTextSections []string          `json:"fullTextSections"`
		ID               int               `json:"id"`
		Images           map[string]string `json:"images,omitempty"`
		Inkwell          bool              `json:"inkwell"`
		Lore             int               `json:"lore,omitempty"`
		Name             string            `json:"name"`
		Number           int               `json:"number"`
		Rarity           string            `json:"rarity"`
		SetCode          string            `json:"setCode"`
		SimpleName       string            `json:"simpleName"`
		Story            string            `json:"story"`
		Strength         int               `json:"strength,omitempty"`
		Subtypes         []string          `json:"subtypes,omitempty"`
		Type             string            `json:"type"`
		Version          string            `json:"version,omitempty"`
		Willpower        int               `json:"willpower,omitempty"`
		KeywordAbilities []string          `json:"keywordAbilities,omitempty"`
		PromoIds         []int             `json:"promoIds,omitempty"`
		Errata           []string          `json:"errata,omitempty"`
		Clarifications   []string          `json:"clarifications,omitempty"`
		Effects          []string          `json:"effects,omitempty"`
		Variant          string            `json:"variant,omitempty"`
		VariantIds       []int             `json:"variantIds,omitempty"`
		MoveCost         int               `json:"moveCost,omitempty"`
		NonPromoID       int               `json:"nonPromoId,omitempty"`
		IsExternalReveal bool              `json:"isExternalReveal,omitempty"`

		ExternalLinks struct {
			TcgPlayerId int `json:"tcgPlayerId"`

			// TcgPlayerExtraIds lists further TCGplayer products that resolve
			// to this same printing, which upstream does not carry: TCGplayer
			// sometimes sells a card's foil under its own product id, and a
			// feed keyed on that id has nothing to match against otherwise.
			// Populated by lorcana-datastore; absent from the upstream
			// file, where it simply stays empty.
			TcgPlayerExtraIds []int `json:"tcgPlayerExtraIds,omitempty"`
		} `json:"externalLinks"`
	} `json:"cards"`

	// Sealed is not part of the upstream file; lorcana-datastore appends
	// every sealed product the TCGplayer catalog files outside the singles
	// type, minting a set entry for the groups upstream has no set for. A
	// datastore built from the plain upstream file simply loads without
	// sealed products.
	Sealed []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		SetCode       string `json:"setCode"`
		ReleaseDate   string `json:"releaseDate"`
		Image         string `json:"image"`
		ExternalLinks struct {
			TcgPlayerId int `json:"tcgPlayerId"`
		} `json:"externalLinks"`
	} `json:"sealed,omitempty"`
}

// Load reads a LorcanaJSON data file from r and returns the parsed
// structure or an error.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload LorcanaJSON
	err := json.NewDecoder(r).Decode(&payload)
	if err != nil {
		return nil, err
	}
	if len(payload.Cards) == 0 || len(payload.Sets) == 0 {
		return nil, errors.New("empty LorcanaJSON file")
	}
	return payload.newBackend(), nil
}

func (lj *LorcanaJSON) newBackend() *mtgmatcher.Backend {
	var b mtgmatcher.Backend

	b.UUIDs = map[string]*mtgmatcher.CardObject{}
	b.Hashes = map[string][]string{}
	b.CanonicalNames = map[string]string{}
	b.ExternalIdentifiers = map[string]string{}
	b.SetSealedUUIDs = map[string][]string{}

	// Load all sets first
	b.Sets = map[string]*mtgmatcher.Set{}
	for code, set := range lj.Sets {
		b.AllSets = append(b.AllSets, code)

		releaseDateTime, _ := time.Parse("2006-01-02", set.ReleaseDate)
		b.Sets[code] = &mtgmatcher.Set{
			Name:            set.Name,
			Code:            code,
			ReleaseDate:     set.ReleaseDate,
			ReleaseDateTime: releaseDateTime,
			Type:            set.Type,
		}
	}
	b.IndexSets()

	// Gather the full reprint list for each name (keyed by normalized name, so
	// case-variant spellings share one list), in first-appearance order. Every
	// card of a name carries the same complete list, mirroring how Magic
	// populates Printings, so Printings4Card works unmodified for Lorcana.
	// All cards of a name share the same backing array; Printings is
	// read-only by contract, as it always has been for Magic.
	printingsByName := map[string][]string{}
	for _, card := range lj.Cards {
		n := mtgmatcher.Normalize(card.FullName)
		if !slices.Contains(printingsByName[n], card.SetCode) {
			printingsByName[n] = append(printingsByName[n], card.SetCode)
		}
	}

	// Load all card names
	for _, card := range lj.Cards {
		// First-seen wins: two Lorcana cards whose names differ only in case
		// ("as"/"As") normalize equal, so last-wins would let a query for one
		// resolve to the other. Keep the first to make the mapping stable.
		if n := mtgmatcher.Normalize(card.FullName); b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.FullName
		}
		if slices.Contains(b.AllCanonicalNames, card.FullName) {
			continue
		}
		b.AllNames = append(b.AllNames, mtgmatcher.Normalize(card.FullName))
		b.AllCanonicalNames = append(b.AllCanonicalNames, card.FullName)
		b.AllLowerNames = append(b.AllLowerNames, card.FullName)
	}
	sort.Strings(b.AllNames)
	sort.Strings(b.AllCanonicalNames)
	sort.Strings(b.AllLowerNames)

	// A product id two different cards both claim names neither of them, so
	// it goes unregistered and a caller sending it falls back to the name it
	// also sent. Mains and extras are counted in one pass: an extra is
	// registered first-wins while a main overwrites, so a main equal to
	// another card's extra would otherwise take it silently.
	claimants := map[int]map[int]bool{}
	claim := func(pid, cardID int) {
		if pid == 0 {
			return
		}
		if claimants[pid] == nil {
			claimants[pid] = map[int]bool{}
		}
		claimants[pid][cardID] = true
	}
	for _, card := range lj.Cards {
		claim(card.ExternalLinks.TcgPlayerId, card.ID)
		for _, extra := range card.ExternalLinks.TcgPlayerExtraIds {
			claim(extra, card.ID)
		}
	}

	// Load all cards and store them in their relative sets
	for _, card := range lj.Cards {
		// Normalize Lorcana's many foil-type names (Silver, Satin, Magma, …) to
		// the matcher's finish constants: "None" is nonfoil, everything else is
		// foil, so output() can select the right (foil) uuid downstream.
		finishes := make([]string, len(card.FoilTypes))
		for i, finish := range card.FoilTypes {
			if strings.EqualFold(finish, "none") {
				finishes[i] = "nonfoil"
			} else {
				finishes[i] = "foil"
			}
		}
		if len(finishes) == 0 {
			finishes = append(finishes, "nonfoil")
		}

		// Ensure no spaces are present for ease of future comparisons
		rarity := strings.Replace(strings.ToLower(card.Rarity), " ", "", -1)

		// Collapse multi and single color info to the same slice, lower case color names
		ogColors := card.Colors
		if len(ogColors) == 0 {
			ogColors = []string{card.Color}
		}
		var colors []string
		for _, color := range ogColors {
			colors = append(colors, strings.ToLower(color))
		}

		// Prepare the card and add it to the main array
		// Since cards are already sorted (by number/id), the order here is preserved
		convertedCard := mtgmatcher.Card{
			UUID: fmt.Sprint(card.ID),

			Name:     card.FullName,
			SetCode:  card.SetCode,
			Finishes: finishes,
			Number:   fmt.Sprintf("%d%s", card.Number, card.Variant),
			Images:   card.Images,

			// The datastore is English-only. Core Match's language filter
			// drops any candidate whose Language differs from English when
			// several survive filtering, so leaving this empty would turn
			// every legitimate multi-candidate result (aliasing) into a
			// bogus wrong-variant error.
			Language: "English",

			Colors: colors,
			Rarity: rarity,

			Subtypes:   card.Subtypes,
			Types:      []string{card.Type},
			Supertypes: []string{card.Story},

			Printings: printingsByName[mtgmatcher.Normalize(card.FullName)],
			IsPromo:   card.NonPromoID != 0,

			OriginalNumber: fmt.Sprintf("%d", card.Number),
		}
		// Register the uuid each finish resolves to. Nonfoil keeps the base
		// uuid and the primary foil keeps "_f", so output()/Match resolve to
		// them; Lorcana's extra foil sub-types (RainbowPillars, …) each get
		// their own uuid keyed by sub-type name so none are dropped. The uuid
		// derives from the sub-type name, not its position, so it is stable
		// across data updates that reorder or add foil types.
		finishUUIDs := map[string]string{}
		type perFinish struct {
			uuid string
			foil bool
			name string
		}
		var stored []perFinish
		foilSeen := false
		for i, finish := range finishes {
			if finish != "foil" {
				finishUUIDs[mtgmatcher.FinishNonfoil] = convertedCard.UUID
				stored = append(stored, perFinish{convertedCard.UUID, false, mtgmatcher.FinishNonfoil})
				continue
			}

			// The verbatim exported finish name, lowercased ("silver",
			// "rainbowpillars", …). Nonfoil above uses the matcher's own
			// constant instead of the export's "None" placeholder.
			finishName := strings.ToLower(card.FoilTypes[i])

			uuid := convertedCard.UUID
			key := mtgmatcher.FinishFoil
			if !foilSeen {
				// Primary foil: "_f", or the base uuid when a foil is the very
				// first finish (a foil-only card).
				foilSeen = true
				if i > 0 {
					uuid += suffixFoil
				}
			} else {
				// Additional sub-types get a name-derived uuid, keyed by their
				// sub-type name in the map.
				key = foilSuffix(card.FoilTypes[i])
				uuid += "_" + key
			}
			finishUUIDs[key] = uuid
			stored = append(stored, perFinish{uuid, true, finishName})
		}
		convertedCard.FoilUUIDs = finishUUIDs

		// A card LorcanaJSON has no TCGplayer link for carries a zero id:
		// registering that would file every one of them under "0" for the
		// next to overwrite, leaving a key that resolves to whichever card
		// happened to load last, and stamping it as an identifier would
		// advertise a product id no product carries.
		if card.ExternalLinks.TcgPlayerId != 0 {
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": fmt.Sprint(card.ExternalLinks.TcgPlayerId),
			}
			if len(claimants[card.ExternalLinks.TcgPlayerId]) == 1 {
				b.ExternalIdentifiers[fmt.Sprint(card.ExternalLinks.TcgPlayerId)] = convertedCard.UUID
			}
		}

		b.Sets[card.SetCode].Cards = append(b.Sets[card.SetCode].Cards, convertedCard)
		// Alternate products for the same printing resolve to the same base
		// uuid; MatchId applies the requested finish to it through output(),
		// so pointing them at the base card is enough to reach the foil. Only
		// the id map grows: no CardObject and no uuid is created here.
		for _, extra := range card.ExternalLinks.TcgPlayerExtraIds {
			if extra == 0 {
				continue
			}
			if _, found := b.ExternalIdentifiers[fmt.Sprint(extra)]; found {
				continue
			}
			if len(claimants[extra]) != 1 {
				continue
			}
			b.ExternalIdentifiers[fmt.Sprint(extra)] = convertedCard.UUID
		}

		// Store a CardObject per finish uuid.
		for _, s := range stored {
			// A genuinely duplicated finish (the same sub-type listed twice)
			// would collide; store each uuid at most once.
			if _, found := b.UUIDs[s.uuid]; found {
				continue
			}
			co := mtgmatcher.CardObject{
				Card:    convertedCard,
				Edition: b.Sets[card.SetCode].Name,
				Foil:    s.foil,
			}
			// co is fresh on every iteration, so the stored pointer is not
			// aliased by later finishes
			co.UUID = s.uuid
			co.Finish = s.name
			b.UUIDs[s.uuid] = &co
			b.AllUUIDs = append(b.AllUUIDs, s.uuid)
			b.Hashes[mtgmatcher.Normalize(card.FullName)] = append(b.Hashes[mtgmatcher.Normalize(card.FullName)], s.uuid)
		}
	}

	// Update any remaining details on Sets after Cards loading
	for code := range b.Sets {
		var rarities, colors []string
		b.Sets[code].IsFoilOnly = true
		b.Sets[code].IsNonFoilOnly = true
		for _, card := range b.Sets[code].Cards {
			if b.Sets[code].BaseSetSize == 0 && card.Rarity == "enchanted" {
				b.Sets[code].BaseSetSize, _ = strconv.Atoi(card.Number)
			}

			if card.HasFinish("nonfoil") {
				b.Sets[code].IsNonFoilOnly = false
			}
			if !card.HasFinish("nonfoil") {
				b.Sets[code].IsFoilOnly = false
			}

			if !slices.Contains(rarities, card.Rarity) {
				rarities = append(rarities, card.Rarity)
			}

			for _, color := range card.Colors {
				if !slices.Contains(colors, lorcanaColorNameMap[color]) {
					colors = append(colors, lorcanaColorNameMap[color])
				}
			}
			if len(card.Colors) == 0 && !slices.Contains(colors, "colorless") {
				colors = append(colors, "colorless")
			}
			if len(card.Colors) > 1 && !slices.Contains(colors, "multicolor") {
				colors = append(colors, "multicolor")
			}
		}

		sort.Slice(rarities, func(i, j int) bool {
			return lorcanaRarityMap[rarities[i]] > lorcanaRarityMap[rarities[j]]
		})
		b.Sets[code].Rarities = rarities

		sort.Strings(colors)
		b.Sets[code].Colors = colors
	}

	// Load sealed products. They live in the sealed namespace throughout:
	// their uuids join AllSealedUUIDs and their names the sealed name
	// index, and the product id is carried as an identifier for
	// BuildSealedProductMap rather than entering the external identifier
	// index, mirroring how Magic and Riftbound keep sealed products out
	// of MatchId's reach.
	var mintedSets bool
	for _, product := range lj.Sealed {
		// The builder mints a set entry for every group it emits sealed
		// from, so an unknown code is a hand-made file; give the product
		// a set to hang off all the same
		if b.Sets[product.SetCode] == nil {
			mintedSets = true
			b.AllSets = append(b.AllSets, product.SetCode)
			releaseDateTime, _ := time.Parse("2006-01-02", product.ReleaseDate)
			b.Sets[product.SetCode] = &mtgmatcher.Set{
				Name:            product.SetCode,
				Code:            product.SetCode,
				ReleaseDate:     product.ReleaseDate,
				ReleaseDateTime: releaseDateTime,
			}
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
		if product.ExternalLinks.TcgPlayerId != 0 {
			card.Identifiers = map[string]string{
				"tcgplayerProductId": fmt.Sprint(product.ExternalLinks.TcgPlayerId),
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
		// The name lists are gated on their own contents rather than on
		// bucket existence: a card can already own the bucket, and the
		// sealed name must still be searchable
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
	if mintedSets {
		sort.Strings(b.AllSets)
		b.IndexSets()
	}

	b.SetRules(Rules{})

	return &b
}

// lorcanaRarityMap ranks the rarities so a set can list them in a stable
// order. The tiers past the base set are ranked by the collector numbers
// LorcanaJSON gives them: from Fabled on, a set runs epic, then enchanted,
// then the two iconic cards that close it out. A rarity absent from here
// ranks 0 and would sort below common, so every printed rarity belongs in
// the table; "special" keeps the top slot it has always held.
var lorcanaRarityMap = map[string]int{
	"common":    1,
	"uncommon":  2,
	"rare":      3,
	"superrare": 4,
	"legendary": 5,
	"epic":      6,
	"enchanted": 7,
	"iconic":    8,
	"special":   9,
}

const suffixFoil = "_f"

// foilSuffix turns a LorcanaJSON foil-type name (Silver, Satin, RainbowPillars,
// …) into a compact, uuid-safe suffix used to give each foil sub-type past the
// primary its own uuid.
func foilSuffix(foilType string) string {
	return strings.ToLower(strings.ReplaceAll(foilType, " ", ""))
}

var lorcanaColorNameMap = map[string]string{
	"W": "white",
	"U": "blue",
	"B": "black",
	"R": "red",
	"G": "green",
}
