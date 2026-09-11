package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestResolvePromoShelfPrinting pins every product the Promo shelf cannot
// place on its own. SCG shelves them all under "Promo" and numbers them for the set the
// league ran beside, so nothing but the sku says which yearly token set they
// belong to.
func TestResolvePromoShelfPrinting(t *testing.T) {
	withMagic(t)

	for _, tt := range []struct {
		desc, sku, name string
		foil            bool
		scryfallID      string
		wantSet, wantNo string
	}{
		{"2012's league ran beside Magic 2013", "SGL-MTG-PRM-LEAG_M13_L01-ENN", "{Goblin Token}", false, "", "L12", "1"},
		{"and beside Return to Ravnica", "SGL-MTG-PRM-LEAG_RTR_L01-ENN", "{Knight Token}", false, "", "L12", "2"},

		// 2013 handed out a Soldier twice, Gatecrash's Boros one and
		// Theros's mono-white one. Only the sku separates them.
		{"Gatecrash's Soldier is L13's first", "SGL-MTG-PRM-LEAG_GTC_L01-ENN", "{Soldier Token}", false, "", "L13", "1"},
		{"Theros's is its fourth", "SGL-MTG-PRM-LEAG_THS_L01-ENN", "{Soldier Token}", false, "", "L13", "4"},
		{"Dragon's Maze", "SGL-MTG-PRM-LEAG_DGM_L01-ENN", "{Bird Token}", false, "", "L13", "2"},
		{"Magic 2014", "SGL-MTG-PRM-LEAG_M14_L01-ENN", "{Sliver Token}", false, "", "L13", "3"},

		{"Born of the Gods", "SGL-MTG-PRM-LEAG_BNG_T01-ENN", "{Soldier Token}", false, "", "L14", "1"},
		{"Journey into Nyx", "SGL-MTG-PRM-LEAG_JOU_L01-ENN", "{Minotaur Token}", false, "", "L14", "2"},
		{"Magic 2015", "SGL-MTG-PRM-LEAG_M15_L01-ENN", "{Squid Token}", false, "", "L14", "3"},
		{"Khans of Tarkir", "SGL-MTG-PRM-LEAG_KTK_L01-ENN", "{Warrior Token}", false, "", "L14", "4"},

		// The one product SCG gave an identifier to was given the set
		// token's, which is what used to price a $34.99 promo as a $0.49
		// common. The sku has to win over it.
		{"Fate Reforged, whose scryfall id names the set token", "SGL-MTG-PRM-LEAG_FRF_L01-ENN", "{Monk Token}", false, "3142cb28-23cc-405f-9db5-7c4d168aab19", "L15", "1"},

		// Both later leagues handed out a foil double-faced token.
		{"Kaladesh", "SGL-MTG-PRM-LEAG_KLD_T05T09-ENF", "{Servo Token} (#5) // {Thopter Token} (#9)", true, "", "L16", "5"},
		{"Aether Revolt", "SGL-MTG-PRM-LEAG_AER_T01T00-ENF", "{Gremlin Token} (#1) // {Energy Reserve}", true, "", "L17", "1"},

		// Cowboy Bebop's Standard Showdown promo, whose scryfall id names
		// the Friday Night Magic 2015 printing instead.
		{"Cowboy Bebop's Disdainful Stroke", "SGL-MTG-PRM-SSD_2024_002b-ENF", "Disdainful Stroke", true, "3711f61d-6381-4c92-a3f5-6deed29aae47", "PCBB", "2"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			finish, group := "Non-foil", "Non-foil"
			if tt.foil {
				finish, group = "Foil", "Foil"
			}
			p := CatalogProduct{
				SKU: tt.sku, Name: tt.name, Game: "Magic: The Gathering",
				Set: "Promo", Rarity: "Promo", ProductType: ProductTypeSingles,
				Finish: finish, FinishGroup: group, Language: "English",
				ScryfallID: tt.scryfallID,
			}
			id, err := resolveProduct(GameMagic, p)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.sku, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNo {
				t.Errorf("%s resolved to %s #%s (%s), want %s #%s",
					tt.sku, co.SetCode, co.Number, co.Name, tt.wantSet, tt.wantNo)
			}
			if co.Foil != tt.foil {
				t.Errorf("%s resolved to foil=%v, want %v", tt.sku, co.Foil, tt.foil)
			}
		})
	}
}

// TestLeagueTokensTableIsExhaustive guards the table against the datastore
// growing a League Tokens printing the catalog would then have nowhere to put.
// These leagues ended in 2017, so a new row means something has changed
// upstream rather than that SCG has started selling one.
func TestLeagueTokensTableIsExhaustive(t *testing.T) {
	withMagic(t)

	mapped := map[string]bool{}
	for _, printing := range promoShelfPrintings {
		out := mtgmatcher.MatchWithNumber("", printing.set, printing.number)
		if len(out) != 1 {
			t.Errorf("%s #%s names %d printings, want exactly 1", printing.set, printing.number, len(out))
			continue
		}
		mapped[out[0].UUID] = true
	}

	for _, code := range []string{"L12", "L13", "L14", "L15", "L16", "L17"} {
		set, err := mtgmatcher.GetSet(code)
		if err != nil {
			t.Errorf("GetSet(%s) = %v", code, err)
			continue
		}
		for _, token := range set.Tokens {
			// The far side of a double-faced token is filed upstream but
			// never loaded: one physical card, one reachable row.
			if _, err := mtgmatcher.GetUUID(token.UUID); err != nil {
				continue
			}
			if !mapped[token.UUID] {
				t.Errorf("%s #%s %q is in no promoShelfPrintings entry", code, token.Number, token.Name)
			}
		}
	}
}
