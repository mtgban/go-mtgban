// Package riftbound loads a Riftbound (League of Legends TCG) datastore.
//
// The datastore is the official card-gallery payload, enriched by
// github.com/mtgban/riftbound-datastore with the TCGplayer product id of
// every printing and with the promotional printings the gallery does not
// carry; that repository publishes a ready-made file daily. The plain
// gallery payload loads too, only without identifiers or promo sets.
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
	// singles type, in the card items' own vocabulary. A datastore built
	// before this was recorded simply loads without sealed products.
	Sealed struct {
		Items []GallerySealed `json:"items"`
	} `json:"sealed"`
}

// GallerySealed is a sealed product: a booster box, a display, a starter
// bundle. It has no collector number, no finish and no gallery entry - the
// TCGplayer product id is its whole identity.
type GallerySealed struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Set  struct {
		Value struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"value"`
	} `json:"set"`
	CardImage struct {
		URL string `json:"url"`
	} `json:"cardImage"`
	TCGplayerProductID int    `json:"tcgplayerProductId"`
	ReleaseDate        string `json:"releaseDate,omitempty"`
}

// GallerySet is one set as the gallery publishes it.
type GallerySet struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	CollectorNumberMax int    `json:"collectorNumberMax"`

	// Type is not part of the official payload; the datastore builder
	// (github.com/mtgban/riftbound-datastore) marks the promotional sets it
	// appends with "promo", which gates how their printings match (see
	// rules.go).
	Type string `json:"type,omitempty"`

	// ReleaseDate is likewise stamped by the builder, from the day the
	// TCGplayer group went on sale ("2006-01-02").
	ReleaseDate string `json:"releaseDate,omitempty"`
}

// GalleryCard is one printing as the gallery publishes it.
type GalleryCard struct {
	ID              string `json:"id"`
	CollectorNumber int    `json:"collectorNumber"`
	Name            string `json:"name"`
	PublicCode      string `json:"publicCode"`
	Orientation     string `json:"orientation"`
	Set             struct {
		Value struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"value"`
	} `json:"set"`
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
	CardImage struct {
		URL string `json:"url"`
	} `json:"cardImage"`
	Tags struct {
		Tags []string `json:"tags"`
	} `json:"tags"`

	// TCGplayerProductID is not part of the official payload; the datastore
	// builder (github.com/mtgban/riftbound-datastore) stamps each card with
	// the TCGplayer product id it maps to, feeding the external identifier
	// index.
	TCGplayerProductID int `json:"tcgplayerProductId,omitempty"`

	// Finishes is likewise stamped by the builder, from the printings the
	// TCGplayer catalog lists for that product. The gallery says nothing
	// about finish, so a datastore built before this was recorded leaves it
	// empty and every card falls back to being sold in both.
	Finishes []string `json:"finishes,omitempty"`

	// PromoTypes carries the parenthetical qualifiers the builder strips
	// from a promotional printing's TCGplayer name ("Sett - The Boss
	// (Metal) (Best Of)" becomes "Sett - The Boss" with promo types
	// "metal" and "best of"), so sibling promos share one clean name and
	// are told apart by number or by the storefront's own wording.
	PromoTypes []string `json:"promoTypes,omitempty"`
}

// Load reads an official card-gallery payload from r and returns a Backend
// for it, or an error when r does not hold a Riftbound card gallery (so
// LoadDatastore's auto-detection can move on to the next registered game).
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload CardGallery
	err := json.NewDecoder(r).Decode(&payload)
	if err != nil {
		return nil, err
	}
	for _, blade := range payload.PageProps.Page.Blades {
		if blade.Type != galleryBladeType {
			continue
		}
		if len(blade.Sets.Items) == 0 || len(blade.Cards.Items) == 0 {
			break
		}
		return blade.newBackend(), nil
	}
	return nil, errors.New("not a Riftbound card-gallery payload")
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
		if mtgmatcher.Normalize(CanonicalNumber(promoType)) == mtgmatcher.Normalize(number) {
			continue
		}
		out = append(out, promoType)
	}
	return out
}

