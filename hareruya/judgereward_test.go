package hareruya

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestJudgeReward pins the set a judge reward is filed in against the set the
// storefront files it under. A judge printing is a set of its own and the
// title names the set the card was first printed in, tagged "-P"; the tag is
// dropped with every other frame tag, so the base set stood and deleted the
// printing the wording was naming. Each pair below is one card sold twice,
// the reward and the original, at prices that are not each other's.
func TestJudgeReward(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the judge reward suite")
	}

	for _, test := range []struct {
		desc    string
		title   string
		wantSet string
		wantNum string
	}{
		{
			desc:    "the judge foil, which the base set was answering",
			title:   "買取：【Foil】《ガイアの揺籃の地/Gaea's Cradle》(ジャッジ褒賞)[USG-P] 土地",
			wantSet: "JGP", wantNum: "3",
		},
		{
			desc:    "and the card it was printed beside",
			title:   "買取：《ガイアの揺籃の地/Gaea's Cradle》[USG] 土地R",
			wantSet: "USG", wantNum: "321",
		},
		{
			desc:    "a reward whose set is not the base set's own promos",
			title:   "買取：【Foil】《記憶の欠落/Memory Lapse》(ジャッジ褒賞)[6ED-P] 青",
			wantSet: "G99", wantNum: "1",
		},
		{
			desc:    "and its original",
			title:   "買取：《記憶の欠落/Memory Lapse》[6ED] 青C",
			wantSet: "6ED", wantNum: "81",
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			in, err := preprocess(test.title)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", test.title, err)
			}
			id, err := mtgmatcher.Match(in)
			if err != nil {
				t.Fatalf("Match(%q) = %v", in, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != test.wantSet || co.Number != test.wantNum {
				t.Errorf("Match(%q) = %s|%s, want %s|%s", in, co.SetCode, co.Number, test.wantSet, test.wantNum)
			}
		})
	}
}
