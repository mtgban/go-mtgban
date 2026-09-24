package cardmarket

import (
	"strings"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestFallbackDefersOnMcmIdCollision pins the bug a formal review of the
// Chronicles/chrVariants gap surfaced: mtgjson's own mcmId identifier links
// all four Chronicles Foreign Black Border (Japanese) arts of Urza's Mine to
// Cardmarket's 272488 - the id of one of the four *plain*, English Chronicles
// arts instead. checkLoadedID's search-by-name-then-filter-by-mcmId therefore
// answers with four Japanese candidates for an English product, none of which
// share a number with each other or with what the product's own Number field
// would carry (Cardmarket does not distinguish these four in that field at
// all - that's what its own "(V.N)" name suffix is for). Before the fix,
// Fallback trusted "the last one found" regardless; the fix defers to
// Preprocess/Match instead, whose own chrVariants table already carries a
// "v.1".."v.4" key for exactly this card.
func TestFallbackDefersOnMcmIdCollision(t *testing.T) {
	b := realDatastore(t)

	product := &cm.Product{
		IDProduct:     272488,
		Name:          "Urza's Mine (V.2)",
		Number:        "252", // Cardmarket's own number for this shelf, shared by all four siblings
		ExpansionName: "Chronicles",
	}

	cardID, cardIDFoil := Fallback(b, product)
	if cardID != "" || cardIDFoil != "" {
		co, _ := b.GetUUID(cardID)
		t.Fatalf("Fallback = (%q, %q), want (\"\", \"\") - kept %s instead of deferring to Preprocess/Match", cardID, cardIDFoil, co)
	}
}

// TestFallbackStillTrustsANumberMatch guards the case Fallback's own doc
// comment already named (mtgjson occasionally stamps one mcmId on sibling
// variants, e.g. a frame-variant pair): when the product's own Number
// actually agrees with one of several candidates sharing an mcmId, Fallback
// must still commit to it - the ambiguity guard only defers when *nothing*
// agrees, never merely because more than one candidate id came back.
func TestFallbackStillTrustsANumberMatch(t *testing.T) {
	b := realDatastore(t)

	// All four BCHR arts share mcmId 272488; "114" (bare, no art-suffix
	// letter) is what PlainNumber reduces every one of their numbers to, so
	// this product's own Number agrees with all of them under EqualFold.
	product := &cm.Product{
		IDProduct:     272488,
		Name:          "Urza's Mine",
		Number:        "114",
		ExpansionName: "Chronicles Foreign Black Border",
	}

	cardID, _ := Fallback(b, product)
	if cardID == "" {
		t.Fatal("Fallback = \"\", want it to still commit to a candidate once one agrees by number")
	}
	co, err := b.GetUUID(cardID)
	if err != nil {
		t.Fatal(err)
	}
	if co.Name != "Urza's Mine" || co.SetCode != "BCHR" {
		t.Errorf("Fallback kept %s, want one of the BCHR Urza's Mine printings", co)
	}
}

// TestPreprocessKeepsVIndexForChronicles pins the other half of the same
// bug: even once Fallback defers, Preprocess's own generic fallback for
// editions with no dedicated case ("Old editions do not have any number
// assigned, if so, then keep the V.1 V.2 etc style and process in
// variants.go") was unconditional on Cardmarket's Number field being
// non-empty, discarding the "(V.2)" suffix Chronicles' own chrVariants table
// needs whenever Cardmarket's Number field is populated at all - which for
// these grouped, same-numbered siblings, it always is. The fix only takes
// the number when magic.VariantsTable has no entry for this exact
// edition/card/variant to consult instead.
func TestPreprocessKeepsVIndexForChronicles(t *testing.T) {
	b := realDatastore(t)
	theCard, err := Preprocess(b, "Urza's Mine (V.2)", "252", "Chronicles")
	if err != nil {
		t.Fatal(err)
	}
	if theCard.Variation != "V.2" {
		t.Errorf("Variation = %q, want \"V.2\" preserved for chrVariants to resolve, not overwritten with Cardmarket's shared number %q", theCard.Variation, "252")
	}
}

// TestMysteryBooster2ReprintsResolvesToPLSTNumber pins mb2PLSTNumber against
// the live MTGJSON datastore: "Mystery Booster 2: Reprints from Across
// Magic's History" products carry no number of their own, and the same card
// name can print at more than one PLST number (Suture Priest at both
// MOC-210 and NPH-25) - without reading which one MB2's own booster
// actually bundles, resolution is an AliasingError instead of a match.
func TestMysteryBooster2ReprintsResolvesToPLSTNumber(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		cardName string
		number   string
	}{
		{"Suture Priest", "NPH-25"},
		{"Terramorphic Expanse", "JMP-78"},
	}
	for _, test := range tests {
		theCard, err := Preprocess(b, test.cardName, "", "Mystery Booster 2: Reprints from Across Magic's History")
		if err != nil {
			t.Fatalf("%s: Preprocess: %v", test.cardName, err)
		}

		cardID, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("%s: Match: %v", test.cardName, err)
		}
		co, _ := b.GetUUID(cardID)
		if co.SetCode != "PLST" || co.Number != test.number {
			t.Errorf("%s: matched %s %s, want PLST %s", test.cardName, co.SetCode, co.Number, test.number)
		}
	}
}

