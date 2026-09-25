package mtgmatcher

import "testing"

func TestNormalizeFinish(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Cold Foil", "coldfoil"},
		{"cold-foil", "coldfoil"},
		{"RainbowPillars", "rainbowpillars"},
		{"FreeForm1", "freeform1"},
		{"", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NormalizeFinish(test.name); got != test.want {
				t.Errorf("NormalizeFinish(%q) = %q, want %q", test.name, got, test.want)
			}
		})
	}
}

// TestFinishSlug pins the one spelling every finish is keyed and asked for by.
// A printing TCGplayer adds later is named rather than refused, and a
// storefront's own words ("Holo", "Foil Etched") name nothing: they are its
// scraper's to translate.
func TestFinishSlug(t *testing.T) {
	for _, test := range []struct{ name, want string }{
		{"Normal", FinishNonfoil},
		{"non-foil", FinishNonfoil},
		{"Foil", FinishFoil},
		{"Etched", FinishEtched},
		{"Cold Foil", "coldfoil"},
		{"1st Edition Holofoil", "1steditionholofoil"},
		{"Prismatic Foil", "prismaticfoil"},
		{"Holo", "holo"},
		{"Foil Etched", "foiletched"},
		{"", ""},
	} {
		if got := FinishSlug(test.name); got != test.want {
			t.Errorf("FinishSlug(%q) = %q, want %q", test.name, got, test.want)
		}
	}
}

// TestFinishTable pins that every row reads back through FinishSlug from both
// its TCGplayer name and its label, and that its treatment is a row of its own
// with no run and the same foilness.
func TestFinishTable(t *testing.T) {
	for _, finish := range Finishes {
		for _, name := range []string{finish.TCGplayer, finish.Label} {
			if got := FinishSlug(name); got != finish.Slug {
				t.Errorf("FinishSlug(%q) = %q, want %q", name, got, finish.Slug)
			}
		}
		treatment, found := FinishOf(finish.Treatment)
		if !found || treatment.Run != "" {
			t.Errorf("%s: treatment %q is not a run-less row", finish.Slug, finish.Treatment)
			continue
		}
		if treatment.Foil != finish.Foil {
			t.Errorf("%s: foil %v, its treatment %q says %v", finish.Slug, finish.Foil, finish.Treatment, treatment.Foil)
		}
	}
	if FinishSlug("Normal") != FinishNonfoil || IsFoilFinish(FinishNonfoil) {
		t.Error("the plain printing is not the nonfoil one")
	}
	// A printing TCGplayer adds later is taken for a treatment until it has a row
	if !IsFoilFinish("galaxyfoil") {
		t.Error("an unknown finish is not taken for a foil")
	}
}

// TestPrintingFinish pins that a source's run and treatment name only what
// TCGplayer sells.
// TestDefaultPrinting pins what a bare flag answers with: the treatment the
// table lists first, in the plainest run, and a printing the table has no row
// for only where nothing else of its foilness is sold.
func TestDefaultPrinting(t *testing.T) {
	for _, test := range []struct {
		finishes      []string
		nonfoil, foil string
	}{
		{[]string{"nonfoil", "reverseholofoil", "holofoil"}, "nonfoil", "holofoil"},
		{[]string{"1stedition", "unlimited", "1steditionholofoil", "unlimitedholofoil"}, "unlimited", "unlimitedholofoil"},
		{[]string{"1steditionholofoil", "unlimitedholofoil"}, "", "unlimitedholofoil"},
		{[]string{"coldfoil", "rainbowfoil", "1steditionnormal", "unlimitededitionnormal"}, "unlimitededitionnormal", "rainbowfoil"},
		{[]string{"prismaticfoil", "coldfoil"}, "", "coldfoil"},
		{[]string{"prismaticfoil"}, "", "prismaticfoil"},
		{[]string{"", "limited"}, "limited", ""},
	} {
		printings := map[string]string{}
		for _, finish := range test.finishes {
			printings[finish] = finish
		}
		for foil, want := range map[bool]string{false: test.nonfoil, true: test.foil} {
			got, found := DefaultPrinting(printings, foil)
			if got != want || found != (want != "") {
				t.Errorf("DefaultPrinting(%q, %v) = %q, %v, want %q", test.finishes, foil, got, found, want)
			}
		}
	}
}

