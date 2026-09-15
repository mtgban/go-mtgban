package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// derivedUUIDs finds every uuid the loader minted rather than read from the
// game's own data, the same way any other consumer would: by the marker
// every derived card carries. There is no dedicated index for this - one
// fact (Identifiers["derivedTokenPair"]), one place to read it.
func derivedUUIDs(b *mtgmatcher.Backend) []string {
	var uuids []string
	for uuid, co := range b.UUIDs {
		if co.Identifiers["derivedTokenPair"] == "true" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids
}

// TestDerivedTokenPairsDoNotTouchTheNameIndex pins the structural guarantee
// the whole design rests on: a derived entity is real (GetUUID answers for
// it) but invisible to everything that resolves a listing by name. A
// combined token name is not unique inside its own set - a single-faced
// token commonly pairs with several different partners on the same sheet -
// so letting one leak into Hashes, CanonicalNames or a set's own Cards/
// Tokens would let an ordinary listing alias against the wrong pairing.
func TestDerivedTokenPairsDoNotTouchTheNameIndex(t *testing.T) {
	realDatastore(t)

	derived := derivedUUIDs(testBackend)
	if len(derived) == 0 {
		t.Skip("no derived token pairs in this datastore")
	}

	allUUIDs := make(map[string]bool, len(testBackend.AllUUIDs))
	for _, uuid := range testBackend.AllUUIDs {
		allUUIDs[uuid] = true
	}
	for _, uuid := range derived {
		if allUUIDs[uuid] {
			t.Errorf("derived uuid %s is in AllUUIDs, want kept out", uuid)
		}
	}

	for _, uuid := range derived {
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Fatalf("GetUUID(%s) = %v, want a derived entity", uuid, err)
		}
		// CanonicalNames may already hold this normalized key for an
		// unrelated real card - Normalize strips "//" the same way it
		// strips a space, so "Zombie // Ogre" normalizes identically to
		// AFR's own "Zombie Ogre". That collision is harmless and expected;
		// what must never happen is the derived uuid being *reachable*
		// through it, which Hashes below is the actual guarantee for.
		norm := mtgmatcher.Normalize(co.Card.Name)
		for _, id := range testBackend.Hashes[norm] {
			if id == uuid {
				t.Errorf("Hashes[%q] contains the derived uuid %s, want it absent", norm, uuid)
			}
		}
		set, found := testBackend.Sets[co.SetCode]
		if !found {
			continue
		}
		for _, card := range set.Cards {
			if card.UUID == uuid {
				t.Errorf("%s's Cards contains derived uuid %s, want it absent", co.SetCode, uuid)
			}
		}
		for _, card := range set.Tokens {
			if card.UUID == uuid {
				t.Errorf("%s's Tokens contains derived uuid %s, want it absent", co.SetCode, uuid)
			}
		}
	}
}

// TestDerivedTokenPairResolvesByProductID pins the happy path against real
// TCGplayer products, one of each shape this feature has to get right.
func TestDerivedTokenPairResolvesByProductID(t *testing.T) {
	realDatastore(t)

	for _, probe := range []struct {
		desc    string
		tcgID   string
		name    string
		setCode string
		number  string
	}{
		{"same-set pairing", "278823", "Eldrazi Scion // Boar", "T2X2", "1 // 15"},
		{"cross-set pairing (AFC + AFR)", "244277", "Illusion // Skeleton", "TAFR", "3 // 6"},
		{"multi-id pairing, first id", "200319", "Goat // Food", "TELD", "1 // 16"},
		{"multi-id pairing, second id", "200320", "Goat // Food", "TELD", "1 // 16"},
	} {
		uuid := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, probe.tcgID)
		if uuid == "" {
			t.Errorf("%s: ConvertID(%s) = \"\", want a derived uuid (data may have drifted - re-check the id is still a live pairing)", probe.desc, probe.tcgID)
			continue
		}
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Errorf("%s: GetUUID(%s) = %v", probe.desc, uuid, err)
			continue
		}
		if co.Card.Name != probe.name || co.SetCode != probe.setCode || co.Number != probe.number {
			t.Errorf("%s: id %s = %s|%s|#%s, want %s|%s|#%s",
				probe.desc, probe.tcgID, co.Card.Name, co.SetCode, co.Number, probe.name, probe.setCode, probe.number)
		}
		if co.Identifiers["derivedTokenPair"] != "true" {
			t.Errorf("%s: Identifiers[derivedTokenPair] = %q, want \"true\"", probe.desc, co.Identifiers["derivedTokenPair"])
		}
	}

	// Both ids of the multi-id pair must land on the very same uuid, not
	// two different entities for one physical pairing.
	u1 := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, "200319")
	u2 := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, "200320")
	if u1 == "" || u1 != u2 {
		t.Errorf("ConvertID(200319) = %s, ConvertID(200320) = %s, want equal and non-empty", u1, u2)
	}

	id, err := mtgmatcher.Match(&mtgmatcher.InputCard{ID: "278823"})
	if err != nil {
		t.Fatalf("Match(id=278823) = %v", err)
	}
	if co, _ := mtgmatcher.GetUUID(id); co.Card.Name != "Eldrazi Scion // Boar" {
		t.Errorf("Match(id=278823) = %s, want Eldrazi Scion // Boar", co.Card.Name)
	}
}

