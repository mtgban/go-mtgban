package coolstuffinc

import (
	"testing"
)

// TestPreprocessBuylistResolvesTokenPairing pins a two-sided token buylist
// row resolving to the combined entity mtgmatcher/magic derives for it,
// anchored by the row's own Code and Number - CSI's buylist JSON carries
// both, even though CSIPriceEntry did not decode Code until this fix and
// nothing downstream of PreprocessBuylist knew to route a "X (Token) //
// Y (Token)" name through them instead of trying to match the whole
// two-face string as one card name. "Angel // Cat" is Commander 2014's own
// derived pairing (no native mtgjson entity, no usable TCGplayer id).
func TestPreprocessBuylistResolvesTokenPairing(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	card := CSIPriceEntry{
		PID: "209796", Name: "Angel // Cat (Token)", ItemSet: "Commander 2014 Edition",
		Price: "0.05", Number: "001/002", RarityName: "Fixed", Code: "C14",
	}
	out, err := PreprocessBuylist(b, card)
	if err != nil {
		t.Fatalf("PreprocessBuylist(%s) = %v", card.PID, err)
	}
	co, err := b.GetUUID(out.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", out.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Angel // Cat" {
		t.Errorf("resolved to %q, want \"Angel // Cat\"", co.Name)
	}
}

// TestPreprocessBuylistResolvesCompoundNumberPairing pins the same anchor
// working off a compound "020/023" Number - one collector number per face,
// CSI's own convention for most two-sided token rows - trying the first
// half against the pairing's first face.
func TestPreprocessBuylistResolvesCompoundNumberPairing(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	card := CSIPriceEntry{
		PID: "209802", Name: "Beast (Token) // Elf Druid (Token)", ItemSet: "Commander 2014 Edition",
		Price: "0.50", Number: "020/023", RarityName: "Fixed", Code: "C14",
	}
	out, err := PreprocessBuylist(b, card)
	if err != nil {
		t.Fatalf("PreprocessBuylist(%s) = %v", card.PID, err)
	}
	co, err := b.GetUUID(out.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", out.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Beast // Elf Druid" {
		t.Errorf("resolved to %q, want \"Beast // Elf Druid\"", co.Name)
	}
}

// TestPreprocessBuylistRefusesUnresolvedTokenPairing pins the reverse:
// "Horror // Zombie" is a real Archenemy: Nicol Bolas buylist row, but
// mtgjson carries no Zombie token under E01 (or its token sheet) at all -
// no tokenProducts record links the two faces, so there is nothing to
// anchor. A refusal here is correct per this project's own standing rule
// (drop what mtgjson has no record of, without logging it) rather than a
// gap to chase further.
func TestPreprocessBuylistRefusesUnresolvedTokenPairing(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	card := CSIPriceEntry{
		PID: "243749", Name: "Horror // Zombie (Token)", ItemSet: "Archenemy: Nicol Bolas",
		Price: "0.02", Number: "003", RarityName: "Fixed", Code: "E01",
	}
	_, err := PreprocessBuylist(b, card)
	if err == nil {
		t.Fatal("PreprocessBuylist succeeded, want a refusal: no mtgjson record links Horror to Zombie in this edition")
	}
}

// TestPreprocessBuylistIgnoresSingleFaceTokenDash pins that a single-faced
// token whose own name happens to carry " - " (CSI's own way of telling
// two same-name token variants apart, e.g. Core Set 2021's two different
// Cat tokens) is never routed through the two-sided pairing path: the
// trigger requires " // ", not just a hyphen, and preprocessTokenPairBuylist
// would refuse this name outright since it never splits on " - " without a
// leading "(Token)" name on both sides.
func TestPreprocessBuylistIgnoresSingleFaceTokenDash(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	card := CSIPriceEntry{
		PID: "299568", Name: "Cat (Token) - #20", ItemSet: "Core Set 2021",
		Price: "0.05", Number: "020", RarityName: "Fixed", Code: "M21",
	}
	_, err := PreprocessBuylist(b, card)
	if err != nil {
		t.Fatalf("PreprocessBuylist(%s) = %v, want the ordinary single-token path to still resolve it", card.PID, err)
	}
}
