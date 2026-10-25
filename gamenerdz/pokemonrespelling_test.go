package gamenerdz

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/pokemon"
)

// respellingDatastore is the published Pokemon datastore cut down to the
// printings this storefront misspells or numbers in its own way, every row
// copied verbatim from it. The names are what the vendor's own product body
// calls these cards, and the TCGplayer id each row carries is the one the
// body carries beside the misspelt display name.
const respellingDatastore = `{"data": {
 "game": "pokemon",
 "sets": {
  "DRI": {"abbreviation": "DRI", "baseSetSize": 182, "name": "SV10: Destined Rivals", "releaseDate": "2025-05-30"},
  "EVO": {"abbreviation": "EVO", "baseSetSize": 108, "name": "XY - Evolutions", "releaseDate": "2016-11-02"},
  "LTR-1465": {"abbreviation": "LTR", "name": "Legendary Treasures: Radiant Collection", "releaseDate": "2013-11-06"},
  "MCAP": {"abbreviation": "MCAP", "name": "Miscellaneous Cards & Products", "releaseDate": "1999-01-09"},
  "PAR": {"abbreviation": "PAR", "baseSetSize": 182, "name": "SV04: Paradox Rift", "releaseDate": "2023-11-03"},
  "PRE": {"abbreviation": "PRE", "baseSetSize": 131, "name": "SV: Prismatic Evolutions", "releaseDate": "2025-01-17"},
  "PWCP": {"abbreviation": "PWCP", "name": "Pikachu World Collection Promos", "releaseDate": "2000-09-12", "type": "promo"},
  "SVP": {"abbreviation": "SVP", "name": "SV: Scarlet & Violet Promo Cards", "releaseDate": "2023-03-31", "type": "promo"},
  "SVI": {"abbreviation": "SVI", "baseSetSize": 198, "name": "SV01: Scarlet & Violet Base Set", "releaseDate": "2023-03-31"},
  "SWSD": {"abbreviation": "SWSD", "name": "SWSH: Sword & Shield Promo Cards", "releaseDate": "2019-11-15", "type": "promo"},
  "SWSH02": {"abbreviation": "SWSH02", "baseSetSize": 192, "name": "SWSH02: Rebel Clash", "releaseDate": "2020-05-01"}
 },
 "cards": [
  {"externalLinks": {"tcgPlayerId": 632917}, "finish": "Normal", "id": "109-182_632917", "name": "Arven's Toedscool", "number": "109", "rarity": "Common", "setCode": "DRI", "total": "182", "type": "Fighting"},
  {"externalLinks": {"tcgPlayerId": 632917}, "finish": "Reverse Holofoil", "id": "109-182_632917_reverseholofoil", "name": "Arven's Toedscool", "number": "109", "rarity": "Common", "setCode": "DRI", "total": "182", "type": "Fighting"},
  {"externalLinks": {"tcgPlayerId": 632944}, "finish": "Normal", "id": "137-182_632944", "name": "Marnie's Morpeko", "number": "137", "rarity": "Common", "setCode": "DRI", "total": "182", "type": "Darkness"},
  {"externalLinks": {"tcgPlayerId": 632925}, "finish": "Normal", "id": "117-182_632925", "name": "Team Rocket's Nidoran M", "number": "117", "rarity": "Common", "setCode": "DRI", "total": "182", "type": "Darkness"},
  {"externalLinks": {"tcgPlayerId": 523644}, "finish": "Normal", "id": "035-182_523644", "name": "Feebas", "number": "035", "rarity": "Common", "setCode": "PAR", "total": "182", "type": "Water"},
  {"externalLinks": {"tcgPlayerId": 488053}, "finish": "Holofoil", "id": "158-198_488053_holofoil", "name": "Oinkologne ex", "number": "158", "originalName": "Oinkologne ex - 158/198", "rarity": "Double Rare", "setCode": "SVI", "total": "198", "type": "Colorless"},
  {"externalLinks": {"tcgPlayerId": 488074}, "finish": "Normal", "id": "169-198_488074", "name": "Defiance Band", "number": "169", "rarity": "Uncommon", "setCode": "SVI", "total": "198", "type": "Tool"},
  {"externalLinks": {"tcgPlayerId": 488075}, "finish": "Normal", "id": "170-198_488075", "name": "Electric Generator", "number": "170", "rarity": "Uncommon", "setCode": "SVI", "total": "198", "type": "Item"},
  {"externalLinks": {"tcgPlayerId": 213283}, "finish": "Normal", "id": "172-192_213283", "name": "Horror P Energy", "number": "172", "rarity": "Uncommon", "setCode": "SWSH02", "total": "192", "type": "Special Energy"},
  {"externalLinks": {"tcgPlayerId": 215159}, "finish": "Holofoil", "id": "pw-7_215159_holofoil", "language": "French", "name": "Pikachu", "number": "PW7", "rarity": "Promo", "setCode": "PWCP", "type": "Lightning", "types": ["Lightning"], "variant": "French"},
  {"externalLinks": {"tcgPlayerId": 85639}, "finish": "Holofoil", "id": "rc10-rc25_85639_holofoil", "name": "Gardevoir", "number": "RC10", "rarity": "Uncommon", "setCode": "LTR-1465", "total": "RC25"},
  {"externalLinks": {"tcgPlayerId": 124122, "tcgdexId": "xy12-109"}, "finish": "Normal", "id": "109-108_124122", "name": "Exeggutor", "number": "109", "rarity": "Secret Rare", "setCode": "EVO", "total": "108", "type": "Grass", "types": ["Grass"]},
  {"externalLinks": {"tcgPlayerId": 167223}, "finish": "Normal", "id": "016-107_167223", "name": "Deoxys", "number": "016", "promoTypes": ["movie"], "rarity": "Promo", "setCode": "MCAP", "total": "107", "type": "Psychic", "types": ["Psychic"], "variant": "Movie Normal Forme", "watermark": "normal forme"},
  {"externalLinks": {"tcgPlayerId": 247300, "tcgdexId": "swshp-SWSH167"}, "finish": "Normal", "id": "swsh167_247300", "name": "Professor Burnet", "number": "SWSH167", "promoTypes": ["25thanniversary"], "rarity": "Promo", "setCode": "SWSD", "type": "Supporter", "variant": "25th Anniversary"},
  {"externalLinks": {"tcgPlayerId": 610478, "tcgdexId": "sv08.5-123"}, "finish": "Normal", "id": "123-131_610478", "name": "Professor's Research", "number": "123", "rarity": "Common", "setCode": "PRE", "total": "131", "type": "Trainer - Supporter", "variant": "Professor Elm", "watermark": "professor elm"},
  {"externalLinks": {"tcgPlayerId": 610631, "tcgdexId": "sv08.5-123"}, "finish": "Holofoil", "id": "123-131_610631_holofoil", "name": "Professor's Research", "number": "123", "promoTypes": ["pokeballpattern"], "rarity": "Common", "setCode": "PRE", "total": "131", "type": "Trainer - Supporter", "variant": "Professor Elm Poke Ball Pattern", "watermark": "professor elm"},
  {"externalLinks": {"tcgPlayerId": 477182, "tcgdexId": "svp-002"}, "finish": "Holofoil", "id": "002_477182_holofoil", "name": "Fuecoco", "number": "002", "rarity": "Promo", "setCode": "SVP", "type": "Fire", "types": ["Fire"]},
  {"externalLinks": {"tcgPlayerId": 487755, "tcgdexId": "svp-007"}, "finish": "Holofoil", "id": "007_487755_holofoil", "name": "Hawlucha", "number": "007", "promoTypes": ["prerelease"], "rarity": "Promo", "setCode": "SVP", "type": "Fighting", "types": ["Fighting"], "variant": "Prerelease"}
 ]
}}`

