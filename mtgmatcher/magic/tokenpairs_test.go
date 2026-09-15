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
		{"sibling-set duplicate id (AFR/OAFR dungeon)", "242785", "Goblin // Dungeon of the Mad Mage", "TAFR", "12 // 20"},
	} {
		uuid := testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, probe.tcgID)
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
	u1 := testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, "200319")
	u2 := testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, "200320")
	if u1 == "" || u1 != u2 {
		t.Errorf("ConvertID(200319) = %s, ConvertID(200320) = %s, want equal and non-empty", u1, u2)
	}

	id, err := testBackend.Match(&mtgmatcher.InputCard{ID: "278823"})
	if err != nil {
		t.Fatalf("Match(id=278823) = %v", err)
	}
	if co, _ := testBackend.GetUUID(id); co.Card.Name != "Eldrazi Scion // Boar" {
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
	uuid := testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, "94180")
	if uuid == "" {
		t.Skip("94180 not present in this datastore")
	}
	co, err := testBackend.GetUUID(uuid)
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

// TestDerivedTokenPairsSurviveSiblingSetDuplicateIDs pins idCanonicalKey
// directly: OAFR (Forgotten Realms Oversized Cards, a memorabilia sibling
// of AFR with ParentCode "AFR") repeats AFR's own dungeon-card ids under
// its own uuids, so a naive "an id claimed by more than one pairing
// answers for neither" refusal would leave every AFR dungeon pairing
// unresolved even though CK and SCG both sell them. All three dungeon
// cards, each paired with both a token and with each other, must resolve -
// and, critically, must resolve to the ordinary AFR/TAFR uuid as the
// winning tokenPairPartA/B, not OAFR's: TokenPairIndex only ever indexes
// whichever uuid actually won, so picking the memorabilia sibling here
// would silently leave a real vendor listing - anchored on the ordinary
// set's own scryfall_id, the same one CK and SCG both publish - unable to
// find the pairing at all even though a derived entity for it exists. The
// first pass of this fix got exactly that wrong (picked whichever uuid
// sorted first, which happened to be OAFR's); MatchTokenPairing against
// CK's own real scryfall_id is what actually catches it.
func TestDerivedTokenPairsSurviveSiblingSetDuplicateIDs(t *testing.T) {
	realDatastore(t)

	for _, probe := range []struct {
		desc       string
		tcgID      string
		name       string
		scryfallID string
		listing    string
	}{
		{"dungeon // token, id shared by AFR and OAFR's own Dungeon of the Mad Mage", "242785", "Goblin // Dungeon of the Mad Mage", "6f509dbe-6ec7-4438-ab36-e20be46c9922", "Dungeon of the Mad Mage // Goblin Token"},
		{"dungeon // token, id shared by AFR and OAFR's own Lost Mine of Phandelver", "242783", "Skeleton // Lost Mine of Phandelver", "59b11ff8-f118-4978-87dd-509dc0c8c932", "Lost Mine of Phandelver // Skeleton Token"},
		{"dungeon // token, id shared by AFR and OAFR's own Tomb of Annihilation", "242784", "The Atropal // Tomb of Annihilation", "70b284bd-7a8f-4b60-8238-f746bdc5b236", "Tomb of Annihilation // The Atropal"},
	} {
		uuid := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, probe.tcgID)
		if uuid == "" {
			t.Skip("AFR dungeon pairing not present in this datastore, cannot verify")
		}
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			t.Fatalf("%s: GetUUID(%s) = %v", probe.desc, uuid, err)
		}
		if co.Card.Name != probe.name {
			t.Errorf("%s: id %s = %q, want %q", probe.desc, probe.tcgID, co.Card.Name, probe.name)
		}
		if co.SetCode != "TAFR" {
			t.Errorf("%s: id %s resolved to set %q, want the ordinary TAFR filing, not the OAFR memorabilia sibling", probe.desc, probe.tcgID, co.SetCode)
		}

		// The real regression: a vendor's own scryfall_id for the dungeon
		// face (always the ordinary AFR printing in practice) must find
		// this pairing through MatchTokenPairing, the same path
		// cardkingdom and starcitygames actually call.
		if got := MatchTokenPairing(probe.scryfallID, probe.listing, false); got != probe.tcgID {
			t.Errorf("%s: MatchTokenPairing(%s, %q) = %q, want %q", probe.desc, probe.scryfallID, probe.listing, got, probe.tcgID)
		}
	}

	for _, probe := range []struct {
		desc  string
		tcgID string
		name  string
	}{
		{"dungeon // dungeon, id shared 3 ways across AFR/OAFR combinations", "244297", "Dungeon of the Mad Mage // Lost Mine of Phandelver"},
		{"dungeon // dungeon, id shared 3 ways across AFR/OAFR combinations", "247304", "Dungeon of the Mad Mage // Tomb of Annihilation"},
	} {
		uuid := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, probe.tcgID)
		if uuid == "" {
			t.Skip("AFR dungeon pairing not present in this datastore, cannot verify")
		}
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			t.Fatalf("%s: GetUUID(%s) = %v", probe.desc, uuid, err)
		}
		if co.Card.Name != probe.name {
			t.Errorf("%s: id %s = %q, want %q", probe.desc, probe.tcgID, co.Card.Name, probe.name)
		}
		if co.SetCode != "TAFR" {
			t.Errorf("%s: id %s resolved to set %q, want the ordinary TAFR filing, not the OAFR memorabilia sibling", probe.desc, probe.tcgID, co.SetCode)
		}
	}

	// The one real collision this measurement found (A25's Fish/Kraken
	// token pair sharing an id with the unrelated, already-modeled "Fish
	// // Kraken" double_faced_token foil/nonfoil twins) must stay refused:
	// its claimants don't agree on face names, so it is genuinely
	// ambiguous, not a sibling-set duplicate.
	if uuid := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, "162899"); uuid != "" {
		if co, err := mtgmatcher.GetUUID(uuid); err == nil && co.Identifiers["derivedTokenPair"] == "true" {
			t.Errorf("id 162899 resolved to a derived Fish/Kraken pairing (%s), want it to stay refused: it also names the unrelated already-modeled \"Fish // Kraken\" double_faced_token", co.Card.Name)
		}
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
	if uuid := testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, "278823"); uuid == "" {
		t.Skip("Eldrazi Scion // Boar (id 278823) not derived in this datastore")
	}

	_, err := testBackend.Match(&mtgmatcher.InputCard{
		Name:    "Eldrazi Scion // Boar",
		Edition: "Double Masters 2022 Tokens",
	})
	if err == nil {
		t.Error("Match(\"Eldrazi Scion // Boar\") succeeded, want an error: the combined name is not unique in its own set")
	}

	// The ordinary single-faced token, unaffected by its own pairings.
	id, err := testBackend.Match(&mtgmatcher.InputCard{
		Name:    "Eldrazi Scion",
		Edition: "Double Masters 2022 Tokens",
	})
	if err != nil {
		t.Fatalf("Match(\"Eldrazi Scion\") = %v", err)
	}
	co, _ := testBackend.GetUUID(id)
	if co.Card.Name != "Eldrazi Scion" || co.Identifiers["derivedTokenPair"] == "true" {
		t.Errorf("Match(\"Eldrazi Scion\") = %s (derived=%v), want the real single-faced token",
			co.Card.Name, co.Identifiers["derivedTokenPair"] == "true")
	}

	// TCMM prints many Treasure // X pairings; MatchInSet must still answer
	// with only the one real single-faced Treasure token, never any of them.
	if got := len(testBackend.MatchInSet("Treasure", "TCMM")); got != 1 {
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
	if testBackend.ConvertID(mtgmatcher.IDSpaceScryfall, scryfallID) == "" {
		t.Skip("Cat (C17) scryfallId not present in this datastore")
	}

	tcgID := MatchTokenPairing(scryfallID, "Cat Token - Cat Warrior Token", false)
	if tcgID == "" {
		t.Fatal("MatchTokenPairing(Cat, ..Cat Warrior..) = \"\", want the derived Cat // Cat Warrior pairing")
	}
	if uuid := testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, tcgID); uuid != wantUUID {
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
	if testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, wantTCGID) == "" {
		t.Skip("Illusion // Skeleton (id 244277) not derived in this datastore")
	}

	tcgID := MatchTokenPairingBySetNumber("TAFC", "3", "Illusion Token // Skeleton Token", false)
	if tcgID != wantTCGID {
		t.Errorf("MatchTokenPairingBySetNumber(TAFC, 3, ..Skeleton..) = %q, want %q", tcgID, wantTCGID)
	}
}

