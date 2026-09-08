// Package lorcana loads a Lorcana datastore.
//
// The datastore is LorcanaJSON's allCards payload, enriched by
// github.com/mtgban/lorcana-datastore with sealed product and the
// TCGplayer product ids the plain payload does not carry.
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

// AllCards is the top-level structure of the Lorcana data file, named for
// what LorcanaJSON calls it, the way magic names its payload AllPrintings.
type AllCards struct {
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
		// CardCounts is upstream's own tally of the set. Base is the
		// numbered run a card prints its number out of - the 204 of
		// "12/204" - with the enchanted and promo printings counted
		// separately, above it.
		CardCounts struct {
			Base int `json:"base"`
		} `json:"cardCounts"`
	} `json:"sets"`
	Cards []struct {
		Abilities []struct {
			Effect   string `json:"effect"`
			FullText string `json:"fullText"`
			Name     string `json:"name"`
			Type     string `json:"type"`
		} `json:"abilities,omitempty"`
		Artists     []string `json:"artists"`
		ArtistsText string   `json:"artistsText"`
		Code        string   `json:"code"`
		Color       string   `json:"color"`
		Colors      []string `json:"colors"`
		Cost        int      `json:"cost"`
		FlavorText  string   `json:"flavorText,omitempty"`
		FoilTypes   []string `json:"foilTypes,omitempty"`

		// PrintingIDs is the uuid each foil type prices, keyed by the foil
		// type as upstream spells it ("None", "Silver") and published by
		// the builder rather than spelled here. A uuid is what a price is
		// keyed on, and 3,200 of this game's are reached by spelling a
		// foil type through canonicalFinish - so a change to that
		// normalization moves identity that lives outside this package,
		// silently, since a moved uuid resolves to nothing rather than
		// erroring. A datastore that carries none is spelled from below.
		PrintingIDs map[string]string `json:"printingIds,omitempty"`

		// FinishAliases are the other spellings that reach a printing:
		// upstream's own name for a foil ("silver", "rainbowpillars")
		// against the finish TCGplayer sells it under. A storefront naming
		// the treatment is naming a printing, and without these it would
		// land on the standard foil instead of the one it asked for.
		FinishAliases    map[string]string `json:"finishAliases,omitempty"`
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
		PromoIDs         []int             `json:"promoIds,omitempty"`
		Errata           []string          `json:"errata,omitempty"`
		Clarifications   []string          `json:"clarifications,omitempty"`
		Effects          []string          `json:"effects,omitempty"`
		// The datastore spells a promotional printing's provenance in its
		// own fields rather than in the name, which carries no
		// parentheticals at all: where it was handed out, and the finish it
		// was handed out in.
		PromoSourceCategory string `json:"promoSourceCategory,omitempty"`
		VarnishType         string `json:"varnishType,omitempty"`
		// PromoGrouping is the pool a promotional printing was numbered
		// within, which storefronts write behind its number.
		PromoGrouping    string `json:"promoGrouping,omitempty"`
		Variant          string `json:"variant,omitempty"`
		VariantIDs       []int  `json:"variantIds,omitempty"`
		MoveCost         int    `json:"moveCost,omitempty"`
		NonPromoID       int    `json:"nonPromoId,omitempty"`
		IsExternalReveal bool   `json:"isExternalReveal,omitempty"`

		ExternalLinks struct {
			TcgPlayerID int `json:"tcgPlayerId"`

			// CardmarketID and CardTraderId are read only to tell a
			// regionally renamed repeat of a card from a card of its own.
			CardmarketID int `json:"cardmarketId"`
			CardTraderID int `json:"cardTraderId"`

			// TcgPlayerExtraIDs lists further TCGplayer products that resolve
			// to this same printing, which upstream does not carry: TCGplayer
			// sometimes sells a card's foil under its own product id, and a
			// feed keyed on that id has nothing to match against otherwise.
			// Populated by lorcana-datastore; absent from the upstream
			// file, where it simply stays empty.
			TcgPlayerExtraIDs []int `json:"tcgPlayerExtraIds,omitempty"`
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
			TcgPlayerID int `json:"tcgPlayerId"`
		} `json:"externalLinks"`
	} `json:"sealed,omitempty"`
}