// TestPreprocessPokemonRespelling pins that a name this storefront spells
// wrong reaches the printing its own body names, and that the spelling the
// catalog uses is left alone.
func TestPreprocessPokemonRespelling(t *testing.T) {
	b, err := matcher.Open("pokemon", strings.NewReader(respellingDatastore))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		displayName string
		setName     string
		finish      string
		uuid        string
	}{
		{"Arver's Toedscool 109 - SV10 Destined Rivals", "SV10: Destined Rivals", "Normal", "109-182_632917"},
		{"Arver's Toedscool 109 - SV10 Destined Rivals Reverse Holofoil", "SV10: Destined Rivals", "Reverse Holofoil", "109-182_632917_reverseholofoil"},
		{"Marnie's Marpeko 137 - SV10 Destined Rivals", "SV10: Destined Rivals", "Normal", "137-182_632944"},
		{"Team Rocket's Nidorand 117 - SV10 Destined Rivals", "SV10: Destined Rivals", "Normal", "117-182_632925"},
		{"Feebass 35 - SV04 Paradox Rift", "SV04: Paradox Rift", "Normal", "035-182_523644"},
		{"Oinkalogne ex - 158/198 158 - SV01 Scarlet  Violet Base Set Holofoil", "SV01: Scarlet & Violet Base Set", "Holofoil", "158-198_488053_holofoil"},
		{"Defiant Band - 169/198 169 - SV01 Scarlet  Violet Base Set", "SV01: Scarlet & Violet Base Set", "Normal", "169-198_488074"},
		{"Electro Generator - 170/198 170 - SV01 Scarlet  Violet Base Set", "SV01: Scarlet & Violet Base Set", "Normal", "170-198_488075"},
		// An energy's type spelled out where the catalog writes its letter.
		{"Horror Psychic Energy 172/192 - Rebel Clash", "SWSH02: Rebel Clash", "Normal", "172-192_213283"},
		// The label this storefront gives a promo, a size lettered like the
		// number, a bracket in front of the qualifier, and the names it
		// writes its own way.
		{"Professor Burnet - SWSH167 (25th Anniversary Stamp) SWSH167 - SWSH Sword  Shield Promo Cards", "SWSH: Sword & Shield Promo Cards", "Normal", "swsh167_247300"},
		{"Gardevoir RC10/RC25 - Legendary Treasures Holofoil", "Legendary Treasures: Radiant Collection", "Holofoil", "rc10-rc25_85639_holofoil"},
		{"Professor's Research [Professor Elm] (Poke Ball Pattern) 123 - SV Prismatic Evolutions Holofoil", "SV: Prismatic Evolutions", "Holofoil", "123-131_610631_holofoil"},
		{"Professor's Research [Professor Elm] 123 - SV Prismatic Evolutions", "SV: Prismatic Evolutions", "Normal", "123-131_610478"},
		{"Deoxys (Normal Forme) - 16/107 (Movie Promo) 16 - Miscellaneous Cards  Products", "Miscellaneous Cards & Products", "Normal", "016-107_167223"},
		{"ナッシー[Exeggutor] 109/108 - Evolutions", "XY - Evolutions", "Normal", "109-108_124122"},
		// The catalog's own spelling is not a key of the table and still
		// answers with the same printing.
		{"Arven's Toedscool 109 - SV10 Destined Rivals", "SV10: Destined Rivals", "Normal", "109-182_632917"},
	}
	for _, tt := range tests {
		product := GNProduct{
			DisplayName:    tt.displayName,
			SelectedFinish: tt.finish,
			ProductData:    GNProductData{SetName: tt.setName},
		}
		card, err := preprocess(b, product, matcher.GamePokemon)
		if err != nil {
			t.Errorf("%q: unexpected error %v", tt.displayName, err)
			continue
		}
		uuid, err := b.Match(card)
		if err != nil {
			t.Errorf("%q: unexpected error %v", tt.displayName, err)
			continue
		}
		if uuid != tt.uuid {
			t.Errorf("%q: got %q; want %q", tt.displayName, uuid, tt.uuid)
		}
	}
}

