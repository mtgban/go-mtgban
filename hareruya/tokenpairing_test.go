package hareruya

import "testing"

// TestPreprocessResolvesTokenPairing pins a two-sided token listing
// resolving to the combined entity mtgmatcher/magic derives for it,
// anchored by the set code and collector numbers the storefront's own
// product_name already carries: "(026/015)《Ooze+Treasure Token》[NCC]"
// names Ooze as NCC's own token #26 and Treasure as its #15 - confirmed
// against real listings to agree with mtgjson's own token numbering
// exactly.
func TestPreprocessResolvesTokenPairing(t *testing.T) {
	b := realDatastore(t)

	product := Product{
		Product:       "112148",
		ProductName:   "(026/015)《ウーズ+宝物トークン/Ooze+Treasure Token》[NCC] 緑/茶",
		ProductNameEN: "(026/015)《Ooze+Treasure Token》[NCC]",
		CardName:      "Ooze/Treasure Token",
		Language:      "2",
		FoilFlag:      "0",
	}
	out, err := Preprocess(b, product)
	if err != nil {
		t.Fatalf("Preprocess(%s) = %v", product.Product, err)
	}
	co, err := b.GetUUID(out.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", out.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Treasure // Ooze" {
		t.Errorf("resolved to %q, want \"Treasure // Ooze\"", co.Name)
	}
}

// TestPreprocessRefusesTokenPairingOnNameMismatch pins the reverse: this
// real Commander 2020 listing spells its second face "Human Soldier", but
// C20's own token #5 - the number this same listing names - is plainly
// "Soldier" with no "Human" in mtgjson's own record. The set+number anchor
// finds nothing at that number under the name this listing actually gives,
// so it must refuse rather than assume the storefront's own wording and
// mtgjson's own name are the same token.
func TestPreprocessRefusesTokenPairingOnNameMismatch(t *testing.T) {
	b := realDatastore(t)

	product := Product{
		Product:       "85429",
		ProductName:   "(009/005)《ゾンビ+人間・兵士トークン/Zombie+Human Soldier Token》[C20] 黒/白",
		ProductNameEN: "(009/005)《Zombie+Human Soldier Token》[C20]",
		CardName:      "Zombie Token/Human Soldier Token",
		Language:      "2",
		FoilFlag:      "0",
	}
	_, err := Preprocess(b, product)
	if err == nil {
		t.Fatal("Preprocess succeeded, want a refusal: C20's own token #5 is named \"Soldier\", not \"Human Soldier\"")
	}
}

// TestPreprocessIgnoresOrdinarySplitCardSlash pins that an ordinary split
// card's own card_name, which carries the same bare "/" a two-sided token
// pairing does ("Trial/Error", not "Trial // Error"), is never mistaken for
// one: the gate also requires "Token" in the name, which a real split
// card's name never carries. Without that second condition this real
// listing would be routed into preprocessTokenPair and refused outright
// instead of resolving normally.
func TestPreprocessIgnoresOrdinarySplitCardSlash(t *testing.T) {
	b := realDatastore(t)

	product := Product{
		Product:       "34307",
		ProductName:   "《試行+錯誤/Trial+Error》[C16] 分U",
		ProductNameEN: "《Trial+Error》[C16]",
		CardName:      "Trial/Error",
		Language:      "2",
		FoilFlag:      "0",
	}
	out, err := Preprocess(b, product)
	if err != nil {
		t.Fatalf("Preprocess(%s) = %v, want the ordinary split-card path to still resolve it", product.Product, err)
	}
	if out.Name != "Trial/Error" {
		t.Errorf("resolved name %q, want the untouched \"Trial/Error\": the token-pair gate must not have fired", out.Name)
	}
}
