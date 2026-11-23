package pokemon

import (
	"flag"
	"testing"

	"github.com/mtgban/go-mtgban/internal/gametest"
	"github.com/mtgban/go-mtgban/matcher"
)

var updatePokemon = flag.Bool("update-pokemon", false,
	"rewrite testdata/pokemon_test_data.json from what Match returns")

// pokemonSeeds pin the contract edges the sampled corpus cannot reach: the
// metal and jumbo printings a listing is held to, the shelves storefronts
// spell the promo runs and pooled exclusives with, the subsets filed inside
// a set, the stamps, lettered numbers and number markers that tell siblings
// apart, and the refusals where the wording does not say which copy it means.
var pokemonSeeds = []gametest.Case{
	{
		Desc: "a metal card wording reaches the metal printing, not the set's own",
		In:   matcher.InputCard{Name: "Arceus V", Edition: "Brilliant Stars", Variation: "122/172 Metal Card"},
	},
	{
		Desc: "a metal card with no number reaches the one metal printing",
		In:   matcher.InputCard{Name: "Pikachu", Edition: "Celebrations", Variation: "Metal Card"},
	},
	{
		Desc: "negative: a metal card wording where no metal printing is sold",
		In:   matcher.InputCard{Name: "Charizard ex", Edition: "Scarlet & Violet 151", Variation: "199 Metal Card"},
	},
	{
		Desc: "an oversized wording reaches the jumbo, not the ordinary card",
		In:   matcher.InputCard{Name: "Koraidon ex", Edition: "Scarlet & Violet", Variation: "125 Jumbo Oversized | 125"},
	},
	{
		Desc: "an oversized wording reaches the jumbo past a pooled shelf",
		In:   matcher.InputCard{Name: "Salamence ex", Edition: "Theme Deck & Blisters Exclusives", Variation: "114 Jumbo Oversized | 114/159"},
	},
	{
		Desc: "a LEGEND pair's two numbers name the one jumbo",
		In:   matcher.InputCard{Name: "Darkrai & Cresselia Legend - 99/102 & 100/102", Edition: "Jumbo Cards", Variation: "Single Oversized Promo"},
	},
	{
		Desc: "negative: an oversized wording with no number to tell two jumbos apart",
		In:   matcher.InputCard{Name: "Pikachu", Edition: "XY Promos", Variation: "XY-P Jumbo Oversized | XY-P"},
	},
	{
		Desc: "a black star promo shelf the catalog files under its era's title",
		In:   matcher.InputCard{Name: "Cherrim", Edition: "SWSH Black Star Promos", Variation: "088 Prerelease Promo | SWSH088"},
	},
	{
		Desc: "the wizards black star promos reach the WoTC promo set",
		In:   matcher.InputCard{Name: "Mewtwo", Edition: "Wizards Black Star Promos", Variation: "014 14"},
	},
	{
		Desc: "negative: a per-set promos shelf naming no set does not widen",
		In:   matcher.InputCard{Name: "Moltres ex", Edition: "Plasma Storm Promos", Variation: "014/135 Holo Promo | 14/135"},
	},
	{
		Desc: "the bare prize pack series pools into its one set",
		In:   matcher.InputCard{Name: "Grotle", Edition: "Play! Pokémon Prize Pack Series", Variation: "007 BRS 007"},
	},
	{
		Desc: "a pooled exclusives shelf lands a kit promo in deck exclusives",
		In:   matcher.InputCard{Name: "Chi-Yu", Edition: "Theme Deck & Blisters Exclusives", Variation: "029h Holo Build & Battle Kit | 029/182"},
	},
	{
		Desc: "negative: a card the pooled shelf does not carry",
		In:   matcher.InputCard{Name: "Chansey", Edition: "Theme Deck & Blisters Exclusives", Variation: "003 Cosmos Holo | 003/102"},
	},
	{
		Desc: "a Trainer Gallery card is reached from its parent set's name",
		In:   matcher.InputCard{Name: "Flareon - TG01/TG30", Edition: "SWSH Brilliant Stars"},
	},
	{
		Desc: "a Trainer Gallery shelf with a marker hung off the number",
		In:   matcher.InputCard{Name: "Abomasnow", Edition: "SWSH10: Astral Radiance Trainer Gallery", Variation: "TG001C 1st Edition"},
	},
	{
		Desc: "a Radiant Collection card is reached from its parent set's name",
		In:   matcher.InputCard{Name: "Cinccino - RC19/RC25", Edition: "BW Legendary Treasures"},
	},
	{
		Desc: "a set code shaped like a subset's is not one",
		In:   matcher.InputCard{Name: "Gyarados - 7/95", Edition: "Call of Legends", Variation: "7"},
	},
	{
		Desc: "a stamp names the stamped copy before its mark",
		In:   matcher.InputCard{Name: "Archaludon - 107/142", Edition: "Miscellaneous Cards & Products", Variation: "107/142 Cosmos Holo Stellar Crown Stamp"},
	},
	{
		Desc: "a stamp in the name reaches the stamped promo",
		In:   matcher.InputCard{Name: "Psyduck (Detective Pikachu Stamp) - SM199", Edition: "Promo"},
	},
	{
		Desc: "negative: a stamp wording refuses the unstamped printing",
		In:   matcher.InputCard{Name: "Rapidash", Edition: "Wizards Black Star Promos", Variation: "051pc Pokemon Center Stamped | 51"},
	},
	{
		Desc: "a lettered number reaches the alternate art filed apart",
		In:   matcher.InputCard{Name: "Garbodor", Edition: "SM Guardians Rising", Variation: "51a/145 Alt Art"},
	},
	{
		Desc: "the same number unlettered keeps the set's own card",
		In:   matcher.InputCard{Name: "Garbodor", Edition: "SM Guardians Rising", Variation: "51/145"},
	},
	{
		Desc: "a lettered number's holo copy",
		In:   matcher.InputCard{Name: "Professor Sycamore", Edition: "XY BREAKpoint", Variation: "107a/122 Alt Art HOLO", Foil: true},
	},
	{
		Desc: "a lettered number's non-holo twin, which the catalog prices apart",
		In:   matcher.InputCard{Name: "Professor Sycamore", Edition: "XY BREAKpoint", Variation: "107a/122 Alt Art Non-Holo"},
	},
	{
		Desc: "a number marker names the pattern printing",
		In:   matcher.InputCard{Name: "Umbreon", Edition: "SV: Prismatic Evolutions", Variation: "059M 1st Edition Master Ball Pattern"},
	},
	{
		Desc: "the same number with no marker keeps the plain printing",
		In:   matcher.InputCard{Name: "Umbreon", Edition: "SV: Prismatic Evolutions", Variation: "059 1st Edition"},
	},
	{
		Desc: "a wording writing no number means the unnumbered twin",
		In:   matcher.InputCard{Name: "Metal Energy - 2008", Edition: "World Championship Decks", Variation: "Tristan Robinson"},
	},
	{
		Desc: "a placement outranks the year the plain copy is marked with",
		In:   matcher.InputCard{Name: "Champions Festival", Edition: "XY Promos", Variation: "XY91 2015 Quarter Finalist"},
	},
	{
		Desc: "negative: a placement no copy at the number wears",
		In:   matcher.InputCard{Name: "Professor Turo's Scenario - 171/182", Edition: "SV04: Paradox Rift", Variation: "171/182 Staff"},
	},
	{
		Desc: "an energy's copyright year reaches the set the catalog filed it in",
		In:   matcher.InputCard{Name: "Fire Energy", Edition: "Champion's Path", Variation: "©2020 Reverse Holo | ©2020", Foil: true},
	},
	{
		Desc: "a holo wording picks the product sold only as holofoil",
		In:   matcher.InputCard{Name: "Blaziken", Edition: "EX Battle Stadium", Variation: "3 Holo"},
	},
	{
		Desc: "a non-holo wording picks the product sold as nonfoil",
		In:   matcher.InputCard{Name: "Blaziken", Edition: "EX Battle Stadium", Variation: "3 Non-Holo"},
	},
	{
		Desc: "the catalog's misspelling reaches the corrected name",
		In:   matcher.InputCard{Name: "Dark Exeggcutor", Edition: "Neo Destiny", Variation: "033 033/105"},
	},
	{
		Desc: "the shelf's card count over a card that keeps its original total",
		In:   matcher.InputCard{Name: "Blastoise", Edition: "Celebrations: Classic Collection", Variation: "2/25"},
	},
	{
		Desc: "negative: another set's total vetoes the number",
		In:   matcher.InputCard{Name: "Cascoon", Edition: "Platinum", Variation: "44/130"},
	},
	{
		Desc: "negative: a name the catalog does not carry",
		In:   matcher.InputCard{Name: "Not A Pokemon Card At All", Edition: "Base Set", Variation: "4/102"},
	},
}

func TestPokemonMatch(t *testing.T) {
	b := gametest.Backend(t, matcher.GamePokemon)
	gametest.Replay(t, b, pokemonSeeds, *updatePokemon)
}
