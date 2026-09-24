package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The Delighted Halfling borderless->Prerelease rule (AdjustEdition) must
// skip HOC, whose two borderless printings carry no Prerelease sibling.
func TestHalflingBorderlessPrereleaseIsLTROnly(t *testing.T) {
	realDatastore(t)
	b := testBackend

	in := &mtgmatcher.InputCard{
		Name:      "Delighted Halfling",
		Edition:   "The Lord of the Rings: Tales of Middle-earth",
		Variation: "Borderless",
	}
	Rules{}.AdjustEdition(b, in)
	if in.Variation != "Borderless Prerelease" {
		t.Errorf("LTR: Variation = %q, want %q", in.Variation, "Borderless Prerelease")
	}

	in = &mtgmatcher.InputCard{
		Name:      "Delighted Halfling",
		Edition:   "The Hobbit: Eternal-Legal",
		Variation: "Borderless",
	}
	Rules{}.AdjustEdition(b, in)
	if in.Variation != "Borderless" {
		t.Errorf("HOC: Variation = %q, want %q (unchanged)", in.Variation, "Borderless")
	}

	// End to end, with the wording Strike Zone and Hareruya actually send.
	for _, tc := range []struct {
		variation string
		foil      bool
		want      string
	}{
		{"Borderless", false, "11e51251-1d60-5abf-a995-7c89ed80f6ac"},
		{"Borderless", true, "11e51251-1d60-5abf-a995-7c89ed80f6ac_f"},
		{"Borderless Surge Foil", true, "c1b2ac14-984b-5d6f-bea1-e2f449943a06"},
		{"Boxtopper", false, "11e51251-1d60-5abf-a995-7c89ed80f6ac"},
		{"Surge Foil", true, "c1b2ac14-984b-5d6f-bea1-e2f449943a06"},
	} {
		in := &mtgmatcher.InputCard{
			Name:      "Delighted Halfling",
			Edition:   "The Hobbit: Eternal-Legal",
			Variation: tc.variation,
			Foil:      tc.foil,
		}
		id, err := b.Match(in)
		if err != nil {
			t.Errorf("HOC %q: unexpected error: %v", tc.variation, err)
			continue
		}
		if id != tc.want {
			t.Errorf("HOC %q: id = %q, want %q", tc.variation, id, tc.want)
		}
	}
}
