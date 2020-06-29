package mtgdb

import "testing"

type ExtractTest struct {
	In  string
	Out string
}

var NumberTests = []ExtractTest{
	ExtractTest{
		In:  "123",
		Out: "123",
	},
	ExtractTest{
		In:  "(321)",
		Out: "321",
	},
	ExtractTest{
		In:  "#24",
		Out: "24",
	},
	ExtractTest{
		In:  "100 - A",
		Out: "100",
	},
	ExtractTest{
		In:  "1a",
		Out: "1a",
	},
	ExtractTest{
		In:  "*4",
		Out: "*4",
	},
	ExtractTest{
		In:  "37A Text",
		Out: "37a",
	},
	ExtractTest{
		In:  "A08",
		Out: "A08",
	},
	ExtractTest{
		In:  "2000",
		Out: "",
	},
	ExtractTest{
		In:  "M19",
		Out: "",
	},
	ExtractTest{
		In:  "26 April",
		Out: "",
	},
	ExtractTest{
		In:  "181/185",
		Out: "181",
	},
}

func TestExtractNumber(t *testing.T) {
	for _, probe := range NumberTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := ExtractNumber(test.In)
			if out != test.Out {
				t.Errorf("FAIL %s: Expected '%s' got '%s'", test.In, test.Out, out)
				return
			}
			t.Log("PASS:", test.In)
		})
	}
}

var YearTests = []ExtractTest{
	ExtractTest{
		In:  "Judge 2007",
		Out: "2007",
	},
	ExtractTest{
		In:  "Judge Foil (2020)",
		Out: "2020",
	},
	ExtractTest{
		In:  "FNM '06",
		Out: "2006",
	},
	ExtractTest{
		In:  "20 - multiples 2012",
		Out: "2012",
	},
	ExtractTest{
		In:  "not a 96 year",
		Out: "",
	},
	ExtractTest{
		In:  "missing year",
		Out: "",
	},
	ExtractTest{
		In:  "Urza's Saga Arena 1999",
		Out: "1999",
	},
	ExtractTest{
		In:  "M14 Core Set",
		Out: "2014",
	},
	ExtractTest{
		In:  "WCD 2002:",
		Out: "2002",
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

type CutTest struct {
	In  string
	Tag string
	Out []string
}

var CutTests = []CutTest{
	CutTest{
		In:  "A B C",
		Tag: "A",
		Out: []string{"", "A B C"},
	},
	CutTest{
		In:  "A B C",
		Tag: "C",
		Out: []string{"A B", "C"},
	},
	CutTest{
		In:  "A B C",
		Tag: "D",
		Out: []string{"A B C"},
	},
	CutTest{
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
