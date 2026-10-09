package coolstuffinc

import "testing"

// TestPreprocessBuylistMagic pins Magic buylist rows, copied from the feed,
// to the printing each one names.
func TestPreprocessBuylistMagic(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	for _, tt := range []struct {
		desc             string
		card             CSIPriceEntry
		wantSet, wantNum string
	}{
		{
			desc: "a Deckmasters basic read off its letter",
			card: CSIPriceEntry{
				PID: "96682", Name: "Swamp A", ItemSet: "Deckmasters", Code: "DKM", Image: "SwampADKMa",
				Notes: "picture 1. Deckmaster cards are white bordered and have a stylized D as their set\r\nsymbol.",
			},
			wantSet: "DKM", wantNum: "42",
		},
		{
			desc: "a Zendikar block basic told apart by its note",
			card: CSIPriceEntry{
				PID: "229637", Name: "Wastes A - 183", ItemSet: "Oath of the Gatewatch", Code: "OGW",
				Notes: "This is NOT the full art version", Number: "183", Image: "WASTES183PROMO",
			},
			wantSet: "OGW", wantNum: "183a",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in, err := PreprocessBuylist(b, tt.card)
			if err != nil {
				t.Fatalf("PreprocessBuylist() = %v", err)
			}
			id, err := b.Match(in)
			if err != nil {
				t.Fatalf("Match(%+v) = %v", in, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum || co.Foil != (tt.card.IsFoil == 1) {
				t.Errorf("%s -> %s %s foil=%v, want %s %s", tt.card.Name, co.SetCode, co.Number, co.Foil, tt.wantSet, tt.wantNum)
			}
		})
	}
}
