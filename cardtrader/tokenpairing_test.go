package cardtrader

import (
	"fmt"
	"testing"
)

// TestPreprocessResolvesTokenPairing pins a two-sided token sheet blueprint
// ("X // Y") resolving to the combined entity matcher/magic/tokenpairs.go
// derives for the same physical pairing, rather than to whichever single
// face one of the blueprint's own ids happens to name - the same
// "silently prices the whole two-sided product as if it were just the one
// face" risk cardkingdom's and starcitygames's own versions of this check
// exist to avoid (see magic.MatchTokenPairing's doc comment, shared by all
// three). Card Trader differs from both in carrying its own TCGplayerID
// that is frequently the pairing's own product id directly, with no
// scryfall id and no name-splitting needed at all - the second case below.
func TestPreprocessResolvesTokenPairing(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		desc        string
		bp          Blueprint
		wantName    string
		wantSet     string
		wantDerived bool
	}{
		{
			desc: "anchored by the blueprint's own scryfall id",
			bp: Blueprint{
				ID:         159328,
				Name:       "Lost Mine of Phandelver // Skeleton",
				CategoryID: CategoryMagicTokens,
				ScryfallID: "59b11ff8-f118-4978-87dd-509dc0c8c932",
			},
			wantName:    "Skeleton // Lost Mine of Phandelver",
			wantSet:     "TAFR",
			wantDerived: true,
		},
		{
			desc: "no scryfall id at all - the blueprint's own TCGplayerID is the pairing's own product id directly",
			bp: Blueprint{
				ID:          264230,
				Name:        "Illusion // Serra the Benevolent Emblem",
				CategoryID:  CategoryMagicTokens,
				TCGplayerID: 192366,
			},
			wantName:    "Illusion // Serra the Benevolent Emblem",
			wantSet:     "TMH1",
			wantDerived: true,
		},
		{
			// Copy // Horror is mtgjson's own native combined printing
			// (Guild Kit's own "X // Y" name), not a synthetic derived
			// pairing - proving the number-anchored path tries
			// magic.MatchNativeTokenPair first, the same order
			// starcitygames's own sku-anchored fallback already uses.
			desc: "neither id, but the blueprint's own composite collector_number anchors both faces to a native printing",
			bp: func() Blueprint {
				bp := Blueprint{ID: 49703, Name: "Copy // Horror", CategoryID: CategoryMagicTokens}
				bp.Expansion.Name = "GRN Guild Kit"
				bp.Properties.Number = "T 01/02"
				return bp
			}(),
			wantName:    "Copy // Horror",
			wantSet:     "TGK1",
			wantDerived: false,
		},
		{
			desc: "neither id nor a parseable number - both faces' own names, guarded by the blueprint's own claimed edition",
			bp: func() Blueprint {
				bp := Blueprint{ID: 46598, Name: "Cat Warrior // Beast", CategoryID: CategoryMagicTokens}
				bp.Expansion.Name = "Commander 2018"
				return bp
			}(),
			wantName:    "Beast // Cat Warrior",
			wantSet:     "TC18",
			wantDerived: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := Preprocess(b, &tt.bp)
			if err != nil {
				t.Fatalf("Preprocess(%d) = %v", tt.bp.ID, err)
			}
			co, err := b.GetUUID(card.ID)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", card.ID, err)
			}
			if derived := co.Identifiers["derivedTokenPair"] == "true"; derived != tt.wantDerived {
				t.Errorf("blueprint %d resolved to %s (%s), derivedTokenPair=%v, want %v", tt.bp.ID, card.ID, co.Name, derived, tt.wantDerived)
			}
			if co.Name != tt.wantName {
				t.Errorf("blueprint %d resolved to %q, want %q", tt.bp.ID, co.Name, tt.wantName)
			}
			if co.SetCode != tt.wantSet {
				t.Errorf("blueprint %d resolved to set %s, want %s", tt.bp.ID, co.SetCode, tt.wantSet)
			}
		})
	}
}

