// Package fleshandblood loads a Flesh and Blood datastore.
//
// The datastore is built by github.com/mtgban/datastore-gen's
// cmd/fleshandblood from the TCGplayer catalog dump for category 62,
// annotated with the official Legend Story Studios card identifiers where
// the builder could align the two sources. Identity is the catalog's: every
// English single product is one card, and the print run and foil treatment
// combinations it sold in (1st Edition or Unlimited Edition or neither,
// crossed with Normal, Rainbow Foil or Cold Foil) are priced separately, so
// each entry is one printing of one product. The plain Normal entry keeps
// the bare id; the others suffix it with the edition and treatment slugs
// (_1e, _unlrainbow, _cold); sibling entries share their product's
// tcgPlayerId. Extended arts, marvels and the other alternate printings
// share their base card's collector number and are told apart by the
// variant label the builder distills from the product name.
package fleshandblood

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

// The two axes a finish name is built from, as the game's rules spell them:
// a print run - the bare one carries no name at all - and a treatment.
const (
	editionBare      = ""
	edition1st       = "1stedition"
	editionUnlimited = "unlimitededition"

	treatmentNormal      = "normal"
	treatmentRainbowFoil = "rainbowfoil"
	treatmentColdFoil    = "coldfoil"
)

// Datastore is the cmd/fleshandblood output: sets keyed by code, one card
// entry per priced printing, and the sealed products.
type Datastore struct {
	Game string `json:"game"`
	Sets map[string]struct {
		Name        string `json:"name"`
		ReleaseDate string `json:"releaseDate"`
		// Type is "promo" on the sets that hand their cards out rather
		// than sell them in packs, and empty on every other.
		Type string `json:"type,omitempty"`
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

	// Finish is the TCGplayer printing this entry prices, drawn from the
	// closed vocabulary of "1st Edition" or "Unlimited Edition" (or
	// neither) crossed with "Normal", "Rainbow Foil" or "Cold Foil".
	// Entries sharing everything but the finish are the same product sold
	// several ways.
	Finish string `json:"finish"`

	// Variant is the label distilled from the product name's qualifiers:
	// empty for the base printing, "Extended Art", "Marvel", "Golden" or
	// the like for the alternate printings that share its number.
	Variant string `json:"variant,omitempty"`

	// PromoTypes is the same residue as the labels it is made of, which
	// the joined Variant cannot be read back into: "Cold Foil Extended
	// Art" is two labels, and a query naming either has to reach the
	// printing.
	PromoTypes []string `json:"promoTypes,omitempty"`

	// FabID is the official Legend Story Studios card identifier,
	// annotated where the builder could align the two sources.
	FabID string `json:"fabId,omitempty"`

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

// Load reads a Flesh and Blood datastore from r and returns a Backend for
// it, or an error when r holds something else (so LoadDatastore's
// auto-detection can move on to the next registered game). The datastore
// names its game at the root, and every card carries the identity fields
// the backend is built from. The collector number is not among them: the
// set art cards carry none, and a card the catalog sells under no number
// still has to be sold.
func Load(r io.Reader) (*mtgmatcher.Backend, error) {
	var payload Datastore
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Game != "fleshandblood" || len(payload.Sets) == 0 || len(payload.Cards) == 0 {
		return nil, errors.New("not a Flesh and Blood datastore")
	}
	for _, card := range payload.Cards {
		if card.ID == "" || card.Name == "" || card.Finish == "" {
			return nil, errors.New("not a Flesh and Blood datastore")
		}
	}
	return payload.newBackend(), nil
}

// restatesNumber reports whether a label only respells the printing's own
// collector number, which as a tag describes nothing the number does not
// already say. Both sides lose their separators and their leading zeros
// first: the catalog writes "HER0156" where the label says "HER156" and
// "MST158-A" where it says "158-A". A label naming a different printing's
// number stays - "FAB442" on PEN331 is a promo pointing at the printing it
// reprints, which is worth reading.
func restatesNumber(variant, number string) bool {
	label := foldNumber(variant)
	full := foldNumber(number)
	if label == "" || full == "" {
		return false
	}
	if label == full || label == strings.TrimLeft(full, "abcdefghijklmnopqrstuvwxyz") {
		return true
	}
	// The base number of a suffixed printing ("LSS003" on LSS003-CF): what
	// follows the label has to open a suffix rather than carry on the digits,
	// so "HER1" is not read as a respelling of HER156.
	rest := strings.TrimPrefix(full, label)
	if rest == full || rest == "" {
		return false
	}
	return rest[0] < '0' || rest[0] > '9'
}

// foldNumber reduces a collector number to the letters and digits that carry
// it, dropping the zeros each digit run is padded with.
func foldNumber(number string) string {
	var out strings.Builder
	zeros := true
	for _, r := range strings.ToLower(number) {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
			zeros = true
		case r >= '0' && r <= '9':
			if r == '0' && zeros {
				continue
			}
			zeros = false
			out.WriteRune(r)
		}
	}
	return out.String()
}

// setTypePromo is what the builder types a set that hands its cards out.
const setTypePromo = "promo"

// promoTypesOf reads a printing's labels, preferring the list the builder
// distills them into. A datastore built before that list was recorded
// carries only the joined spelling, which stays one label rather than being
// split on spaces: several are two words long ("Extended Art"), and
// splitting would leave halves that name nothing.
func promoTypesOf(card *DatastoreCard) []string {
	if len(card.PromoTypes) > 0 {
		return card.PromoTypes
	}
	if card.Variant == "" {
		return nil
	}
	return []string{card.Variant}
}

// promoTypeSlugs is promoTypesOf as the tokens a query can carry, which is
// what a card stores: a search splits its words apart before a filter sees
// them, so a tag only survives the trip as one.
func promoTypeSlugs(card *DatastoreCard) []string {
	labels := promoTypesOf(card)
	if len(labels) == 0 {
		return nil
	}
	out := make([]string, 0, len(labels))
	for _, label := range labels {
		out = append(out, mtgmatcher.PromoTypeSlug(label))
	}
	return out
}

// describingPromoTypes keeps the labels worth declaring as tags: the ones
// the printing's own finish or number does not already say. The card keeps
// the full list either way, which the matcher still reads to tell sibling
// printings apart; only the declaration is filtered, the same terms
// Riftbound carries its number-restating labels on.
func describingPromoTypes(card *DatastoreCard) []string {
	var out []string
	for _, promoType := range promoTypesOf(card) {
		if describingVariant(promoType, card.Finish, card.Number) == "" {
			continue
		}
		out = append(out, promoType)
	}
	return out
}

// describingVariant drops the labels a printing's finish already says - the
// catalog names a Cold Foil promo "... (Cold Foil)" and prices it as a Cold
// Foil, and a 1st Edition rainbow arrives labelled "Rainbow" - since as a
// tag they describe nothing the printing does not already spell. The label
// is weighed against the finish's own slugs with the trailing "Foil" and
// "Edition" optional rather than by containment, so the one-letter labels an
// art variant carries ("C" beside a Cold Foil) are not read as an
// abbreviation of one. The card keeps the label either way: FilterCards
// tiers the candidates by it, and only the declaration is filtered.
func describingVariant(variant, finish, number string) string {
	if restatesNumber(variant, number) {
		return ""
	}
	label := canonicalFinish(variant)
	if label == "" {
		return ""
	}
	sold := canonicalFinish(finish)
	for _, slug := range []string{
		edition1st, editionUnlimited,
		treatmentNormal, treatmentRainbowFoil, treatmentColdFoil,
	} {
		if !strings.Contains(sold, slug) {
			continue
		}
		bare := strings.TrimSuffix(strings.TrimSuffix(slug, "foil"), "edition")
		if label == slug || label == bare {
			return ""
		}
	}
	return variant
}

// qualifiedName spells a printing the way TCGplayer names the product, the
// card name followed by the label that tells it from the printings sharing
// its number ("Enigma, Ledger of Ancestry (Marvel)"). Empty for a printing
// the catalog qualifies with nothing, or with nothing its finish has not
// already said.
//
// Empty too for a spelling the datastore already carries as a name of its
// own: the pitch color belongs to a Flesh and Blood name ("Dig In (Red)")
// and a few products file it as a label beside the plain name instead, so
// indexing the label's spelling would pour one product's printings into the
// bucket the whole name answers with.
func qualifiedName(card *DatastoreCard, printingsByName map[string][]string) string {
	variant := describingVariant(card.Variant, card.Finish, card.Number)
	if variant == "" {
		return ""
	}
	qualified := card.Name + " (" + variant + ")"
	if printingsByName[mtgmatcher.Normalize(qualified)] != nil {
		return ""
	}
	return qualified
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

	// Each list holds distinct values of its own kind, and two spellings can
	// normalize or lowercase to one string, so each is deduped on what it
	// actually holds: searchFunc adds a matching entry's whole hash bucket,
	// and a key stored twice returns that bucket twice.
	for _, card := range payload.Cards {
		n := mtgmatcher.Normalize(card.Name)
		if b.CanonicalNames[n] == "" {
			b.CanonicalNames[n] = card.Name
		}
		for _, promoType := range describingPromoTypes(&card) {
			slug := mtgmatcher.PromoTypeSlug(promoType)
			if !slices.Contains(b.AllPromoTypes, slug) {
				b.AllPromoTypes = append(b.AllPromoTypes, slug)
			}
			// The builder folds a qualifier to lower case on the way in,
			// so the words are title-cased back and the acronyms looked
			// up. First spelling seen wins: the catalog writes a few of
			// these two ways, and one token can only read back as one.
			if b.PromoTypeLabels[slug] == "" {
				b.PromoTypeLabels[slug] = promoTypeLabel(promoType)
			}
		}
		// Searchable but never canonical: the qualified spelling names one
		// printing where the bare name names the card, and Match reads
		// CanonicalNames to decide whether a name keeps its parentheticals.
		if qualified := qualifiedName(&card, printingsByName); qualified != "" {
			b.AddName(qualified)
		}
		b.AddName(card.Name)
	}
	sort.Strings(b.AllPromoTypes)
	sort.Strings(b.AllNames)
	sort.Strings(b.AllCanonicalNames)
	sort.Strings(b.AllLowerNames)

	// Group sibling entries back into their product: a product priced in
	// several printings is the same card several times, and the matcher
	// wants it once, with FoilUUIDs naming the uuid each printing prices.
	// The builder stamps every entry with its product's tcgPlayerId; an
	// entry left without one falls back to its id with the finish suffix
	// stripped.
	var productOrder []string
	products := map[string][]*DatastoreCard{}
	for i := range payload.Cards {
		card := &payload.Cards[i]
		key := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
		if card.ExternalLinks.TcgPlayerID == 0 {
			key = trimFinishSuffix(card.ID)
		}
		if _, found := products[key]; !found {
			productOrder = append(productOrder, key)
		}
		products[key] = append(products[key], card)
	}

	for _, key := range productOrder {
		group := products[key]

		// The flag-driven defaults are the plainest entry of each foilness
		// class, preferring the bare print run, then Unlimited, then 1st
		// Edition — and Rainbow Foil over Cold Foil for the foil slot.
		nonfoil := pickFinish(group, runsFor(treatmentNormal)...)
		foil := pickFinish(group, append(
			runsFor(treatmentRainbowFoil), runsFor(treatmentColdFoil)...)...)

		// The product id and the set-level card follow the nonfoil default
		// where one exists, exactly as riftbound points a product at its
		// plain printing; MatchID re-resolves the finish from the caller's
		// own flag either way.
		card := nonfoil
		if card == nil {
			card = foil
		}
		if card == nil {
			card = group[0]
		}
		if b.Sets[card.SetCode] == nil {
			continue
		}

		promoTypes := promoTypeSlugs(card)

		// Only the foilness classes the product is actually sold in are
		// registered: output() folds a storefront's unreliable foil flag
		// onto the sold class, and the finish the input names re-keys onto
		// the specific printing.
		var finishes []string
		foilUUIDs := map[string]string{}
		if nonfoil != nil {
			finishes = append(finishes, mtgmatcher.FinishNonfoil)
			foilUUIDs[mtgmatcher.FinishNonfoil] = nonfoil.ID
		}
		if foil != nil {
			finishes = append(finishes, mtgmatcher.FinishFoil)
			foilUUIDs[mtgmatcher.FinishFoil] = foil.ID
		}
		for _, entry := range group {
			foilUUIDs[canonicalFinish(entry.Finish)] = entry.ID
		}

		// A source that names the treatment alone - cardtrader files the
		// print run on the expansion and leaves the listing with "Cold
		// Foil" - means whichever run this product was sold in, so the
		// bare name is registered as a spelling reaching the plainest one
		// it has. Only where the product has no bare printing of its own:
		// an alias is a spelling, and must never shadow a real entry.
		finishAliases := map[string]string{}
		for _, treatment := range []string{treatmentNormal, treatmentRainbowFoil, treatmentColdFoil} {
			if _, sold := foilUUIDs[treatment]; sold {
				continue
			}
			for _, edition := range []string{editionUnlimited, edition1st} {
				if _, found := foilUUIDs[edition+treatment]; found {
					finishAliases[treatment] = edition + treatment
					break
				}
			}
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
			Rarity:     card.Rarity,
			PromoTypes: promoTypes,
			IsPromo:    payload.Sets[card.SetCode].Type == setTypePromo,
			Printings:  printingsByName[mtgmatcher.Normalize(card.Name)],

			OriginalNumber: Rules{}.PlainNumber(card.Number),
		}
		convertedCard.FoilUUIDs = foilUUIDs
		if len(finishAliases) > 0 {
			convertedCard.FinishAliases = finishAliases
		}

		if card.ExternalLinks.TcgPlayerID != 0 {
			pid := fmt.Sprint(card.ExternalLinks.TcgPlayerID)
			convertedCard.Identifiers = map[string]string{
				"tcgplayerProductId": pid,
			}
			if card.FabID != "" {
				convertedCard.Identifiers["fabId"] = card.FabID
			}
			b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer][pid] = card.ID
		}

		b.Sets[card.SetCode].Cards = append(b.Sets[card.SetCode].Cards, convertedCard)

		var qualified string
		if name := qualifiedName(card, printingsByName); name != "" {
			qualified = mtgmatcher.Normalize(name)
		}
		for _, entry := range group {
			finish := canonicalFinish(entry.Finish)
			co := mtgmatcher.CardObject{
				Card:    convertedCard,
				Edition: b.Sets[card.SetCode].Name,
				Foil:    !strings.HasSuffix(finish, treatmentNormal),
			}
			// co is fresh on every iteration, so the stored pointer is not
			// aliased by the sibling printings
			co.UUID = entry.ID
			co.Finish = finish
			b.UUIDs[entry.ID] = &co
			b.AllUUIDs = append(b.AllUUIDs, entry.ID)
			b.Hashes[mtgmatcher.Normalize(card.Name)] = append(b.Hashes[mtgmatcher.Normalize(card.Name)], entry.ID)
			if qualified != "" {
				b.Hashes[qualified] = append(b.Hashes[qualified], entry.ID)
			}
		}
	}

	for code := range b.Sets {
		var rarities []string
		for _, card := range b.Sets[code].Cards {
			if !slices.Contains(rarities, card.Rarity) {
				rarities = append(rarities, card.Rarity)
			}
		}
		sort.Slice(rarities, func(i, j int) bool {
			return fleshandbloodRarityMap[rarities[i]] > fleshandbloodRarityMap[rarities[j]]
		})
		b.Sets[code].Rarities = rarities
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

var fleshandbloodRarityMap = map[string]int{
	"Token":        1,
	"Basic":        2,
	"Common":       3,
	"Rare":         4,
	"Super Rare":   5,
	"Majestic":     6,
	"Legendary":    7,
	"Fabled":       8,
	"Marvel":       9,
	"Gold":         10,
	"Pirate Booty": 11,
	"Promo":        12,
}

// runsFor names one treatment across the print runs, plainest first, the
// order the flag-driven defaults are chosen in.
func runsFor(treatment string) []string {
	return []string{
		editionBare + treatment,
		editionUnlimited + treatment,
		edition1st + treatment,
	}
}

// pickFinish returns the group's first entry of the first printing present,
// in the given preference order, or nil when none of them is.
func pickFinish(group []*DatastoreCard, finishes ...string) *DatastoreCard {
	for _, finish := range finishes {
		for _, entry := range group {
			if canonicalFinish(entry.Finish) == finish {
				return entry
			}
		}
	}
	return nil
}

// trimFinishSuffix strips the finish tail the builder suffixes ids with
// (the plain Normal entry keeps the bare id), the grouping fallback for an
// entry without a tcgPlayerId.
func trimFinishSuffix(id string) string {
	for _, suffix := range []string{
		"_1e", "_1erainbow", "_1ecold",
		"_unl", "_unlrainbow", "_unlcold",
		"_rainbow", "_cold",
	} {
		if strings.HasSuffix(id, suffix) {
			return strings.TrimSuffix(id, suffix)
		}
	}
	return id
}
