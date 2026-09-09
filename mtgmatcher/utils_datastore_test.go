package mtgmatcher_test

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// These two read the datastore - ExtractNumber has to know 7ED names a set
// before it will pass over it for the collector number - so they sit with the
// rest of the datastore-backed suite rather than in the package's own tests.
type extractTest struct {
	In  string
	Out string
}

var numberTests = []extractTest{
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
		Out: "1a",
	},
	{
		In:  "*4",
		Out: "*4",
	},
	{
		In:  "37A Text",
		Out: "37a",
	},
	{
		In:  "A08",
		Out: "a08",
	},
	{
		In:  "2000",
		Out: "",
	},
	{
		In:  "M19",
		Out: "",
	},
	{
		In:  "26 April",
		Out: "",
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
		Out: "",
	},
	{
		In:  "yaviMAYa 25",
		Out: "25",
	},
	{
		In:  "22 January, 2016",
		Out: "",
	},
	{
		In:  "7/4/1999",
		Out: "",
	},
	{
		In:  "37★",
		Out: "37★",
	},
	{
		In:  "1F★",
		Out: "1f★",
	},
	{
		In:  "659Φ",
		Out: "659φ",
	},
	{
		In:  "A25-141",
		Out: "A25-141",
	},
	{
		In:  "P09-008",
		Out: "P09-8",
	},
	{
		In:  "118†s",
		Out: "118†s",
	},
	{
		In:  "8th Edition 332 Julien Nuijten 2004",
		Out: "332",
	},
	{
		In:  "2001 Tom van de Logt 7ED 337",
		Out: "337",
	},
	{
		In:  "30a",
		Out: "30a",
	},
}

func TestExtractNumber(t *testing.T) {
	realDatastore(t)

	for _, probe := range numberTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := mtgmatcher.ExtractNumber(test.In)
			if out != test.Out {
				t.Errorf("FAIL %s: Expected '%s' got '%s'", test.In, test.Out, out)
				return
			}
			t.Log("PASS:", test.In)
		})
	}
}

func TestAlias(t *testing.T) {
	realDatastore(t)

	inCard := &mtgmatcher.InputCard{
		Name:      "Forest",
		Variation: "Full-Art",
		Edition:   "Zendikar",
	}
	// These are ZEN Forest 246, 247, 248, 249
	outCards := []string{
		"8b2b4813-187c-53d1-8ee6-d9109ce4c427",
		"7c0ffc88-34ff-5436-bfe7-ac9f1dd62888",
		"59cea094-ebc9-5afa-bdf3-f0cc832a2136",
		"41d883ae-9018-5218-887e-502b03a2b89f",
	}

	_, err := mtgmatcher.Match(inCard)
	if err == nil {
		t.Error("FAIL: this call is supposed to return an error")
		return
	}

	alias, ok := err.(*mtgmatcher.AliasingError)
	if !ok {
		t.Error("FAIL: the returned error is not mtgmatcher.AliasingError")
		t.Errorf("%s", err.Error())
		return
	}

	dupes := alias.Probe()
	if len(dupes) != len(outCards) {
		t.Error("FAIL: wrong number of dupes returned")
		t.Errorf("%v", dupes)
		return
	}

	for i := range dupes {
		if dupes[i] != outCards[i] {
			t.Error("FAIL: incorrect duplicate returned")
			t.Errorf("%v vs %v", dupes[i], outCards[i])
		}
	}
	t.Log("PASS: Aliasing")
}
