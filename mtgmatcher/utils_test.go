package mtgmatcher

import "testing"

type ExtractTest struct {
	In  string
	Out string
}

var YearTests = []ExtractTest{
	{
		In:  "Judge 2007",
		Out: "2007",
	},
	{
		In:  "Judge Foil (2020)",
		Out: "2020",
	},
	{
		In:  "FNM '06",
		Out: "2006",
	},
	{
		In:  "20 - multiples 2012",
		Out: "2012",
	},
	{
		In:  "not a 96 year",
		Out: "",
	},
	{
		In:  "missing year",
		Out: "",
	},
	{
		In:  "Urza's Saga Arena 1999",
		Out: "1999",
	},
	{
		In:  "M14 Core Set",
		Out: "2014",
	},
	{
		In:  "WCD 2002:",
		Out: "2002",
	},
	{
		In:  "7/4/1999",
		Out: "",
	},
}

func TestExtractYear(t *testing.T) {
	for _, probe := range YearTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := ExtractYear(test.In)
			if out != test.Out {
				t.Errorf("FAIL %s: Expected '%s' got '%s'", test.In, test.Out, out)
				return
			}
			t.Log("PASS:", test.In)
		})
	}
}

var NumericTests = []ExtractTest{
	{
		In:  "123",
		Out: "123",
	},
	{
		In:  "(321)",
		Out: "321",
	},
	{
		In:  "#24",
		Out: "24",
	},
	{
		In:  "100 - A",
		Out: "100",
	},
	{
		In:  "1a",
		Out: "1",
	},
	{
		In:  "*4",
		Out: "4",
	},
	{
		In:  "37A Text",
		Out: "37",
	},
	{
		In:  "A08",
		Out: "8",
	},
	{
		In:  "2000",
		Out: "2000",
	},
	{
		In:  "181/185",
		Out: "181",
	},
	{
		In:  "001/006",
		Out: "1",
	},
	{
		In:  "May 25",
		Out: "25",
	},
	{
		In:  "22 January, 2016",
		Out: "22",
	},
	{
		In:  "7/4/1999",
		Out: "7",
	},
	{
		In:  "37★",
		Out: "37",
	},
	{
		In:  "1F★",
		Out: "1",
	},
	{
		In:  "659Φ",
		Out: "659",
	},
	{
		In:  "118†s",
		Out: "118",
	},
}

func TestExtractNumberValue(t *testing.T) {
	for _, probe := range NumericTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := ExtractNumberValue(test.In)
			if out != test.Out {
				t.Errorf("FAIL %s: Expected '%s' got '%s'", test.In, test.Out, out)
				return
			}
			t.Log("PASS:", test.In)
		})
	}
}

type CutTest struct {
	In  string
	Tag string
	Out []string
}

var CutTests = []CutTest{
	{
		In:  "A B C",
		Tag: "A",
		Out: []string{"", "A B C"},
	},
	{
		In:  "A B C",
		Tag: "C",
		Out: []string{"A B", "C"},
	},
	{
		In:  "A B C",
		Tag: "D",
		Out: []string{"A B C"},
	},
	{
		In:  "A B C D",
		Tag: "B C",
		Out: []string{"A", "B C D"},
	},
}

func TestCut(t *testing.T) {
	for _, probe := range CutTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := Cut(test.In, test.Tag)
			for i := range out {
				if out[i] != test.Out[i] {
					t.Errorf("FAIL %s: Expected '%s' got '%q'", test.In, test.Out, out)
					return
				}
			}
			t.Log("PASS:", test.In)
		})
	}
}

type SplitTest struct {
	In  string
	Out []string
}

var SplitTests = []SplitTest{
	{
		In:  "A",
		Out: []string{"A"},
	},
	{
		In:  "A B",
		Out: []string{"A B"},
	},
	{
		In:  "A (B)",
		Out: []string{"A", "B"},
	},
	{
		In:  "A (B) (C)",
		Out: []string{"A", "B", "C"},
	},
	{
		In:  "A B (C)",
		Out: []string{"A B", "C"},
	},
	{
		In:  "A (B) C",
		Out: []string{"A", "B"},
	},
	{
		In:  "A (B) Token",
		Out: []string{"A Token", "B"},
	},
	{
		In:  "Erase (Not the Urza's Legacy One)",
		Out: []string{"Erase (Not the Urza's Legacy One)"},
	},
	{
		In:  "B.F.M. (Big Furry Monster) (Left)",
		Out: []string{"B.F.M. (Big Furry Monster)", "Left"},
	},
	{
		In:  "A (B)(C)",
		Out: []string{"A", "B", "C"},
	},
	{
		In:  "A (B (C))",
		Out: []string{"A", "B", "C"},
	},
}

