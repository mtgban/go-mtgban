// Package riftbound loads a Riftbound (League of Legends TCG) datastore.
//
// The datastore is the official card-gallery payload, enriched by
// github.com/mtgban/datastore-gen with the TCGplayer product id of every
// printing and with the promotional printings the gallery does not carry;
// that repository publishes a ready-made file daily.
package riftbound

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

// CardGallery is the official Riftbound card-gallery payload served by
// https://riftbound.leagueoflegends.com/_next/data/<buildId>/en-us/card-gallery.json
// (the buildId is embedded in the card-gallery page itself). The gallery blade
// carries every published card inline; only the fields the loader needs are
// declared.
type CardGallery struct {
	PageProps struct {
		Page struct {
			Blades []GalleryBlade `json:"blades"`
		} `json:"page"`
	} `json:"pageProps"`
}

// galleryBladeType identifies the blade holding the card gallery among the
// page's other content blocks (navigation, masthead, ...).
const galleryBladeType = "riftboundCardGallery"

// GalleryBlade is the payload Riot's card gallery serves, named for the page
// section it arrives in.
type GalleryBlade struct {
	Type string `json:"type"`
	Sets struct {
		Items []GallerySet `json:"items"`
	} `json:"sets"`
	Cards struct {
		Items []GalleryCard `json:"items"`
	} `json:"cards"`

	// Sealed is not part of the official payload; the datastore builder
	// appends the sealed products the TCGplayer catalog files outside the
	// singles type, and mints a set for a group sold only sealed.
	Sealed struct {
		Items []GallerySealed `json:"items"`
	} `json:"sealed"`
}

// GallerySealed is a sealed product: a booster box, a display, a starter
// bundle. It has no collector number, no finish and no gallery entry - the
// TCGplayer product id is its whole identity.
type GallerySealed struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SetCode       string `json:"setCode"`
	Image         string `json:"image"`
	ExternalLinks struct {
		TcgPlayerID int `json:"tcgPlayerId"`
	} `json:"externalLinks"`
}

// GallerySet is one set as the gallery publishes it.
type GallerySet struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// BaseSetSize is the size of the numbered run, the gallery's
	// collectorNumberMax under the name every datastore gives it.
	BaseSetSize int `json:"baseSetSize"`

	// Type is not part of the official payload; the datastore builder
	// (github.com/mtgban/datastore-gen) marks the promotional sets it
	// appends with "promo", which gates how their printings match (see
	// rules.go).
	Type string `json:"type,omitempty"`

	// ReleaseDate is likewise stamped by the builder, from the day the
	// TCGplayer group went on sale ("2006-01-02").
	ReleaseDate string `json:"releaseDate,omitempty"`
}

// GalleryCard is one printing as the gallery publishes it.
type GalleryCard struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// SetCode, Number, Image and ExternalLinks are the gallery's set,
	// publicCode, cardImage and the builder's TCGplayer id, under the names
	// every other datastore uses; the builder publishes both spellings.
	SetCode       string `json:"setCode"`
	Number        string `json:"number"`
	Image         string `json:"image"`
	ExternalLinks struct {
		TcgPlayerID int `json:"tcgPlayerId"`
	} `json:"externalLinks"`

	CardType struct {
		Type []struct {
			ID string `json:"id"`
		} `json:"type"`
	} `json:"cardType"`
	Rarity struct {
		Value struct {
			ID string `json:"id"`
		} `json:"value"`
	} `json:"rarity"`
	Domain struct {
		Values []struct {
			ID string `json:"id"`
		} `json:"values"`
	} `json:"domain"`
	Tags struct {
		Tags []string `json:"tags"`
	} `json:"tags"`

	// Printings is what a card's printings are, one entry each, stamped by
	// the builder from the TCGplayer catalog since the gallery says nothing
	// about finish: the finish TCGplayer prices it under ("Normal", "Foil")
	// and the uuid it is quoted by. The uuid is the builder's to publish
	// rather than this package's to spell, since a uuid is what a price is
	// keyed on and a moved one resolves to nothing rather than erroring.
	Printings []struct {
		Finish string `json:"finish"`
		ID     string `json:"id"`
	} `json:"printings,omitempty"`

	// PromoTypes carries the parenthetical qualifiers the builder strips
	// from a promotional printing's TCGplayer name ("Sett - The Boss
	// (Metal) (Best Of)" becomes "Sett - The Boss" with promo types
	// "metal" and "best of"), so sibling promos share one clean name and
	// are told apart by number or by the storefront's own wording.
	PromoTypes []string `json:"promoTypes,omitempty"`
}