// TestTokenPairIndexCollision pins the fix for a real bug: a face commonly
// pairs with several different partners across a sheet, and two of those
// partners can normalize to the identical key (measured: 252 of 1,865
// faces in today's datastore carry at least one such collision). "Bear"
// pairs with four differently-numbered "Food" tokens across Throne of
// Eldraine's own token sheets, all colliding on "food" - a plain
// last-write-wins map would silently pick one and make the other three
// unreachable, so a vendor listing that actually names one of the dropped
// three would resolve to the wrong physical product under the survivor's
// id. The fix must refuse rather than guess: TokenPairIndex itself carries
// no entry for the colliding key, and MatchTokenPairing (the caller every
// vendor package goes through) returns "" for it - never silently
// answering with one of the four candidates.
func TestTokenPairIndexCollision(t *testing.T) {
	realDatastore(t)

	bearScryfallID := "b0f09f9e-e0f9-4ed8-bfc0-5f1a3046106e"
	bearUUID := testBackend.ConvertID(mtgmatcher.IDSpaceScryfall, bearScryfallID)
	if bearUUID == "" {
		t.Skip("Bear (TELD) scryfallId not present in this datastore")
	}

	idx := TokenPairIndex()
	if id, found := idx[bearUUID]["food"]; found {
		t.Errorf(`TokenPairIndex[Bear]["food"] = %q, want no entry (colliding key must stay unresolved, not answer with an arbitrary one of Bear's several Food partners)`, id)
	}

	if id := MatchTokenPairing(bearScryfallID, "Bear Token // Food Token", false); id != "" {
		t.Errorf("MatchTokenPairing(Bear, ..Food..) = %q, want \"\": Bear pairs with multiple differently-numbered Food tokens, none namable from \"Food\" alone", id)
	}
}

