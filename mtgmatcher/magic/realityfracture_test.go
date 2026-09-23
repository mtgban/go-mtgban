package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Reality Fracture (FRA) and Reality Fracture Commander (FRC) both carry
// booster-fun duplicates - the same card reprinted at a second, unrelated
// number - with no promoTypes or frameEffects to tell the plain printing
// from its duplicate: mtgjson does set isAlternative correctly on the
// duplicate, the same signal CMR's own alternates already carry, so a
// listing naming no treatment must still land on the plain (non-alternate)
// printing rather than refusing as ambiguous. Rows below are copied
// verbatim from the loaded datastore (2026-09-20 AllPrintings5).
func TestRealityFractureBoosterfunDuplicates(t *testing.T) {
	realDatastore(t)

	for _, tt := range []struct {
		desc                string
		name, edition, code string
		wantNumber          string
	}{
		{"fra_essence_burn_regular", "Essence Burn", "Reality Fracture", "FRA", "82"},
		{"fra_fatehold_charm_regular", "Fatehold Charm", "Reality Fracture", "FRA", "132"},
		{"fra_fblthp_impossibly_lost_regular", "Fblthp, Impossibly Lost", "Reality Fracture", "FRA", "213"},
		{"frc_venser_fervent_forger_regular", "Venser, Fervent Forger", "Reality Fracture Commander", "FRC", "9"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card := &mtgmatcher.InputCard{
				Name:    tt.name,
				Edition: tt.edition,
			}
			id, err := testBackend.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v, want a landed id", tt.name, err)
			}
			co, err := testBackend.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.code || co.Number != tt.wantNumber {
				t.Errorf("Match(%q) = %s %s, want %s %s", tt.name, co.SetCode, co.Number, tt.code, tt.wantNumber)
			}
		})
	}
}
