package cardmarket

import (
	"errors"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// pokemonForeignExpansions names the Cardmarket Pokemon expansions that are
// Japanese or other non-English catalogs wearing an English set's name, so
// GetSetByName resolves them onto the English set: "Mystery of the Fossils"
// is Japan's Fossil, "Pokémon Jungle" trims down to Jungle, and "XY Promos"
// is the very name the datastore gives the English promos Cardmarket sells
// as "XY Black Star Promos". Without the gate their singles price English
// printings from another market's stock - 115 wrong landings when measured.
// The list is Cardmarket's own naming, which is why it lives here and not
// in the matcher: other storefronts mean the English set by some of these
// spellings.
//
// What proves each entry is the CardTrader bridge: none of the expansions'
// products link to a TCGplayer id in the English Pokemon catalog, while a
// genuinely English expansion's products overwhelmingly do. The bridge
// route stays open for them and answers nothing for the same reason.
//
// The value says whether a product naming a collector number may pass. The
// datastore's "XY Promos" set itself carries twelve Japanese promos under
// their ##/XY-P numbers - Mega Tokyo's Pikachu, the poncho Pikachus - and
// Cardmarket's numbered products land exactly on them; only its unnumbered
// products were landing on English printings. Everywhere else the wrong
// landings carried numbers too, so nothing passes.
var pokemonForeignExpansions = map[string]bool{
	"Advent of Arceus":                       false,
	"Great Detective Pikachu":                false,
	"Magma Gang VS Aqua Gang: Double Crisis": false,
	"Mystery of the Fossils":                 false,
	"Pikachu Legendary Celebration":          false,
	"Pokémon Jungle":                         false,
	"Scarlet & Violet Battle Academy":        false,
	"The Glory of Team Rocket":               false,
	"XY Promos":                              true,
	// A shelf of Japanese vending, CD and magazine promos, the "_____'s
	// Pikachu" and "Imakuni?" kind, with no number on any of them.
	"Unnumbered Promos": false,
}

// pokemonForeign reports whether an expansion is one of those catalogs.
func pokemonForeign(expansion string) bool {
	_, found := pokemonForeignExpansions[expansion]
	return found
}

// pokemonForeignDenied reports whether a Pokemon product from a foreign
// expansion must not reach the name path.
func pokemonForeignDenied(expansion, number string) bool {
	numberedOK, found := pokemonForeignExpansions[expansion]
	if !found {
		return false
	}
	return !(numberedOK && number != "")
}

// pokemonExpansion is what a Cardmarket expansion name means in the
// catalog's terms: the sets its products may be printings of, tried in
// order, and the programme prefix the promo numbers are written without.
type pokemonExpansion struct {
	sets   []string
	prefix string
}

// pokemonExpansions maps the Cardmarket Pokemon expansions the matcher
// resolves to no set onto the sets their bridged products land on. Each
// entry was read off the cardtrader bridge: the products of the expansion
// that carry a TCGplayer id land in the set named, and the name path is
// held to the same set.
var pokemonExpansions = map[string]pokemonExpansion{
	"Scarlet & Violet":          {sets: []string{"SV01: Scarlet & Violet Base Set"}},
	"Sword & Shield":            {sets: []string{"SWSH01: Sword & Shield Base Set"}},
	"Sun & Moon":                {sets: []string{"SM Base Set"}},
	"151":                       {sets: []string{"SV: Scarlet & Violet 151"}},
	"XY Black Star Promos":      {sets: []string{"XY Promos"}, prefix: "XY"},
	"SM Black Star Promos":      {sets: []string{"SM Promos"}, prefix: "SM"},
	"SWSH Black Star Promos":    {sets: []string{"SWSH: Sword & Shield Promo Cards"}, prefix: "SWSH"},
	"SV Black Star Promos":      {sets: []string{"SV: Scarlet & Violet Promo Cards"}},
	"MEP Black Star Promos":     {sets: []string{"ME: Mega Evolution Promo"}},
	"BW Black Star Promos":      {sets: []string{"Black and White Promos"}, prefix: "BW"},
	"DP Black Star Promos":      {sets: []string{"Diamond and Pearl Promos"}, prefix: "DP"},
	"Southeast Asia Promos":     {sets: []string{"Southeast Asia Exclusives"}},
	"Professor Program":         {sets: []string{"Professor Program Promos"}},
	"Best of Game Cards Promos": {sets: []string{"Best of Promos"}},
	"Promos":                    {sets: []string{"League & Championship Cards", "Jumbo Cards"}},
	"XY Trainer Kit":            {sets: []string{"XY Trainer Kit: Sylveon & Noivern"}},
	"Futsal Promos":             {sets: []string{"Miscellaneous Cards & Products"}},
}

// pokemonAdditionals are the sets an "Additionals" expansion draws on: the
// stamped, cosmos, deck and blister printings the base set's own products
// are not. The base set is left out on purpose - the products here that
// are its own printings are its pattern reprints, which the name cannot
// tell from the plain card.
var pokemonAdditionals = []string{"Miscellaneous Cards & Products", "Deck Exclusives", "Blister Exclusives", "Jumbo Cards"}

var (
	pokemonPrizePack = "Play! Pokémon Prize Pack Series "
	pokemonWorlds    = regexp.MustCompile(`^WCD (\d{4})$`)
	pokemonBracket   = regexp.MustCompile(`\s*\[[^\]]*\]\s*$`)
)

// pokemonEditions answers the sets a Cardmarket expansion may hold and the
// number prefix its promos need.
func pokemonEditions(expansion string) ([]string, string) {
	if alias, found := pokemonExpansions[expansion]; found {
		return alias.sets, alias.prefix
	}
	if strings.HasPrefix(expansion, pokemonPrizePack) {
		return []string{"Prize Pack Series Cards"}, ""
	}
	if m := pokemonWorlds.FindStringSubmatch(expansion); m != nil {
		return []string{m[1] + " World Championship Decks"}, ""
	}
	if base, found := strings.CutSuffix(expansion, ": Additionals"); found {
		if _, err := mtgmatcher.GetSetByName(base); err == nil {
			return pokemonAdditionals, ""
		}
	}
	return []string{expansion}, ""
}

// pokemonName reads the card's name off a product name: Cardmarket tells
// same-name products apart by their attacks in brackets ("Tapu Koko [Flying
// Flip | Electric Ball]"), writes an energy symbol inside a name as its
// letter in brackets ("Magnetic [M] Energy", "Unit Energy [LPM]") where the
// catalog spells one letter out and keeps several as they are, and pads
// some names with a trailing space.
func pokemonName(name string) string {
	name = pokemonBracket.ReplaceAllString(name, "")
	name = pokemonSymbol.ReplaceAllStringFunc(name, func(symbol string) string {
		letters := strings.Trim(symbol, "[]")
		if word, found := pokemonEnergyLetters[letters]; found {
			return word
		}
		return letters
	})
	return strings.Join(strings.Fields(name), " ")
}

var pokemonSymbol = regexp.MustCompile(`\[[A-Z]+\]`)

var pokemonEnergyLetters = map[string]string{
	"G": "Grass", "R": "Fire", "W": "Water", "L": "Lightning", "P": "Psychic",
	"F": "Fighting", "D": "Darkness", "M": "Metal", "Y": "Fairy", "N": "Dragon", "C": "Colorless",
}

// pokemonCodeCard reports whether a product is a code for the online game
// rather than a card.
func pokemonCodeCard(name string) bool {
	return strings.Contains(name, "Code Card")
}

// twinsAmong refuses the by-name results whose printing a product the game
// calls the same in the same expansion already holds. Cardmarket sells a
// stamped, cosmos or deck variant as a second product with the same name and
// number and nothing else to tell it by, an unnumbered product beside a
// numbered one of the same name, and a print run as a version index; the
// name reaches the plain printing for all of them. An id-resolved sibling
// holds its printing outright; among by-name siblings the first holds it
// and the rest give way. A printing held by a product of another name is
// left to the inventory to refuse out loud, since that is a disagreement
// worth reading.
func twinsAmong(results []resolved, same func(a, b *MKMProduct) bool, face func(product *MKMProduct, cardID string) bool) {
	held := map[string][]int{}
	var holders []int
	for i, r := range results {
		if r.err == nil && r.cardID != "" && !r.byName {
			held[r.cardID] = append(held[r.cardID], i)
			holders = append(holders, i)
		}
	}
	for i, r := range results {
		if r.err == nil && r.cardID != "" && r.byName {
			twin := false
			for _, j := range held[r.cardID] {
				if same(results[j].product, r.product) || (face != nil && face(r.product, r.cardID)) {
					twin = true
					break
				}
			}
			if twin {
				results[i].cardID, results[i].cardIDFoil, results[i].err = "", "", errTwin
				continue
			}
			held[r.cardID] = append(held[r.cardID], i)
			holders = append(holders, i)
		}
	}
	// A miss beside a same-named product that holds a printing is the
	// same twin: the legacy unnumbered product of a card the expansion
	// sells numbered too, or the version of a promo the name alone cannot
	// tell from the versions already priced.
	for i, r := range results {
		if r.err == nil || errors.Is(r.err, errForeign) {
			continue
		}
		for _, j := range holders {
			if same(results[j].product, r.product) || (face != nil && face(r.product, results[j].cardID)) {
				results[i].err = errTwin
				break
			}
		}
	}
}

// pokemonSameProduct reports whether two products are named alike: the same
// card name once the shelf's own decorations are off it - the "(Theme
// Deck)" parenthetical, the "Basic" an energy is sometimes written with -
// and the same number, or none, once the letter a shelf hangs off it and
// the zeros it pads with are off.
func pokemonSameProduct(a, b *MKMProduct) bool {
	if pokemonPlainName(a.Name) != pokemonPlainName(b.Name) {
		return false
	}
	return a.Number == "" || b.Number == "" || pokemonPlainNumber(a.Number) == pokemonPlainNumber(b.Number)
}

var pokemonParenthetical = regexp.MustCompile(`\s*\([^)]*\)`)

func pokemonPlainName(name string) string {
	name = pokemonParenthetical.ReplaceAllString(pokemonName(name), "")
	name = strings.TrimPrefix(name, "Basic ")
	return mtgmatcher.Normalize(name)
}

func pokemonPlainNumber(number string) string {
	if m := pokemonPromoNumber.FindStringSubmatch(strings.TrimSpace(number)); m != nil {
		number = m[2]
	}
	number = strings.TrimRight(strings.TrimSpace(number), "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if trimmed := strings.TrimLeft(number, "0"); trimmed != "" {
		number = trimmed
	}
	return number
}

// pokemonPromoCodes are the programmes a promo's number is written with
// when the card is sold on another shelf ("Flareon V SWSH 149" in a prize
// pack series): the number names the promo set outright.
var pokemonPromoCodes = map[string]string{
	"SWSH": "SWSH Black Star Promos",
	"SVP":  "SV Black Star Promos",
	"SM":   "SM Black Star Promos",
	"XY":   "XY Black Star Promos",
	"MEP":  "MEP Black Star Promos",
	"BW":   "BW Black Star Promos",
	"DP":   "DP Black Star Promos",
}

var pokemonPromoNumber = regexp.MustCompile(`^([A-Z]+) (\S+)$`)

// pokemonLettered matches a collector number with a letter hung off it,
// which is how the alternate-art promos are numbered after the card they
// reprint ("92a").
var pokemonLettered = regexp.MustCompile(`^\d+[a-z]$`)

// matchPokemon names a Pokemon product's printing from what the catalog
// says of it, held to the sets its expansion may hold.
// matchPokemon names a Pokemon product's printing from what the catalog
// says of it, held to the sets its expansion may hold. An expansion naming
// no set of ours is a catalog we do not carry, and a miss in one is said so
// rather than reported product by product.
func (mkm *Index) matchPokemon(product *MKMProduct) (string, error) {
	if pokemonForeignDenied(product.ExpansionName, product.Number) {
		return "", errForeign
	}
	name := pokemonName(product.Name)
	type candidate struct {
		edition, number string
		prefixed        bool
	}
	var candidates []candidate
	editions, prefix := pokemonEditions(product.ExpansionName)
	number := product.Number
	if prefix != "" && number != "" {
		number = prefix + number
	}
	for _, edition := range editions {
		candidates = append(candidates, candidate{edition, number, prefix != ""})
	}
	if pokemonLettered.MatchString(number) {
		candidates = append(candidates, candidate{"Alternate Art Promos", number, false})
	}
	if m := pokemonPromoNumber.FindStringSubmatch(product.Number); m != nil {
		if shelf, found := pokemonPromoCodes[m[1]]; found {
			promo := pokemonExpansions[shelf]
			candidates = append(candidates, candidate{promo.sets[0], promo.prefix + m[2], promo.prefix != ""})
		}
	}
	carried := false
	for _, c := range candidates {
		set, err := mtgmatcher.GetSetByName(c.edition)
		if err != nil {
			continue
		}
		carried = true
		id, err := mtgmatcher.Match(&mtgmatcher.InputCard{Name: name, Edition: c.edition, Variation: c.number})
		if err != nil {
			continue
		}
		co, err := mtgmatcher.GetUUID(id)
		if err != nil || !strings.EqualFold(co.SetCode, set.Code) {
			continue
		}
		if c.prefixed && !strings.EqualFold(co.Number, c.number) {
			continue
		}
		return id, nil
	}
	if !carried || pokemonForeign(product.ExpansionName) {
		return "", errForeign
	}
	return "", errNoPrinting
}