// Load reads an official card-gallery payload from r and returns a Backend
// for it, or an error when r does not hold a Riftbound card gallery.
//
// It reads the {"meta":...,"data":...} envelope the builders publish, where
// "data" holds the payload; a bare document reads as empty and is refused.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var envelope struct {
		Data CardGallery `json:"data"`
	}
	if err := json.NewDecoder(r).Decode(&envelope); err != nil {
		return nil, err
	}
	payload := envelope.Data
	for _, blade := range payload.PageProps.Page.Blades {
		if blade.Type != galleryBladeType {
			continue
		}
		if len(blade.Sets.Items) == 0 || len(blade.Cards.Items) == 0 {
			break
		}
		for _, card := range blade.Cards.Items {
			// No printing is a card with nothing to price, not a wrong file.
			if card.ID == "" || card.Name == "" {
				return nil, errors.New("not a Riftbound datastore")
			}
		}
		return blade.newBackend(), nil
	}
	return nil, errors.New("not a Riftbound datastore")
}

// qualifiedName spells a printing the way a storefront selling it does, the
// card name followed by the qualifiers that tell it from its siblings
// ("Teemo, Swift Scout (Metal Best Of)"). Several promotional printings
// share one name and one collector number, and those qualifiers are the only
// thing between them. Empty when a printing carries none, which the bare
// name already describes.
// productName rebuilds the name the catalog sells a printing under. The
// builder splits a promotional product's name into the base name and the
// qualifiers behind it, and putting them back gives the storefront's own
// spelling: "Teemo, Swift Scout" and [metal, best of] is "Teemo, Swift Scout
// (Metal) (Best Of)". Normalize drops the parentheses and the case, so the
// rebuilt name answers to the catalog's spelling exactly, and the qualifiers
// are taken before they are slugged because it is the words that spell it.
func productName(name string, promoTypes []string) string {
	if len(promoTypes) == 0 {
		return ""
	}
	return name + " (" + strings.Join(promoTypes, " ") + ")"
}

// slugPromoTypes renders every qualifier as the token that identifies it, so
// one word names it wherever it is read.
func slugPromoTypes(promoTypes []string) []string {
	var out []string
	for _, promoType := range promoTypes {
		out = append(out, mtgmatcher.PromoTypeSlug(promoType))
	}
	return out
}

// signaturePromoType is the qualifier a starred collector number stands for.
// The gallery marks the signed showcase printings with a star on the public
// code ("SFD-235*/221") and says nothing else about them: they carry no
// qualifier of their own, and the star is the whole of what separates them
// from the printing they share a name and a number with.
const signaturePromoType = "signature"

// signedPromoTypes is a card's own qualifiers plus the one its number
// implies, so the star reads as a tag rather than only as a number the
// storefronts have to spell exactly.
func signedPromoTypes(promoTypes []string, number string) []string {
	if !strings.HasSuffix(number, "*") {
		return promoTypes
	}
	for _, promoType := range promoTypes {
		if mtgmatcher.PromoTypeSlug(promoType) == signaturePromoType {
			return promoTypes
		}
	}
	return append(slices.Clone(promoTypes), signaturePromoType)
}

// describingPromoTypes drops the qualifiers that only repeat the collector
// number - the catalog names a rune variant "Fury Rune (R01c)", and the
// number field already says R01c, so as a tag it describes nothing and would
// read as one in a label. The card keeps the full list, which the matcher
// still reads to tell such printings apart.
func describingPromoTypes(promoTypes []string, number string) []string {
	var out []string
	for _, promoType := range promoTypes {
		// Both sides go through the number canonicalization first: the
		// qualifier keeps the catalog's zeros ("R06c") where the number has
		// already lost them ("r6c").
		if mtgmatcher.Normalize(canonicalNumber(promoType)) == mtgmatcher.Normalize(number) {
			continue
		}
		out = append(out, promoType)
	}
	return out
}

