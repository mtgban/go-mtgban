package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestFabSameTreatment pins that a Flesh and Blood listing lands only on a
// printing in its own treatment, whatever the print run: Volzar's Regular
// copies are not its Cold Foil token (ROS021), since a Normal printing
// carries that number as a face. A treatment no printing at the number is
// sold in names nothing, so the blueprint's own product stands, unless a
// promo printing was reached by name rather than by that product.
func TestFabSameTreatment(t *testing.T) {
	b := &mtgmatcher.Backend{UUIDs: map[string]*mtgmatcher.CardObject{
		"ros021_coldfoil":           {Card: mtgmatcher.Card{Finish: "Cold Foil", Number: "ROS021"}, Foil: true},
		"ros009_ros021_normal":      {Card: mtgmatcher.Card{Finish: "Normal", Number: "ROS009//ROS021"}},
		"her009_1steditioncoldfoil": {Card: mtgmatcher.Card{Finish: "1st Edition Cold Foil", Number: "HER009"}, Foil: true},
		"wtr001_normal":             {Card: mtgmatcher.Card{Finish: "Normal", Number: "WTR001"}},
		"wtr001_unlimitednormal":    {Card: mtgmatcher.Card{Finish: "Unlimited Edition Normal", Number: "WTR001"}},
		"sea245_rainbowfoil":        {Card: mtgmatcher.Card{Finish: "Rainbow Foil", Number: "SEA245", PromoTypes: []string{"treasure"}, Identifiers: map[string]string{"tcgplayerProductId": "633612"}}, Foil: true},
	}}
	b.AllUUIDs = []string{"ros021_coldfoil", "ros009_ros021_normal", "her009_1steditioncoldfoil", "wtr001_normal", "wtr001_unlimitednormal", "sea245_rainbowfoil"}
	siblings := fabNumbers(b)
	for _, tc := range []struct {
		cardID, treatment string
		tcgID             int
		want              bool
	}{
		{"ros021_coldfoil", "Regular", 0, false},
		{"ros021_coldfoil", "Cold Foil", 0, true},
		{"ros021_coldfoil", "Gold Foil", 0, true},
		{"her009_1steditioncoldfoil", "Cold Foil", 0, true},
		{"wtr001_normal", "", 0, true},
		{"wtr001_normal", "Rainbow Foil", 0, true},
		{"wtr001_normal", "Cold Foil", 0, true},
		{"wtr001_unlimitednormal", "Regular", 0, true},
		{"sea245_rainbowfoil", "Regular", 0, false},
		{"sea245_rainbowfoil", "Regular", 633612, true},
		{"unknown", "Regular", 0, false},
	} {
		var p Product
		p.Properties.FabFoilNew = tc.treatment
		got := fabSameTreatment(b, siblings, tc.cardID, tc.tcgID, p)
		if got != tc.want {
			t.Errorf("%q listed %q: same treatment %v, want %v", tc.cardID, tc.treatment, got, tc.want)
		}
	}
}