// TestMatchTokenPairingRequiresBothFacesInRequestedFinish pins the fix for a
// second real bug: the derived pairing's own Finishes is deliberately the
// UNION of both faces' independent finish lists (unionFinishes, above - a
// fine tradeoff for its own original purpose, keeping
// mtgmatcher.MatchIDFinish from erroring on a finish only one face happens
// to carry). Trusting that union to answer "was this specific two-sided
// PRODUCT sold in this finish" is a different question the union was never
// built to answer: Boar was never sold foil on its own, its TKHM sheet
// partner Spirit was, and the union claims foil regardless. A foil request
// anchored on Boar's own scryfall_id must be refused, not silently answered
// with Spirit's foil-ness - the same "don't know, refuse" discipline used
// everywhere else two-sided token matching cannot verify a vendor's own
// claim.
func TestMatchTokenPairingRequiresBothFacesInRequestedFinish(t *testing.T) {
	realDatastore(t)

	boarScryfallID := "8ef6aca1-2e66-48fa-a446-6ec052b1e596"
	if testBackend.ConvertID(mtgmatcher.IDSpaceScryfall, boarScryfallID) == "" {
		t.Skip("Boar (TKHM) scryfallId not present in this datastore")
	}

	if id := MatchTokenPairing(boarScryfallID, "Boar Token // Spirit Token", true); id != "" {
		t.Errorf("MatchTokenPairing(Boar, ..Spirit.., foil=true) = %q, want \"\": Boar itself was never sold foil, only its sheet partner Spirit was", id)
	}
	if id := MatchTokenPairing(boarScryfallID, "Boar Token // Spirit Token", false); id == "" {
		t.Error("MatchTokenPairing(Boar, ..Spirit.., foil=false) = \"\", want the real nonfoil pairing id")
	}
}