func (gallery *GalleryBlade) newBackend() *mtgmatcher.Backend {
	var b mtgmatcher.Backend

	// Keep the semantic name of the Vendetta T04 recruit stable when the
	// gallery omits its faction qualifier. The same card is named Recruit
	// (NX) in Origins, and callers use that qualifier to distinguish it from
	// the DE and ZN recruits. This is a published-name correction, not a
	// datastore-generator concern.
	for i := range gallery.Cards.Items {
		gallery.Cards.Items[i].Name = canonicalGalleryName(gallery.Cards.Items[i])
	}

	b.UUIDs = map[string]*mtgmatcher.CardObject{}
	b.Hashes = map[string][]string{}
	b.PromoTypeLabels = map[string]string{}
	b.CanonicalNames = map[string]string{}
	b.ExternalIdentifiers = map[string]map[string]string{mtgmatcher.IDSpaceTCGplayer: {}}
	b.SetSealedUUIDs = map[string][]string{}

	// Load all sets first
	b.Sets = map[string]*mtgmatcher.Set{}
	for _, set := range gallery.Sets.Items {
		b.AllSets = append(b.AllSets, set.ID)
		releaseDateTime, _ := time.Parse("2006-01-02", set.ReleaseDate)
		b.Sets[set.ID] = &mtgmatcher.Set{
			Name:            set.Name,
			Code:            set.ID,
			BaseSetSize:     set.BaseSetSize,
			Type:            set.Type,
			ReleaseDate:     set.ReleaseDate,
			ReleaseDateTime: releaseDateTime,
		}
	}
	sort.Strings(b.AllSets)
	b.IndexSets()

	// Gather the full reprint list for each name (keyed by normalized name),
	// in first-appearance order. Every card of a name carries the same
	// complete list, mirroring how Magic populates Printings, so
	// Printings4Card works unmodified for Riftbound. All cards of a name
	// share the same backing array; Printings is read-only by contract.
	printingsByName := map[string][]string{}
	for _, card := range gallery.Cards.Items {
		n := mtgmatcher.Normalize(card.Name)
		if !slices.Contains(printingsByName[n], card.SetCode) {
			printingsByName[n] = append(printingsByName[n], card.SetCode)
		}
	}

	// Load all card names. First-seen wins, mirroring the Lorcana loader.
	//
	// AllNames holds the normalized name, and 29 pairs of Riftbound names
	// normalize to one string - the promos spell an epithet off a dash
	// where the main sets use a comma, so "Teemo - Scout" and "Teemo,
	// Scout" both become "teemocout". Appending once per distinct spelling
	// put that entry in the list twice, and searchFunc adds a matching
	// entry's whole hash bucket, so a search returned every printing of
	// such a name once per spelling.
	for _, card := range gallery.Cards.Items {
		n := mtgmatcher.Normalize(card.Name)
		if b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.Name
		}
		number := collectorNumber(card.Number)
		for _, promoType := range describingPromoTypes(signedPromoTypes(card.PromoTypes, number), number) {
			slug := mtgmatcher.PromoTypeSlug(promoType)
			if !slices.Contains(b.AllPromoTypes, slug) {
				b.AllPromoTypes = append(b.AllPromoTypes, slug)
			}
			// The builder folds a qualifier to lower case on the way in, so
			// the spelling is looked up rather than guessed - title-casing
			// would render "GG EZ" as "Gg Ez".
			if label := promoTypeLabels[slug]; label != "" && b.PromoTypeLabels[slug] == "" {
				b.PromoTypeLabels[slug] = label
			}
		}
		// The catalog's own spelling is searchable but never canonical: it
		// names one printing where the bare name names the card, and Match
		// reads CanonicalNames to decide whether a name keeps its
		// parentheticals.
		if product := productName(card.Name, card.PromoTypes); product != "" {
			b.AddName(product)
		}
		b.AddName(card.Name)
	}
	sort.Strings(b.AllPromoTypes)
	sort.Strings(b.AllNames)
	sort.Strings(b.AllCanonicalNames)
	sort.Strings(b.AllLowerNames)

	// Load all cards and store them in their relative sets
	for _, card := range gallery.Cards.Items {
		setCode := card.SetCode
		if b.Sets[setCode] == nil {
			continue
		}
		// A card published with no printing has no uuid to price.
		finishes := cardFinishes(card)
		if len(finishes) == 0 {
			continue
		}

		var types []string
		for _, cardType := range card.CardType.Type {
			types = append(types, cardType.ID)
		}
		var colors []string
		for _, domain := range card.Domain.Values {
			colors = append(colors, domain.ID)
		}

		number := collectorNumber(card.Number)

		promoTypes := slugPromoTypes(signedPromoTypes(card.PromoTypes, number))
		convertedCard := mtgmatcher.Card{
			UUID: card.ID,

			Name:     card.Name,
			SetCode:  setCode,
			Finishes: finishes,
			Number:   number,
			Images: map[string]string{
				"full":      card.Image,
				"thumbnail": card.Image,
			},

			// The datastore is English-only. Core Match's language filter
			// drops any candidate whose Language differs from English when
			// several survive filtering, so leaving this empty would turn
			// every legitimate multi-candidate result (aliasing) into a
			// bogus wrong-variant error.
			Language: "English",

			// The promotional sets are typed as such by the builder, and
			// that is the whole of what makes a Riftbound printing a promo:
			// the gallery sets carry no such printings at all.
			IsPromo: b.Sets[setCode].Type == "promo",

			Colors: colors,
			Rarity: card.Rarity.Value.ID,

			Types:      types,
			Subtypes:   card.Tags.Tags,
			PromoTypes: promoTypes,

			IsOversized: slices.Contains(promoTypes, "oversized"),

			Printings: printingsByName[mtgmatcher.Normalize(card.Name)],

			PlainNumber: Rules{}.PlainNumber(number),
		}
		// Register the uuid each finish resolves to, spelling the finish out
		// in the uuid itself, so output()/Match resolve to them.
		convertedCard.FoilUUIDs = map[string]string{}
		for _, finish := range convertedCard.Finishes {
			convertedCard.FoilUUIDs[finish] = printingUUID(card, finish)
		}

		if card.ExternalLinks.TcgPlayerID != 0 {
			pid := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": pid,
			}
			// The product id names the printing, not one of its finishes, so
			// it points at the plain one where that exists, at the foil when
			// the card is only sold foil, and at whatever finish it is sold
			// in when that is a treatment alone. MatchID re-resolves the
			// finish from the caller's own flag either way.
			uuid, found := convertedCard.FoilUUIDs[mtgmatcher.FinishNonfoil]
			if !found {
				uuid, found = convertedCard.FoilUUIDs[mtgmatcher.FinishFoil]
			}
			if !found && len(convertedCard.Finishes) > 0 {
				uuid = convertedCard.FoilUUIDs[convertedCard.Finishes[0]]
			}
			b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer][pid] = uuid
		}

		b.Sets[setCode].Cards = append(b.Sets[setCode].Cards, convertedCard)

		// Store a CardObject per finish uuid, over the finishes the printing
		// is actually sold in rather than both: a card sold in one finish
		// has no uuid for the other, and reaching for it would file a
		// CardObject under the empty string. Every finish but the plain one
		// is a foil to the flag, the way Lorcana reads its treatments:
		// CanonicalFinish places a printing TCGplayer adds later, and one
		// sold in a treatment alone is not sold plain.
		for _, finish := range convertedCard.Finishes {
			s := struct {
				uuid string
				foil bool
				name string
			}{convertedCard.FoilUUIDs[finish], finish != mtgmatcher.FinishNonfoil, finish}
			if _, found := b.UUIDs[s.uuid]; found {
				continue
			}
			co := mtgmatcher.CardObject{
				Card:    convertedCard,
				Edition: b.Sets[setCode].Name,
				Foil:    s.foil,
			}
			// co is fresh on every iteration, so the stored pointer is not
			// aliased by later finishes
			co.UUID = s.uuid
			co.Finish = s.name
			b.UUIDs[s.uuid] = &co
			b.AllUUIDs = append(b.AllUUIDs, s.uuid)
			b.Hashes[mtgmatcher.Normalize(card.Name)] = append(b.Hashes[mtgmatcher.Normalize(card.Name)], s.uuid)
			if product := productName(card.Name, card.PromoTypes); product != "" {
				pn := mtgmatcher.Normalize(product)
				b.Hashes[pn] = append(b.Hashes[pn], s.uuid)
			}
		}
	}

	// Update any remaining details on Sets after Cards loading
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
			return riftboundRarityMap[rarities[i]] > riftboundRarityMap[rarities[j]]
		})
		b.Sets[code].Rarities = rarities

		sort.Strings(colors)
		b.Sets[code].Colors = colors
	}

	// Load sealed products. They live in the sealed namespace throughout:
	// their uuids join AllSealedUUIDs and their names the sealed name
	// index, and the product id is carried as an identifier for
	// BuildSealedProductMap rather than entering the external identifier
	// index, mirroring how Magic keeps sealed out of MatchID's reach.
	// Sealed products live in the sealed namespace throughout; AddSealed
	// is what files them there.
	for _, product := range gallery.Sealed.Items {
		b.AddSealed(product.ID, product.Name, product.SetCode, product.Image, product.ExternalLinks.TcgPlayerID)
	}
	b.SortSealed()

	b.IndexSetUUIDs()

	b.SetRules(Rules{})

	return &b
}

