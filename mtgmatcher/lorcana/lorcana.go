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
		} `json:"externalLinks"`
	} `json:"cards"`
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

	b.UUIDs = map[string]mtgmatcher.CardObject{}
	b.Hashes = map[string][]string{}
	b.CanonicalNames = map[string]string{}
	b.ExternalIdentifiers = map[string]string{}

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

			Identifiers: map[string]string{
				"tcgplayerProductId": fmt.Sprint(card.ExternalLinks.TcgPlayerId),
			},
		}
		b.Sets[card.SetCode].Cards = append(b.Sets[card.SetCode].Cards, convertedCard)

		b.ExternalIdentifiers[fmt.Sprint(card.ExternalLinks.TcgPlayerId)] = convertedCard.UUID

		// Split cards per finish
		for i, finish := range finishes {
			co := mtgmatcher.CardObject{
				Card:    convertedCard,
				Edition: b.Sets[card.SetCode].Name,
			}

			// The main/first version keeps the same uuid of the card in the Cards array
			uuid := convertedCard.UUID
			if finish != "nonfoil" {
				co.Foil = true
				if i > 0 {
					uuid += suffixFoil
				}
			}

			// Cards with several foil sub-types collapse onto one foil uuid;
			// store it once so AllUUIDs and the name hash stay duplicate-free
			if _, found := b.UUIDs[uuid]; found {
				continue
			}

			// Update uuid and store
			co.UUID = uuid
			b.UUIDs[uuid] = co

			// Save uuid in the array of uuids and
			b.AllUUIDs = append(b.AllUUIDs, uuid)
			b.Hashes[mtgmatcher.Normalize(card.FullName)] = append(b.Hashes[mtgmatcher.Normalize(card.FullName)], uuid)
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

	b.SetRules(Rules{})

	return &b
}

var lorcanaRarityMap = map[string]int{
	"common":    1,
	"uncommon":  2,
	"rare":      3,
	"superrare": 4,
	"legendary": 5,
	"enchanted": 6,
	"special":   7,
}

const suffixFoil = "_f"

var lorcanaColorNameMap = map[string]string{
	"W": "white",
	"U": "blue",
	"B": "black",
	"R": "red",
	"G": "green",
}