func (gallery *GalleryBlade) newBackend() *mtgmatcher.Backend {
	var b mtgmatcher.Backend

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
			BaseSetSize:     set.CollectorNumberMax,
			Type:            set.Type,
			ReleaseDate:     set.ReleaseDate,
			ReleaseDateTime: releaseDateTime,
		}
	}
	// A sealed product can belong to a group the gallery has no set for
	// (an accessories-only group, a set sold before its cards are
	// published); give it a set to hang off
	for _, product := range gallery.Sealed.Items {
		if b.Sets[product.Set.Value.ID] != nil {
			continue
		}
		b.AllSets = append(b.AllSets, product.Set.Value.ID)
		releaseDateTime, _ := time.Parse("2006-01-02", product.ReleaseDate)
		b.Sets[product.Set.Value.ID] = &mtgmatcher.Set{
			Name:            product.Set.Value.Label,
			Code:            product.Set.Value.ID,
			ReleaseDate:     product.ReleaseDate,
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
		if !slices.Contains(printingsByName[n], card.Set.Value.ID) {
			printingsByName[n] = append(printingsByName[n], card.Set.Value.ID)
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
		number := numberFromPublicCode(card.PublicCode)
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
		setCode := card.Set.Value.ID
		if b.Sets[setCode] == nil {
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

		number := numberFromPublicCode(card.PublicCode)

		convertedCard := mtgmatcher.Card{
			UUID: card.ID,

			Name:     card.Name,
			SetCode:  setCode,
			Finishes: cardFinishes(card),
			Number:   number,
			Images: map[string]string{
				"full":      card.CardImage.URL,
				"thumbnail": card.CardImage.URL,
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
			PromoTypes: slugPromoTypes(signedPromoTypes(card.PromoTypes, number)),

			Printings: printingsByName[mtgmatcher.Normalize(card.Name)],

			OriginalNumber: Rules{}.PlainNumber(number),
		}
		// Register the uuid each finish resolves to, spelling the finish out
		// in the uuid itself, so output()/Match resolve to them.
		convertedCard.FoilUUIDs = map[string]string{}
		for _, finish := range convertedCard.Finishes {
			convertedCard.FoilUUIDs[finish] = card.ID + "_" + finish
		}

		if card.TCGplayerProductID != 0 {
			pid := fmt.Sprint(card.TCGplayerProductID)
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": pid,
			}
			// The product id names the printing, not one of its finishes, so
			// it points at the plain one where that exists and at the foil
			// when the card is only sold foil. MatchID re-resolves the finish
			// from the caller's own flag either way.
			uuid, found := convertedCard.FoilUUIDs[mtgmatcher.FinishNonfoil]
			if !found {
				uuid = convertedCard.FoilUUIDs[mtgmatcher.FinishFoil]
			}
			b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer][pid] = uuid
		}

		b.Sets[setCode].Cards = append(b.Sets[setCode].Cards, convertedCard)

		// Store a CardObject per finish uuid, over the finishes the printing
		// is actually sold in rather than both: a card sold in one finish
		// has no uuid for the other, and reaching for it would file a
		// CardObject under the empty string.
		for _, finish := range convertedCard.Finishes {
			s := struct {
				uuid string
				foil bool
				name string
			}{convertedCard.FoilUUIDs[finish], finish == mtgmatcher.FinishFoil, finish}
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
		b.AddSealed(product.ID, product.Name, product.Set.Value.ID, product.CardImage.URL, product.TCGplayerProductID)
	}
	b.SortSealed()

	b.SetRules(Rules{})

	return &b
}

var riftboundRarityMap = map[string]int{
	"common":   1,
	"uncommon": 2,
	"rare":     3,
	"epic":     4,
	"showcase": 5,
}

// cardFinishes returns the finishes a printing is sold in. The gallery says
// nothing about finish, so the builder stamps what the TCGplayer catalog
// lists for the product it maps to; most of the game is sold in one finish
// only, promotional printings being foil and starter cards plain.
//
// A datastore built before that was recorded says nothing, and the honest
// answer there is both: it is the assumption the whole game was loaded under
// until now, and narrowing on no evidence would strand real printings.
func cardFinishes(card GalleryCard) []string {
	var out []string
	for _, finish := range card.Finishes {
		switch finish {
		case mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil:
			out = append(out, finish)
		}
	}
	if len(out) == 0 {
		return []string{mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil}
	}
	return out
}

// numberFromPublicCode extracts the collector number from a card's public
// code: what follows the set prefix, without the "/total" tail, canonicalized
// ("OGN-066a/298" -> "66a", "UNL-T01" -> "T1", "SFD-227*/221" -> "227*").
// The letter suffixes and prefixes are real: variants share their base card's
// numeric collectorNumber and are told apart only here.
func numberFromPublicCode(publicCode string) string {
	code := publicCode
	if idx := strings.IndexByte(code, '-'); idx >= 0 {
		code = code[idx+1:]
	}
	code = strings.Split(code, "/")[0]
	return CanonicalNumber(code)
}

// CanonicalNumber strips leading zeros from the digit run of a collector
// number, preserving any letter prefix ("T01" -> "T1") and any suffix
// ("066a" -> "66a"). An all-zero run stays "0" so a genuine zero input
// errors instead of silently disabling the number filter.
func CanonicalNumber(number string) string {
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