// TestNormalizeTokenFaceStripsBraceWrapping pins the brace-stripping
// generalization directly, independent of any vendor package: SCG wraps
// every face name in its own "{curly braces}", which must come off (and
// still leave the artist parenthetical and " Token" suffix handled
// correctly) for its wording to compare equal against the datastore's own
// unwrapped names. No datastore needed - pure string logic.
func TestNormalizeTokenFaceStripsBraceWrapping(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
	}{
		{"{Angel Token}", "angel"},
		{"{Eldrazi Spawn Token (Briclot)}", "eldrazi spawn"},
		{"Angel Token", "angel"}, // unaffected where a vendor never wraps
	} {
		if got := NormalizeTokenFace(tt.name); got != tt.want {
			t.Errorf("NormalizeTokenFace(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestMatchNativeTokenPair pins the native-combined-printing path directly,
// independent of any vendor package: mtgjson sometimes files a two-sided
// token sheet as one ordinary printing of its own under a combined "X // Y"
// name (Guild Kit's "Copy // Horror" at TGK1 #1) rather than as a derived
// pairing, and a vendor's own listing order does not always agree with
// mtgjson's, so both face orders must resolve to the same printing.
func TestMatchNativeTokenPair(t *testing.T) {
	realDatastore(t)

	if len(testBackend.MatchInSetNumber("Copy // Horror", "TGK1", "1")) != 1 {
		t.Skip("Copy // Horror not present at TGK1 #1 in this datastore")
	}

	for _, listing := range []string{
		"{Copy Token} // {Horror Token}",
		"{Horror Token} // {Copy Token}",
	} {
		uuid := MatchNativeTokenPair("TGK1", "1", listing)
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Fatalf("MatchNativeTokenPair(TGK1, 1, %q) = %q, GetUUID: %v", listing, uuid, err)
		}
		if co.Card.Name != "Copy // Horror" || co.SetCode != "TGK1" || co.Number != "1" {
			t.Errorf("MatchNativeTokenPair(TGK1, 1, %q) = %s #%s [%s], want Copy // Horror #1 [TGK1]",
				listing, co.Card.Name, co.Number, co.SetCode)
		}
		if co.Identifiers["derivedTokenPair"] == "true" {
			t.Errorf("MatchNativeTokenPair(TGK1, 1, %q) resolved to a synthetic derived pairing, want mtgjson's own native combined printing", listing)
		}
	}
}

// TestMatchTokenPairingByNamesAndEdition pins the mechanism a listing with
// neither an id nor a filing set/number resolves through: both faces'
// names alone, guarded by requiring the vendor's own claimed edition to
// independently agree with the match (see the function's own doc comment
// for why - a generic pairing name recurs across more than one set's own
// token sheet, and without this guard whichever one the datastore
// currently derives would win regardless of which set the listing
// actually names).
func TestMatchTokenPairingByNamesAndEdition(t *testing.T) {
	realDatastore(t)

	tcgID := MatchTokenPairingByNamesAndEdition("Cat Warrior // Beast", "Commander 2018", false)
	if tcgID == "" {
		t.Fatal("MatchTokenPairingByNamesAndEdition(Cat Warrior // Beast, Commander 2018) = \"\", want a match")
	}
	uuid := testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, tcgID)
	co, err := testBackend.GetUUID(uuid)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", uuid, err)
	}
	if co.SetCode != "TC18" {
		t.Errorf("resolved to set %s, want TC18", co.SetCode)
	}

	// Bird // Myr is a real derived pairing (Modern Horizons' own token
	// sheet), but Commander 2016 never sold that exact pairing - its own
	// Bird // Myr partners are different tokens on TC16's own sheet. The
	// two face names alone would resolve to Modern Horizons' pairing
	// regardless; the edition check must catch the disagreement.
	if got := MatchTokenPairingByNamesAndEdition("Bird // Myr", "Commander 2016", false); got != "" {
		t.Errorf("MatchTokenPairingByNamesAndEdition(Bird // Myr, Commander 2016) = %q, want \"\": the listing's own edition disagrees with the name-only match", got)
	}
}

// TestTokenPairIDByBothNamesCollision pins the same collision-blanking
// discipline byFace already has, one level down: "Knight" and "Zombie"
// pair with each other on more than one set's own token sheet, so the
// unordered name pair alone must stay unresolved rather than answering
// with an arbitrary one of them.
func TestTokenPairIDByBothNamesCollision(t *testing.T) {
	realDatastore(t)

	key := [2]string{NormalizeTokenFace("Knight"), NormalizeTokenFace("Zombie")}
	if id, found := TokenPairIDByBothNames()[key]; found {
		t.Errorf(`TokenPairIDByBothNames[Knight,Zombie] = %q, want no entry (colliding key must stay unresolved)`, id)
	}
}

