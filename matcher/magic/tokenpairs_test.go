package magic

import (
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// derivedUUIDs finds every uuid the loader minted rather than read from the
// game's own data, the same way any other consumer would: by the marker
// every derived card carries. There is no dedicated index for this - one
// fact (Identifiers["derivedTokenPair"]), one place to read it.
func derivedUUIDs(b *matcher.Backend) []string {
	var uuids []string
	for uuid, co := range b.UUIDs {
		if co.Identifiers["derivedTokenPair"] == "true" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids
}

// TestDerivedTokenPairsAreSearchedNotMatched pins what a derived entity is
// to a name: a search finds it by its own name, either face first, and by
// each face's, while Match never answers with one. A combined token name is
// not unique inside its own set - a single-faced token commonly pairs with
// several partners on the same sheet - so a listing matched by name would
// alias against the wrong pairing.
func TestDerivedTokenPairsAreSearchedNotMatched(t *testing.T) {
	realDatastore(t)

	derived := derivedUUIDs(testBackend)
	if len(derived) == 0 {
		t.Fatal("no derived token pairs in this datastore")
	}

	for _, uuid := range derived {
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Fatalf("GetUUID(%s) = %v, want a derived entity", uuid, err)
		}
		if !slices.Contains(testBackend.GetUUIDsInSet(co.SetCode), uuid) {
			t.Errorf("%s is not among the uuids of %s", uuid, co.SetCode)
		}
		first, second, _ := strings.Cut(co.Name, " // ")
		for _, name := range []string{co.Name, second + " // " + first, first, second} {
			found, _ := testBackend.SearchEquals(name)
			if !slices.Contains(found, uuid) {
				t.Errorf("SearchEquals(%q) does not find %s", name, uuid)
			}
		}
		for _, name := range []string{co.Name, first} {
			id, err := testBackend.Match(&matcher.InputCard{Name: name, Edition: co.Edition, Foil: co.Foil})
			if err == nil && slices.Contains(derived, id) {
				t.Errorf("Match(%q in %q) = %s, a derived pairing", name, co.Edition, id)
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
		{"same-set pairing", "278823", "Eldrazi Scion // Boar", "T2X2", "1//15"},
		{"cross-set pairing (AFC + AFR)", "244277", "Illusion // Skeleton", "TAFR", "3//6"},
		{"multi-id pairing, first id", "200319", "Goat // Food", "TELD", "1//16"},
		{"multi-id pairing, second id", "200320", "Goat // Food", "TELD", "1//16"},
		{"sibling-set duplicate id (AFR/OAFR dungeon)", "242785", "Goblin // Dungeon of the Mad Mage", "TAFR", "12//20"},
	} {
		uuid := testBackend.ConvertID(matcher.IDSpaceTCGplayer, probe.tcgID)
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
	u1 := testBackend.ConvertID(matcher.IDSpaceTCGplayer, "200319")
	u2 := testBackend.ConvertID(matcher.IDSpaceTCGplayer, "200320")
	if u1 == "" || u1 != u2 {
		t.Errorf("ConvertID(200319) = %s, ConvertID(200320) = %s, want equal and non-empty", u1, u2)
	}

	id, err := testBackend.Match(&matcher.InputCard{ID: "278823"})
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

	// A25's star Fish // Kraken carries 162899 as its own
	// tcgplayerProductId; the Fish #5 // Kraken #6 pairing that also names
	// 162899 must lose that id entirely rather than steal it.
	uuid := testBackend.ConvertID(matcher.IDSpaceTCGplayer, "162899")
	if uuid == "" {
		t.Fatal("162899 not present in this datastore")
	}
	co, err := testBackend.GetUUID(uuid)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", uuid, err)
	}
	if co.Identifiers["derivedTokenPair"] == "true" {
		t.Error("id 162899 resolves to a derived entity, want the real A25 Fish // Kraken it already named")
	}
	if co.Card.Name != "Fish // Kraken" || co.Number != "5★" {
		t.Errorf("id 162899 = %s|#%s, want Fish // Kraken|#5★", co.Card.Name, co.Number)
	}
}

// TestDerivedTokenPairTakesItsFaceProduct pins a product mtgjson files on
// one face, or both, while tokenProducts names it only as the two-sided
// product: the pairing owns the id and the faces lose it. A face's own
// single-sided product stays the face's.
func TestDerivedTokenPairTakesItsFaceProduct(t *testing.T) {
	realDatastore(t)

	for _, probe := range []struct {
		tcgID  string
		foil   bool
		name   string
		number string
		faces  []string
	}{
		{"94180", false, "Angel // Cat", "1//2", []string{"20ac0df6-bb34-549c-8fa5-5d43da39e9de"}},
		{"94696", false, "Elf Warrior // Gargoyle", "24//27", []string{"07163036-315e-554a-bfc0-16028503d1be"}},
		{"533867", true, "Dog // Cat", "1516//1517", []string{"b038f379-b2fa-52ba-b9f9-b689ab164edd", "d455d547-bb4c-5999-8561-b7cbe0ac2871"}},
	} {
		id, err := testBackend.MatchID(probe.tcgID, probe.foil)
		if err != nil {
			t.Errorf("MatchID(%s, foil=%v) = %v", probe.tcgID, probe.foil, err)
			continue
		}
		co, err := testBackend.GetUUID(id)
		if err != nil {
			t.Errorf("GetUUID(%s) = %v", id, err)
			continue
		}
		if co.Identifiers["derivedTokenPair"] != "true" || co.Card.Name != probe.name || co.Number != probe.number || co.Foil != probe.foil {
			t.Errorf("id %s = %s|#%s foil=%v, want the pairing %s|#%s foil=%v", probe.tcgID, co.Card.Name, co.Number, co.Foil, probe.name, probe.number, probe.foil)
		}
		if VerifyTokenPairingFinish(testBackend, probe.tcgID, probe.foil) != probe.tcgID {
			t.Errorf("VerifyTokenPairingFinish(%s, foil=%v) refuses the pairing", probe.tcgID, probe.foil)
		}
		for _, face := range probe.faces {
			fco, err := testBackend.GetUUID(face)
			if err != nil {
				t.Errorf("GetUUID(%s) = %v", face, err)
				continue
			}
			if fco.Identifiers["tcgplayerProductId"] != "" {
				t.Errorf("face %s|#%s still carries product %s", fco.Card.Name, fco.Number, fco.Identifiers["tcgplayerProductId"])
			}
		}
	}

	// 2XM Angel #3 is sold alone as 220044, beside the pairings it is a face of
	for _, foil := range []bool{false, true} {
		id, err := testBackend.MatchID("220044", foil)
		if err != nil {
			t.Errorf("MatchID(220044, foil=%v) = %v", foil, err)
			continue
		}
		co, _ := testBackend.GetUUID(id)
		if co.Identifiers["mtgjsonId"] != "20ed6cbc-ff63-5be7-acee-c6e34d490685" || co.Foil != foil {
			t.Errorf("id 220044 foil=%v = %s|#%s, want the 2XM Angel #3 token", foil, co.Card.Name, co.Number)
		}
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
// winning tokenPairPartA/B, not OAFR's: the pairing index only ever indexes
// whichever uuid actually won, so picking the memorabilia sibling here
// would silently leave a real vendor listing - anchored on the ordinary
// set's own scryfall_id, the same one CK and SCG both publish - unable to
// find the pairing at all even though a derived entity for it exists.
// Picking whichever uuid sorts first lands on OAFR's; MatchTokenPairing
// against CK's own real scryfall_id is what catches it.
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
		uuid := testBackend.ConvertID(matcher.IDSpaceTCGplayer, probe.tcgID)
		if uuid == "" {
			t.Fatal("AFR dungeon pairing not present in this datastore, cannot verify")
		}
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Fatalf("%s: GetUUID(%s) = %v", probe.desc, uuid, err)
		}
		if co.Card.Name != probe.name {
			t.Errorf("%s: id %s = %q, want %q", probe.desc, probe.tcgID, co.Card.Name, probe.name)
		}
		if co.SetCode != "TAFR" {
			t.Errorf("%s: id %s resolved to set %q, want the ordinary TAFR filing, not the OAFR memorabilia sibling", probe.desc, probe.tcgID, co.SetCode)
		}

		// A vendor's own scryfall_id for the dungeon
		// face (always the ordinary AFR printing in practice) must find
		// this pairing through MatchTokenPairing, the same path
		// cardkingdom and starcitygames actually call.
		if got := MatchTokenPairing(testBackend, probe.scryfallID, probe.listing, false); got != probe.tcgID {
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
		uuid := testBackend.ConvertID(matcher.IDSpaceTCGplayer, probe.tcgID)
		if uuid == "" {
			t.Fatal("AFR dungeon pairing not present in this datastore, cannot verify")
		}
		co, err := testBackend.GetUUID(uuid)
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
	if uuid := testBackend.ConvertID(matcher.IDSpaceTCGplayer, "162899"); uuid != "" {
		if co, err := testBackend.GetUUID(uuid); err == nil && co.Identifiers["derivedTokenPair"] == "true" {
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
	if uuid := testBackend.ConvertID(matcher.IDSpaceTCGplayer, "278823"); uuid == "" {
		t.Fatal("Eldrazi Scion // Boar (id 278823) not derived in this datastore")
	}

	_, err := testBackend.Match(&matcher.InputCard{
		Name:    "Eldrazi Scion // Boar",
		Edition: "Double Masters 2022 Tokens",
	})
	if err == nil {
		t.Error("Match(\"Eldrazi Scion // Boar\") succeeded, want an error: the combined name is not unique in its own set")
	}

	// The ordinary single-faced token, unaffected by its own pairings.
	id, err := testBackend.Match(&matcher.InputCard{
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
	const wantUUID = "8eb8dc6d-0fe3-594e-91ad-92ffc189e4a9"
	if testBackend.ConvertID(matcher.IDSpaceScryfall, scryfallID) == "" {
		t.Fatal("Cat (C17) scryfallId not present in this datastore")
	}

	tcgID := MatchTokenPairing(testBackend, scryfallID, "Cat Token - Cat Warrior Token", false)
	if tcgID == "" {
		t.Fatal("MatchTokenPairing(Cat, ..Cat Warrior..) = \"\", want the derived Cat // Cat Warrior pairing")
	}
	if uuid := testBackend.ConvertID(matcher.IDSpaceTCGplayer, tcgID); uuid != wantUUID {
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
	if testBackend.ConvertID(matcher.IDSpaceTCGplayer, wantTCGID) == "" {
		t.Fatal("Illusion // Skeleton (id 244277) not derived in this datastore")
	}

	tcgID := MatchTokenPairingBySetNumber(testBackend, "TAFC", "3", "Illusion Token // Skeleton Token", false)
	if tcgID != wantTCGID {
		t.Errorf("MatchTokenPairingBySetNumber(TAFC, 3, ..Skeleton..) = %q, want %q", tcgID, wantTCGID)
	}
}

// TestMatchTokenPairingBySetNumberFindsContestedName pins matchNumberedFace's
// own reason for existing: the loader already merges a token-type set's own
// Tokens into its Cards, so TAFC above resolves through Cards alone and
// reading Tokens adds nothing there - but "Shapeshifter" collides with a real
// card elsewhere in the game (one of a small, known set - Ninja,
// Ornithopter, Storm Crow, Faerie Dragon, Kobolds of Kher Keep are the
// others), so the merged Cards copy is suffixed "Shapeshifter Token" to
// stay unambiguous while its own Tokens entry keeps the plain name.
// A listing spelling the suffix itself ("Shapeshifter Token // ...", the
// shape TestMatchTokenPairingBySetNumber above uses) tries "Shapeshifter
// Token" as StripFaceWrapping's own unstripped first attempt and hits the
// Cards copy directly, Tokens or not - the shape that actually needs
// Tokens is a parenthesized one ("Shapeshifter (Token) // ...", the real
// shape Cool Stuff Inc's own buylist sends, PID 262248): both
// StripFaceWrapping and CleanFaceName strip the parenthetical down to the
// plain "Shapeshifter", which Cards alone (MatchInSetNumber) never has.
func TestMatchTokenPairingBySetNumberFindsContestedName(t *testing.T) {
	realDatastore(t)

	const wantTCGID = "173808"
	if testBackend.ConvertID(matcher.IDSpaceTCGplayer, wantTCGID) == "" {
		t.Fatal("Shapeshifter // Zombie (id 173808) not derived in this datastore")
	}

	if cards := testBackend.MatchInSetNumber("Shapeshifter", "TC18", "2"); len(cards) != 0 {
		t.Fatalf("MatchInSetNumber(Shapeshifter, TC18, 2) = %d cards, want 0: this pin needs Cards alone to still miss it", len(cards))
	}

	tcgID := MatchTokenPairingBySetNumber(testBackend, "TC18", "2", "Shapeshifter (Token) // Zombie (Token)", false)
	if tcgID != wantTCGID {
		t.Errorf("MatchTokenPairingBySetNumber(TC18, 2, ..Zombie..) = %q, want %q", tcgID, wantTCGID)
	}
}

// TestTokenPairIndexCollision pins a refusal: a face commonly pairs with
// several different partners across a sheet, and two of those partners can
// normalize to the identical key (measured: 252 of 1,865 faces in today's
// datastore carry at least one such collision). "Bear" pairs with four
// differently-numbered "Food" tokens across Throne of Eldraine's own token
// sheets, all colliding on "food" - a plain last-write-wins map would silently
// pick one and make the other three unreachable, so a vendor listing that
// actually names one of the dropped three would resolve to the wrong physical
// product under the survivor's id. The index must refuse rather than guess: it
// carries no entry for the colliding key, and MatchTokenPairing (the caller
// every vendor package goes through) returns "" for it - never silently
// answering with one of the four candidates.
func TestTokenPairIndexCollision(t *testing.T) {
	realDatastore(t)

	bearScryfallID := "b0f09f9e-e0f9-4ed8-bfc0-5f1a3046106e"
	bearUUID := testBackend.ConvertID(matcher.IDSpaceScryfall, bearScryfallID)
	if bearUUID == "" {
		t.Fatal("Bear (TELD) scryfallId not present in this datastore")
	}

	if id, found := testBackend.TokenPairIndex[bearUUID]["food"]; found {
		t.Errorf(`TokenPairIndex[Bear]["food"] = %q, want no entry (colliding key must stay unresolved, not answer with an arbitrary one of Bear's several Food partners)`, id)
	}

	if id := MatchTokenPairing(testBackend, bearScryfallID, "Bear Token // Food Token", false); id != "" {
		t.Errorf("MatchTokenPairing(Bear, ..Food..) = %q, want \"\": Bear pairs with multiple differently-numbered Food tokens, none namable from \"Food\" alone", id)
	}
}

// TestMatchTokenPairingRequiresBothFacesInRequestedFinish pins a second
// refusal: the derived pairing's own Finishes is deliberately the UNION of both
// faces' independent finish lists (unionFinishes, above - a fine tradeoff for
// its own purpose, keeping matcher.MatchIDFinish from erroring on a finish
// only one face happens to carry). Trusting that union to answer "was this
// specific two-sided PRODUCT sold in this finish" is a different question the
// union was never built to answer: Boar was never sold foil on its own, its
// TKHM sheet partner Spirit was, and the union claims foil regardless. A foil
// request anchored on Boar's own scryfall_id must be refused, not silently
// answered with Spirit's foil-ness - the same "don't know, refuse" discipline
// used everywhere else two-sided token matching cannot verify a vendor's own
// claim.
func TestMatchTokenPairingRequiresBothFacesInRequestedFinish(t *testing.T) {
	realDatastore(t)

	boarScryfallID := "8ef6aca1-2e66-48fa-a446-6ec052b1e596"
	if testBackend.ConvertID(matcher.IDSpaceScryfall, boarScryfallID) == "" {
		t.Fatal("Boar (TKHM) scryfallId not present in this datastore")
	}

	if id := MatchTokenPairing(testBackend, boarScryfallID, "Boar Token // Spirit Token", true); id != "" {
		t.Errorf("MatchTokenPairing(Boar, ..Spirit.., foil=true) = %q, want \"\": Boar itself was never sold foil, only its sheet partner Spirit was", id)
	}
	if id := MatchTokenPairing(testBackend, boarScryfallID, "Boar Token // Spirit Token", false); id == "" {
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
		t.Fatal("Copy // Horror not present at TGK1 #1 in this datastore")
	}

	for _, listing := range []string{
		"{Copy Token} // {Horror Token}",
		"{Horror Token} // {Copy Token}",
	} {
		uuid := MatchNativeTokenPair(testBackend, "TGK1", "1", listing)
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

	tcgID := MatchTokenPairingByNamesAndEdition(testBackend, "Cat Warrior // Beast", "Commander 2018", false)
	if tcgID == "" {
		t.Fatal("MatchTokenPairingByNamesAndEdition(Cat Warrior // Beast, Commander 2018) = \"\", want a match")
	}
	uuid := testBackend.ConvertID(matcher.IDSpaceTCGplayer, tcgID)
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
	if got := MatchTokenPairingByNamesAndEdition(testBackend, "Bird // Myr", "Commander 2016", false); got != "" {
		t.Errorf("MatchTokenPairingByNamesAndEdition(Bird // Myr, Commander 2016) = %q, want \"\": the listing's own edition disagrees with the name-only match", got)
	}
}

// TestMatchTokenPairingByNamesWithoutProduct pins that a pair with no
// TCGplayer product, indexed under its own uuid, is found by its names and
// edition, and still refused under another set's.
func TestMatchTokenPairingByNamesWithoutProduct(t *testing.T) {
	realDatastore(t)

	checked := 0
	for names, id := range testBackend.TokenPairIDByBothNames {
		co, err := testBackend.GetUUID(id)
		if err != nil {
			continue
		}
		edition := testBackend.Sets[co.SetCode].Name
		if EditionTokenSetCode(testBackend, edition) != co.SetCode {
			continue
		}
		checked++

		listing := names[0] + " // " + names[1]
		got := MatchTokenPairingByNamesAndEdition(testBackend, listing, edition, false)
		if got != id {
			t.Errorf("MatchTokenPairingByNamesAndEdition(%s, %s) = %q, want %q", listing, edition, got, id)
		}

		other := "Commander 2018"
		if co.SetCode == "TC18" {
			other = "Commander 2016"
		}
		got = MatchTokenPairingByNamesAndEdition(testBackend, listing, other, false)
		if got != "" {
			t.Errorf("MatchTokenPairingByNamesAndEdition(%s, %s) = %q, want \"\"", listing, other, got)
		}
	}
	if checked == 0 {
		t.Skip("No token pair without a TCGplayer product is reachable by names")
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
	if id, found := testBackend.TokenPairIDByBothNames[key]; found {
		t.Errorf(`TokenPairIDByBothNames[Knight,Zombie] = %q, want no entry (colliding key must stay unresolved)`, id)
	}
}

// TestDeriveTokenPairsMintsWithNoUsableID pins a pairing minted without its id:
// mtgjson's own tokenProducts entry links Fish (TA25 #5) and Kraken (TA25 #6)
// under tcgplayerProductId 162899, but that id is also claimed by the star
// Fish // Kraken, a printing outside the pairing, so deriveTokenPairs's own
// "don't let one id price two different things" guard refuses to hand it to
// this pairing. The pairing itself, which the tokenProducts entry confirms is
// real, id or no id, must still be minted rather than the whole row dropped.
func TestDeriveTokenPairsMintsWithNoUsableID(t *testing.T) {
	realDatastore(t)

	fish := testBackend.MatchInSetNumber("Fish", "TA25", "5")
	kraken := testBackend.MatchInSetNumber("Kraken", "TA25", "6")
	if len(fish) != 1 || len(kraken) != 1 {
		t.Fatal("Fish/Kraken TA25 #5/#6 not present in this datastore")
	}

	id := MatchTokenPairingByUUIDs(testBackend, fish[0].UUID, kraken[0].UUID, false)
	if id == "" {
		t.Fatal("MatchTokenPairingByUUIDs(Fish, Kraken) = \"\", want the derived pairing mtgjson's own tokenProducts entry confirms - a contested id must not drop the row it belongs to")
	}
	co, err := testBackend.GetUUID(id)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", id, err)
	}
	if _, found := co.Identifiers["tcgplayerProductId"]; found {
		t.Errorf("resolved to %s with a tcgplayerProductId, want none: id 162899 is claimed by another printing, so this pairing must mint without one", co.Name)
	}
}

// TestDeriveTokenPairsCrossSetFallsBackRatherThanDrops pins that a pairing with
// no home set is kept too: homeSet requires the two faces' sets to share an
// mtgjson parent-set ancestor, which no real pairing lacks today, so the
// two sets here are made up. When homeSet returns "", the row falls back to
// the first face's token set rather than being dropped.
func TestDeriveTokenPairsCrossSetFallsBackRatherThanDrops(t *testing.T) {
	face := func(uuid, setCode, name string) *matcher.CardObject {
		return &matcher.CardObject{Card: matcher.Card{
			UUID: uuid, SetCode: setCode, Name: name, Number: "1", Layout: "token",
		}}
	}
	const a, b = "00000000-0000-5000-8000-00000000000a", "00000000-0000-5000-8000-00000000000b"
	uuids := map[string]*matcher.CardObject{
		a: face(a, "TAAA", "Dinosaur"),
		b: face(b, "TBBB", "Gnome"),
	}
	sets := map[string]*Set{
		"AAA": {Code: "AAA", TokenSetCode: "TAAA"},
		"TAAA": {Code: "TAAA", ParentCode: "AAA", Tokens: []Card{{UUID: a, TokenProducts: []TokenProduct{{
			Identifiers: map[string]string{"tcgplayerProductId": "1"},
			TokenParts:  []TokenPart{{UUID: a}, {UUID: b}},
		}}}}},
		"BBB":  {Code: "BBB", TokenSetCode: "TBBB"},
		"TBBB": {Code: "TBBB", ParentCode: "BBB"},
	}

	derived := deriveTokenPairs(sets, uuids, map[string]string{})
	if len(derived) != 1 {
		t.Fatalf("deriveTokenPairs minted %d pairings, want 1: a pairing whose faces share no ancestor must fall back rather than drop the row", len(derived))
	}
	if derived[0].SetCode != "TAAA" {
		t.Errorf("derived pairing filed under %q, want the first face's token set TAAA", derived[0].SetCode)
	}
}

// heldFaceFixture builds two token faces a and b that tokenProducts names
// together as product 1, with a holding 1 as its own id, plus any extra
// printings also holding 1. single names 1 again as a's one-part product.
func heldFaceFixture(single bool, extra ...string) (map[string]*Set, map[string]*matcher.CardObject) {
	const a, b = "00000000-0000-5000-8000-00000000000a", "00000000-0000-5000-8000-00000000000b"
	printing := func(uuid, name string, ids map[string]string) *matcher.CardObject {
		ids["mtgjsonId"] = uuid
		return &matcher.CardObject{Card: matcher.Card{
			UUID: uuid, SetCode: "TAAA", Name: name, Number: "1", Layout: "token", Identifiers: ids,
		}}
	}
	uuids := map[string]*matcher.CardObject{
		a: printing(a, "Dinosaur", map[string]string{"tcgplayerProductId": "1"}),
		b: printing(b, "Gnome", map[string]string{}),
	}
	for _, uuid := range extra {
		uuids[uuid] = printing(uuid, "Dinosaur", map[string]string{"tcgplayerProductId": "1"})
	}
	products := []TokenProduct{{
		Identifiers: map[string]string{"tcgplayerProductId": "1"},
		TokenParts:  []TokenPart{{UUID: a}, {UUID: b}},
	}}
	if single {
		products = append(products, TokenProduct{
			Identifiers: map[string]string{"tcgplayerProductId": "1"},
			TokenParts:  []TokenPart{{UUID: a}},
		})
	}
	sets := map[string]*Set{
		"TAAA": {Code: "TAAA", Tokens: []Card{{UUID: a, TokenProducts: products}, {UUID: b}}},
	}
	return sets, uuids
}

// TestDeriveTokenPairsLeavesFaceProductOfAnotherShape pins that a face keeps
// an id tokenProducts also names as the face sold alone: the pairing mints
// without it.
func TestDeriveTokenPairsLeavesFaceProductOfAnotherShape(t *testing.T) {
	const a = "00000000-0000-5000-8000-00000000000a"
	sets, uuids := heldFaceFixture(true)
	tcgIDs := map[string]string{"1": a}

	derived := deriveTokenPairs(sets, uuids, tcgIDs)
	if len(derived) != 1 {
		t.Fatalf("deriveTokenPairs minted %d pairings, want 1", len(derived))
	}
	if id, found := derived[0].Identifiers["tcgplayerProductId"]; found {
		t.Errorf("pairing took product %s, want none: 1 also sells the face alone", id)
	}
	if tcgIDs["1"] != a || uuids[a].Identifiers["tcgplayerProductId"] != "1" {
		t.Errorf("product 1 files at %q and the face holds %q, want both left on the face", tcgIDs["1"], uuids[a].Identifiers["tcgplayerProductId"])
	}
}

// TestDeriveTokenPairsRefusesOutsideHolder pins that a pairing takes no id a
// printing outside it also holds, whichever holder filed the id first.
func TestDeriveTokenPairsRefusesOutsideHolder(t *testing.T) {
	const a, c = "00000000-0000-5000-8000-00000000000a", "00000000-0000-5000-8000-00000000000c"
	for _, first := range []string{a, c} {
		sets, uuids := heldFaceFixture(false, c)
		tcgIDs := map[string]string{"1": first}

		derived := deriveTokenPairs(sets, uuids, tcgIDs)
		if len(derived) != 1 {
			t.Fatalf("filed at %s: deriveTokenPairs minted %d pairings, want 1", first, len(derived))
		}
		if id, found := derived[0].Identifiers["tcgplayerProductId"]; found {
			t.Errorf("filed at %s: pairing took product %s, want none: %s holds it too", first, id, c)
		}
		if tcgIDs["1"] != first || uuids[a].Identifiers["tcgplayerProductId"] != "1" {
			t.Errorf("filed at %s: product 1 files at %q and the face holds %q, want both unchanged", first, tcgIDs["1"], uuids[a].Identifiers["tcgplayerProductId"])
		}
	}
}

// TestAdoptOwnTokenProducts pins when a printing with no TCGplayer product
// takes the one tokenProducts names as selling it alone: only when it names
// exactly one, no other printing names the same one as its own, and no
// printing or sealed product carries it under any id.
func TestAdoptOwnTokenProducts(t *testing.T) {
	own := func(uuid string, ids ...string) Card {
		card := Card{UUID: uuid, Name: uuid, Number: uuid, Identifiers: map[string]string{}}
		for _, id := range ids {
			card.TokenProducts = append(card.TokenProducts, TokenProduct{
				Identifiers: map[string]string{"tcgplayerProductId": id},
				TokenParts:  []TokenPart{{UUID: uuid}},
			})
		}
		return card
	}
	pair := own("pair")
	pair.TokenProducts = []TokenProduct{{
		Identifiers: map[string]string{"tcgplayerProductId": "6"},
		TokenParts:  []TokenPart{{UUID: "pair"}, {UUID: "other"}},
	}}
	priced := own("priced", "7")
	priced.Identifiers["tcgplayerProductId"] = "70"
	etched := own("etched")
	etched.Identifiers["tcgplayerEtchedProductId"] = "8"
	// A double-faced token's two face records, both naming the product as
	// their pair
	dfc := TokenProduct{
		Identifiers: map[string]string{"tcgplayerProductId": "11"},
		TokenParts:  []TokenPart{{UUID: "dfc b"}, {UUID: "dfc a"}},
	}
	dfcA := Card{UUID: "dfc a", Name: "X // Y", Number: "11", Side: "a", Identifiers: map[string]string{}, TokenProducts: []TokenProduct{dfc}}
	dfcB := Card{UUID: "dfc b", Name: "X // Y", Number: "11", Side: "b", Identifiers: map[string]string{}, TokenProducts: []TokenProduct{dfc}}
	// A sheet of two different tokens
	sheet := TokenProduct{
		Identifiers: map[string]string{"tcgplayerProductId": "12"},
		TokenParts:  []TokenPart{{UUID: "sheet x"}, {UUID: "sheet y"}},
	}
	sheetX := own("sheet x")
	sheetX.TokenProducts = []TokenProduct{sheet}
	sheetY := own("sheet y")
	sheetY.TokenProducts = []TokenProduct{sheet}
	foreign := own("foreign")
	foreign.TokenProducts = []TokenProduct{{
		Identifiers: map[string]string{"tcgplayerProductId": "9"},
		TokenParts:  []TokenPart{{UUID: "someone else"}},
	}}

	sets := map[string]*Set{"TAAA": {Code: "TAAA", SealedProduct: []SealedProduct{{
		UUID:        "box",
		Identifiers: map[string]string{"tcgplayerProductId": "10"},
	}}, Cards: []Card{
		own("alone", "1"),
		own("twice", "2", "3"),
		own("shared a", "4"),
		own("shared b", "4"),
		own("repeated", "5", "5"),
		pair,
		priced,
		etched,
		own("owned", "8"),
		own("boxed", "10"),
		foreign,
		dfcA,
		dfcB,
		sheetX,
		sheetY,
	}}, "GAAA": {Code: "GAAA", Tokens: []Card{
		// A set's tokens before the loader files them with its cards
		own("kit token", "13"),
	}}}

	adoptOwnTokenProducts(sets)

	want := map[string]string{
		"alone":     "1",
		"repeated":  "5",
		"priced":    "70",
		"dfc a":     "11",
		"kit token": "13",
	}
	for _, card := range append(sets["TAAA"].Cards, sets["GAAA"].Tokens...) {
		got := card.Identifiers["tcgplayerProductId"]
		if got != want[card.UUID] {
			t.Errorf("%s: tcgplayerProductId %q, want %q", card.UUID, got, want[card.UUID])
		}
	}
}
