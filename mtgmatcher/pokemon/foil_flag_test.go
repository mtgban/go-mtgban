package pokemon

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestFoilFlagAgreesWithFinish pins that the flag and the finish name say the
// same thing. The flag is what a caller with no finish vocabulary reads - the
// arbitrage filters and the website's foil column among them - and the loader
// set the finish without ever setting the flag, so every holo printing in the
// game was filed as a plain one.
func TestFoilFlagAgreesWithFinish(t *testing.T) {
	path := os.Getenv("POKEMON_PATH")
	if path == "" {
		t.Skip("Need POKEMON_PATH set to run this test")
	}
	f, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(b)

	var foil, total, disagree int
	for _, uuid := range mtgmatcher.GetUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil || co.Sealed {
			continue
		}
		total++
		if co.Foil {
			foil++
		}
		if co.Foil != isFoilFinish(co.Finish) {
			if disagree < 5 {
				t.Errorf("%s is %q with Foil=%v", uuid, co.Finish, co.Foil)
			}
			disagree++
		}
	}
	if disagree > 0 {
		t.Errorf("%d of %d printings disagree with their own finish", disagree, total)
	}
	if foil == 0 {
		t.Errorf("no printing of %d is foil, and the game is mostly holo", total)
	}
	t.Logf("%d of %d printings are foil", foil, total)
}
