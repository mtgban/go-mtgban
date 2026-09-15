package starcitygames

import "testing"

// TestResolveTwoSidedTokenPairing pins a two-sided token listing resolving
// to the combined entity mtgmatcher/magic/tokenpairs.go derives for the same
// physical pairing, anchored by SCG's own scryfall_id rather than guessed
// from the name alone - the same mechanism cardkingdom uses (see
// magic.MatchTokenPairing), reached here through SCG's own brace-
// wrapped face names ("{X Token}") and ahead of both the promo-shelf table
// and the plain identifier lookup, either of which would otherwise resolve
// only the one face the id happens to anchor.
func TestResolveTwoSidedTokenPairing(t *testing.T) {
	b := withMagic(t)

	for _, tt := range []struct {
		desc       string
		sku        string
		name       string
		foil       bool
		scryfallID string
		wantName   string
		wantSet    string
	}{
		{
			desc:       "a nonfoil pairing",
			sku:        "SGL-MTG-AKH-T1T18-ENN",
			name:       "{Angel of Sanctions Token} // {Drake Token}",
			foil:       false,
			scryfallID: "a1786cb3-6f38-4f10-b5f1-8feb20e3baaf",
			wantName:   "Angel of Sanctions Token // Drake",
			wantSet:    "TAKH",
		},
		{
			// The derived pairing's own uuid carries no separate foil
			// identity a bare MatchID(tcgID) would fall back to, so this
			// also pins that the foil flag actually reaches the right
			// finish rather than silently pricing the nonfoil id. Soldier
			// pairs with exactly one partner (Rebel) in today's datastore -
			// picked deliberately over a face like Bear (which pairs with
			// four differently-numbered Food tokens on the same sheet, all
			// colliding on the same normalized name) so this test can never
			// pass by an arbitrary pick among several correct-looking
			// answers; TestTokenPairIndexCollision below pins that case.
			desc:       "a foil pairing",
			sku:        "SGL-MTG-FIC-T01T02-ENF",
			name:       "{Soldier Token} // {Rebel Token}",
			foil:       true,
			scryfallID: "c459f2ec-2aa3-44f6-999f-b1467dd4e27c",
			wantName:   "Soldier // Rebel",
			wantSet:    "TFIC",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			finish, group := "Non-foil", "Non-foil"
			if tt.foil {
				finish, group = "Foil", "Foil"
			}
			p := CatalogProduct{
				SKU: tt.sku, Name: tt.name, Game: "Magic: The Gathering",
				Set: "Promo", Rarity: "Token", ProductType: ProductTypeSingles,
				Finish: finish, FinishGroup: group, Language: "English",
				ScryfallID: tt.scryfallID,
			}
			id, err := resolveProductID(b, GameMagic, p)
			if err != nil {
				t.Fatalf("resolveProductID(%s) = %v", tt.sku, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.Identifiers["derivedTokenPair"] != "true" {
				t.Errorf("%s resolved to %s (%s), want a derived token pairing", tt.sku, id, co.Card.Name)
			}
			if co.Card.Name != tt.wantName || co.SetCode != tt.wantSet {
				t.Errorf("%s resolved to %s [%s], want %s [%s]", tt.sku, co.Card.Name, co.SetCode, tt.wantName, tt.wantSet)
			}
			if co.Foil != tt.foil {
				t.Errorf("%s resolved to foil=%v, want %v", tt.sku, co.Foil, tt.foil)
			}
		})
	}
}

// TestResolveNativeTokenPair pins a two-sided token listing SCG never
// publishes a scryfall_id for at all, resolving instead to a real printing
// mtgjson already files under one combined "X // Y" name of its own - not a
// synthetic tokenProducts-derived pairing (magic.MatchNativeTokenPair,
// not MatchTokenPairing) - anchored by the sku's own filing set and number,
// with the datastore's own face order ("Weird // Goblin") differing from
// SCG's own listing order ("Goblin Token} // {Weird Token").
func TestResolveNativeTokenPair(t *testing.T) {
	b := withMagic(t)

	p := CatalogProduct{
		SKU: "SGL-MTG-GK1-T01-ENN", Name: "{Copy Token} // {Horror Token}",
		Game: "Magic: The Gathering", Set: "Guild Kit - Guilds of Ravnica",
		Rarity: "Token", ProductType: ProductTypeSingles,
		Finish: "Non-foil", FinishGroup: "Non-foil", Language: "English",
	}
	id, err := resolveProductID(b, GameMagic, p)
	if err != nil {
		t.Fatalf("resolveProductID(%s) = %v", p.SKU, err)
	}
	co, err := b.GetUUID(id)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", id, err)
	}
	if co.Identifiers["derivedTokenPair"] == "true" {
		t.Errorf("%s resolved to a synthetic derived pairing (%s), want mtgjson's own native combined printing", p.SKU, id)
	}
	if co.Card.Name != "Copy // Horror" || co.SetCode != "TGK1" || co.Number != "1" {
		t.Errorf("%s resolved to %s #%s [%s], want Copy // Horror #1 [TGK1]", p.SKU, co.Card.Name, co.Number, co.SetCode)
	}
}

