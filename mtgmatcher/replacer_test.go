package mtgmatcher

import "testing"

type NormalizeTest struct {
	In  string
	Out string
}

var NormalizeTests = []NormalizeTest{
	NormalizeTest{
		In:  "Lotus Bloom",
		Out: "lotubloom",
	},
	NormalizeTest{
		In:  "Lotus Blossom",
		Out: "lotublossom",
	},
	NormalizeTest{
		In:  "Tangle Asp",
		Out: "tangleasp",
	},
	NormalizeTest{
		In:  "Tanglesap",
		Out: "tangleap",
	},
	NormalizeTest{
		In:  "Ghazbán Ogre",
		Out: "ghazbanogre",
	},
	NormalizeTest{
		In:  "Ghazban Ogress",
		Out: "ghazbanogress",
	},
	NormalizeTest{
		In:  "Flame Slash",
		Out: "flameslash",
	},
	NormalizeTest{
		In:  "Flame Lash",
		Out: "flamelah",
	},
	NormalizeTest{
		In:  "Waste Land",
		Out: "waste land",
	},
	NormalizeTest{
		In:  "Wasteland",
		Out: "wateland",
	},
	NormalizeTest{
		In:  "  spaaaaaaace  ",
		Out: "paaaaaaace",
	},
	NormalizeTest{
		In:  "Ætherflux reservoir",
		Out: "aetherfluxreervoir",
	},
	NormalizeTest{
		In:  "forest b",
		Out: "foretb",
	},
	NormalizeTest{
		In:  "them the removed",
		Out: "themremoved",
	},
	NormalizeTest{
		In:  "Jakub Šlem",
		Out: "jakublem",
	},
}

func TestNormalize(t *testing.T) {
	for _, probe := range NormalizeTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := Normalize(test.In)
			if out != test.Out {
				t.Errorf("FAIL %s: Expected '%s' got '%s'", test.In, test.Out, out)
				return
			}
			t.Log("PASS:", test.In)
		})
	}
}
