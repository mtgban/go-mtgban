package mtgmatcher_test

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

// The finishes every game shares answer without a datastore holding them,
// which is what a caller gets before one is installed.
func TestIsFoilFinishSharedNames(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{"foil", true},
		{"Foil", true},
		{"COLD-FOIL", false}, // a name only a game can place
		{"etched", true},
		{"nonfoil", false},
		{"normal", false},
		{"Normal", false},
	} {
		if got := mtgmatcher.IsFoilFinish(tt.name); got != tt.want {
			t.Errorf("IsFoilFinish(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// A game naming its own finishes is answered from the printings that carry
// them, so nobody writes the vocabulary down twice.
//
// Flesh and Blood is the game that needs this: it keys a printing by its
// print run and treatment together, so "1steditionnormal" sits beside
// "1steditioncoldfoil" and no spelling rule tells them apart. Every printing
// already says which it is through Foil and Etched, and this checks the name
// gives the same answer on all of them.
func TestIsFoilFinishReadsThePrintings(t *testing.T) {
	path := os.Getenv("FLESHANDBLOOD_PATH")
	if path == "" {
		t.Skip("Need FLESHANDBLOOD_PATH set to run this test")
	}
	b, err := datastore.Read("fleshandblood", path)
	if err != nil {
		t.Fatal(err)
	}
	previous := mtgmatcher.GlobalDatastore()
	mtgmatcher.SetGlobalDatastore(b)
	t.Cleanup(func() { mtgmatcher.SetGlobalDatastore(previous) })

	var checked, disagreed int
	finishes := map[string]bool{}
	for _, uuid := range mtgmatcher.GetUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil || co.Sealed || co.Finish == "" {
			continue
		}
		checked++
		finishes[co.Finish] = true
		want := co.Foil || co.Etched
		if got := mtgmatcher.IsFoilFinish(co.Finish); got != want {
			disagreed++
			if disagreed < 4 {
				t.Errorf("%s %s #%s: IsFoilFinish(%q) = %v, but the printing says %v",
					co.Name, co.SetCode, co.Number, co.Finish, got, want)
			}
		}
	}
	if checked == 0 {
		t.Fatal("the datastore loaded no printings")
	}
	if disagreed != 0 {
		t.Errorf("%d of %d printings disagree with the name they carry", disagreed, checked)
	}

	// The pair that has no spelling rule between them.
	for name, want := range map[string]bool{
		"1steditionnormal":   false,
		"1steditioncoldfoil": true,
	} {
		if !finishes[name] {
			t.Skipf("this datastore sells no %s printing", name)
		}
		if got := mtgmatcher.IsFoilFinish(name); got != want {
			t.Errorf("IsFoilFinish(%q) = %v, want %v", name, got, want)
		}
	}
}
