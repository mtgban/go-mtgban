package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"

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
	realDatastore(t)

	product := &cm.Product{
		IDProduct:     272488,
		Name:          "Urza's Mine (V.2)",
		Number:        "252", // Cardmarket's own number for this shelf, shared by all four siblings
		ExpansionName: "Chronicles",
	}

	cardID, cardIDFoil := Fallback(product)
	if cardID != "" || cardIDFoil != "" {
		co, _ := mtgmatcher.GetUUID(cardID)
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
	realDatastore(t)

	// All four BCHR arts share mcmId 272488; "114" (bare, no art-suffix
	// letter) is what PlainNumber reduces every one of their numbers to, so
	// this product's own Number agrees with all of them under EqualFold.
	product := &cm.Product{
		IDProduct:     272488,
		Name:          "Urza's Mine",
		Number:        "114",
		ExpansionName: "Chronicles Foreign Black Border",
	}

	cardID, _ := Fallback(product)
	if cardID == "" {
		t.Fatal("Fallback = \"\", want it to still commit to a candidate once one agrees by number")
	}
	co, err := mtgmatcher.GetUUID(cardID)
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
	theCard, err := Preprocess("Urza's Mine (V.2)", "252", "Chronicles")
	if err != nil {
		t.Fatal(err)
	}
	if theCard.Variation != "V.2" {
		t.Errorf("Variation = %q, want \"V.2\" preserved for chrVariants to resolve, not overwritten with Cardmarket's shared number %q", theCard.Variation, "252")
	}
}

// TestResolveMagicLandsCorrectChroniclesArt replays Fallback and Preprocess
// together the way resolveMagic calls them, and confirms the fix actually
// lands product 272488 on the English printing Cardmarket sells it as
// (114b) rather than any of the four Japanese Chronicles Foreign Black
// Border arts mtgjson's mislinked mcmId would otherwise keep.
func TestResolveMagicLandsCorrectChroniclesArt(t *testing.T) {
	realDatastore(t)

	product := &cm.Product{
		IDProduct:     272488,
		Name:          "Urza's Mine (V.2)",
		Number:        "252",
		ExpansionName: "Chronicles",
	}

	cardID, _ := Fallback(product)
	if cardID != "" {
		t.Fatalf("Fallback kept %q, want it to defer", cardID)
	}

	theCard, err := Preprocess(product.Name, product.Number, product.ExpansionName)
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}

	id, err := mtgmatcher.Match(theCard)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	co, err := mtgmatcher.GetUUID(id)
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
	realDatastore(t)

	tests := []struct {
		name       string
		wantNumber string
	}{
		{"Plains (V.1)", "364alt"},
		{"Plains (V.2)", "365alt"},
		{"Plains (V.3)", "366alt"},
	}
	for _, tt := range tests {
		theCard, err := Preprocess(tt.name, "175", "Fourth Edition: Alternate")
		if err != nil {
			t.Fatalf("%s: Preprocess: %v", tt.name, err)
		}
		if theCard.Variation == "175" {
			t.Fatalf("%s: Variation was overwritten with Cardmarket's shared number instead of kept as the (V.N) tag", tt.name)
		}
		id, err := mtgmatcher.Match(theCard)
		if err != nil {
			t.Fatalf("%s: Match: %v", tt.name, err)
		}
		co, err := mtgmatcher.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.Number != tt.wantNumber {
			t.Errorf("%s: Match landed on %s, want number %s", tt.name, co, tt.wantNumber)
		}
	}
}
