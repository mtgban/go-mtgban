package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// fixtureBackendWithPairing hand-builds a Backend carrying exactly one
// derived token pairing, with no datastore involved: bearUUID's own
// two-sided partner is partnerUUID, named partnerName, filed under tcgID.
// Every fixture shares the literal uuid "bear-uuid" for the Bear half and
// the same scryfall id for it, the way a reload of the same physical sheet
// would - so the two backends built by this test are what "two datastores
// in one process" or "one reloaded on top of another" actually look like,
// not two unrelated fixtures that could never collide by construction.
func fixtureBackendWithPairing(partnerUUID, partnerName, tcgID string) *mtgmatcher.Backend {
	const bearUUID = "bear-uuid"
	pairUUID := bearUUID + derivedTokenPairSuffix + partnerUUID
	b := &mtgmatcher.Backend{
		UUIDs: map[string]*mtgmatcher.CardObject{
			bearUUID:    {Card: mtgmatcher.Card{Name: "Bear"}},
			partnerUUID: {Card: mtgmatcher.Card{Name: partnerName}},
			pairUUID: {Card: mtgmatcher.Card{
				Name: "Bear // " + partnerName,
				Identifiers: map[string]string{
					"derivedTokenPair":   "true",
					"tokenPairPartA":     bearUUID,
					"tokenPairPartB":     partnerUUID,
					"tcgplayerProductId": tcgID,
				},
			}},
		},
		ExternalIdentifiers: map[mtgmatcher.IDSpace]map[string]string{
			mtgmatcher.IDSpaceScryfall: {"bear-scryfall": bearUUID},
		},
	}
	tokenPairs := buildTokenPairIndices(b)
	b.TokenPairIndex = tokenPairs.byFace
	b.TokenPairIDByUUIDs = tokenPairs.byUUIDPair
	b.TokenPairIDByBothNames = tokenPairs.byBothNames
	return b
}

// TestTokenPairIndexIsPerBackend pins the bug this whole feature exists to
// kill: TokenPairIndex used to be a sync.OnceValue built once against
// whichever datastore the process loaded first, so a second Backend built
// later in the same process silently answered with the first one's
// pairings. Two hand-built backends here derive the SAME Bear (same uuid,
// same scryfall id) into two DIFFERENT pairings - b1's Bear pairs with
// Food, b2's with Ogre - and each backend's own TokenPairIndex, and
// MatchTokenPairing run against it, must answer only its own pairing,
// never the other backend's.
func TestTokenPairIndexIsPerBackend(t *testing.T) {
	b1 := fixtureBackendWithPairing("food-uuid", "Food", "1001")
	b2 := fixtureBackendWithPairing("ogre-uuid", "Ogre", "2002")

	if got := b1.TokenPairIndex["bear-uuid"]["food"]; got != "1001" {
		t.Fatalf(`b1.TokenPairIndex["bear-uuid"]["food"] = %q, want "1001"`, got)
	}
	if got := b2.TokenPairIndex["bear-uuid"]["ogre"]; got != "2002" {
		t.Fatalf(`b2.TokenPairIndex["bear-uuid"]["ogre"] = %q, want "2002"`, got)
	}
	if _, found := b1.TokenPairIndex["bear-uuid"]["ogre"]; found {
		t.Error("b1.TokenPairIndex carries b2's Ogre pairing, want isolation")
	}
	if _, found := b2.TokenPairIndex["bear-uuid"]["food"]; found {
		t.Error("b2.TokenPairIndex carries b1's Food pairing, want isolation")
	}

	if got := MatchTokenPairing(b1, "bear-scryfall", "Bear Token // Food Token", false); got != "1001" {
		t.Errorf("MatchTokenPairing(b1, ..Food..) = %q, want %q", got, "1001")
	}
	if got := MatchTokenPairing(b1, "bear-scryfall", "Bear Token // Ogre Token", false); got != "" {
		t.Errorf(`MatchTokenPairing(b1, ..Ogre..) = %q, want "": b1 never derived that pairing`, got)
	}
	if got := MatchTokenPairing(b2, "bear-scryfall", "Bear Token // Ogre Token", false); got != "2002" {
		t.Errorf("MatchTokenPairing(b2, ..Ogre..) = %q, want %q", got, "2002")
	}
	if got := MatchTokenPairing(b2, "bear-scryfall", "Bear Token // Food Token", false); got != "" {
		t.Errorf(`MatchTokenPairing(b2, ..Food..) = %q, want "": b2 never derived that pairing`, got)
	}
}
