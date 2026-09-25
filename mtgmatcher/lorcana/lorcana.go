// Package lorcana loads a Lorcana datastore.
//
// The datastore is LorcanaJSON's allCards payload, enriched by
// github.com/mtgban/datastore-gen with sealed product and the
// TCGplayer product ids the plain payload does not carry.
package lorcana

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
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
		// BaseSetSize is the numbered run a card prints its number out of,
		// the 204 of "12/204": upstream's cardCounts.base under the name
		// every datastore gives it.
		BaseSetSize int `json:"baseSetSize"`
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

		// PrintingIDs is the uuid each finish prices, keyed by the finish
		// as TCGplayer prices it ("Normal", "Cold Foil", "Holofoil") and
		// published by the builder rather than spelled here: a uuid is what
		// a price is keyed on, and a spelling of this package's would move
		// identity that lives outside it, silently, since a moved uuid
		// resolves to nothing rather than erroring.
		PrintingIDs map[string]string `json:"-"`

		// Printings is what a card's printings are, one entry each: the
		// finish TCGplayer prices it under, the uuid it is quoted by, and
		// the treatments that printing is the printing of. adoptPrintings
		// reads PrintingIDs, above, and the treatments off it.
		Printings []struct {
			Finish     string   `json:"finish"`
			ID         string   `json:"id"`
			PromoTypes []string `json:"promoTypes,omitempty"`
		} `json:"printings,omitempty"`
		FullIdentifier   string            `json:"fullIdentifier"`
		FullName         string            `json:"fullName"`
		FullText         string            `json:"fullText"`
		FullTextSections []string          `json:"fullTextSections"`
		ID               int               `json:"id"`
		Images           map[string]string `json:"images,omitempty"`
		Inkwell          bool              `json:"inkwell"`
		Lore             int               `json:"lore,omitempty"`
		Name             string            `json:"name"`
		// A string, as every other game's datastore writes it. It is the
		// only spelling that can be absent, and absence is a fact of the
		// card: the puzzle inserts, the lore cards and the oversized
		// components print no collector number and the datastore files
		// them with none. An integer gives that the same 0 a card really
		// printing 0/204 has, and "Bruno Madrigal - Undetected Uncle" is
		// that card.
		Number string `json:"number"`
		// Total is the denominator the face prints after the number: the
		// set's size on a card of the set, the run's own label on a promo
		// ("1/P1" beside "1/204"). It is what tells the two apart, since a
		// set's promos are numbered from 1 alongside its own cards.
		Total            string   `json:"total"`
		Rarity           string   `json:"rarity"`
		SetCode          string   `json:"setCode"`
		SimpleName       string   `json:"simpleName"`
		Story            string   `json:"story"`
		Strength         int      `json:"strength,omitempty"`
		Subtypes         []string `json:"subtypes,omitempty"`
		Type             string   `json:"type"`
		Version          string   `json:"version,omitempty"`
		Willpower        int      `json:"willpower,omitempty"`
		KeywordAbilities []string `json:"keywordAbilities,omitempty"`
		PromoIDs         []int    `json:"promoIds,omitempty"`
		Errata           []string `json:"errata,omitempty"`
		Clarifications   []string `json:"clarifications,omitempty"`
		Effects          []string `json:"effects,omitempty"`
		// Where a promotional printing was handed out, read for whether
		// there was a promotion at all rather than for what it was called.
		PromoSourceCategory string `json:"promoSourceCategory,omitempty"`
		// PromoTypes are the promotions themselves, as the builder distils
		// them. It leaves out a varnish every card of a rarity wears, leaves
		// out the "Promo" category that says only what the set says, and
		// carries the foil treatment a printing is sold in, which upstream
		// keeps in a field of its own and this package reads as a finish.
		// None of that is worked out again here: a card it labelled with
		// nothing is a card with no promotion.
		PromoTypes       []string `json:"promoTypes,omitempty"`
		Variant          string   `json:"variant,omitempty"`
		VariantIDs       []int    `json:"variantIds,omitempty"`
		MoveCost         int      `json:"moveCost,omitempty"`
		NonPromoID       int      `json:"nonPromoId,omitempty"`
		IsExternalReveal bool     `json:"isExternalReveal,omitempty"`

		// Language is the builder's, not upstream's: it is written on the
		// exclusives TCGplayer prices in no English sku. Empty means English.
		Language string `json:"language,omitempty"`

		ExternalLinks struct {
			TcgPlayerID int `json:"tcgPlayerId"`

			// CardmarketID is indexed where it names one card alone.
			// CardTraderID is read only to tell a regionally renamed repeat
			// of a card from a card of its own.
			CardmarketID int `json:"cardmarketId"`
			CardTraderID int `json:"cardTraderId"`

			// TcgPlayerExtraIDs lists further TCGplayer products that resolve
			// to this same printing, which upstream does not carry: TCGplayer
			// sometimes sells a card's foil under its own product id, and a
			// feed keyed on that id has nothing to match against otherwise.
			// Populated by datastore-gen; absent from the upstream file.
			TcgPlayerExtraIDs []int `json:"tcgPlayerExtraIds,omitempty"`
		} `json:"externalLinks"`
	} `json:"cards"`

	// Sealed is not part of the upstream file; datastore-gen appends every
	// sealed product the TCGplayer catalog files outside the singles type,
	// minting a set entry for the groups upstream has no set for. A
	// datastore without it loads with no sealed products.
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

	// treatments are the promo types each printing carries, by uuid, which
	// Rules keeps to tell a card's printings apart by the treatment a
	// listing names.
	treatments map[string][]string
}

