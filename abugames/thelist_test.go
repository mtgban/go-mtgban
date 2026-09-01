package abugames

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestTheListNumber pins what a listing reaches when its wording names a
// reprint and its number names a card of the set it is filed under. The
// storefront carries both for one card, the reprint set holds it once, and
// nothing in either listing says which of them is the reprint.
func TestTheListNumber(t *testing.T) {
	for _, test := range []struct {
		desc    string
		card    ABUCard
		wantSet string
		wantNum string
	}{
		{"the number echoing the reprint's own tail", ABUCard{
			DisplayTitle: "Savage Lands (The List)", Edition: "Commander Masters", Number: "228"}, "PLST", "ALA-228"},
		{"and another", ABUCard{
			DisplayTitle: "Armorcraft Judge (The List)", Edition: "Commander Masters", Number: "144"}, "PLST", "KLD-144"},
		{"a number the reprint set spells whole", ABUCard{
			DisplayTitle: "Lonis, Cryptozoologist (The List)", Edition: "Modern Horizons 2", Number: "MH2-204"}, "PLST", "MH2-204"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := test.card
			in, err := preprocess(&card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", card.DisplayTitle, err)
			}
			id, err := mtgmatcher.Match(in)
			if err != nil {
				t.Fatalf("Match(%q) = %v", in, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != test.wantSet || co.Number != test.wantNum {
				t.Errorf("Match(%q) = %s|%s, want %s|%s", in, co.SetCode, co.Number, test.wantSet, test.wantNum)
			}
		})
	}
}

// TestTheListConflict pins the listings whose number names a card of their own
// set exactly, contradicting the reprint their wording names. Both cannot be
// true, so neither is priced.
func TestTheListConflict(t *testing.T) {
	for _, test := range []struct {
		desc string
		card ABUCard
	}{
		{"a Commander Masters card wearing the reprint's words", ABUCard{
			DisplayTitle: "Savage Lands (The List)", Edition: "Commander Masters", Number: "1025"}},
		{"a Theros card whose reprint is filed under another set", ABUCard{
			DisplayTitle: "Heliod's Intervention (The List)", Edition: "Theros Beyond Death", Number: "19"}},
		{"and a Foundations one", ABUCard{
			DisplayTitle: "Adamant Will (The List)", Edition: "Foundations", Number: "488"}},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := test.card
			in, err := preprocess(&card)
			if !errors.Is(err, errConflictingNumber) {
				t.Errorf("preprocess(%q) = %v, %v, want %v", card.DisplayTitle, in, err, errConflictingNumber)
			}
		})
	}
}
