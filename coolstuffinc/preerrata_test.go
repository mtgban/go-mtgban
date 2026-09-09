package coolstuffinc

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// promoTypesSpell reports whether a printing's tags spell the label wanted,
// as one tag or as the parts of one.
//
// A datastore publishing a shelf's whole product name files a printing
// under a single tag, "parallelpreerrata"; one publishing the facts apart
// files the same printing under "parallel" and "preerrata", with the
// instalment beside them as a mark. It is the same printing either way, so
// a test names the spellings it accepts and each tag of one is matched
// whole.
func promoTypesSpell(promoTypes []string, spellings ...[]string) bool {
	for _, spelling := range spellings {
		named := len(spelling) > 0
		for _, tag := range spelling {
			named = named && slices.Contains(promoTypes, tag)
		}
		if named {
			return true
		}
	}
	return false
}

// parallelPreErrata is the corrected parallel run, in the two spellings a
// datastore files it under.
var parallelPreErrata = [][]string{{"parallelpreerrata"}, {"parallel", "preerrata"}}

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
		// Either spelling of the run: one tag where the datastore
		// publishes a shelf's whole product name, two where it publishes
		// the facts apart.
		wantPromos [][]string
	}{
		{"Dracule Mihawk (070) (Parallel)", "OP01-070", "OP01 - Romance Dawn", parallelPreErrata},
		{`Eustass"Captain"Kid (051) (Parallel)`, "OP01-051", "OP01 - Romance Dawn", parallelPreErrata},
		{"King (096) (Parallel)", "OP01-096", "OP01 - Romance Dawn", parallelPreErrata},
		{"Ulti (093) (Parallel)", "OP01-093", "OP01 - Romance Dawn", parallelPreErrata},
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
			if !promoTypesSpell(co.PromoTypes, test.wantPromos...) {
				t.Errorf("Match(%q) = %v, want %v spelled among them",
					card, co.PromoTypes, test.wantPromos)
			}
		})
	}
}