// Load reads a LorcanaJSON data file from r and returns the parsed
// structure or an error.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload AllCards
	err := json.NewDecoder(r).Decode(&payload)
	if err != nil {
		return nil, err
	}
	if len(payload.Cards) == 0 || len(payload.Sets) == 0 {
		return nil, errors.New("empty LorcanaJSON file")
	}
	return payload.newBackend(), nil
}

// englishCards drops the entries that repeat a card already listed under
// another name. Upstream files the regionally renamed printings twice -
// "Vaiana - Adventurer of Land and Sea" beside "Moana - Adventurer of Land
// and Sea" - and the two carry one set, one number and one id at every
// storefront, so keeping both splits a single product's prices across two
// uuids and leaves its product id claimed by two cards. Identity is the
// storefront ids rather than the name, so a card renamed in some other
// language folds away too; the datastore is the English program, and the
// English name is the one upstream lists first.
func (ac *AllCards) englishCards() []int {
	seen := map[string]bool{}
	var keep []int
	for i, card := range ac.Cards {
		el := card.ExternalLinks
		if el.TcgPlayerID == 0 && el.CardmarketID == 0 && el.CardTraderID == 0 {
			keep = append(keep, i)
			continue
		}
		// The collector number as printed, letter included: that letter is
		// all that separates the same-numbered art siblings ("4a" to "4e"),
		// and the number alone would file them under one identity.
		identity := fmt.Sprintf("%d|%d|%d|%s|%d%s",
			el.TcgPlayerID, el.CardmarketID, el.CardTraderID, card.SetCode, card.Number, card.Variant)
		if seen[identity] {
			continue
		}
		seen[identity] = true
		keep = append(keep, i)
	}
	return keep
}

// slugTags renders every tag as the token that names it, the form a search
// query can carry.
func slugTags(tags []string) []string {
	var out []string
	for _, tag := range tags {
		out = append(out, mtgmatcher.PromoTypeSlug(tag))
	}
	return out
}

// promoTags names what tells a promotional printing from the ordinary one of
// the same name. Lorcana writes none of this into the name - not one of its
// card names carries a parenthesis - so the tags come from the fields the
// datastore keeps them in.
//
// The pool is among them because it is the only thing that tells two promos
// of one card apart when they also share a number: the datastore numbers each
// pool from one, so "Maleficent - Monstrous Dragon" is card 5 of both the P1
// pool and the P3 one. Storefronts print it where a set card writes its set
// size - "5/P3" against "87/204" - and the tag is what lets that be read.
func promoTags(sourceCategory, varnishType, grouping string) []string {
	var tags []string
	if sourceCategory != "" {
		tags = append(tags, sourceCategory)
	}
	if varnishType != "" {
		tags = append(tags, varnishType)
	}
	if grouping != "" {
		tags = append(tags, grouping)
	}
	return tags
}

