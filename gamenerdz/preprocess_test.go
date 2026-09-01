package gamenerdz

import (
	"testing"
)

func TestPreprocess(t *testing.T) {
	tests := []struct {
		game    string
		product GNProduct

		name      string
		edition   string
		variation string
		finish    string
		foil      bool
		err       bool
	}{
		{
			game: GameMagic,
			product: GNProduct{
				DisplayName:    "Teferi, Time Raveler (PPELD-221P) - War of the Spark",
				SelectedFinish: "nonfoil",
				ProductData:    GNProductData{SetName: "War of the Spark"},
			},
			name: "Teferi, Time Raveler", edition: "War of the Spark", variation: "221P",
		},
		{
			// The name stops at the first parenthesis, and the number tag is
			// the last one, past the bare display number the name also shows.
			game: GameMagic,
			product: GNProduct{
				DisplayName:    "Aang, Air Nomad (0265) (TLE-265) - Avatar: The Last Airbender: Eternal Decks Foil",
				SelectedFinish: "foil",
				ProductData:    GNProductData{Set: "TLE"},
			},
			name: "Aang, Air Nomad", edition: "TLE", variation: "265", foil: true,
		},
		{
			game: GameMagic,
			product: GNProduct{
				DisplayName:    "Aang's Shelter - Teferi's Protection (Borderless) (TLE-007) - Avatar",
				SelectedFinish: "nonfoil",
				ProductData:    GNProductData{Set: "TLE"},
			},
			name: "Aang's Shelter - Teferi's Protection", edition: "TLE", variation: "007",
		},
		{
			game: GameLorcana,
			product: GNProduct{
				DisplayName:    "4*Town - Hottest Band of the Year (17/204) - Attack of the Vine",
				SelectedFinish: "Normal",
				ProductData:    GNProductData{SetName: "Attack of the Vine"},
			},
			name: "4*Town - Hottest Band of the Year", edition: "Attack of the Vine", variation: "17",
		},
		{
			game: GameLorcana,
			product: GNProduct{
				DisplayName:    "99 Puppies (24/204) - Into the Inklands Cold Foil",
				SelectedFinish: "Cold Foil",
				ProductData:    GNProductData{SetName: "Into the Inklands"},
			},
			name: "99 Puppies", edition: "Into the Inklands", variation: "24",
			finish: "Cold Foil", foil: true,
		},
		{
			game: GamePokemon,
			product: GNProduct{
				DisplayName:    "Abra 65/130 - Base Set 2",
				SelectedFinish: "Normal",
				ProductData:    GNProductData{SetName: "Base Set 2"},
			},
			name: "Abra", edition: "Base Set 2", variation: "65/130",
		},
		{
			game: GamePokemon,
			product: GNProduct{
				DisplayName:    "Abra 63 - SV Scarlet and Violet 151 Reverse Holofoil",
				SelectedFinish: "Reverse Holofoil",
				ProductData:    GNProductData{SetName: "SV: Scarlet & Violet 151"},
			},
			name: "Abra", edition: "SV: Scarlet & Violet 151", variation: "63",
			finish: "Reverse Holofoil",
		},
		{
			// The finishing place is written in brackets where every other
			// qualifier gets parentheses. Left as it is the matcher reads no
			// place at all and answers with the printing awarded none, which
			// at this number is a $4 card standing against a $558 one.
			game: GameOnePiece,
			product: GNProduct{
				DisplayName: "Monkey.D.Luffy (Offline Regional 2024 Vol. 2) [Finalist] (P-041) One Piece Promotion Cards",
				ProductData: GNProductData{SetName: "One Piece Promotion Cards"},
			},
			name:      "Monkey.D.Luffy (Offline Regional 2024 Vol. 2) (Finalist)",
			edition:   "One Piece Promotion Cards",
			variation: "P-041",
		},
		{
			// Variant wording before the code stays in the name for the
			// matcher to read.
			game: GameOnePiece,
			product: GNProduct{
				DisplayName:    "Ama no Murakumo Sword (Jolly Roger Foil) (OP06-056) Premium Booster -The Best-",
				SelectedFinish: "Foil",
				ProductData:    GNProductData{SetName: "Premium Booster -The Best-"},
			},
			name: "Ama no Murakumo Sword (Jolly Roger Foil)", edition: "Premium Booster -The Best-",
			variation: "OP06-056", foil: true,
		},
		{
			// The Premium Booster listings write the code twice, around the
			// variant wording; the earlier copy is shed, the wording stays.
			game: GameOnePiece,
			product: GNProduct{
				DisplayName:    "Baby 5 (OP04-032) (Jolly Roger Foil) (OP04-032) Premium Booster -The Best- Foil",
				SelectedFinish: "Foil",
				ProductData:    GNProductData{SetName: "Premium Booster -The Best-"},
			},
			name: "Baby 5 (Jolly Roger Foil)", edition: "Premium Booster -The Best-",
			variation: "OP04-032", foil: true,
		},
		{
			game: GameFleshAndBlood,
			product: GNProduct{
				DisplayName:    "Aether Sink (ARC017) Arcane Rising 1st Edition Rainbow Foil",
				SelectedFinish: "1st Edition Rainbow Foil",
				ProductData:    GNProductData{SetName: "Arcane Rising"},
			},
			name: "Aether Sink", edition: "Arcane Rising", variation: "ARC017",
			finish: "1st Edition Rainbow Foil", foil: true,
		},
		{
			// The pitch color is part of the card's name, not a variant.
			game: GameFleshAndBlood,
			product: GNProduct{
				DisplayName:    "Aether Quickening (Yellow) (FAB113) Flesh and Blood: Promo Cards Rainbow Foil",
				SelectedFinish: "Rainbow Foil",
				ProductData:    GNProductData{SetName: "Flesh and Blood: Promo Cards"},
			},
			name: "Aether Quickening (Yellow)", edition: "Flesh and Blood: Promo Cards",
			variation: "FAB113", finish: "Rainbow Foil", foil: true,
		},
		{
			game:    GameLorcana,
			product: GNProduct{DisplayName: "Illumineer's Trove - Sapphire and Steel"},
			err:     true,
		},
	}
	for _, tt := range tests {
		card, err := preprocess(tt.product, tt.game)
		if tt.err {
			if err == nil {
				t.Errorf("%s/%s: expected an error", tt.game, tt.product.DisplayName)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s/%s: unexpected error %v", tt.game, tt.product.DisplayName, err)
			continue
		}
		if card.Name != tt.name || card.Edition != tt.edition ||
			card.Variation != tt.variation || card.Finish != tt.finish || card.Foil != tt.foil {
			t.Errorf("%s/%s: got %q/%q/%q/%q/%v; want %q/%q/%q/%q/%v",
				tt.game, tt.product.DisplayName,
				card.Name, card.Edition, card.Variation, card.Finish, card.Foil,
				tt.name, tt.edition, tt.variation, tt.finish, tt.foil)
		}
	}
}
