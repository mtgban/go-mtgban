package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestTreatmentClaim pins the border and the frame being read beside the
// promo types rather than after them. Deciding them last leaves them speaking
// to a list the promo types have already settled: The Hobbit's Gleaming
// Splendor stands at #15 in a black border and at #239 and #275 borderless,
// the latter two apart on poster and surge foil, and a listing naming only
// the border was answered by #15 once those two had each dropped the
// printings the listing had not claimed.
//
// The denial is a separate rule and stays where it ran: a listing silent
// about the border keeps the borderless printings until the filters that
// read that silence have had their say, so a plain Ninja Pizza [TMC] listing
// still answers with the borderless #93 rather than #32.
func TestTreatmentClaim(t *testing.T) {
	for _, tt := range []struct {
		desc            string
		in              mtgmatcher.InputCard
		wantSet         string
		wantNum         string
		wantBorderless  bool
		wantExtendedArt bool
	}{
		{
			desc:    "a listing naming the border reaches the printing wearing it",
			in:      mtgmatcher.InputCard{Name: "Bard, King of Dale", Edition: "The Hobbit", Variation: "Borderless"},
			wantSet: "HOB", wantNum: "244", wantBorderless: true,
		},
		{
			desc:    "and the one that names nothing keeps the bordered printing beside it",
			in:      mtgmatcher.InputCard{Name: "Bard, King of Dale", Edition: "The Hobbit"},
			wantSet: "HOB", wantNum: "144",
		},
		{
			desc:    "the border is read beside the promo types the other printings carry",
			in:      mtgmatcher.InputCard{Name: "Gleaming Splendor", Edition: "The Hobbit", Variation: "Borderless"},
			wantSet: "HOB", wantNum: "239", wantBorderless: true,
		},
		{
			desc:    "a double-faced name reads the same way",
			in:      mtgmatcher.InputCard{Name: "Ajani, Nacatl Pariah // Ajani, Nacatl Avenger", Edition: "Modern Horizons 3", Variation: "Borderless"},
			wantSet: "MH3", wantNum: "442", wantBorderless: true,
		},
		{
			desc:    "a listing naming the frame reaches the extended art printing",
			in:      mtgmatcher.InputCard{Name: "Lifecraft Engine", Edition: "Aetherdrift", Variation: "Extended Art", Foil: true},
			wantSet: "DFT", wantNum: "423", wantExtendedArt: true,
		},
		{
			desc:    "and silence about the frame is the printing without one",
			in:      mtgmatcher.InputCard{Name: "Lifecraft Engine", Edition: "Aetherdrift"},
			wantSet: "DFT", wantNum: "234",
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
			if borderless := co.BorderColor == BorderColorBorderless; borderless != tt.wantBorderless {
				t.Errorf("Match(%v) = %s|%s borderless %v, want %v", tt.in, co.SetCode, co.Number, borderless, tt.wantBorderless)
			}
			if extended := co.HasFrameEffect(FrameEffectExtendedArt); extended != tt.wantExtendedArt {
				t.Errorf("Match(%v) = %s|%s extended art %v, want %v", tt.in, co.SetCode, co.Number, extended, tt.wantExtendedArt)
			}
		})
	}

	// The word narrows to the printings wearing the treatment and no
	// further: where the set holds two of them and only one is sold in the
	// finish asked for, the finish decides, and where both are, nothing does.
	t.Run("two borderless printings and a word that cannot choose between them", func(t *testing.T) {
		in := mtgmatcher.InputCard{Name: "Bard, King of Dale", Edition: "The Hobbit", Variation: "Borderless", Foil: true}
		id, err := testBackend.Match(&in)
		if err == nil {
			co, _ := testBackend.GetUUID(id)
			t.Errorf("Match(%v) = %s|%s, want an aliasing error", in, co.SetCode, co.Number)
		}
	})
}
