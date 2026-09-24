package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestCatalogTreatment pins the patterned holos the storefront and the catalog
// disagree on. The disagreement is not cosmetic: "foil" is a finish the
// matcher already reads, so a Master Ball asked for by the storefront's name
// answers with the set's reverse holo rather than missing outright.
func TestCatalogTreatment(t *testing.T) {
	for _, tt := range []struct {
		desc, in, want string
	}{
		{"the storefront calls the pattern a foil",
			"024/086 Master Ball Foil", "024/086 Master Ball Pattern"},
		{"and the other pattern too",
			"059/131 Poke Ball Foil", "059/131 Poke Ball Pattern"},
		{"an ordinary foil is left alone",
			"099 Reverse Foil", "099 Reverse Foil"},
		{"so is a plain number",
			"024/086", "024/086"},
		{"and a wording neither spells",
			"Holo Promo", "Holo Promo"},
		{"an empty variation stays empty", "", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := catalogTreatment(tt.in); got != tt.want {
				t.Errorf("catalogTreatment(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestBuylistEtchedQualifier pins a Secret Lair "(Foil-Etched)" buylist row
// whose qualifier lives only in the name, not the notes PreprocessBuylist
// otherwise reads for it.
func TestBuylistEtchedQualifier(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	for _, tt := range []struct {
		desc       string
		card       CSIPriceEntry
		wantNumber string
		wantEtched bool
	}{
		{
			desc: "the etched qualifier lives in the name, not the notes",
			card: CSIPriceEntry{
				PID: "345549", Name: "Talisman of Hierarchy (Foil-Etched)",
				ItemSet: "Secret Lair", Number: "1057", IsFoil: 1,
				Notes: "Secret Lair: Dan Frazier Is Back Again: The Enemy Talismans",
			},
			wantNumber: "1057", wantEtched: true,
		},
		{
			desc: "a plain Secret Lair listing still lands nonfoil",
			card: CSIPriceEntry{
				PID: "345548", Name: "Talisman of Hierarchy",
				ItemSet: "Secret Lair", Number: "1057", IsFoil: 0,
				Notes: "Secret Lair: Dan Frazier Is Back Again: The Enemy Talismans",
			},
			wantNumber: "1057", wantEtched: false,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := PreprocessBuylist(b, tt.card)
			if err != nil {
				t.Fatalf("PreprocessBuylist(%+v) = %v", tt.card, err)
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%+v) = %v", card, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.Number != tt.wantNumber || co.Etched != tt.wantEtched {
				t.Errorf("got number=%s etched=%v, want number=%s etched=%v",
					co.Number, co.Etched, tt.wantNumber, tt.wantEtched)
			}
		})
	}
}
