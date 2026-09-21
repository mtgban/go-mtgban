package mintcard

import "testing"

// TestPreprocessResolvesTokenPairingWithDenominator pins a two-sided token
// listing resolving to the combined entity mtgmatcher/magic derives for
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
// token name, which contains "Token" only once, is never routed through
// the two-sided pairing path: the gate is strings.Count(cardName,
// "Token") > 1, so a plain token keeps going through the ordinary
// preprocess pipeline below.
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
