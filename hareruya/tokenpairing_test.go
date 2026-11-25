package hareruya

import "testing"

// TestPreprocessResolvesTokenPairing pins a two-sided token listing
// resolving to the combined entity matcher/magic derives for it,
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

// TestPreprocessTitleSheetNeedsBothNumbers pins a title whose first face
// pairs, by name, with its partner at another number: CMM's Spider #36
// pairs with Elemental #24, not the #37 this title row gives, so it is no
// pairing of this listing.
func TestPreprocessTitleSheetNeedsBothNumbers(t *testing.T) {
	b := realDatastore(t)

	product := Product{ProductName: "(036/037)《蜘蛛+エレメンタルトークン/Spider+Elemental Token》[CMM]", ProductNameEN: "(036/037)《Spider+Elemental Token》[CMM]", CardName: "Spider Token", Language: "2", FoilFlag: "0"}
	out, err := Preprocess(b, product)
	if err == nil && out.ID != "" {
		co, _ := b.GetUUID(out.ID)
		t.Errorf("%s resolved to the pairing %s, want none", product.ProductNameEN, co)
	}
}

// TestPreprocessReadsTokenSheetFromTitle pins the sheets whose card_name
// names one face, or spells "token" small, while the English title names
// both: each lands on the pairing, not on its first face, and a finish the
// pairing was never sold in goes unpriced. The rows with no product id
// are title rows of the storefront's shape for sheets it stocks none of
// today.
func TestPreprocessReadsTokenSheetFromTitle(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		product Product
		want    string
	}{
		{Product{Product: "92277", ProductName: "(015/016)《猫+兵士トークン/Cat+Soldier Token》[CMR] 白/白", ProductNameEN: "(015/016)《Cat+Soldier Token》[CMR]", CardName: "Cat Token", Language: "2", FoilFlag: "0"}, "Cat // Soldier"},
		{Product{Product: "152913", ProductName: "(020/030)《蛇+変異トークン/Snake+Morph token》[MKC] 緑/無", ProductNameEN: "(020/030)《Snake+Morph token》[MKC]", CardName: "Snake/Morph token", Language: "2", FoilFlag: "0"}, "Snake // Morph"},
		{Product{Product: "75753", ProductName: "(010/029)《ゾンビ+炉の再覚醒、オブニクシリス紋章トークン/Zombie+Emblem Ob Nixilis Regnited Token》[C19] 黒/無", ProductNameEN: "(010/029)《Zombie+Emblem Ob Nixilis Regnited Token》[C19]", CardName: "Zombie Token", Language: "2", FoilFlag: "0"}, "Zombie // Ob Nixilis Reignited Emblem"},
		{Product{Product: "75753", ProductName: "【Foil】(010/029)《ゾンビ+炉の再覚醒、オブニクシリス紋章トークン/Zombie+Emblem Ob Nixilis Regnited Token》[C19] 黒/無", ProductNameEN: "【Foil】(010/029)《Zombie+Emblem Ob Nixilis Regnited Token》[C19]", CardName: "Zombie Token", Language: "2", FoilFlag: "1"}, ""},
		{Product{ProductName: "(007/020)《恐竜+宝物トークン/Dinosaur+Treasure Token》[MOM]", ProductNameEN: "(007/020)《Dinosaur+Treasure Token》[MOM]", CardName: "Dinosaur Token", Language: "2", FoilFlag: "0"}, "Dinosaur // Treasure"},
		{Product{ProductName: "(011/002)《エレメンタル+鳥トークン/Elemental+Bird Token》[DMU]", ProductNameEN: "(011/002)《Elemental+Bird Token》[DMU]", CardName: "Elemental Token", Language: "2", FoilFlag: "0"}, "Bird // Elemental"},
		{Product{Product: "90664", ProductName: "(007/009)《ビースト+苗木トークン/Beast+Sapoling Token》[ZNC] 緑/緑", ProductNameEN: "(007/009)《Beast+Sapoling Token》[ZNC]", CardName: "Beast Token", Language: "2", FoilFlag: "0"}, "Beast // Saproling"},
	} {
		out, err := Preprocess(b, tt.product)
		id := ""
		if err == nil {
			id = out.ID
		}
		if err == nil && id == "" {
			id, err = b.Match(out)
		}
		if tt.want == "" {
			if err == nil {
				t.Errorf("%s resolved to %s, want it refused", tt.product.ProductNameEN, id)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", tt.product.ProductNameEN, err)
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Errorf("GetUUID(%s) = %v", id, err)
			continue
		}
		if co.Identifiers["derivedTokenPair"] != "true" || co.Name != tt.want {
			t.Errorf("%s resolved to %q, want the pairing %q", tt.product.ProductNameEN, co.Name, tt.want)
		}
	}
}