func (ac *AllCards) newBackend() *mtgmatcher.Backend {
	var b mtgmatcher.Backend

	b.UUIDs = map[string]*mtgmatcher.CardObject{}
	b.Hashes = map[string][]string{}
	b.PromoTypeLabels = map[string]string{}
	b.CanonicalNames = map[string]string{}
	b.ExternalIdentifiers = map[string]map[string]string{mtgmatcher.IDSpaceTCGplayer: {}}
	b.SetSealedUUIDs = map[string][]string{}

	cards := ac.englishCards()

	// Load all sets first
	b.Sets = map[string]*mtgmatcher.Set{}
	for code, set := range ac.Sets {
		b.AllSets = append(b.AllSets, code)

		releaseDateTime, _ := time.Parse("2006-01-02", set.ReleaseDate)
		b.Sets[code] = &mtgmatcher.Set{
			Name:            set.Name,
			Code:            code,
			ReleaseDate:     set.ReleaseDate,
			ReleaseDateTime: releaseDateTime,
			Type:            set.Type,
			BaseSetSize:     set.CardCounts.Base,
		}
	}
	b.IndexSets()

	// Which printings are promotional, which no single field says. Upstream
	// stopped publishing nonPromoId, the back-pointer this used to read, and
	// states the relationship the other way round: a card lists its own promo
	// printings in promoIds, and a promo says where it came from in
	// promoSourceCategory. Neither reaches the two sets that are wholly
	// promotional - D23 Promos and Disney Lorcana Promo Cards carry neither
	// field on any of their 23 cards - so the set's own type answers there.
	// A minted printing is outside all three, having no upstream entry to
	// carry a field or a relationship at all, and is known by its rarity.
	promoPrintings := map[int]bool{}
	for _, card := range ac.Cards {
		for _, id := range card.PromoIDs {
			promoPrintings[id] = true
		}
	}

	// Gather the full reprint list for each name (keyed by normalized name, so
	// case-variant spellings share one list), in first-appearance order. Every
	// card of a name carries the same complete list, mirroring how Magic
	// populates Printings, so Printings4Card works unmodified for Lorcana.
	// All cards of a name share the same backing array; Printings is
	// read-only by contract, as it always has been for Magic.
	printingsByName := map[string][]string{}
	for _, i := range cards {
		card := ac.Cards[i]
		n := mtgmatcher.Normalize(card.FullName)
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
	// Load all card names. AllNames holds the normalized name, and the
	// case-variant pairs below normalize to one string, so appending once
	// per distinct spelling put that entry in the list twice. searchFunc
	// adds a matching entry's whole hash bucket, so a search returned every
	// printing of such a name once per spelling.
	for _, i := range cards {
		card := ac.Cards[i]
		// First-seen wins: two Lorcana cards whose names differ only in case
		// ("as"/"As") normalize equal, so last-wins would let a query for one
		// resolve to the other. Keep the first to make the mapping stable.
		n := mtgmatcher.Normalize(card.FullName)
		if b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.FullName
		}
		for _, tag := range promoTags(card.PromoSourceCategory, card.VarnishType, card.PromoGrouping) {
			slug := mtgmatcher.PromoTypeSlug(tag)
			if !slices.Contains(b.AllPromoTypes, slug) {
				b.AllPromoTypes = append(b.AllPromoTypes, slug)
			}
			if b.PromoTypeLabels[slug] == "" {
				b.PromoTypeLabels[slug] = tag
			}
		}
		b.AddName(card.FullName)
	}
	sort.Strings(b.AllPromoTypes)
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
	for _, i := range cards {
		card := ac.Cards[i]
		claim(card.ExternalLinks.TcgPlayerID, card.ID)
		for _, extra := range card.ExternalLinks.TcgPlayerExtraIDs {
			claim(extra, card.ID)
		}
	}

	// Load all cards and store them in their relative sets
	for _, i := range cards {
		card := ac.Cards[i]
		// Normalize Lorcana's many foil-type names (Silver, Satin, Magma, …) to
		// the matcher's finish constants: "None" is nonfoil, everything else is
		// foil, so output() can select the right (foil) uuid downstream. Which
		// foil each of them is stays on the uuid carrying it, below.
		finishes := make([]string, len(card.FoilTypes))
		for i, finish := range card.FoilTypes {
			finishes[i] = mtgmatcher.FinishFoil
			if canonicalFinish(finish) == mtgmatcher.FinishNonfoil {
				finishes[i] = mtgmatcher.FinishNonfoil
			}
		}
		if len(finishes) == 0 {
			finishes = append(finishes, mtgmatcher.FinishNonfoil)
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

		// A set wholly of promos says so once, rather than every card in it
		// repeating a field upstream does not set there.
		promoSet := false
		if set, found := b.Sets[card.SetCode]; found {
			promoSet = set.Type == "promo"
		}

		// Prepare the card and add it to the main array
		// Since cards are already sorted (by number/id), the order here is preserved
		convertedCard := mtgmatcher.Card{
			UUID: cardUUID(card.ID),

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

			Printings:  printingsByName[mtgmatcher.Normalize(card.FullName)],
			IsPromo:    promoPrintings[card.ID] || card.PromoSourceCategory != "" || promoSet || rarity == "promo",
			PromoTypes: slugTags(promoTags(card.PromoSourceCategory, card.VarnishType, card.PromoGrouping)),

			PlainNumber: Rules{}.PlainNumber(fmt.Sprintf("%d%s", card.Number, card.Variant)),
		}
		// Register the uuid each finish prices. Where the datastore names
		// them - which it does in TCGplayer's own words, the vocabulary
		// prices arrive in - they are read; where it does not, they are
		// spelled from the foil type as they were before it did.
		finishUUIDs := map[string]string{}
		finishAliases := map[string]string{}
		type perFinish struct {
			uuid string
			foil bool
			name string
		}
		var stored []perFinish
		baseUUID := convertedCard.UUID
		if len(card.PrintingIDs) > 0 {
			for finish, uuid := range card.PrintingIDs {
				finishUUIDs[finish] = uuid
				stored = append(stored, perFinish{uuid, finish != mtgmatcher.FinishNonfoil, finish})
			}
			// A bare foil flag has to reach a printing. A card sold only in
			// a treatment has no standard foil for it to land on, so the
			// treatment answers it - which is what the caller meant, there
			// being nothing else foil about the card.
			// The coarse key the flag form resolves through, which a card
			// sold only in a treatment has no printing of its own for.
			// Pokemon files the same key the same way. The named form is
			// not answered by it: FinishUUID refuses a key whose printing
			// is sold in another finish, so a caller pricing a Cold Foil
			// sku this card does not have is told so.
			if _, found := finishUUIDs[mtgmatcher.FinishFoil]; !found {
				if uuid, found := finishUUIDs[finishHolofoil]; found {
					finishUUIDs[mtgmatcher.FinishFoil] = uuid
				}
			}
			// Finishes is the coarse pair output() reads, not the names
			// above: a card sold in a treatment is sold foil.
			var coarse []string
			for _, s := range stored {
				name := mtgmatcher.FinishNonfoil
				if s.foil {
					name = mtgmatcher.FinishFoil
				}
				if !slices.Contains(coarse, name) {
					coarse = append(coarse, name)
				}
			}
			sort.Strings(coarse)
			convertedCard.Finishes = coarse
			for name, finish := range card.FinishAliases {
				if _, sold := finishUUIDs[finish]; sold {
					finishAliases[name] = finish
				}
			}
		} else {
			foilSeen := false
			for i, finish := range finishes {
				// The foil type this position was read from, which is the key
				// the builder publishes a uuid under. A card upstream lists
				// none is the plain printing alone and has no key to look up.
				var foilType string
				if i < len(card.FoilTypes) {
					foilType = card.FoilTypes[i]
				}
				if finish != mtgmatcher.FinishFoil {
					uuid := published(card.PrintingIDs, foilType, baseUUID)
					finishUUIDs[mtgmatcher.FinishNonfoil] = uuid
					stored = append(stored, perFinish{uuid, false, mtgmatcher.FinishNonfoil})
					continue
				}

				// The exported foil type as the vocabulary spells it ("silver",
				// "rainbowpillars", …). Nonfoil above uses the matcher's own
				// constant instead of the export's "None" placeholder.
				finishName := canonicalFinish(foilType)

				uuid := published(card.PrintingIDs, foilType, baseUUID+"_"+finishName)
				// The printing's first foil answers the plain foil flag; the
				// sub-types past it are keyed by their own name, which is what
				// keeps a flag from reaching a treatment nobody asked for.
				key := mtgmatcher.FinishFoil
				if foilSeen {
					key = finishName
				}
				foilSeen = true
				finishUUIDs[key] = uuid
				stored = append(stored, perFinish{uuid, true, finishName})

				// The standard foil is keyed under the shared constant whatever
				// the printing's foil type is called, so its own name is
				// registered as a spelling that reaches it.
				if finishName != key {
					finishAliases[finishName] = key
				}
				// TCGplayer prices a Lorcana printing in up to four printings:
				// Normal, Foil and Cold Foil for the silver foil almost every
				// card is foiled in, and Holofoil for the treatment past it.
				// Which uuid that names is the printing's own business: the
				// sub-type where there is one, since the foil types are visited
				// in exported order and it wins over the standard foil.
				if finishName != standardFoil {
					finishAliases[tcgSpecialFoil] = key
				}
			}
		}
		convertedCard.FoilUUIDs = finishUUIDs
		// The printing is identified by the entry a bare flag resolves to,
		// which is its nonfoil where it has one and its first foil where it
		// does not. A foil-only printing no longer holds the bare uuid, so
		// naming it here is what keeps the set listing and the identifier
		// index pointing at a card that exists.
		// Read it off the map rather than the order the finishes were
		// listed in: a printing that lists a foil first would otherwise be
		// identified by that foil while carrying a nonfoil.
		if uuid, found := finishUUIDs[mtgmatcher.FinishNonfoil]; found {
			convertedCard.UUID = uuid
		} else if uuid, found := finishUUIDs[mtgmatcher.FinishFoil]; found {
			convertedCard.UUID = uuid
		}
		// A printing foiled one way only answers Holofoil with that foil
		// whatever LorcanaJSON calls it: 418 products in the catalog carry a
		// single sku TCGplayer names Holofoil, and 15 of them are foiled in
		// plain silver, so reading the name as "a treatment past the silver"
		// refuses the only sku those products have.
		//
		// Only where the finishes were spelled from foil types. A datastore
		// naming them in TCGplayer's words sells Holofoil as a finish of its
		// own, and aliasing the name onto the standard foil would answer a
		// caller pricing the treatment with the sku beside it.
		if _, named := finishUUIDs[finishHolofoil]; !named {
			if _, found := finishAliases[tcgSpecialFoil]; !found {
				if _, sold := finishUUIDs[mtgmatcher.FinishFoil]; sold {
					finishAliases[tcgSpecialFoil] = mtgmatcher.FinishFoil
				}
			}
		}
		if len(finishAliases) > 0 {
			convertedCard.FinishAliases = finishAliases
		}

		// A card LorcanaJSON has no TCGplayer link for carries a zero id:
		// registering that would file every one of them under "0" for the
		// next to overwrite, leaving a key that resolves to whichever card
		// happened to load last, and stamping it as an identifier would
		// advertise a product id no product carries.
		if card.ExternalLinks.TcgPlayerID != 0 {
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": fmt.Sprint(card.ExternalLinks.TcgPlayerID),
			}
			if len(claimants[card.ExternalLinks.TcgPlayerID]) == 1 {
				b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer][fmt.Sprint(card.ExternalLinks.TcgPlayerID)] = convertedCard.UUID
			}
		}

		b.Sets[card.SetCode].Cards = append(b.Sets[card.SetCode].Cards, convertedCard)
		// Alternate products for the same printing resolve to the same base
		// uuid; MatchID applies the requested finish to it through output(),
		// so pointing them at the base card is enough to reach the foil. Only
		// the id map grows: no CardObject and no uuid is created here.
		for _, extra := range card.ExternalLinks.TcgPlayerExtraIDs {
			if extra == 0 {
				continue
			}
			if _, found := b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer][fmt.Sprint(extra)]; found {
				continue
			}
			if len(claimants[extra]) != 1 {
				continue
			}
			b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer][fmt.Sprint(extra)] = convertedCard.UUID
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
	// of MatchID's reach.
	var mintedSets bool
	// Sealed products live in the sealed namespace throughout; AddSealed
	// is what files them there.
	for _, product := range ac.Sealed {
		// The builder mints a set entry for every group it emits sealed
		// from, so an unknown code is a hand-made file; give the product
		// a set to hang off all the same, or AddSealed would drop it.
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
		b.AddSealed(product.ID, product.Name, product.SetCode, product.Image, product.ExternalLinks.TcgPlayerID)
	}
	b.SortSealed()
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

