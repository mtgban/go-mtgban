package mtgmatcher

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

func TestAlias(t *testing.T) {
	inCard := &Card{
		Name:      "Forest",
		Variation: "Full-Art",
		Edition:   "Zendikar",
	}
	outCards := []Card{
		Card{
			Id:      "8b2b4813-187c-53d1-8ee6-d9109ce4c427",
			Name:    "Forest",
			Edition: "Zendikar",
			Number:  "246",
		},
		Card{
			Id:      "7c0ffc88-34ff-5436-bfe7-ac9f1dd62888",
			Name:    "Forest",
			Edition: "Zendikar",
			Number:  "247",
		},
		Card{
			Id:      "59cea094-ebc9-5afa-bdf3-f0cc832a2136",
			Name:    "Forest",
			Edition: "Zendikar",
			Number:  "248",
		},
		Card{
			Id:      "41d883ae-9018-5218-887e-502b03a2b89f",
			Name:    "Forest",
			Edition: "Zendikar",
			Number:  "249",
		},
	}

	_, err := Match(inCard)
	if err == nil {
		t.Errorf("FAIL: this call is supposed to return an error")
		return
	}

	alias, ok := err.(*AliasingError)
	if !ok {
		t.Errorf("FAIL: the returned error is not AliasingError")
		t.Errorf("%s", err.Error())
		return
	}

	dupes := alias.Probe()
	if len(dupes) != len(outCards) {
		t.Errorf("FAIL: wrong number of dupes returned")
		t.Errorf("%v", dupes)
		return
	}

	for i, dupe := range dupes {
		if dupe != outCards[i] {
			t.Errorf("FAIL: incorrect duplicate returned")
			t.Errorf("%v vs %v", dupe, outCards[i])
		}
	}
	t.Log("PASS: Aliasing")
}

type SplitTest struct {
	In  string
	Out []string
}

var SplitTests = []SplitTest{
	SplitTest{
		In:  "A B",
		Out: []string{"A B"},
	},
	SplitTest{
		In:  "A (B)",
		Out: []string{"A", "B"},
	},
	SplitTest{
		In:  "A (B) (C)",
		Out: []string{"A", "B", "C"},
	},
	SplitTest{
		In:  "A B (C)",
		Out: []string{"A B", "C"},
	},
	SplitTest{
		In:  "A (B) C",
		Out: []string{"A", "B"},
	},
}

func TestSplit(t *testing.T) {
	for _, probe := range SplitTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := SplitVariants(test.In)
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
