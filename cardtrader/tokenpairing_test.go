package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
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
	realDatastore(t)

	for _, tt := range []struct {
		desc     string
		bp       Blueprint
		wantName string
		wantSet  string
	}{
		{
			desc: "anchored by the blueprint's own scryfall id",
			bp: Blueprint{
				ID:         159328,
				Name:       "Lost Mine of Phandelver // Skeleton",
				CategoryID: CategoryMagicTokens,
				ScryfallID: "59b11ff8-f118-4978-87dd-509dc0c8c932",
			},
			wantName: "Skeleton // Lost Mine of Phandelver",
			wantSet:  "TAFR",
		},
		{
			desc: "no scryfall id at all - the blueprint's own TCGplayerID is the pairing's own product id directly",
			bp: Blueprint{
				ID:          264230,
				Name:        "Illusion // Serra the Benevolent Emblem",
				CategoryID:  CategoryMagicTokens,
				TCGplayerID: 192366,
			},
			wantName: "Illusion // Serra the Benevolent Emblem",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := Preprocess(&tt.bp)
			if err != nil {
				t.Fatalf("Preprocess(%d) = %v", tt.bp.ID, err)
			}
			co, err := mtgmatcher.GetUUID(card.ID)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", card.ID, err)
			}
			if co.Identifiers["derivedTokenPair"] != "true" {
				t.Errorf("blueprint %d resolved to %s (%s), want a derived token pairing", tt.bp.ID, card.ID, co.Name)
			}
			if tt.wantSet != "" && co.SetCode != tt.wantSet {
				t.Errorf("blueprint %d resolved to set %s, want %s", tt.bp.ID, co.SetCode, tt.wantSet)
			}
		})
	}
}