// TestMintVerifiedPairsResolves pins a real vendorVerifiedPair entity -
// no tcgplayerProductId at all - resolving through the same
// MatchTokenPairingByUUIDs a caller that has already anchored both faces
// by identity already uses for an ordinary derived pairing. Angel (TAVR)
// and Demon (TAVR) have no mtgjson tokenProducts entry linking them (Star
// City Games's own composite sku is the only thing that confirms this
// pairing is real), so if buildDerivedCard's nil-usableIDs handling or
// tokenPairingFinishOK's bare-uuid resolution ever regresses, this fails.
func TestMintVerifiedPairsResolves(t *testing.T) {
	realDatastore(t)

	angel := testBackend.MatchInSetNumber("Angel", "TAVR", "1")
	demon := testBackend.MatchInSetNumber("Demon", "TAVR", "5")
	if len(angel) != 1 || len(demon) != 1 {
		t.Skip("Angel/Demon TAVR #1/#5 not present in this datastore")
	}

	id := MatchTokenPairingByUUIDs(angel[0].UUID, demon[0].UUID, false)
	if id == "" {
		t.Fatal("MatchTokenPairingByUUIDs(Angel, Demon) = \"\", want the vendorVerifiedPair entity")
	}
	uuid := testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, id)
	if uuid != "" {
		t.Fatalf("id %q resolved through IDSpaceTCGplayer, want a bare uuid (no real tcgplayerProductId for a vendorVerifiedPair entity)", id)
	}
	co, err := testBackend.GetUUID(id)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", id, err)
	}
	if co.Identifiers["vendorVerifiedPair"] != "true" {
		t.Errorf("resolved to %s, want Identifiers[vendorVerifiedPair] = true", co.Name)
	}
	if _, found := co.Identifiers["tcgplayerProductId"]; found {
		t.Errorf("vendorVerifiedPair entity %s carries a tcgplayerProductId, want none", co.Name)
	}
}

// TestVerifiedPairCollisionRefusesRatherThanGuess pins the reason
// verifiedNoUpstreamPairs' own doc comment gives for why adding entries
// can correctly reduce some other resolution's confidence: Final Fantasy
// really does file its own native "Wizard // Bird" token pairing, but a
// second, different Bird/Wizard pairing this table also confirms is real
// shares the identical normalized name pair - a listing with neither an
// id nor a set/number anchor, naming only "Bird // Wizard", cannot tell
// the two apart and must refuse rather than guess.
func TestVerifiedPairCollisionRefusesRatherThanGuess(t *testing.T) {
	realDatastore(t)

	bird := testBackend.MatchInSetNumber("Bird", "TFIN", "17")
	// TFIN's own real derived pairing's own Wizard partner.
	realWizard := testBackend.MatchInSetNumber("Wizard", "TFIN", "15")
	// This table's own, different Wizard this same Bird also verifiably
	// pairs with - two real, distinct physical products.
	verifiedWizard := testBackend.MatchInSetNumber("Wizard", "TFIN", "14")
	if len(bird) != 1 || len(realWizard) != 1 || len(verifiedWizard) != 1 {
		t.Skip("Bird/Wizard TFIN #17/#20/#14 not present in this datastore")
	}
	if realWizard[0].UUID == verifiedWizard[0].UUID {
		t.Skip("TFIN's own Wizard and this table's verified Wizard are the same printing in this datastore - the collision this test pins no longer exists")
	}

	// Both faces already anchored by identity (not by name) is
	// unambiguous even between two colliding uuid pairs: a caller with
	// both uuids in hand already knows which physical pairing it means,
	// so it isn't asking the name-keyed indices anything at all.
	if id := MatchTokenPairingByUUIDs(bird[0].UUID, realWizard[0].UUID, false); id == "" {
		t.Error("MatchTokenPairingByUUIDs(TFIN Bird, TFIN's own Wizard) = \"\", want a match: this exact uuid pair is unambiguous regardless of what else Bird's name collides with")
	}
	if id := MatchTokenPairingByUUIDs(bird[0].UUID, verifiedWizard[0].UUID, false); id == "" {
		t.Error("MatchTokenPairingByUUIDs(TFIN Bird, this table's verified Wizard) = \"\", want a match: this exact uuid pair is unambiguous too")
	}

	// The real pin: a caller with only Bird anchored and the bare name
	// "Wizard" to go on (TokenPairIndex/byFace, what
	// MatchTokenPairingBySetNumber actually asks) cannot tell TFIN's own
	// Wizard from this table's different one, and must refuse rather
	// than pick either arbitrarily.
	if id, found := TokenPairIndex()[bird[0].UUID]["wizard"]; found {
		t.Errorf("TokenPairIndex[Bird][wizard] = %q, want no entry: TFIN's own Bird pairs with more than one real Wizard across this table plus mtgjson's own tokenProducts, and the name alone cannot tell them apart", id)
	}

	// The same collision, for a caller with neither face anchored at all
	// (TokenPairIDByBothNames, what MatchTokenPairingByNamesAndEdition
	// asks before its own edition check ever runs).
	key := [2]string{NormalizeTokenFace("Bird"), NormalizeTokenFace("Wizard")}
	if key[1] < key[0] {
		key = [2]string{key[1], key[0]}
	}
	if id, found := TokenPairIDByBothNames()[key]; found {
		t.Errorf("TokenPairIDByBothNames[Bird,Wizard] = %q, want no entry", id)
	}
}
