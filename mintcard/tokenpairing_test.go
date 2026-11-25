package mintcard

import "testing"

// TestPreprocessResolvesTokenPairingWithDenominator pins a two-sided token
// listing resolving to the combined entity matcher/magic derives for
// it, anchored by the catalog's own set code and each face's own number:
// "Bird Token (2/21) // Saproling Token (16/21)" names Bird as C16's own
// token #2 and Saproling as its #16 - confirmed against real listings to
// agree with mtgjson's own token numbering exactly. The "/21" is the
// sheet's own token count, no part of either real number.
func TestPreprocessResolvesTokenPairingWithDenominator(t *testing.T) {
	b := realDatastore(t)

	out, err := preprocess(b, "Bird Token (2/21) // Saproling Token (16/21)", "", "", "English", "Commander 2016", "C16")
	if err != nil {
		t.Fatalf("preprocess() = %v", err)
	}
	co, err := b.GetUUID(out.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", out.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Bird // Saproling" {
		t.Errorf("resolved to %q, want \"Bird // Saproling\"", co.Name)
	}
}

// TestPreprocessResolvesTokenPairingBareNumber pins the same anchor working
// off a bare, zero-padded number with no denominator at all - some sets
// (measured: Aetherdrift) spell it this way instead: "Elephant Token
// (006) // Insect Token (007)" names Elephant as DFT's own token #6 and
// Insect as its #7.
func TestPreprocessResolvesTokenPairingBareNumber(t *testing.T) {
	b := realDatastore(t)

	out, err := preprocess(b, "Elephant Token (006) // Insect Token (007)", "", "", "English", "Aetherdrift", "DFT")
	if err != nil {
		t.Fatalf("preprocess() = %v", err)
	}
	co, err := b.GetUUID(out.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", out.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Elephant // Insect" {
		t.Errorf("resolved to %q, want \"Elephant // Insect\"", co.Name)
	}
}

// TestPreprocessResolvesTokenPairingMissingSpace pins that a listing
// missing the space before its own parenthetical number - measured on
// Commander 2014/2015 rows ("Spirit Token(22/24)") - still resolves: the
// shared magic.StripFaceWrapping only trims starting at " (", so without
// normalizing the space first the face name would keep its parenthetical
// attached and never reduce to the plain name the datastore carries.
func TestPreprocessResolvesTokenPairingMissingSpace(t *testing.T) {
	b := realDatastore(t)

	out, err := preprocess(b, "Soldier Token(06/36) // Spirit Token(07/36)", "", "", "English", "Commander 2014", "C14")
	if err != nil {
		t.Fatalf("preprocess() = %v", err)
	}
	co, err := b.GetUUID(out.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", out.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Soldier // Spirit" {
		t.Errorf("resolved to %q, want \"Soldier // Spirit\"", co.Name)
	}
}

// TestPreprocessRefusesTokenPairingWithNoMTGJSONLink pins the reverse:
// "Horror // Zombie" is a real Archenemy: Nicol Bolas listing (also seen
// on Cool Stuff Inc's own buylist, see project memory), but mtgjson
// carries no Zombie token under E01 at all - no tokenProducts record
// links the two faces, so there is nothing to anchor. A refusal is
// correct rather than a gap to chase further.
func TestPreprocessRefusesTokenPairingWithNoMTGJSONLink(t *testing.T) {
	b := realDatastore(t)

	_, err := preprocess(b, "Horror Token (3/5) // Zombie Token (20/25)", "", "", "English", "Archenemy: Nicol Bolas", "E01")
	if err == nil {
		t.Fatal("preprocess() succeeded, want a refusal: no mtgjson record links Horror to Zombie in this edition")
	}
}

// TestPreprocessIgnoresSingleFacedToken pins that an ordinary single-faced
// token name, which names no second face, is never routed through the
// two-sided pairing path: the gate is namesTokenSheet, so a plain token
// keeps going through the ordinary preprocess pipeline below.
func TestPreprocessIgnoresSingleFacedToken(t *testing.T) {
	b := realDatastore(t)

	out, err := preprocess(b, "Zombie Token", "", "", "English", "Commander 2016", "C16")
	if err != nil {
		t.Fatalf("preprocess(%s) = %v, want the ordinary single-token path to still resolve it", "Zombie Token", err)
	}
	if out.Name != "Zombie Token" {
		t.Errorf("resolved name %q, want \"Zombie Token\": the token-pair gate must not have fired", out.Name)
	}
}

// TestPreprocessTokenPairNeedsTheTokenSetWalk pins that resolving against
// the raw parent code Mint Card's own catalog gives genuinely depends on
// magic.SetTokenSetCode's own walk to the token sheet: mtgjson's loader
// moves a set's tokens out of its own Cards entirely once it has a
// distinct token set code, so a plain MatchInSetNumber against the parent
// ("C16") alone - what preprocessTokenPair would do without that walk -
// finds nothing, even though the pairing itself resolves.
func TestPreprocessTokenPairNeedsTheTokenSetWalk(t *testing.T) {
	b := realDatastore(t)

	if cards := b.MatchInSetNumber("Bird", "C16", "2"); len(cards) != 0 {
		t.Fatalf("MatchInSetNumber(Bird, C16, 2) = %d cards, want 0: this pin needs the parent code alone to miss", len(cards))
	}

	out, err := preprocess(b, "Bird Token (2/21) // Saproling Token (16/21)", "", "", "English", "Commander 2016", "C16")
	if err != nil {
		t.Fatalf("preprocess() = %v, want the SetTokenSetCode walk to still find it", err)
	}
	co, err := b.GetUUID(out.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", out.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
}

// TestPreprocessResolvesTokenEmblemSheet pins the sheets whose second face
// is an emblem, which the listing spells word first and with "Token" once:
// each lands on the pairing the datastore derives, misspellings and all.
func TestPreprocessResolvesTokenEmblemSheet(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		name, finish, edition, setCode, want string
	}{
		{"Treasure Token (15/19) // Emblem Lolth, Spider Queen (17/19)", "Foil", "Adventures in the Forgotten Realms", "AFR", "Treasure // Lolth, Spider Queen Emblem"},
		{"Emblem Wrenn and Six (24/24) // Spirit Token (8/24)", "Regular", "Double Masters 2022", "2X2", "Spirit // Wrenn and Six Emblem"},
		{"Dwarf Berserker Token (12/23) // Emblem Tibalt, Cosmic Imposter (21/23)", "Foil", "Kaldheim", "KHM", "Dwarf Berserker // Tibalt, Cosmic Impostor Emblem"},
	} {
		out, err := preprocess(b, tt.name, "", tt.finish, "English", tt.edition, tt.setCode)
		if err != nil {
			t.Errorf("preprocess(%q) = %v", tt.name, err)
			continue
		}
		co, err := b.GetUUID(out.ID)
		if err != nil {
			t.Errorf("GetUUID(%s) = %v", out.ID, err)
			continue
		}
		if co.Identifiers["derivedTokenPair"] != "true" || co.Name != tt.want {
			t.Errorf("preprocess(%q) resolved to %q, want the pairing %q", tt.name, co.Name, tt.want)
		}
	}
}

// TestProcessEntryMovesSkuFaceToSheet pins a sku the catalog resolves to
// one face of the sheet it sells: the listing lands on the pairing.
func TestProcessEntryMovesSkuFaceToSheet(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		card             Card
		face             string
		edition, setCode string
		want             string
	}{
		{Card{ID: "738567", Name: "Spirit Token (2/19) // Treasure Token (17/19)", Quantity: 1, Price: "0.49", TCGplayerID: 5434620}, "af3f43ba-5ccd-5df8-bb7b-5b26307c67b1", "Kamigawa: Neon Dynasty", "NEO", "Spirit // Treasure"},
	} {
		mint := NewScraper(b)
		mint.processEntry(map[int]string{tt.card.TCGplayerID: tt.face}, tt.card, "Mint", "Regular", "English", tt.edition, tt.setCode, "")
		if len(mint.inventory) != 1 {
			t.Errorf("%q priced on %d cards, want 1", tt.card.Name, len(mint.inventory))
			continue
		}
		for cardID := range mint.inventory {
			co, err := b.GetUUID(cardID)
			if err != nil {
				t.Errorf("GetUUID(%s) = %v", cardID, err)
				continue
			}
			if co.Identifiers["derivedTokenPair"] != "true" || co.Name != tt.want {
				t.Errorf("%q priced on %q, want the pairing %q", tt.card.Name, co.Name, tt.want)
			}
		}
	}
}
