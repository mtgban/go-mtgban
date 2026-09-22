package gamenerdz

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
)

// respellingDatastore is the published Pokemon datastore cut down to the
// printings this storefront misspells, every row copied verbatim from it. The
// names are what the vendor's own product body calls these cards, and the
// TCGplayer id each row carries is the one the body carries beside the
// misspelt display name.
const respellingDatastore = `{
 "game": "pokemon",
 "sets": {
  "DRI": {"abbreviation": "DRI", "baseSetSize": 182, "name": "SV10: Destined Rivals", "releaseDate": "2025-05-30"},
  "PAR": {"abbreviation": "PAR", "baseSetSize": 182, "name": "SV04: Paradox Rift", "releaseDate": "2023-11-03"},
  "SVI": {"abbreviation": "SVI", "baseSetSize": 198, "name": "SV01: Scarlet & Violet Base Set", "releaseDate": "2023-03-31"}
 },
 "cards": [
  {"externalLinks": {"tcgPlayerId": 632917}, "finish": "Normal", "id": "109-182_632917", "name": "Arven's Toedscool", "number": "109", "rarity": "Common", "setCode": "DRI", "total": "182", "type": "Fighting"},
  {"externalLinks": {"tcgPlayerId": 632917}, "finish": "Reverse Holofoil", "id": "109-182_632917_reverseholofoil", "name": "Arven's Toedscool", "number": "109", "rarity": "Common", "setCode": "DRI", "total": "182", "type": "Fighting"},
  {"externalLinks": {"tcgPlayerId": 632944}, "finish": "Normal", "id": "137-182_632944", "name": "Marnie's Morpeko", "number": "137", "rarity": "Common", "setCode": "DRI", "total": "182", "type": "Darkness"},
  {"externalLinks": {"tcgPlayerId": 632925}, "finish": "Normal", "id": "117-182_632925", "name": "Team Rocket's Nidoran M", "number": "117", "rarity": "Common", "setCode": "DRI", "total": "182", "type": "Darkness"},
  {"externalLinks": {"tcgPlayerId": 523644}, "finish": "Normal", "id": "035-182_523644", "name": "Feebas", "number": "035", "rarity": "Common", "setCode": "PAR", "total": "182", "type": "Water"},
  {"externalLinks": {"tcgPlayerId": 488053}, "finish": "Holofoil", "id": "158-198_488053_holofoil", "name": "Oinkologne ex", "number": "158", "originalName": "Oinkologne ex - 158/198", "rarity": "Double Rare", "setCode": "SVI", "total": "198", "type": "Colorless"},
  {"externalLinks": {"tcgPlayerId": 488074}, "finish": "Normal", "id": "169-198_488074", "name": "Defiance Band", "number": "169", "rarity": "Uncommon", "setCode": "SVI", "total": "198", "type": "Tool"},
  {"externalLinks": {"tcgPlayerId": 488075}, "finish": "Normal", "id": "170-198_488075", "name": "Electric Generator", "number": "170", "rarity": "Uncommon", "setCode": "SVI", "total": "198", "type": "Item"}
 ]
}`

// TestPreprocessPokemonRespelling pins that a name this storefront spells
// wrong reaches the printing its own body names, and that the spelling the
// catalog uses is left alone.
func TestPreprocessPokemonRespelling(t *testing.T) {
	b, err := mtgmatcher.Open("pokemon", strings.NewReader(respellingDatastore))
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
		card, err := preprocess(b, product, mtgban.GamePokemon)
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
