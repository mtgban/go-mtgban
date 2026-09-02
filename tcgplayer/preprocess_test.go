package tcgplayer

import (
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
	"github.com/mtgban/go-tcgplayer"
)

func TestMain(m *testing.M) {
	allprintingsPath := os.Getenv("ALLPRINTINGS5_PATH")
	if allprintingsPath == "" {
		log.Fatalln("Need ALLPRINTINGS5_PATH variable set to run tests")
	}

	reader, err := os.Open(allprintingsPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer reader.Close()

	ds, err := magic.Load(reader)
	if err != nil {
		log.Fatalln(err)
	}
	mtgmatcher.SetGlobalDatastore(ds)

	os.Exit(m.Run())
}

// TestPreprocessJapanesePromoTokens pins which sheet a Japanese promo token
// is filed under. Six sets print one and all of them are sold from the same
// catalog group, so the qualifier is what tells them apart. Reading every one
// as Dominaria United's reported the Wilds of Eldraine bird as the Dominaria
// bird, which would have overwritten a right id upstream with a wrong one.
func TestPreprocessJapanesePromoTokens(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		name      string
		cleanName string
		setCode   string
	}{
		{"the qualifier names the set the sheet belongs to", "Bird Token (JP WOE Exclusive)", "Bird Token JP WOE Exclusive", "WWOE"},
		{"and another set's qualifier names another sheet", "Zombie Token (JP MKM Exclusive)", "Zombie Token JP MKM Exclusive", "WMKM"},
		// Dominaria United printed the first of these sheets, and its
		// qualifier names no set at all
		{"the bare qualifier is still Dominaria United's", "Bird Token (JP Exclusive)", "Bird Token JP Exclusive", "WDMU"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			product := tcgplayer.Product{Name: tt.name, CleanName: tt.cleanName}
			editions := map[int]string{0: "Unique and Miscellaneous Promos"}

			theCard, err := Preprocess(&product, editions)
			if err != nil {
				t.Fatalf("Preprocess(%q) = %v", tt.name, err)
			}
			cardID, err := mtgmatcher.Match(theCard)
			if err != nil {
				t.Fatalf("Match(%v) = %v", theCard, err)
			}
			co, err := mtgmatcher.GetUUID(cardID)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", cardID, err)
			}
			if co.SetCode != tt.setCode {
				t.Errorf("Preprocess(%q) matched %s %s #%s, want a printing in %s",
					tt.name, co.SetCode, co.Card.Name, co.Number, tt.setCode)
			}
		})
	}
}