// TestNamedFinish pins what a listing naming a run, a treatment or both is
// answered with: the plainest printing sold in every axis named, and nothing
// where the printing is sold in no such finish - one treatment never answers
// for another.
func TestNamedFinish(t *testing.T) {
	wotc := []string{"1stedition", "1steditionholofoil", "unlimited", "unlimitedholofoil"}
	fab := []string{"nonfoil", "rainbowfoil", "1steditionrainbowfoil", "unlimitededitionrainbowfoil"}
	for _, test := range []struct {
		finishes             []string
		run, treatment, want string
	}{
		{[]string{"nonfoil", "reverseholofoil", "holofoil"}, "", "Holofoil", "holofoil"},
		{[]string{"nonfoil", "reverseholofoil", "holofoil"}, "", "Reverse Holofoil", "reverseholofoil"},
		{[]string{"nonfoil", "reverseholofoil"}, "", "Holofoil", ""},
		{wotc, Run1stEdition, "", "1stedition"},
		{wotc, Run1stEdition, "Holofoil", "1steditionholofoil"},
		{wotc, "", "Holofoil", "unlimitedholofoil"},
		{[]string{"1steditionholofoil", "unlimitedholofoil"}, Run1stEdition, "", "1steditionholofoil"},
		{fab, "", "Rainbow Foil", "rainbowfoil"},
		{fab, Run1stEdition, "Rainbow Foil", "1steditionrainbowfoil"},
		{fab, "", "Normal", "nonfoil"},
		{fab, RunUnlimited, "Cold Foil", ""},
		{[]string{"nonfoil", "foil", "1stedition", "unlimited", "limited"}, RunLimited, "", "limited"},
		{[]string{"prismaticfoil"}, "", "Prismatic Foil", ""},
	} {
		printings := map[string]bool{}
		for _, finish := range test.finishes {
			printings[finish] = true
		}
		if got := NamedFinish(printings, test.run, test.treatment); got != test.want {
			t.Errorf("NamedFinish(%q, %q, %q) = %q, want %q", test.finishes, test.run, test.treatment, got, test.want)
		}
	}
}

func TestPrintingFinish(t *testing.T) {
	for _, tt := range []struct{ run, treatment, want string }{
		{"1st Edition", "Rainbow Foil", "1st Edition Rainbow Foil"},
		{"Unlimited Edition", "Normal", "Unlimited Edition Normal"},
		{"Unlimited Edition", "Cold Foil", "Cold Foil"},
		{"", "Cold Foil", "Cold Foil"},
	} {
		if got := PrintingFinish(tt.run, tt.treatment); got != tt.want {
			t.Errorf("PrintingFinish(%q, %q) = %q, want %q", tt.run, tt.treatment, got, tt.want)
		}
	}
}

// TestOtherRun pins the three moves of the run rule and the two it refuses.
func TestOtherRun(t *testing.T) {
	printing := func(finishes map[string]string) (*Backend, *Card) {
		b := &Backend{UUIDs: map[string]*CardObject{}, knownFinishes: map[string]bool{}, Hashes: map[string][]string{}}
		card := &Card{Name: "Alakazam", Number: "001", FoilUUIDs: map[string]string{}}
		for finish, uuid := range finishes {
			card.FoilUUIDs[finish] = uuid
			b.UUIDs[uuid] = &CardObject{Card: Card{UUID: uuid, Name: card.Name, Number: card.Number, Finish: finish}}
			b.Hashes["alakazam"] = append(b.Hashes["alakazam"], uuid)
		}
		for _, finish := range Finishes {
			b.knownFinishes[finish.Slug] = finish.Slug != "limited"
		}
		return b, card
	}
	for _, tt := range []struct {
		desc      string
		finishes  map[string]string
		ask       string
		want      string
		elsewhere map[string]string
	}{
		{"a run the card is printed in on another product is refused", map[string]string{"holofoil": "d"},
			"1steditionholofoil", "", map[string]string{"1steditionholofoil": "s"}},
		{"no run named reaches the unlimited run", map[string]string{"1stedition": "a", "unlimited": "b"}, "nonfoil", "b", nil},
		{"then the first edition", map[string]string{"1steditionholofoil": "a"}, "holofoil", "a", nil},
		{"a run named on a product filed in none is dropped", map[string]string{"nonfoil": "c", "holofoil": "d"}, "1steditionholofoil", "d", nil},
		{"one run never answers for another", map[string]string{"1stedition": "a"}, "unlimited", "", nil},
		{"a product with a treatment sold only in runs is filed in them", map[string]string{"nonfoil": "c", "1steditionholofoil": "e"}, "1stedition", "", nil},
		{"a finish this datastore sells nowhere is not answered", map[string]string{"nonfoil": "c"}, "limited", "", nil},
	} {
		b, card := printing(tt.finishes)
		for finish, uuid := range tt.elsewhere {
			b.UUIDs[uuid] = &CardObject{Card: Card{UUID: uuid, Name: card.Name, Number: card.Number, Finish: finish}}
			b.Hashes["alakazam"] = append(b.Hashes["alakazam"], uuid)
		}
		if got := b.otherRun(card, tt.ask); got != tt.want {
			t.Errorf("%s: otherRun(%q) = %q, want %q", tt.desc, tt.ask, got, tt.want)
		}
	}
}
