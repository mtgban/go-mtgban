package mtgmatcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

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
	} `json:"cards"`
}

func LoadLorcana(r io.Reader) (DataStore, error) {
	var payload LorcanaJSON
	err := json.NewDecoder(r).Decode(&payload)
	if err != nil {
		return nil, err
	}
	if len(payload.Cards) == 0 || len(payload.Sets) == 0 {
		return nil, errors.New("empty LorcanaJSON file")
	}
	return payload, nil
}

func (lj LorcanaJSON) Load() cardBackend {
	var backend cardBackend

	backend.UUIDs = map[string]CardObject{}
	backend.Hashes = map[string][]string{}

	// Load all sets first
	backend.Sets = map[string]*Set{}
	for code, set := range lj.Sets {
		backend.AllSets = append(backend.AllSets, code)

		backend.Sets[code] = &Set{
			Name:        set.Name,
			Code:        code,
			ReleaseDate: set.ReleaseDate,
			Type:        set.Type,
		}
	}

	// Load all card names
	for _, card := range lj.Cards {
		if slices.Contains(backend.AllCanonicalNames, card.FullName) {
			continue
		}
		backend.AllNames = append(backend.AllNames, Normalize(card.FullName))
		backend.AllCanonicalNames = append(backend.AllCanonicalNames, card.FullName)
		backend.AllLowerNames = append(backend.AllLowerNames, card.FullName)
	}
	sort.Strings(backend.AllNames)
	sort.Strings(backend.AllCanonicalNames)
	sort.Strings(backend.AllLowerNames)

	// Load all cards and store them in their relative sets
	for _, card := range lj.Cards {
		// Make finishes lowercase, and assume that if missing it's nonfoil
		finishes := card.FoilTypes
		for i := range finishes {
			finishes[i] = strings.ToLower(finishes[i])
			if finishes[i] == "none" {
				finishes[i] = "nonfoil"
			}
		}
		if len(finishes) == 0 {
			finishes = append(finishes, "nonfoil")
		}

		// Ensure no spaces are present for ease of future comparisons
		rarity := strings.Replace(strings.ToLower(card.Rarity), " ", "", -1)

		// Prepare the card and add it to the main array
		// Since cards are already sorted (by number/id), the order here is preserved
		convertedCard := Card{
			UUID: fmt.Sprint(card.ID),

			Name:     card.FullName,
			SetCode:  card.SetCode,
			Finishes: finishes,
			Number:   fmt.Sprintf("%d%s", card.Number, card.Variant),
			Images:   card.Images,

			Colors: []string{strings.ToLower(card.Color)},
			Rarity: rarity,

			Subtypes:   card.Subtypes,
			Types:      []string{card.Type},
			Supertypes: []string{card.Story},

			Printings: []string{card.SetCode},
			IsPromo:   card.NonPromoID != 0,
		}
		backend.Sets[card.SetCode].Cards = append(backend.Sets[card.SetCode].Cards, convertedCard)

		// Split cards per finish
		for i, finish := range finishes {
			co := CardObject{
				Card:    convertedCard,
				Edition: backend.Sets[card.SetCode].Name,
			}

			// The main/first version keeps the same uuid of the card in the Cards array
			uuid := convertedCard.UUID
			if finish != "nonfoil" {
				co.Foil = true
				if i > 0 {
					uuid += suffixFoil
				}
			}

			// Update uuid and store
			co.UUID = uuid
			backend.UUIDs[uuid] = co

			// Save uuid in the array of uuids and
			backend.AllUUIDs = append(backend.AllUUIDs, uuid)
			backend.Hashes[Normalize(card.FullName)] = append(backend.Hashes[Normalize(card.FullName)], uuid)
		}
	}

	// Update any reamining details on Sets after Cards loading
	for code := range backend.Sets {
		var rarities, colors []string
		backend.Sets[code].IsFoilOnly = true
		backend.Sets[code].IsNonFoilOnly = true
		for _, card := range backend.Sets[code].Cards {
			if backend.Sets[code].BaseSetSize == 0 && card.Rarity == "enchanted" {
				backend.Sets[code].BaseSetSize, _ = strconv.Atoi(card.Number)
			}

			if card.HasFinish("nonfoil") {
				backend.Sets[code].IsNonFoilOnly = false
			}
			if !card.HasFinish("nonfoil") {
				backend.Sets[code].IsFoilOnly = false
			}

			if !slices.Contains(rarities, card.Rarity) {
				rarities = append(rarities, card.Rarity)
			}
			if len(card.Colors) > 0 && !slices.Contains(colors, card.Colors[0]) {
				colors = append(colors, card.Colors[0])
			}
		}

		sort.Slice(rarities, func(i, j int) bool {
			return lorcanaRarityMap[rarities[i]] > lorcanaRarityMap[rarities[j]]
		})
		backend.Sets[code].Rarities = rarities

		sort.Strings(colors)
		backend.Sets[code].Colors = colors
	}

	return backend
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

func SimpleSearch(cardName, number string, foil bool) (string, error) {
	number = strings.TrimLeft(number, "0")
	number = strings.Split(number, "/")[0]

	cardName = SplitVariants(cardName)[0]

	uuids, err := SearchEquals(cardName)
	if err != nil {
		return "", err
	}

	if len(uuids) == 1 {
		return uuids[0], nil
	}

	var cardIds []string
	for _, uuid := range uuids {
		co, err := GetUUID(uuid)
		if err != nil {
			continue
		}

		if foil && !co.Foil {
			continue
		} else if !foil && co.Foil {
			continue
		}

		if number != "" && number != co.Number {
			continue
		}
		cardIds = append(cardIds, uuid)
	}

	if len(cardIds) < 1 {
		return "", ErrCardWrongVariant
	}

	if len(cardIds) > 1 {
		alias := newAliasingError()
		alias.dupes = uuids
		return "", err
	}

	return cardIds[0], nil
}
