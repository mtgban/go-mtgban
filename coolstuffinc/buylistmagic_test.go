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
		{
			// No notes: the number is the promo pack's, in the set the
			// image names.
			desc: "a noteless Promo row keeps its number",
			card: CSIPriceEntry{
				PID: "310976", Name: "Usher of the Fallen", ItemSet: "Promo", Number: "401",
				Image: "khm401usherofthefallen", IsFoil: 1,
			},
			wantSet: "KHM", wantNum: "401",
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

// TestPreprocessFinishSplitRetail pins the sale listing of a product whose
// note offers one number per finish: the image names the foil number, and
// the offer row's finish picks the printing.
func TestPreprocessFinishSplitRetail(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	for _, tt := range []struct {
		foil    bool
		wantNum string
	}{
		{false, "265"},
		{true, "210"},
	} {
		in, err := preprocess(b, "Aang, Air Nomad", "Avatar: The Last Airbender Eternal Legal Cards", "Card number can be 210 or 265", "https://s.cf.net/i/TLE0210.jpg", tt.foil)
		if err != nil {
			t.Fatalf("preprocess() = %v", err)
		}
		id, err := b.Match(in)
		if err != nil {
			t.Fatalf("Match(%+v) = %v", in, err)
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != "TLE" || co.Number != tt.wantNum || co.Foil != tt.foil {
			t.Errorf("foil=%v -> %s %s foil=%v, want TLE %s", tt.foil, co.SetCode, co.Number, co.Foil, tt.wantNum)
		}
	}

	// A note naming a language is not split, so the listing stays foreign.
	in, err := preprocess(b, "Aang, Air Nomad", "Avatar: The Last Airbender Eternal Legal Cards", "Card number can be 210 or 265 - Japanese", "https://s.cf.net/i/TLE0210.jpg", false)
	if err == nil {
		id, err := b.Match(in)
		if err == nil {
			t.Errorf("Japanese listing matched %s, want a refusal", id)
		}
	}
}

// TestShelfFinishSplit pins where the sale listing's split looks: on the
// shelf's own set where its name resolves, so a card filed at the same number
// in two sets lands on the shelf's, and anywhere when the name names no set.
func TestShelfFinishSplit(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	for _, tt := range []struct {
		name, edition, notes string
		foil                 bool
		wantSet, wantNum     string
	}{
		// Clone is #45 in both Magic 2013 and Magic 2010.
		{"Clone", "Magic 2013", "Card number can be 45 or 46", false, "M13", "45"},
		{"Clone", "Magic 2010", "Card number can be 45 or 46", false, "M10", "45"},
		{"Clone", "Core Sets", "Card number can be 45 or 46", false, "", ""},
		{"Aang, Air Nomad", "Avatar Eternal", "Card number can be 210 or 265", false, "TLE", "265"},
	} {
		got := shelfFinishSplit(b, tt.name, tt.edition, tt.notes, tt.foil)
		switch {
		case tt.wantSet == "" && got != nil:
			t.Errorf("%s on %q split to %s %s, want no split", tt.name, tt.edition, got.Edition, got.Variation)
		case tt.wantSet != "" && (got == nil || got.Edition != tt.wantSet || got.Variation != tt.wantNum):
			t.Errorf("%s on %q split to %+v, want %s %s", tt.name, tt.edition, got, tt.wantSet, tt.wantNum)
		}
	}
}
