package abugames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestFrameNumber pins the printing a listing reaches when this storefront's
// own collector number and the frame its wording names disagree. The number is
// appended last, so it buried the frame a listing named - and it is wrong the
// other way round too, naming a frame on a listing that names none.
func TestFrameNumber(t *testing.T) {
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
		{"a set keeping its two finishes at two numbers", "Aang's Defense (211)", "Avatar: The Last Airbender", "211", "TLE", "266"},
		{"a ripple foil, which the number knows nothing of", "Talon Gates of Madara (Ripple) - FOIL", "Modern Horizons 3 Commander", "134", "M3C", "82★"},
		{"a retro frame sold only in etched", "Aeromoeba (Retro Frame ETCHED) - FOIL", "Modern Horizons 2", "37", "MH2", "389"},
		{"an etched card the wording names outright", "Six (ETCHED) - FOIL", "Modern Horizons 3", "169", "MH3", "484"},
		{"a borderless card already on its own number", "Avacyn, Angel of Hope (Borderless)", "Innistrad Remastered", "482", "INR", "482"},
		{"a showcase already on its own number", "Marina Vendrell (Showcase) - FOIL", "Duskmourn: House of Horror", "360", "DSK", "360"},
		{"a plain listing carrying the frame's own number", "Nimble Trapfinder", "Zendikar Rising", "332", "ZNR", "72"},
		{"and its sibling, swapped the same way", "Master of Winds", "Zendikar Rising", "331", "ZNR", "68"},
		{"a commander deck numbering its frames first", "Wave Goodbye", "The Lost Caverns of Ixalan Commander", "47", "LCC", "79"},
		{"a borderless land's number on the plain land", "Underground River - FOIL", "The Brothers' War", "300", "BRO", "267"},
		{"a set marking every printing keeps its number", "Nexus of Fate", "Special Guests", "122", "SPG", "122"},
		{"a surge foil spelling its own number", "The Sea Devils (Surge 713) - FOIL", "Doctor Who", "713", "WHO", "713"},
		{"the storefront's own misspelling still names a frame", "Discontinuity (Extented Art) - FOIL", "Core Set 2021 / M21", "349", "M21", "349"},
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

// TestFinishNamed pins the flag a printing answers this storefront's FOIL
// with: etched counts, because the storefront spells it the same way.
func TestFinishNamed(t *testing.T) {
	for _, test := range []struct {
		co   mtgmatcher.CardObject
		want bool
	}{
		{mtgmatcher.CardObject{Foil: true}, true},
		{mtgmatcher.CardObject{Etched: true}, true},
		{mtgmatcher.CardObject{}, false},
	} {
		if got := finishNamed(&test.co); got != test.want {
			t.Errorf("finishNamed(foil=%v etched=%v) = %v, want %v", test.co.Foil, test.co.Etched, got, test.want)
		}
	}
}