// Load reads a LorcanaJSON data file from r and returns the parsed
// structure or an error.
//
// It reads the {"meta":...,"data":...} envelope the builders publish, where
// "data" holds the payload; a bare document reads as empty and is refused.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var envelope struct {
		Data AllCards `json:"data"`
	}
	if err := json.NewDecoder(r).Decode(&envelope); err != nil {
		return nil, err
	}
	payload := envelope.Data
	if len(payload.Cards) == 0 || len(payload.Sets) == 0 {
		return nil, errors.New("not a Lorcana datastore")
	}
	for _, card := range payload.Cards {
		// No printing is a card with nothing to price, not a wrong file.
		if card.ID == 0 || card.FullName == "" {
			return nil, errors.New("not a Lorcana datastore")
		}
	}
	payload.adoptPrintings()
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
		identity := fmt.Sprintf("%d|%d|%d|%s|%s%s",
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

// promoTypeLabels are the words behind a token. Decoration is this side's
// job - the datastore publishes a slug and nothing else, and a slug cannot
// give back the boundaries it dropped, so "verticalwave" reads as
// "Verticalwave" to a title-caser and has to be written down instead.
//
// It is also the only place a promotion is renamed. The token stays what the
// datastore published - it is the vocabulary a query is written in, and a
// matcher answering to one spelling while the datastore publishes another
// agree on nothing - so "serialnumbered" keeps its name and is merely shown
// as "Serialized".
//
// Only the tokens a title-caser gets wrong are here. A single word it gets
// right on its own, and so does an initialism the caser leaves alone
// ("d23"), so listing those would only be a second place to keep them in
// step with the first.
var promoTypeLabels = map[string]string{
	"alternateart": "Alternate Art",
	"calendarwave": "Calendar Wave",
	// The two exclusives are a language's, and the card says which in a
	// field of its own: "Chinese (S)" and "Japanese". The initials are what
	// TCGplayer prints in the product name, and they are not worth showing
	// a reader who has not seen the shelf they came off.
	"csexclusive":             "Simplified Chinese Exclusive",
	"disney100":               "Disney 100",
	"disneycruise":            "Disney Cruise",
	"disneyparksstores":       "Disney Parks & Stores",
	"extendedart":             "Extended Art",
	"freeform":                "Free Form",
	"giftbox":                 "Gift Box",
	"illumineersquest":        "Illumineer's Quest",
	"jpexclusive":             "Japanese Exclusive",
	"magicalplaces":           "Magical Places",
	"mattehotfoil":            "Matte Hot Foil",
	"organizedplay":           "Organized Play",
	"rainbowhotfoil":          "Rainbow Hot Foil",
	"rainbowpillars":          "Rainbow Pillars",
	"seawave":                 "Sea Wave",
	"serialnumbered":          "Serialized",
	"stitchcollectorsgiftset": "Stitch Collector's Gift Set",
	"storechampionship":       "Store Championship",
	"top128":                  "Top 128",
	"verticalwave":            "Vertical Wave",
}

// promoTypeLabel is the words a token is shown as, from the table above
// where a title-caser cannot work them out.
func promoTypeLabel(slug string) string {
	if label, found := promoTypeLabels[slug]; found {
		return label
	}
	return mtgmatcher.Title(slug)
}

// adoptPrintings reads a card's printings into the uuid each finish prices,
// and puts the treatments its printings carry on the card as promo types. A
// Card here is the card, not one of its printings: a query for
// "rainbowpillars" is asking for the card that has such a printing, and
// which printing that is stays with the treatments, by uuid.
func (ac *AllCards) adoptPrintings() {
	ac.treatments = map[string][]string{}
	for i := range ac.Cards {
		card := &ac.Cards[i]
		card.PrintingIDs = make(map[string]string, len(card.Printings))
		for _, printing := range card.Printings {
			card.PrintingIDs[printing.Finish] = printing.ID
			for _, treatment := range printing.PromoTypes {
				ac.treatments[printing.ID] = append(ac.treatments[printing.ID], mtgmatcher.NormalizeFinish(treatment))
				if !slices.Contains(card.PromoTypes, treatment) {
					card.PromoTypes = append(card.PromoTypes, treatment)
				}
			}
		}
	}
}

func (ac *AllCards) newBackend() *mtgmatcher.Backend {
	var b mtgmatcher.Backend

	b.UUIDs = map[string]*mtgmatcher.CardObject{}
	b.Hashes = map[string][]string{}
	b.PromoTypeLabels = map[string]string{}
	b.CanonicalNames = map[string]string{}
	b.ExternalIdentifiers = map[string]map[string]string{
		mtgmatcher.IDSpaceTCGplayer:  {},
		mtgmatcher.IDSpaceCardmarket: {},
	}
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
			BaseSetSize:     set.BaseSetSize,
		}
	}
	sort.Strings(b.AllSets)
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

	// Each list holds distinct values of its own kind. AllNames holds the
	// normalized name, and the case-variant pairs below normalize to one
	// string, so appending once per distinct spelling put that entry in the
	// list twice; searchFunc adds a matching entry's whole hash bucket, so
	// a search returned every printing of such a name once per spelling.
	for _, i := range cards {
		card := ac.Cards[i]
		// First-seen wins: two Lorcana cards whose names differ only in case
		// ("as"/"As") normalize equal, so last-wins would let a query for one
		// resolve to the other. Keep the first to make the mapping stable.
		n := mtgmatcher.Normalize(card.FullName)
		if b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.FullName
		}
		// The labels are the builder's, and nothing is worked out from
		// the fields it read them off. It is the half that can see whether
		// a varnish belongs to a rarity or to a printing, so a card it
		// gave no label is a card with no promotion - not one to derive
		// labels for, which is how HighGloss and "Promo" kept coming back
		// after being dropped on purpose.
		for _, tag := range card.PromoTypes {
			slug := mtgmatcher.PromoTypeSlug(tag)
			if !slices.Contains(b.AllPromoTypes, slug) {
				b.AllPromoTypes = append(b.AllPromoTypes, slug)
			}
			if b.PromoTypeLabels[slug] == "" {
				b.PromoTypeLabels[slug] = promoTypeLabel(slug)
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
	// Cardmarket ids are counted apart, being another vendor's integers:
	// Moana and Vaiana both claim 801862.
	cardmarketClaims := map[int]int{}
	for _, i := range cards {
		card := ac.Cards[i]
		claim(card.ExternalLinks.TcgPlayerID, card.ID)
		for _, extra := range card.ExternalLinks.TcgPlayerExtraIDs {
			claim(extra, card.ID)
		}
		if card.ExternalLinks.CardmarketID != 0 {
			cardmarketClaims[card.ExternalLinks.CardmarketID]++
		}
	}

	// Load all cards and store them in their relative sets
	for _, i := range cards {
		card := ac.Cards[i]
		// A card published with no printing has no uuid to price.
		if len(card.Printings) == 0 {
			continue
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

		promoTypes := slugTags(card.PromoTypes)

		// Prepare the card and add it to the main array
		// Since cards are already sorted (by number/id), the order here is preserved
		convertedCard := mtgmatcher.Card{
			UUID: cardUUID(card.ID),

			Name:    card.FullName,
			SetCode: card.SetCode,
			Number:  card.Number + card.Variant,
			Images:  card.Images,

			// English unless the printing says otherwise: core's filter drops
			// a candidate that is not English whenever two survive, empty ones
			// included.
			Language: cmp.Or(card.Language, "English"),

			Colors: colors,
			Rarity: rarity,

			Subtypes:   card.Subtypes,
			Types:      []string{card.Type},
			Supertypes: []string{card.Story},

			Printings:  printingsByName[mtgmatcher.Normalize(card.FullName)],
			IsPromo:    promoPrintings[card.ID] || card.PromoSourceCategory != "" || promoSet || rarity == "promo",
			PromoTypes: promoTypes,

			// IsOversized mirrors core's own "oversized" promoType.
			IsOversized: slices.Contains(promoTypes, "oversized"),

			PlainNumber: Rules{}.PlainNumber(card.Number + card.Variant),

			// The face's own denominator, which for Lorcana is not always
			// a size: a promo prints its run where a card of the set
			// prints the set's size, and the number alone named two cards
			// on 155 of the game's (set, number) pairs without it.
			SetTotal: card.Total,
		}
		// Register the uuid each finish prices, keyed by the name TCGplayer
		// prices it under, which is the name the datastore gives it.
		finishUUIDs := map[string]string{}
		type perFinish struct {
			uuid string
			foil bool
			name string
		}
		var stored []perFinish
		// Walked in order so the first of two names folding to one key
		// keeps it on every load.
		for _, name := range slices.Sorted(maps.Keys(card.PrintingIDs)) {
			finish := mtgmatcher.FinishSlug(name)
			if _, placed := finishUUIDs[finish]; placed {
				continue
			}
			uuid := card.PrintingIDs[name]
			finishUUIDs[finish] = uuid
			stored = append(stored, perFinish{uuid, mtgmatcher.IsFoilFinish(finish), finish})
		}
		// The CardObjects below are registered in this order, so
		// AllUUIDs and the name hashes come out the same on every load.
		sort.Slice(stored, func(i, j int) bool { return stored[i].name < stored[j].name })
		// A bare foil flag answers with the standard foil, or with the
		// treatment on a card sold only in one.
		if _, found := finishUUIDs[mtgmatcher.FinishFoil]; !found {
			if uuid, found := mtgmatcher.DefaultPrinting(finishUUIDs, true); found {
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
		if card.ExternalLinks.CardmarketID != 0 {
			mcmID := fmt.Sprint(card.ExternalLinks.CardmarketID)
			if convertedCard.Identifiers == nil {
				convertedCard.Identifiers = map[string]string{}
			}
			convertedCard.Identifiers["mcmId"] = mcmID
			if cardmarketClaims[card.ExternalLinks.CardmarketID] == 1 {
				b.ExternalIdentifiers[mtgmatcher.IDSpaceCardmarket][mcmID] = convertedCard.UUID
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
			if card.HasFinish(mtgmatcher.FinishNonfoil) {
				b.Sets[code].IsFoilOnly = false
			}
			if card.HasFinish(mtgmatcher.FinishFoil) {
				b.Sets[code].IsNonFoilOnly = false
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

	b.IndexSetUUIDs()

	b.SetRules(Rules{treatments: ac.treatments})

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
