package fleshandblood

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// fabIDFixture holds rows from the published datastore: a printing
// TCGplayer numbers apart from its Legend Story Studios id (Star Fall/
// Spider's Bite share ARA002 between them), and a lettered pair sharing one
// fabId (Spectral Shield MST158-A/-B).
const fabIDFixture = `{
	"game": "fleshandblood",
	"sets": {
		"ARA": {"name": "Blitz Deck: Rosetta - Aurora", "releaseDate": "2024-09-20"},
		"BDOA": {"name": "Blitz Deck: Outsiders - Arakni", "releaseDate": "2023-03-24"},
		"MST": {"name": "Part the Mistveil", "releaseDate": "2024-05-17"}
	},
	"cards": [
		{"artist": "Ramza Ardyputra", "externalLinks": {"fabId": "AUA002", "tcgPlayerId": 577175}, "fabId": "AUA002", "finish": "Normal", "id": "ara002_577175", "image": "https://tcgplayer-cdn.tcgplayer.com/product/577175_400w.jpg", "name": "Star Fall", "number": "ARA002", "rarity": "Common", "setCode": "ARA"},
		{"artist": "Marcus Reyno", "externalLinks": {"fabId": "ARA002", "tcgPlayerId": 489117}, "fabId": "ARA002", "finish": "Normal", "id": "ara002_489117", "image": "https://tcgplayer-cdn.tcgplayer.com/product/489117_400w.jpg", "name": "Spider's Bite", "number": "ARA002", "rarity": "Common", "setCode": "BDOA"},
		{"artist": "Asur Misoa", "externalLinks": {"fabId": "MST158", "tcgPlayerId": 552843}, "fabId": "MST158", "finish": "Normal", "id": "mst158-a_552843", "image": "https://tcgplayer-cdn.tcgplayer.com/product/552843_400w.jpg", "name": "Spectral Shield", "number": "MST158-A", "rarity": "Common", "setCode": "MST", "variant": "158-A"},
		{"artist": "Asur Misoa", "externalLinks": {"fabId": "MST158", "tcgPlayerId": 552844}, "fabId": "MST158", "finish": "Normal", "id": "mst158-b_552844", "image": "https://tcgplayer-cdn.tcgplayer.com/product/552844_400w.jpg", "name": "Spectral Shield", "number": "MST158-B", "rarity": "Token", "setCode": "MST", "variant": "158-B"}
	]
}`

// TestFabIDAlternateNumber pins a Legend Story Studios id read as an
// alternate collector number: reachable where the catalog numbered the
// printing apart from it, still an alias beside the catalog's own number,
// and never letting a label stem two printings happen to share (both
// halves of MST158's lettered pair carry fabId "MST158") fold them
// together.
func TestFabIDAlternateNumber(t *testing.T) {
	b, err := Load(strings.NewReader(fabIDFixture))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		desc, name, variation string
		want                  string
	}{
		{"a storefront sku following the fabId reaches TCGplayer's own number", "Star Fall", "AUA002", "ara002_577175"},
		{"the fabId is still an alias beside the catalog's own number", "Star Fall", "ARA002", "ara002_577175"},
		{"a different card at the same catalog number is unaffected", "Spider's Bite", "ARA002", "ara002_489117"},
		// SCG's own sku spells the lettered pair "MST158a"/"MST158b" with
		// no dash; both halves share fabId "MST158".
		{"a shared fabId stem does not pull in the lettered pair's other half", "Spectral Shield", "MST158a", "mst158-a_552843"},
		{"and its sibling keeps its own letter too", "Spectral Shield", "MST158b", "mst158-b_552844"},
	} {
		in := mtgmatcher.InputCard{Name: tt.name, Variation: tt.variation}
		got, err := b.Match(&in)
		if err != nil {
			t.Errorf("%s: Match(%q, %q) = %v", tt.desc, tt.name, tt.variation, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: Match(%q, %q) = %q, want %q", tt.desc, tt.name, tt.variation, got, tt.want)
		}
	}
}
