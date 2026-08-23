package cardmarket

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