// TestPreprocessPokemonPromoNumbers pins the two promos this storefront
// numbers in its own way: the Pikachu World Collection ones by the language
// they are printed in, and the Scarlet & Violet ones by a code in the name
// or by the sku alone.
func TestPreprocessPokemonPromoNumbers(t *testing.T) {
	b, err := matcher.Open("pokemon", strings.NewReader(respellingDatastore))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		displayName string
		setName     string
		sku         string
		uuid        string
	}{
		{"Pikachu (French) PW - Pikachu World Collection Promos Holofoil", "Pikachu World Collection Promos", "PKM-PWCP-PW 7-H-3QSRE7LUY9", "pw-7_215159_holofoil"},
		{"Hawlucha (SVP-007) (Scarlet & Violet Base Set) Holofoil", "SV: Scarlet & Violet Promo Cards", "PKM-SS&VPC-007-H-XV2SJUDZGC", "007_487755_holofoil"},
		{"Fuecoco (Scarlet & Violet Base Set) Holofoil", "SV: Scarlet & Violet Promo Cards", "PKM-SS&VPC-002-H-VLDF24NPY4", "002_477182_holofoil"},
	}
	for _, tt := range tests {
		product := GNProduct{
			DisplayName:    tt.displayName,
			SelectedFinish: "Holofoil",
			ProductData:    GNProductData{SetName: tt.setName},
			RetailVariants: []GNRetailVariant{{SKU: tt.sku}},
		}
		card, err := preprocess(b, product, matcher.GamePokemon)
		if err != nil {
			t.Errorf("%q: unexpected error %v", tt.displayName, err)
			continue
		}
		uuid, err := b.Match(card)
		if err != nil {
			t.Errorf("%q: unexpected error %v", tt.displayName, err)
			continue
		}
		if uuid != tt.uuid {
			t.Errorf("%q: got %q; want %q", tt.displayName, uuid, tt.uuid)
		}
	}
}