// TestResolveTokenPairingBySetNumber pins a two-sided token listing SCG
// never publishes a scryfall_id for, resolving to a synthetic
// tokenProducts-derived pairing (magic.MatchTokenPairingBySetNumber)
// anchored by the sku's own filing set and number for the first face alone
// - the same mechanism cardkingdom's own Mystery Booster/List fallback
// uses, reached here through a sku that names its filing set directly
// rather than bundling two unrelated sets into one sku.
func TestResolveTokenPairingBySetNumber(t *testing.T) {
	b := withMagic(t)

	p := CatalogProduct{
		SKU: "SGL-MTG-AFC-T03_AFR_T06-ENN", Name: "{Illusion Token} // {Skeleton Token}",
		Game: "Magic: The Gathering", Set: "Adventures in the Forgotten Realms Commander",
		Rarity: "Token", ProductType: ProductTypeSingles,
		Finish: "Non-foil", FinishGroup: "Non-foil", Language: "English",
	}
	id, err := resolveProductID(b, GameMagic, p)
	if err != nil {
		t.Fatalf("resolveProductID(%s) = %v", p.SKU, err)
	}
	co, err := b.GetUUID(id)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", id, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("%s resolved to %s (%s), want a derived token pairing", p.SKU, id, co.Card.Name)
	}
	if co.Card.Name != "Illusion // Skeleton" || co.SetCode != "TAFR" {
		t.Errorf("%s resolved to %s [%s], want Illusion // Skeleton [TAFR]", p.SKU, co.Card.Name, co.SetCode)
	}
}

// TestResolveDungeonPairings pins AFR's dungeon cards, a shape neither
// face of which is a token: SCG spells its own dungeon-card listings the
// same brace-and-suffix way it spells tokens ("{X Dungeon}" rather than
// "{X Token}"), which the two-sided trigger and cleanFaceName both had to
// learn about specifically, and which OAFR (Forgotten Realms Oversized
// Cards, a memorabilia sibling of AFR) duplicates the ids of under its
// own uuids - see mtgmatcher/magic/tokenpairs.go's idCanonicalKey. The
// foil case (no scryfall_id at all) exercises MatchTokenPairingBySetNumber
// rather than MatchTokenPairing; the third case pins catalogNames' own
// fixup for a real SCG typo ("Lost Mine of THE Phandelver" - the real
// card carries no "the") that would otherwise make this exact pairing
// unreachable regardless of how well the rest of the matching works.
func TestResolveDungeonPairings(t *testing.T) {
	b := withMagic(t)

	for _, tt := range []struct {
		desc       string
		sku        string
		name       string
		foil       bool
		scryfallID string
		wantName   string
	}{
		{
			desc:     "dungeon // dungeon, no token on either face",
			sku:      "SGL-MTG-AFR-T20T22-ENN",
			name:     "{Dungeon of the Mad Mage Dungeon} // {Tomb of Annihilation Dungeon}",
			foil:     false,
			wantName: "Dungeon of the Mad Mage // Tomb of Annihilation",
		},
		{
			desc:     "dungeon // token, no scryfall_id (foil)",
			sku:      "SGL-MTG-AFR-T20T12-ENF",
			name:     "{Dungeon of the Mad Mage Dungeon} // {Goblin Token}",
			foil:     true,
			wantName: "Goblin // Dungeon of the Mad Mage",
		},
		{
			desc:     "dungeon // dungeon, SCG's own \"Lost Mine of THE Phandelver\" typo",
			sku:      "SGL-MTG-AFR-T20T21-ENN",
			name:     "{Dungeon of the Mad Mage Dungeon} // {Lost Mine of the Phandelver Dungeon}",
			foil:     false,
			wantName: "Dungeon of the Mad Mage // Lost Mine of Phandelver",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			finish, group := "Non-foil", "Non-foil"
			if tt.foil {
				finish, group = "Foil", "Foil"
			}
			p := CatalogProduct{
				SKU: tt.sku, Name: tt.name, Game: "Magic: The Gathering",
				Set: "Adventures in the Forgotten Realms", Rarity: "Token", ProductType: ProductTypeSingles,
				Finish: finish, FinishGroup: group, Language: "English",
				ScryfallID: tt.scryfallID,
			}
			id, err := resolveProductID(b, GameMagic, p)
			if err != nil {
				t.Fatalf("resolveProductID(%s) = %v", tt.sku, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.Identifiers["derivedTokenPair"] != "true" {
				t.Errorf("%s resolved to %s (%s), want a derived token pairing", tt.sku, id, co.Card.Name)
			}
			if co.Card.Name != tt.wantName || co.SetCode != "TAFR" {
				t.Errorf("%s resolved to %s [%s], want %s [TAFR]", tt.sku, co.Card.Name, co.SetCode, tt.wantName)
			}
			if co.Foil != tt.foil {
				t.Errorf("%s resolved to foil=%v, want %v", tt.sku, co.Foil, tt.foil)
			}
		})
	}
}