func TestSplit(t *testing.T) {
	for _, probe := range SplitTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := SplitVariants(test.In)
			if len(out) != len(test.Out) {
				t.Errorf("FAIL %s: Expected '%q' got '%q'", test.In, test.Out, out)
				return
			}
			for i := range out {
				if out[i] != test.Out[i] {
					t.Errorf("FAIL %s: Expected '%q' got '%q'", test.In, test.Out, out)
					return
				}
			}
			t.Log("PASS:", test.In)
		})
	}
}

var TitleTests = []ExtractTest{
	{
		In:  "abc abc",
		Out: "Abc Abc",
	},
	{
		In:  "Abc abC",
		Out: "Abc Abc",
	},
	{
		In:  "ABC ABC",
		Out: "Abc Abc",
	},
	// An ordinal keeps its letters down. A title-caser capitalises them
	// because they follow a digit, which is the one thing that says they are
	// an ordinal at all.
	{
		In:  "1st place",
		Out: "1st Place",
	},
	{
		In:  "2nd place",
		Out: "2nd Place",
	},
	{
		In:  "3rd place",
		Out: "3rd Place",
	},
	{
		In:  "10th anniversary",
		Out: "10th Anniversary",
	},
	{
		In:  "25th anniversary edition",
		Out: "25th Anniversary Edition",
	},
	// A capital after a digit is otherwise the one wanted, and stays.
	{
		In:  "3d text",
		Out: "3D Text",
	},
	// The letters alone are not an ordinal, so they keep their capital.
	{
		In:  "nd",
		Out: "Nd",
	},
	{
		In:  "the stars",
		Out: "The Stars",
	},
	{
		In:  "duel of destiny",
		Out: "Duel of Destiny",
	},
	{
		In:  "legacy of the duelist",
		Out: "Legacy of the Duelist",
	},
	{
		In:  "back to duel",
		Out: "Back to Duel",
	},
	// A small word opening the phrase is not small: it is the first word.
	{
		In:  "the sacred cards",
		Out: "The Sacred Cards",
	},
	{
		In:  "a",
		Out: "A",
	},
	// Nothing small in it, and the ordinal still comes back down.
	{
		In:  "25th anniversary edition",
		Out: "25th Anniversary Edition",
	},
	// A mark hands what follows a phrase of its own, and a small word
	// opening that phrase is no longer inside one.
	{
		In:  "tag force: the world",
		Out: "Tag Force: The World",
	},
	{
		In:  "duel of destiny - the movie",
		Out: "Duel of Destiny - The Movie",
	},
	{
		In:  "battles of legend: the crystal",
		Out: "Battles of Legend: The Crystal",
	},
	// The mark has to end the word before, so a hyphen inside one says
	// nothing about what follows.
	{
		In:  "yu-gi-oh the movie",
		Out: "Yu-Gi-Oh the Movie",
	},
}

func TestTitle(t *testing.T) {
	for _, probe := range TitleTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := Title(test.In)
			if out != test.Out {
				t.Errorf("FAIL %s: Expected '%s' got '%s'", test.In, test.Out, out)
				return
			}
			t.Log("PASS:", test.In)
		})
	}
}

// TestIsPromoHeading pins the question two games ask of an edition, and that
// asking it does not depend on when the package finished initializing: the
// table is built on first use because Normalize memoizes through a map this
// package sets up in its own init.
func TestIsPromoHeading(t *testing.T) {
	for _, tt := range []struct {
		edition string
		want    bool
	}{
		{"Promo", true},
		{"Promos", true},
		{"Promotional Cards", true},
		{"promo cards", true},
		{"D23 Promos", false},
		{"Disney Lorcana Promo Cards", false},
		{"Origins", false},
		{"", false},
	} {
		if got := IsPromoHeading(tt.edition); got != tt.want {
			t.Errorf("IsPromoHeading(%q) = %v, want %v", tt.edition, got, tt.want)
		}
	}
}