// TestDerivedTokenPairExclusions pins two of the exclusion ladder's rungs
// against real data: an id a real printing already owns must keep pointing
// at that printing, never get shadowed by a derived entity.
func TestDerivedTokenPairExclusions(t *testing.T) {
	realDatastore(t)

	// C14's own Angel #1 carries 94180 as its own tcgplayerProductId; a
	// pairing that also names 94180 must lose that id entirely rather than
	// steal it.
	uuid := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, "94180")
	if uuid == "" {
		t.Skip("94180 not present in this datastore")
	}
	co, err := mtgmatcher.GetUUID(uuid)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", uuid, err)
	}
	if co.Identifiers["derivedTokenPair"] == "true" {
		t.Error("id 94180 resolves to a derived entity, want the real C14 Angel it already named")
	}
	if co.Card.Name != "Angel" || co.SetCode != "TC14" {
		t.Errorf("id 94180 = %s|%s, want Angel|TC14", co.Card.Name, co.SetCode)
	}
}

// TestDerivedTokenPairsAreNotNameMatchable is the safety half of the
// structural test above, exercised through the public Match path a
// scraper actually uses rather than the index directly.
func TestDerivedTokenPairsAreNotNameMatchable(t *testing.T) {
	realDatastore(t)

	// Derived names are never in CanonicalNames by design (that is what
	// this test proves), so existence there can't gate the skip; check the
	// pairing is actually derived by asking for it by its own product id.
	if uuid := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, "278823"); uuid == "" {
		t.Skip("Eldrazi Scion // Boar (id 278823) not derived in this datastore")
	}

	_, err := mtgmatcher.Match(&mtgmatcher.InputCard{
		Name:    "Eldrazi Scion // Boar",
		Edition: "Double Masters 2022 Tokens",
	})
	if err == nil {
		t.Error("Match(\"Eldrazi Scion // Boar\") succeeded, want an error: the combined name is not unique in its own set")
	}

	// The ordinary single-faced token, unaffected by its own pairings.
	id, err := mtgmatcher.Match(&mtgmatcher.InputCard{
		Name:    "Eldrazi Scion",
		Edition: "Double Masters 2022 Tokens",
	})
	if err != nil {
		t.Fatalf("Match(\"Eldrazi Scion\") = %v", err)
	}
	co, _ := mtgmatcher.GetUUID(id)
	if co.Card.Name != "Eldrazi Scion" || co.Identifiers["derivedTokenPair"] == "true" {
		t.Errorf("Match(\"Eldrazi Scion\") = %s (derived=%v), want the real single-faced token",
			co.Card.Name, co.Identifiers["derivedTokenPair"] == "true")
	}

	// TCMM prints many Treasure // X pairings; MatchInSet must still answer
	// with only the one real single-faced Treasure token, never any of them.
	if got := len(mtgmatcher.MatchInSet("Treasure", "TCMM")); got != 1 {
		t.Errorf("len(MatchInSet(\"Treasure\", \"TCMM\")) = %d, want 1 (a derived pairing leaked in)", got)
	}
}

