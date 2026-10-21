package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestFabSameTreatment pins that a Flesh and Blood listing lands only on a
// printing in its own treatment, whatever the print run: Volzar's Regular
// copies are not its Cold Foil token (ROS021), and a Gold Foil copy is no
// printing the datastore files.
func TestFabSameTreatment(t *testing.T) {
	b := &mtgmatcher.Backend{UUIDs: map[string]*mtgmatcher.CardObject{
		"ros021_coldfoil":           {Card: mtgmatcher.Card{Finish: "Cold Foil"}, Foil: true},
		"her009_1steditioncoldfoil": {Card: mtgmatcher.Card{Finish: "1st Edition Cold Foil"}, Foil: true},
		"wtr001_normal":             {Card: mtgmatcher.Card{Finish: "Normal"}},
		"wtr001_unlimitednormal":    {Card: mtgmatcher.Card{Finish: "Unlimited Edition Normal"}},
	}}
	for _, tc := range []struct {
		cardID, treatment string
		want              bool
	}{
		{"ros021_coldfoil", "Regular", false},
		{"ros021_coldfoil", "Cold Foil", true},
		{"ros021_coldfoil", "Gold Foil", false},
		{"her009_1steditioncoldfoil", "Cold Foil", true},
		{"wtr001_normal", "", true},
		{"wtr001_normal", "Rainbow Foil", false},
		{"wtr001_unlimitednormal", "Regular", true},
		{"unknown", "Regular", false},
	} {
		var p Product
		p.Properties.FabFoilNew = tc.treatment
		got := fabSameTreatment(b, tc.cardID, p)
		if got != tc.want {
			t.Errorf("%q listed %q: same treatment %v, want %v", tc.cardID, tc.treatment, got, tc.want)
		}
	}
}
