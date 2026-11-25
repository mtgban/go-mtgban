package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/matcher"
)

// TestResolveMagicResolvesTokenPairing pins a two-sided token product
// resolving to the combined entity matcher/magic derives for it, anchored
// by both face names plus the product's own edition, its Number naming the
// same faces. Preprocess/Match alone is not built to read "X Token (...) //
// Y Token (...)" and refuses every one of these listings outright.
func TestResolveMagicResolvesTokenPairing(t *testing.T) {
	b := realDatastore(t)
	res := &resolver{backend: b, gameID: cm.GameMagic}

	product := &cm.Product{
		IDProduct: 362664, Name: "Manifest Token // Angel Token (W 4/4)",
		ExpansionName: "Commander 2018", Number: "T 01/03", // as the catalog lists it
	}
	cardID, _, byName, err := resolveMagic(res, product)
	if err != nil {
		t.Fatalf("resolveMagic(%d) = %v", product.IDProduct, err)
	}
	if cardID == "" {
		t.Fatal("resolveMagic returned no id, want the derived Manifest // Angel pairing")
	}
	if !byName {
		t.Error("resolveMagic said the pairing was found by id, want it named")
	}
	co, err := b.GetUUID(cardID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", cardID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Manifest // Angel" {
		t.Errorf("resolved to %q, want \"Manifest // Angel\"", co.Name)
	}
}

// TestResolveMagicRefusesAmbiguousTokenPairing pins the reverse: Commander
// 2016 prints more than one "Bird" token and more than one "Spirit" token,
// so a listing naming only "Bird" and "Spirit" cannot say which pairing it
// means - the same collision TestTokenPairIndexCollision already pins at
// the matcher/magic level. A refusal here is correct: guessing one of
// several candidate pairings would risk pricing the wrong physical product.
// The product's Number, "T 2/6", does say which; the name alone cannot.
func TestResolveMagicRefusesAmbiguousTokenPairing(t *testing.T) {
	b := realDatastore(t)
	res := &resolver{backend: b, gameID: cm.GameMagic}

	product := &cm.Product{
		IDProduct: 294192, Name: "Bird Token (W 1/1) // Spirit Token (W 1/1)",
		ExpansionName: "Commander 2016",
	}
	cardID, cardIDFoil, _, err := resolveMagic(res, product)
	if err != nil {
		t.Fatalf("resolveMagic(%d) = %v", product.IDProduct, err)
	}
	if cardID != "" || cardIDFoil != "" {
		t.Fatalf("resolveMagic(%d) = (%q, %q), want both empty: Commander 2016 prints several Bird and Spirit tokens, name+edition alone cannot say which pairing this is", product.IDProduct, cardID, cardIDFoil)
	}
}

// TestResolveMagicRefusesOtherFaces pins TMH3's two Copy // Eldrazi Angel
// products onto the one pair the datastore has: "CT 1/2" lands on it, and
// "CT 37/2", Copy #37's, names no printing of ours.
func TestResolveMagicRefusesOtherFaces(t *testing.T) {
	b := realDatastore(t)
	res := &resolver{backend: b, gameID: cm.GameMagic}
	for _, tt := range []struct {
		number string
		landed bool
	}{
		{"CT 1/2", true},
		{"CT 37/2", false},
	} {
		product := &cm.Product{Name: "Copy Token // Eldrazi Angel Token (C 4/4)", ExpansionName: "Modern Horizons 3: Tokens", Number: tt.number}
		cardID, cardIDFoil, _, err := resolveMagic(res, product)
		if err != nil || (cardID != "") != tt.landed || (cardIDFoil != "") != tt.landed {
			t.Errorf("resolveMagic(%q) = (%q, %q, %v), want landed %v", tt.number, cardID, cardIDFoil, err, tt.landed)
		}
	}
}

// TestResolveMagicAnchorsTokenPairAtFaces pins two-sided token products
// whose face names pair in several sets onto the pair their own edition's
// token set holds at the faces' numbers, including a sheet whose faces are
// not both named tokens and whose pair spans two token sets.
func TestResolveMagicAnchorsTokenPairAtFaces(t *testing.T) {
	b := realDatastore(t)
	res := &resolver{backend: b, gameID: cm.GameMagic}
	for _, tt := range []struct {
		product *cm.Product
		want    string
	}{
		{&cm.Product{
			IDProduct: 286424, Name: "Zombie Token (B 2/2) // Elephant Token (G 3/3)",
			ExpansionName: "Commander 2015", Number: "T 14/8", Rarity: "Token",
		}, "e0ba2c37-fcb7-540a-80a5-cb0e94491667"}, // TC15 8//14
		{&cm.Product{
			IDProduct: 294192, Name: "Bird Token (W 1/1) // Spirit Token (W 1/1)",
			ExpansionName: "Commander 2016", Number: "T 2/6", Rarity: "Token",
		}, "21844837-1928-5553-8723-256d30cf5b08"}, // TC16 2//6
		{&cm.Product{
			IDProduct: 652363, Name: "Lightning Rager (R 5/1) // Rogue (B 2/2)",
			ExpansionName: "Commander: Streets of New Capenna: Extras", Number: "T 19/7", Rarity: "Token",
		}, "ce2ae58c-c9e2-5c99-8318-11103660194c"}, // TSNC 7//19, Lightning Rager in TNCC
	} {
		cardID, _, _, err := resolveMagic(res, tt.product)
		if err != nil || cardID != tt.want {
			t.Errorf("resolveMagic(%d) = (%q, %v), want %s", tt.product.IDProduct, cardID, err, tt.want)
		}
	}
}

// TestPlaceMappedTokenPairOverFace pins two-sided token products mtgjson
// links to one face onto the pairing instead: off the face where the
// datastore pairs it with the other face, else at the product's own
// numbers, here Guild Kits' combined Wurm // Saproling where the link
// names a List Saproling.
func TestPlaceMappedTokenPairOverFace(t *testing.T) {
	b := realDatastore(t)
	res := &resolver{backend: b, gameID: cm.GameMagic}
	for _, tt := range []struct {
		id      int
		mapped  cm.CatalogProduct
		edition string
		want    string
	}{
		{719964, cm.CatalogProduct{Name: "Plant Token (G 0/2) // Chandra, Roaring Flame Emblem", Number: "T 6/14", Rarity: "Token"},
			"Secret Lair Commander Deck: From Cute to Brute", "befe10ef-5a07-5aa0-a52b-1e011f46ebf0"},
		{366031, cm.CatalogProduct{Name: "Wurm Token (G 5/5 Trample) // Saproling Token (G 1/1)", Number: "T 09/08", Rarity: "Token"},
			"Guilds of Ravnica: Guild Kits", "bdab3e0b-5513-5899-a4fa-f19ef6fc6d87"},
	} {
		got := res.placeMapped(tt.id, tt.mapped, cm.Expansion{Name: tt.edition})
		if got.cardID != tt.want {
			t.Errorf("placeMapped(%d) = %q, want %s", tt.id, got.cardID, tt.want)
		}
	}
}

// TestOtherFaces pins the face numbers a token product's Number states
// against its pair's: TMH3 sells Copy #1 and Copy #37 beside Eldrazi Angel
// #2, and only the first is the pair the datastore has.
func TestOtherFaces(t *testing.T) {
	b := &matcher.Backend{UUIDs: map[string]*matcher.CardObject{
		"pair":  {Card: matcher.Card{Number: "1//2"}},
		"other": {Card: matcher.Card{Number: "2//15"}},
	}}
	res := &resolver{backend: b, gameID: cm.GameMagic}
	for _, tt := range []struct {
		number, cardID string
		want           bool
	}{
		{"CT 1/2", "pair", false},
		{"CT 2/1", "pair", false},
		{"CT 37/2", "pair", true},
		{"T 15/REX02", "other", false},
		{"", "pair", false},
		{"T 1", "pair", false},
	} {
		got := res.otherFaces(&cm.Product{Number: tt.number}, tt.cardID)
		if got != tt.want {
			t.Errorf("otherFaces(%q, %s) = %v, want %v", tt.number, tt.cardID, got, tt.want)
		}
	}
}
