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
		{
			desc: "the nonfoil row of a note offering one number per finish",
			card: CSIPriceEntry{
				PID: "421807", Name: "Aang, Air Nomad", ItemSet: "Avatar: The Last Airbender Eternal Legal Cards",
				Code: "TLE", Notes: "Card number can be 210 or 265", Image: "TLE0210",
			},
			wantSet: "TLE", wantNum: "265",
		},
		{
			desc: "the foil row of a note offering one number per finish",
			card: CSIPriceEntry{
				PID: "421807", Name: "Aang, Air Nomad", ItemSet: "Avatar: The Last Airbender Eternal Legal Cards",
				Code: "TLE", Notes: "Card number can be 210 or 265", Image: "TLE0210", IsFoil: 1,
			},
			wantSet: "TLE", wantNum: "210",
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

// TestFinishSplitNote pins the notes read as one number per finish, and the
// range and list shapes that name more printings than that.
func TestFinishSplitNote(t *testing.T) {
	for _, tt := range []struct {
		note          string
		first, second string
	}{
		{"Card number can be 210 or 265", "210", "265"},
		{"Card number can be 0522 or 0842", "522", "842"},
		{"Nonfoil is number 0765, Foil is 0720 - Not Available in Play Boosters", "765", "720"},
		{"Card number can be 289 - 296", "", ""},
		{"Artist Pace Wilder - Card number can be 859, 861, 863, or 865", "", ""},
		{"Card number can be 1 or 2 or 3", "", ""},
	} {
		m := finishSplitNote.FindStringSubmatch(tt.note)
		var first, second string
		if m != nil {
			first, second = m[1]+m[2], m[3]
		}
		if first != tt.first || second != tt.second {
			t.Errorf("%q read as %q/%q, want %q/%q", tt.note, first, second, tt.first, tt.second)
		}
	}
}