// TestNormalizeTokenFace pins the pure string logic directly, independent
// of any vendor package: the artist parenthetical must come off before the
// " Token" suffix trim runs, or "X Token (Artist)" never has the suffix
// found at all (it only ever finds " Token" at the very end of the
// string). No datastore needed - this is a string transform, not a lookup.
func TestNormalizeTokenFace(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
	}{
		{"Angel Token", "angel"},
		{"Angel", "angel"},
		{"Eldrazi Spawn Token (Briclot)", "eldrazi spawn"},
		{"Cat Warrior Token", "cat warrior"},
		{"  Bear  Token  ", "bear"},
	} {
		if got := NormalizeTokenFace(tt.name); got != tt.want {
			t.Errorf("NormalizeTokenFace(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestSplitTokenPairName pins the separator-preference logic directly: "//"
// is tried before "-", so a listing carrying both never has its faces split
// at the wrong one. No datastore needed.
func TestSplitTokenPairName(t *testing.T) {
	for _, tt := range []struct {
		name       string
		wantFirst  string
		wantSecond string
	}{
		{"Angel Token // Drake Token", "Angel Token", "Drake Token"},
		{"Angel Token - Cat Token", "Angel Token", "Cat Token"},
		{"Angel - Drake Token // Cat Token", "Angel - Drake Token", "Cat Token"},
		{"Plain Card Name", "Plain Card Name", ""},
	} {
		first, second := SplitTokenPairName(tt.name)
		if first != tt.wantFirst || second != tt.wantSecond {
			t.Errorf("SplitTokenPairName(%q) = (%q, %q), want (%q, %q)",
				tt.name, first, second, tt.wantFirst, tt.wantSecond)
		}
	}
}

// TestMatchTokenPairingAnchorsEitherHalf pins MatchTokenPairing's own
// anchor-swap logic directly, independent of any vendor package's sku
// parsing: a vendor's scryfall_id usually names the FIRST half of its own
// listing name, but not always - "Cat Token - Cat Warrior Token" carries
// the id against the Cat Warrior face, so the Cat half is the partner to
// look up, the opposite of what naming order alone would suggest.
// cardkingdom's own TestPreprocessTokenPairingSecondHalfAnchored pins the
// same real pairing end to end through Preprocess(); this pins the
// matcher's own behavior directly.
func TestMatchTokenPairingAnchorsEitherHalf(t *testing.T) {
	realDatastore(t)

	const scryfallID = "29c4e4f2-0040-4490-b357-660d729ad9cc"
	const wantUUID = "7a13db1f-523c-5b19-80e5-d4d6f0121c6b_tp_7e5dc858-2163-5de0-95cb-f0e2933a7f7f"
	if mtgmatcher.ConvertID(mtgmatcher.IDSpaceScryfall, scryfallID) == "" {
		t.Skip("Cat (C17) scryfallId not present in this datastore")
	}

	tcgID := MatchTokenPairing(scryfallID, "Cat Token - Cat Warrior Token")
	if tcgID == "" {
		t.Fatal("MatchTokenPairing(Cat, ..Cat Warrior..) = \"\", want the derived Cat // Cat Warrior pairing")
	}
	if uuid := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, tcgID); uuid != wantUUID {
		t.Errorf("MatchTokenPairing(Cat, ..Cat Warrior..) = %s (%s), want %s", tcgID, uuid, wantUUID)
	}
}

// TestMatchTokenPairingBySetNumber pins the sku-anchored fallback directly,
// independent of any vendor package: MatchTokenPairingBySetNumber anchors
// the FIRST face by its own filing set and number (rather than a
// scryfall_id) and tries both a suffix-kept and a suffix-stripped form of
// that face's own name against it, since most token Card.Names drop the
// " Token" suffix a vendor spells but the wording alone can't say whether
// this one does. Reuses the same AFC/AFR cross-set pairing
// TestDerivedTokenPairResolvesByProductID already pins as ground truth -
// the derived entity's own SetCode is the pairing's home set (TAFR), but
// Illusion, the face being anchored here, is actually filed under TAFC.
func TestMatchTokenPairingBySetNumber(t *testing.T) {
	realDatastore(t)

	const wantTCGID = "244277"
	if mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, wantTCGID) == "" {
		t.Skip("Illusion // Skeleton (id 244277) not derived in this datastore")
	}

	tcgID := MatchTokenPairingBySetNumber("TAFC", "3", "Illusion Token // Skeleton Token")
	if tcgID != wantTCGID {
		t.Errorf("MatchTokenPairingBySetNumber(TAFC, 3, ..Skeleton..) = %q, want %q", tcgID, wantTCGID)
	}
}
