package coolstuffinc

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestConditionPrintingReachesTheRun pins the printing this storefront sells
// as an offer of its own, naming it where a condition would go. The rows were
// refused as an unsupported condition and the listing dropped, though the
// catalog carries every run they name.
func TestConditionPrintingReachesTheRun(t *testing.T) {
	withGameDatastore(t, "onepiece", "ONEPIECE_PATH")

	// The four rows the last runs refused, each naming a parallel whose
	// pre-errata run the catalog files on its own.
	for _, test := range []struct {
		name, number, edition string
		wantPromos            []string
	}{
		{"Dracule Mihawk (070) (Parallel)", "OP01-070", "OP01 - Romance Dawn", []string{"parallelpreerrata"}},
		{`Eustass"Captain"Kid (051) (Parallel)`, "OP01-051", "OP01 - Romance Dawn", []string{"parallelpreerrata"}},
		{"King (096) (Parallel)", "OP01-096", "OP01 - Romance Dawn", []string{"parallelpreerrata"}},
		{"Ulti (093) (Parallel)", "OP01-093", "OP01 - Romance Dawn", []string{"parallelpreerrata"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			printing := conditionPrinting("PRE-ERRATA  PRE-ERRATA ")
			if printing == "" {
				t.Fatal("the wording named no printing")
			}
			card := &mtgmatcher.InputCard{
				Name:      onePieceSpelling(test.name),
				Edition:   test.edition,
				Variation: eventNamed(test.number+" "+nameQualifiers(test.name)) + " " + printing,
			}
			id, err := mtgmatcher.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v", card, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range test.wantPromos {
				if !co.HasPromoType(want) {
					t.Errorf("Match(%q) = %v, want %q among them", card, co.PromoTypes, want)
				}
			}
		})
	}
}
