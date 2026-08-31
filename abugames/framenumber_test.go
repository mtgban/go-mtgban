package abugames

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestFrameNumber pins the printing a listing reaches when it names a frame
// this storefront files under the plain card's own collector number. The
// number is appended last, so it buried the frame the wording named and every
// treatment of a card answered with the plain one.
func TestFrameNumber(t *testing.T) {
	path := os.Getenv("ALLPRINTINGS5_PATH")
	if path == "" {
		t.Skip("Need ALLPRINTINGS5_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		desc    string
		title   string
		edition string
		number  string
		wantSet string
		wantNum string
	}{
		{"an extended art filed under the plain number", "Platoon Dispenser (Extended Art)", "The Brothers' War", "36", "BRO", "310"},
		{"a showcase god", "Heliod, Sun-Crowned (Showcase)", "Theros Beyond Death", "18", "THB", "259"},
		{"a borderless planeswalker", "Oko, Thief of Crowns (Borderless) - FOIL", "Throne of Eldraine", "197", "ELD", "271"},
		{"a retro frame in etched", "Soul Snare (Retro Frame ETCHED) - FOIL", "Modern Horizons 2", "266", "MH2", "387"},
		{"a set numbering its frames below the plain card", "Polygoyf (Extended Art)", "Modern Horizons 3 Commander", "117", "M3C", "65"},
		{"a frame the storefront numbers correctly stays put", "Kona, Rescue Beastie (Showcase) - FOIL", "Duskmourn: House of Horror", "358", "DSK", "358"},
		{"a listing spelling its own number stays put", "Salvation Engine (Borderless First-Place 517) - FOIL", "Aetherdrift", "517", "DFT", "517"},
		{"a Secret Lair keeps the number naming its drop", "Ashiok, Dream Render (Borderless) - FOIL", "Secret Lair Drop", "399", "SLD", "399"},
		{"and another, where the drop is the whole identity", "Boros Charm (Borderless) - FOIL", "Secret Lair Drop", "217", "SLD", "217"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := ABUCard{DisplayTitle: test.title, Edition: test.edition, Number: test.number}
			in, err := preprocess(&card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", test.title, err)
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

// TestNamesTreatment pins that nothing but a frame is read this way: the
// retry costs two matches, and a wording naming no frame has nothing to find.
func TestNamesTreatment(t *testing.T) {
	for _, v := range []string{"Extended Art", "Borderless", "Showcase", "Retro Frame"} {
		if !namesTreatment(v) {
			t.Errorf("namesTreatment(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"", "Prerelease", "JSS Foil", "Buy-A-Box"} {
		if namesTreatment(v) {
			t.Errorf("namesTreatment(%q) = true, want false", v)
		}
	}
}