// TestResolveMagicLandsCorrectChroniclesArt replays Fallback and Preprocess
// together the way resolveMagic calls them, and confirms the fix actually
// lands product 272488 on the English printing Cardmarket sells it as
// (114b) rather than any of the four Japanese Chronicles Foreign Black
// Border arts mtgjson's mislinked mcmId would otherwise keep.
func TestResolveMagicLandsCorrectChroniclesArt(t *testing.T) {
	b := realDatastore(t)

	product := &cm.Product{
		IDProduct:     272488,
		Name:          "Urza's Mine (V.2)",
		Number:        "252",
		ExpansionName: "Chronicles",
	}

	cardID, _ := Fallback(b, product)
	if cardID != "" {
		t.Fatalf("Fallback kept %q, want it to defer", cardID)
	}

	theCard, err := Preprocess(b, product.Name, product.Number, product.ExpansionName)
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}

	id, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	co, err := b.GetUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	if co.SetCode != "CHR" || co.Number != "114b" || co.Language != "English" {
		t.Errorf("Match landed on %s, want the plain Chronicles 114b (English)", co)
	}
}

// TestFourthEditionAlternateKeepsVIndex pins the same default-case fix for
// an edition whose Cardmarket name ("Fourth Edition: Alternate") is not the
// matcher's own ("Alternate Fourth Edition") - the VariantsTable lookup that
// guards the fallback has to try the alias mtgmatcher/magic/editions.go's
// EditionTable resolves it to, the same one AdjustEdition applies later, or
// it never finds ed4Variants at all and clobbers the "(V.N)" tag same as
// Chronicles did. Each of Plains's three "(V.N)" siblings must land on its
// own distinct printing, not all three on whichever one FilterCards happens
// to see first.
func TestFourthEditionAlternateKeepsVIndex(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		name       string
		wantNumber string
	}{
		{"Plains (V.1)", "364"},
		{"Plains (V.2)", "365"},
		{"Plains (V.3)", "366"},
	}
	for _, tt := range tests {
		theCard, err := Preprocess(b, tt.name, "175", "Fourth Edition: Alternate")
		if err != nil {
			t.Fatalf("%s: Preprocess: %v", tt.name, err)
		}
		if theCard.Variation == "175" {
			t.Fatalf("%s: Variation was overwritten with Cardmarket's shared number instead of kept as the (V.N) tag", tt.name)
		}
		id, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("%s: Match: %v", tt.name, err)
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.Number != tt.wantNumber {
			t.Errorf("%s: Match landed on %s, want number %s", tt.name, co, tt.wantNumber)
		}
	}
}

