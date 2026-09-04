package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestMisprintTwin pins the two things a star at the end of a collector
// number can mean. The catalog files one on the misprints it knows about,
// and a listing that never says "Misprint" is not selling those. But where a
// set filed the foil as a card of its own the loader gives the twin a star
// too, and dropping that one takes the foil printing of an ordinary card:
// the Turtles listings saying only "Surge Foil" lost their black-bordered
// printing that way and answered with the borderless one standing beside it,
// three stages before the border was ever read.
func TestMisprintTwin(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		in      mtgmatcher.InputCard
		wantSet string
		wantNum string
	}{
		{
			desc:    "a listing silent about the border reaches the foil twin of the plain printing",
			in:      mtgmatcher.InputCard{Name: "Ninja Pizza", Edition: "Teenage Mutant Ninja Turtles Eternal", Variation: "Surge Foil", Foil: true},
			wantSet: "TMC", wantNum: "32★",
		},
		{
			desc:    "and the one that names the border still reaches the borderless printing",
			in:      mtgmatcher.InputCard{Name: "Ninja Pizza", Edition: "Teenage Mutant Ninja Turtles Eternal", Variation: "Borderless Surge Foil", Foil: true},
			wantSet: "TMC", wantNum: "93",
		},
		{
			desc:    "a star the catalog itself files is a misprint, and silence is not selling one",
			in:      mtgmatcher.InputCard{Name: "Shadow Lance", Edition: "Guildpact", Foil: true},
			wantSet: "GPT", wantNum: "14",
		},
		{
			desc:    "the listing that does say so reaches it",
			in:      mtgmatcher.InputCard{Name: "Shadow Lance", Edition: "Guildpact", Variation: "Misprint", Foil: true},
			wantSet: "GPT", wantNum: "14★",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := testBackend.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, err := testBackend.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("Match(%v) = %s|%s, want %s|%s", tt.in, co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
		})
	}

	// The star the catalog files sits on a printing sold in a finish its base
	// also sells, which is what tells it from a twin: the pair a set split by
	// finish shares none.
	t.Run("the two stars are told apart by the finishes they share", func(t *testing.T) {
		for _, tc := range []struct {
			setCode, number string
			wantTwin        bool
		}{
			{"TMC", "32★", true},
			{"GPT", "14★", false},
		} {
			cards := testBackend.MatchInSetNumber("", tc.setCode, tc.number)
			if len(cards) == 0 {
				cards = testBackend.MatchWithNumber("", tc.setCode, tc.number)
			}
			if len(cards) == 0 {
				t.Fatalf("no printing at %s #%s", tc.setCode, tc.number)
			}
			if twin := testBackend.IsFinishTwin(cards[0].UUID); twin != tc.wantTwin {
				t.Errorf("IsFinishTwin(%s #%s) = %v, want %v", tc.setCode, tc.number, twin, tc.wantTwin)
			}
		}
	})
}