func canonicalGalleryName(card GalleryCard) string {
	if card.SetCode == "VEN" && card.Number == "T04" &&
		mtgmatcher.Equals(card.Name, "Recruit") {
		return "Recruit (NX)"
	}
	return card.Name
}

var riftboundRarityMap = map[string]int{
	"common":   1,
	"uncommon": 2,
	"rare":     3,
	"epic":     4,
	"showcase": 5,
}

// printingUUID is the uuid the datastore publishes for a finish.
func printingUUID(card GalleryCard, finish string) string {
	// Keyed by the datastore's own spelling, which is TCGplayer's, so the
	// key is placed the same way the finish it answers for was.
	for _, printing := range card.Printings {
		if printing.ID != "" && (Rules{}).CanonicalFinish(printing.Finish) == finish {
			return printing.ID
		}
	}
	return ""
}

// cardFinishes returns the finishes a printing is sold in, placed through
// CanonicalFinish from the TCGplayer names its printings carry - which also
// places a printing TCGplayer adds later without being taught it first. Most
// of the game is sold in one finish only, promotional printings being foil
// and starter cards plain. A printing published without a uuid has none to
// price and is left out.
func cardFinishes(card GalleryCard) []string {
	var out []string
	for _, printing := range card.Printings {
		finish := (Rules{}).CanonicalFinish(printing.Finish)
		if printing.ID == "" || finish == "" || slices.Contains(out, finish) {
			continue
		}
		out = append(out, finish)
	}
	return out
}

// collectorNumber canonicalizes the number a card is published under,
// the first of a two-faced token's pair ("066a" -> "66a", "T02//T03" ->
// "T2", "227*" -> "227*"). The letters are real: variants share their base
// card's digits and are told apart only by them.
func collectorNumber(number string) string {
	number, _, _ = strings.Cut(number, "/")
	return canonicalNumber(number)
}

// canonicalNumber strips leading zeros from the digit run of a collector
// number, preserving any letter prefix ("T01" -> "T1") and any suffix
// ("066a" -> "66a"). An all-zero run stays "0" so a genuine zero input
// errors instead of silently disabling the number filter.
func canonicalNumber(number string) string {
	i := 0
	for i < len(number) && (number[i] < '0' || number[i] > '9') {
		i++
	}
	prefix, rest := number[:i], number[i:]
	trimmed := strings.TrimLeft(rest, "0")
	if trimmed == "" && rest != "" {
		trimmed = "0"
	}
	return prefix + trimmed
}