// TestFallbackDefersOnImplausibleWCDCandidate pins a live-confirmed mtgjson
// linking mistake, a single-candidate variant of the same class of bug as
// TestFallbackDefersOnMcmIdCollision above: id 249617, "Phyrexian Processor
// (V.2)" under Cardmarket's "WCD 2000: Janosch Kühn", carries mtgjson's
// mcmId 249617 - but on a completely unrelated printing, The Brothers' War
// Retro Artifacts' foil Phyrexian Processor, not any of the real World
// Championship Decks printings (set codes WC97-WC04). id 249533, "Duress
// (V.2)" under "WCD 2001: Antoine Ruel", carries the same shape onto a
// starred Seventh Edition Duress. Neither is an ambiguous multi-candidate
// case the existing number-disagreement guard would catch - there was only
// ever the one candidate, and it simply names the wrong card. Fallback now
// also distrusts a WCD product's lone candidate when that candidate is not
// itself a WC-family printing, deferring to Preprocess/Match instead (which
// already resolves these by name/edition).
func TestFallbackDefersOnImplausibleWCDCandidate(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		id      int
		name    string
		edition string
	}{
		{249617, "Phyrexian Processor (V.2)", "WCD 2000: Janosch Kühn"},
		{249533, "Duress (V.2)", "WCD 2001: Antoine Ruel"},
	}
	for _, tt := range tests {
		product := &cm.Product{IDProduct: tt.id, Name: tt.name, ExpansionName: tt.edition}
		cardID, cardIDFoil := Fallback(b, product)
		if cardID != "" || cardIDFoil != "" {
			co, _ := b.GetUUID(cardID)
			t.Errorf("%d: Fallback = (%q, %q), want (\"\", \"\") - kept %s, not a World Championship Decks printing", tt.id, cardID, cardIDFoil, co)
		}

		theCard, err := Preprocess(b, product.Name, product.Number, product.ExpansionName)
		if err != nil {
			t.Fatalf("%d: Preprocess: %v", tt.id, err)
		}
		id, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("%d: Match: %v", tt.id, err)
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(co.SetCode, "WC") {
			t.Errorf("%d: Match landed on %s, want a World Championship Decks printing", tt.id, co)
		}
	}
}

// TestSLDCommanderDeckCardDisambiguatesByNumber pins sldCommanderDeckCard's
// disambiguation rule: a deck that reprints one name under more than one
// distinct printing - four Shapeshifter Tokens (SLD 1906-1909) in
// "Everyone's Invited!", four Plains (SLD 1348-1351) in "Angels: They're
// Just Like Us but Cooler" - must not pick the board's first one merely
// because the name matched. A product number that agrees with one of them
// settles it; without one, and with more than one distinct printing under
// that name, the caller's own route decides instead. A name the deck lists
// under only one printing still resolves without a number, whether the
// deck holds one copy of it (Sol Ring) or several (the ten-copy Plains of
// "Raining Cats and Dogs" - GetPicksForDeck lists one entry per physical
// copy, not per printing, so the count itself must not read as ambiguity).
func TestSLDCommanderDeckCardDisambiguatesByNumber(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		name       string
		expansion  string
		cardName   string
		number     string
		wantSet    string
		wantNumber string
	}{
		{
			"a numbered token settles on its own printing, not the board's first",
			"Secret Lair Commander Deck: Everyone's Invited!",
			"Shapeshifter Token", "1907", "SLD", "1907",
		},
		{
			"the same token with no number is ambiguous and refuses",
			"Secret Lair Commander Deck: Everyone's Invited!",
			"Shapeshifter Token", "", "", "",
		},
		{
			"a numbered basic land settles on its own printing, not the board's first",
			"Secret Lair Commander Deck: Angels: They're Just Like Us but Cooler",
			"Plains", "1350", "SLD", "1350",
		},
		{
			"the same basic land with no number is ambiguous and refuses",
			"Secret Lair Commander Deck: Angels: They're Just Like Us but Cooler",
			"Plains", "", "", "",
		},
		{
			"a name the deck lists once still resolves without a number",
			"Secret Lair Commander Deck: Everyone's Invited!",
			"Sol Ring", "", "SLD", "1905",
		},
		{
			"a basic land printed once but held as ten copies is not ambiguous",
			"Secret Lair Commander Deck: Raining Cats and Dogs",
			"Plains", "", "SLD", "1513",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := sldCommanderDeckCard(b, tt.expansion, tt.cardName, tt.number)
			if id == "" {
				if tt.wantSet != "" {
					t.Fatalf("sldCommanderDeckCard(%q, %q) = \"\", want %s %s", tt.cardName, tt.number, tt.wantSet, tt.wantNumber)
				}
				return
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantSet == "" {
				t.Fatalf("sldCommanderDeckCard(%q, %q) = %s, want \"\" (ambiguous)", tt.cardName, tt.number, co)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
				t.Errorf("sldCommanderDeckCard(%q, %q) = %s, want %s %s", tt.cardName, tt.number, co, tt.wantSet, tt.wantNumber)
			}
		})
	}
}
