package mtgmatcher

import "testing"

type NormalizeTest struct {
	In  string
	Out string
}

var NormalizeTests = []NormalizeTest{
	{
		In:  "Lotus Bloom",
		Out: "lotubloom",
	},
	{
		In:  "Lotus Blossom",
		Out: "lotublossom",
	},
	{
		In:  "Tangle Asp",
		Out: "tangleasp",
	},
	{
		In:  "Tanglesap",
		Out: "tangleap",
	},
	{
		In:  "Ghazbán Ogre",
		Out: "ghazbanogre",
	},
	{
		In:  "Ghazban Ogress",
		Out: "ghazbanogress",
	},
	{
		In:  "Flame Slash",
		Out: "flameslash",
	},
	{
		In:  "Flame Lash",
		Out: "flamelah",
	},
	{
		In:  "Waste Land",
		Out: "waste land",
	},
	{
		In:  "Wasteland",
		Out: "wateland",
	},
	{
		In:  "  spaaaaaaace  ",
		Out: "paaaaaaace",
	},
	{
		In:  "Ætherflux reservoir",
		Out: "aetherfluxreervoir",
	},
	{
		In:  "forest b",
		Out: "foretb",
	},
	{
		In:  "them the removed",
		Out: "themremoved",
	},
	{
		In:  "Jakub Šlem",
		Out: "jakublem",
	},
	{
		In:  "Fire // Ice",
		Out: "fireice",
	},
	{
		In:  "Commit to Memory",
		Out: "commitmemory",
	},
	{
		In:  "Trial // Error",
		Out: "trialerror",
	},
	{
		In:  "Trial and Error",
		Out: "trial and error",
	},
	{
		In:  "Will o' the Wisp",
		Out: "willowip",
	},
	{
		In:  "Reverse the Polarity",
		Out: "reverse the polarity",
	},
	{
		In:  "Welcome to...",
		Out: "welcometo",
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

var SealedNormalizeTests = []NormalizeTest{
	{
		In:  "2010 Core Set Booster Box",
		Out: "magic2010booterbox",
	},
	{
		In:  "Commander (2017 Edition) - Complete Set of 4",
		Out: "commander2017diplay",
	},
	{
		In:  "Warhammer 40k Commander Deck: Collector Edition - Set of 4",
		Out: "warhammer40000deckcollectoreditiondiplay",
	},
	{
		In:  "D&D: Adventures in the Forgotten Realms - Commander Deck - Set of 4",
		Out: "adventureinforgottenrealmdeckdiplay",
	},
	{
		In:  "March of the Machine: The Aftermath - Epilogue Collector Booster Box",
		Out: "marchofmachineaftermathepiloguebooterbox",
	},
	{
		In:  "Secret Lair Drop Series Special Guest Kozyndan - The Lands",
		Out: "ecretlairkozyndanland",
	},
	{
		In:  "The Brothers' War - Bundle Gift Edition",
		Out: "brotherwargiftbundle",
	},
}

func TestSealedNormalize(t *testing.T) {
	for _, probe := range SealedNormalizeTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out := SealedNormalize(test.In)
			if out != test.Out {
				t.Errorf("FAIL %s: Expected '%s' got '%s'", test.In, test.Out, out)
				return
			}
			t.Log("PASS:", test.In)
		})
	}
}

type ContainsTest struct {
	First  string
	Second string
	Result bool
}

var ContainsTests = []ContainsTest{
	{
		First:  "Name",
		Second: "ame",
		Result: true,
	},
	{
		First:  "Object",
		Second: "Name",
		Result: false,
	},
	{
		First:  "Empty",
		Second: "",
		Result: true,
	},
	{
		First:  "",
		Second: "Test",
		Result: false,
	},
}

func TestContains(t *testing.T) {
	for _, probe := range ContainsTests {
		test := probe
		t.Run(test.First, func(t *testing.T) {
			t.Parallel()
			out := Contains(test.First, test.Second)
			if out != test.Result {
				t.Errorf("FAIL %s: Expected '%v'", test.First, test.Result)
				return
			}
			t.Log("PASS:", test.First)
		})
	}
}

var SealedContainsTests = []ContainsTest{
	{
		First:  "Secret Lair Drop Series: Read the Fine Print Edition",
		Second: "Read the Fine Print Foil",
		Result: false,
	},
	{
		First:  "Secret Lair Drop Series: Read the Fine Print Foil Edition",
		Second: "Read the Fine Print Foil Edition",
		Result: true,
	},
	{
		First:  "Secret Lair Drop Series: Read the Fine Print Foil Etched Edition",
		Second: "Read the Fine Print Foil",
		Result: false,
	},
}

func TestSealedContains(t *testing.T) {
	for _, probe := range SealedContainsTests {
		test := probe
		t.Run(test.First, func(t *testing.T) {
			t.Parallel()
			out := SealedContains(test.First, test.Second)
			if out != test.Result {
				t.Errorf("FAIL %s: Expected '%v'\nTested %s -> %s", test.First, test.Result, SealedNormalize(test.First), SealedNormalize(test.Second))
				return
			}
			t.Log("PASS:", test.First)
		})
	}
}