// TestPreprocessRefusesNamePairEditionMismatch pins the safety guard on
// magic.MatchTokenPairingByNamesAndEdition's own use here: "Bird // Myr" is
// a real derived pairing (Modern Horizons' own token sheet), but Commander
// 2016 - a blueprint with neither a scryfall_id nor a tcgplayer_id to
// anchor either face by identity - never sold that exact pairing. Without
// the edition check this would resolve to Modern Horizons' Bird // Myr
// regardless of what the blueprint's own edition actually names; with it,
// the disagreement must fall through to a refusal rather than namedID
// silently resolving one bare face instead.
func TestPreprocessRefusesNamePairEditionMismatch(t *testing.T) {
	b := realDatastore(t)

	bp := Blueprint{ID: 46731, Name: "Bird // Myr", CategoryID: CategoryMagicTokens}
	bp.Expansion.Name = "Commander 2016"

	card, err := Preprocess(b, &bp)
	if err != nil {
		// A hard refusal (no id, no number, no set-anchor for namedID's
		// own name-based Match fallback either) is an acceptable outcome
		// here too - the only unacceptable one is silently resolving to
		// the wrong derived pairing.
		return
	}
	co, err := b.GetUUID(card.ID)
	if err == nil && co.Identifiers["derivedTokenPair"] == "true" {
		t.Errorf("blueprint %d resolved to derived pairing %s [%s], want a refusal: Commander 2016 never sold Modern Horizons' Bird // Myr pairing", bp.ID, co.Name, co.SetCode)
	}
}

// TestTokenPairNumbers pins the real shapes measured against Card
// Trader's own collector_number for a two-sided token blueprint - see
// tokenPairNumberRes's own comment for where each one came from.
func TestTokenPairNumbers(t *testing.T) {
	tests := []struct {
		desc       string
		number     string
		n1, n2     string
		wantParsed bool
	}{
		{"leading T marker", "T 01/02", "1", "2", true},
		{"leading F marker", "F 1/3", "1", "3", true},
		{"leading CT marker", "CT 01/10", "1", "10", true},
		{"no marker at all", "13/15", "13", "15", true},
		{"both numbers then a shared total and marker", "024-027/031 T", "24", "27", true},
		{"each face's own number-total-T marker", "05-014T / 03-014T", "5", "3", true},
		{"a bare single number names only one face", "011", "", "", false},
		{"empty", "", "", "", false},
		{"a set code fused into the number", "T 13/REX02", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			n1, n2, ok := tokenPairNumbers(tt.number)
			if ok != tt.wantParsed {
				t.Fatalf("tokenPairNumbers(%q) ok = %v, want %v", tt.number, ok, tt.wantParsed)
			}
			if !ok {
				return
			}
			if n1 != tt.n1 || n2 != tt.n2 {
				t.Errorf("tokenPairNumbers(%q) = (%q, %q), want (%q, %q)", tt.number, n1, n2, tt.n1, tt.n2)
			}
		})
	}
}

// TestPreprocessResolvesPairingWithItsFaceProduct pins a real blueprint
// resolving to a derived pairing whose TCGplayer product mtgjson files on
// one of its faces, C17 Cat Dragon #9: the pairing takes that id from the
// face (see deriveTokenPairs). No Card Trader-specific code exists for this
// case: the same number-anchored resolution that already finds an
// ordinary, id-bearing derived pairing reaches this one too.
func TestPreprocessResolvesPairingWithItsFaceProduct(t *testing.T) {
	b := realDatastore(t)

	bp := Blueprint{ID: 46667, Name: "Dragon // Cat Dragon", CategoryID: CategoryMagicTokens}
	bp.Expansion.Name = "Commander 2017"
	bp.Properties.Number = "07/09"

	card, err := Preprocess(b, &bp)
	if err != nil {
		t.Fatalf("Preprocess(%d) = %v", bp.ID, err)
	}
	co, err := b.GetUUID(card.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", card.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("blueprint %d resolved to %s (%s), want a derived token pairing", bp.ID, card.ID, co.Name)
	}
	if co.Identifiers["tcgplayerProductId"] != "139794" {
		t.Errorf("blueprint %d resolved to %s (%s) with product %q, want 139794", bp.ID, card.ID, co.Name, co.Identifiers["tcgplayerProductId"])
	}
	if co.SetCode != "TC17" {
		t.Errorf("blueprint %d resolved to set %s, want TC17", bp.ID, co.SetCode)
	}
}

