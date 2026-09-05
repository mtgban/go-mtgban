package mtgmatcher

import "testing"

type NormalizeTest struct {
	In  string
	Out string
}

var NormalizeTests = []NormalizeTest{
	{
		In:  "Lotus Bloom",
		Out: "lotusbloom",
	},
	{
		In:  "Lotus Blossom",
		Out: "lotusblossom",
	},
	{
		In:  "Tangle Asp",
		Out: "tangleasp",
	},
	{
		In:  "Tanglesap",
		Out: "tanglesap",
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
		Out: "flamelash",
	},
	{
		In:  "Waste Land",
		Out: "waste land",
	},
	{
		In:  "Wasteland",
		Out: "wasteland",
	},
	{
		In:  "  spaaaaaaace  ",
		Out: "spaaaaaaace",
	},
	{
		In:  "Ætherflux reservoir",
		Out: "aetherfluxreservoir",
	},
	{
		In:  "forest b",
		Out: "forestb",
	},
	{
		// An article is a word of the name like any other
		In:  "them the removed",
		Out: "themtheremoved",
	},
	{
		In:  "Jakub Šlem",
		Out: "jakubslem",
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
		// However it is typeset, and the card is Will-o'-the-Wisp
		In:  "Will o' the Wisp",
		Out: "willothewisp",
	},
	{
		In:  "Will-o'-the-Wisp",
		Out: "willothewisp",
	},
	{
		// Kept its spaces while a rule shielded it from the article step
		In:  "Reverse the Polarity",
		Out: "reversethepolarity",
	},
	{
		In:  "Welcome to...",
		Out: "welcometo",
	},
	{
		In:  "Henzie &quot;Toolbox&quot; Torre",
		Out: "henzietoolboxtorre",
	},
	{
		In:  "Jeong Jeong, the Deserter",
		Out: "jeongjeongthedeserter",
	},
	{
		In:  "Jeong Jeong's Deserters",
		Out: "jeongjeongsdeserters",
	},
	{
		In:  "Sword of War and Peace",
		Out: "swordofwarpeace",
	},
	{
		In:  "Word of War",
		Out: "wordofwar",
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
