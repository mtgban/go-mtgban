package cardtrader

import (
	"testing"
)

// TestPreprocessResolvesTokenPairing pins a two-sided token sheet blueprint
// ("X // Y") resolving to the combined entity mtgmatcher/magic/tokenpairs.go
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