// published is the uuid the datastore names for a foil type, and the
// spelled one where it names none - which is how every uuid here was
// reached before the builder began publishing them.
func published(ids map[string]string, foilType, spelled string) string {
	if uuid := ids[foilType]; uuid != "" {
		return uuid
	}
	return spelled
}

// standardFoil is LorcanaJSON's name for the cold foil almost every Lorcana
// card is foiled in (2717 of the 3242 printings in the datastore at the time
// of writing); every other foil type is a treatment on top of it.
const standardFoil = "silver"

// tcgSpecialFoil is the one name TCGplayer prices any such treatment under.
const tcgSpecialFoil = "holofoil"

// finishHolofoil is the third name TCGplayer prices a Lorcana printing
// under, beside Normal and Cold Foil, and the one a datastore naming its
// finishes uses for every treatment past the standard foil.
const finishHolofoil = "holofoil"

var lorcanaColorNameMap = map[string]string{
	"W": "white",
	"U": "blue",
	"B": "black",
	"R": "red",
	"G": "green",
}

// cardUUID spells a card's id as the uuid everything downstream addresses
// it by. The datastore mints a printing upstream does not carry under the
// negated product id, which keeps this build's ids provably clear of
// LorcanaJSON's own - it counts from one - but a uuid is not only compared,
// it is put in a URL, a query string and a spreadsheet cell, and a leading
// minus is escaped, dropped or read as a formula in turn. The minted ones
// are therefore said as "m-512519", and the ids upstream publishes stay the
// digits they always were.
func cardUUID(id int) string {
	if id < 0 {
		return fmt.Sprintf("m-%d", -id)
	}
	return strconv.Itoa(id)
}