// TestIsTwoSidedTokenBlueprintSpacing pins the separator match against Card
// Trader's own inconsistent spacing around "//": blueprint 276925 ("Thopter
// //Dog") lost its silencing when the check moved from theCard's own
// Match-normalized Name to the blueprint's raw one, since the raw vendor
// text carries no space after the slashes.
func TestIsTwoSidedTokenBlueprintSpacing(t *testing.T) {
	tests := []struct {
		desc string
		bp   *Blueprint
		name string
		want bool
	}{
		{"normal spacing", &Blueprint{CategoryID: CategoryMagicTokens}, "Hero // Hero", true},
		{"no space after the slashes, blueprint 276925", &Blueprint{CategoryID: CategoryMagicTokens}, "Thopter //Dog", true},
		{"no separator at all", &Blueprint{CategoryID: CategoryMagicTokens}, "Solo Token", false},
		{"separator present but outside Tokens", &Blueprint{CategoryID: CategoryMagicSingles}, "Turn // Burn", false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := isTwoSidedTokenBlueprint(tt.bp, tt.name); got != tt.want {
				t.Errorf("isTwoSidedTokenBlueprint(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestProcessProductsSilencesForeignTokenPairing pins the token-pairing
// silence check against the blueprint's own name rather than theCard's.
// Match mutates theCard's Name while validating a foreign-language listing's
// id-path language - resetting it to the single face the id names, "Hero",
// where the blueprint itself still says "Hero // Hero". An English listing
// lands first here and a Spanish listing of the same real derived pairing
// follows, the order a live run sees them in.
func TestProcessProductsSilencesForeignTokenPairing(t *testing.T) {
	b := realDatastore(t)

	bp := &Blueprint{
		ID:          341108,
		Name:        "Hero // Hero",
		GameID:      GameMagic,
		CategoryID:  CategoryMagicTokens,
		TCGplayerID: 638098,
	}
	bp.Expansion.Name = "FINAL FANTASY"
	bp.Properties.Number = "T 2/7"

	var logged []string
	ct := &Market{
		backend:    b,
		gameID:     GameMagic,
		blueprints: map[int]*Blueprint{bp.ID: bp},
		logCallback: func(format string, a ...any) {
			logged = append(logged, fmt.Sprintf(format, a...))
		},
	}

	newProduct := func(lang string) Product {
		var p Product
		p.BlueprintID = bp.ID
		p.Quantity = 1
		p.Price.Cents = 100
		p.Price.Currency = "USD"
		p.Properties.Condition = "Near Mint"
		p.Properties.Number = bp.Properties.Number
		p.Properties.MTGLanguage = lang
		return p
	}

	ch := make(chan resultChan, 2)
	ct.processProducts(ch, bp.ID, []Product{newProduct("en"), newProduct("es")})
	close(ch)

	var landed int
	for range ch {
		landed++
	}
	if landed != 1 {
		t.Errorf("landed %d products, want 1 (the English listing only)", landed)
	}
	if len(logged) != 0 {
		t.Errorf("logged %v for the Spanish listing, want silence: a foreign listing of a real derived pairing has no supported printing to land on, the same as the pairing's own English refusal case", logged)
	}
}

// tokenBlueprint is a Magic token blueprint as FormatBlueprints leaves it.
func tokenBlueprint(id int, name, edition, number, scryfallID string) Blueprint {
	bp := Blueprint{ID: id, Name: name, CategoryID: CategoryMagicTokens, ScryfallID: scryfallID}
	bp.Expansion.Name = edition
	bp.Properties.Number = number
	return bp
}

// TestPreprocessAnchorsTheSecondFaceNumber pins a two-sided token blueprint
// whose collector number anchors only its second face: the pairing is the
// one carrying both numbers, and a pairing that matches one number and the
// partner's name alone is not it.
func TestPreprocessAnchorsTheSecondFaceNumber(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		bp       Blueprint
		wantName string
		wantSet  string
	}{
		{
			// Vanguard Suppressor #10 pairs with two Soldiers, so only
			// Soldier #4 names one pairing.
			bp:       tokenBlueprint(218243, "Vanguard Suppressor // Soldier", "Universes Beyond: Warhammer 40,000", "T10/4", "e5d319cb-4982-4fb9-93d4-1cc846b03c30"),
			wantName: "Soldier // Vanguard Suppressor Token",
			wantSet:  "T40K",
		},
		{
			// The scryfall id names the Spirit, a face of neither.
			bp:       tokenBlueprint(3610, "Soldier // Beast", "Archenemy: Nicol Bolas", "1/5", "5b2c2f3d-9489-4e7b-85e3-db1f2ee0f601"),
			wantName: "Soldier // Beast",
			wantSet:  "TE01",
		},
		{
			bp:       tokenBlueprint(60562, "Goat // Food", "Throne of Eldraine Promos", "T01/15", ""),
			wantName: "Goat // Food",
			wantSet:  "TELD",
		},
		{
			// Card Trader spaces this separator its own way.
			bp:       tokenBlueprint(169067, "Icy Manalith //Dwarf Berserker", "Kaldheim Promos", "T 17/12", ""),
			wantName: "Dwarf Berserker // Icy Manalith",
			wantSet:  "TKHM",
		},
	} {
		card, err := Preprocess(b, &tt.bp)
		if err != nil {
			t.Fatalf("Preprocess(%d) = %v", tt.bp.ID, err)
		}
		co, err := b.GetUUID(card.ID)
		if err != nil {
			t.Fatalf("blueprint %d: GetUUID(%q) = %v", tt.bp.ID, card.ID, err)
		}
		if co.Name != tt.wantName || co.SetCode != tt.wantSet || co.Identifiers["derivedTokenPair"] != "true" {
			t.Errorf("blueprint %d resolved to %s %q, want the %s pairing %q", tt.bp.ID, co.SetCode, co.Name, tt.wantSet, tt.wantName)
		}
	}

	// Cat #5 pairs with Elemental #24, not with the Elemental #25 quoted.
	bp := tokenBlueprint(256138, "Elemental // Cat", "Commander Masters", "T 25/5", "")
	bp.TCGplayerID = 510097
	card, err := Preprocess(b, &bp)
	if err == nil && card.ID != "" {
		co, err := b.GetUUID(card.ID)
		if err == nil && co.Identifiers["derivedTokenPair"] == "true" {
			t.Errorf("blueprint %d resolved to %s %q #%s, want no pairing", bp.ID, co.SetCode, co.Name, co.Number)
		}
	}
}

// TestProcessProductsRefusesTokenSheetOnAFace pins a sheet of two different
// tokens whose pairing the datastore does not mint: its ids name the Horror
// face alone, and the listing is refused rather than priced as that face.
func TestProcessProductsRefusesTokenSheetOnAFace(t *testing.T) {
	b := realDatastore(t)

	bp := tokenBlueprint(3609, "Horror // Zombie", "Archenemy: Nicol Bolas", "3/20", "8cfb1a9c-3f34-4362-bc2d-4cd584488820")
	bp.GameID = GameMagic
	bp.TCGplayerID = 134946

	ct := &Market{
		backend:     b,
		gameID:      GameMagic,
		blueprints:  map[int]*Blueprint{bp.ID: &bp},
		logCallback: func(string, ...any) {},
	}
	var p Product
	p.ID = 1
	p.BlueprintID = bp.ID
	p.Quantity = 1
	p.Price.Cents = 100
	p.Price.Currency = "USD"
	p.Properties.Condition = "Near Mint"
	p.Properties.MTGLanguage = "en"

	ch := make(chan resultChan, 1)
	ct.processProducts(ch, bp.ID, []Product{p})
	close(ch)
	for result := range ch {
		co, err := b.GetUUID(result.cardID)
		if err == nil {
			t.Errorf("blueprint %d landed on %s %q #%s, want a refusal", bp.ID, co.SetCode, co.Name, co.Number)
		}
	}
}

// TestTokenSheetOnFaceReadsCardTraderSpacing pins the refusal reading a pair
// name however Card Trader spaces its separator: two different tokens on one
// face are refused, the same token on both sides of its sheet is kept.
func TestTokenSheetOnFaceReadsCardTraderSpacing(t *testing.T) {
	b := realDatastore(t)

	bp := tokenBlueprint(276925, "Thopter //Dog", "Murders at Karlov Manor Collectors", "T 20/1", "")
	bp.TCGplayerID = 537562
	card, err := Preprocess(b, &bp)
	if err != nil {
		t.Fatalf("Preprocess(%d) = %v", bp.ID, err)
	}
	pair, err := b.GetUUID(card.ID)
	if err != nil {
		t.Fatalf("blueprint %d: GetUUID(%q) = %v", bp.ID, card.ID, err)
	}
	face := pair.Identifiers["tokenPairPartA"]
	if !tokenSheetOnFace(b, &bp, face) {
		t.Errorf("blueprint %d on its face %s is kept, want it refused", bp.ID, face)
	}
	if tokenSheetOnFace(b, &bp, card.ID) {
		t.Errorf("blueprint %d on its pairing %s is refused, want it kept", bp.ID, card.ID)
	}

	// Blueprint 149464's own name is "Walker // Walker", spaced here the
	// way 276925 is; SLD Walker #149 is its printing.
	walker := tokenBlueprint(149464, "Walker //Walker", "Secret Lair Drop Series", "149", "b4c637c4-c1cc-4852-9561-78add390ef52")
	if tokenSheetOnFace(b, &walker, "50fcb456-82d6-5452-8318-8a58a6ee5f0f") {
		t.Errorf("blueprint %d on its one printing is refused, want it kept", walker.ID)
	}
}
