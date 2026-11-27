package bazaarofmagic

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

func TestPreprocessMatch(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		set    Set
		title  string
		finish string
		want   string // SET#number, foil marked with *
		cond   mtgban.Condition
	}{
		{Set{"4ed", "4th Edition"}, "Strip Mine (EX)", "0", "4ED#363", mtgban.SP},
		{Set{"4ed", "4th Edition"}, "Lightning Bolt (BB) (Chinese)", "0", "4BBZHT#208", mtgban.NM},
		{Set{"4ed", "4th Edition"}, "Library of Leng (BB) (Japanese)", "0", "4BB#333", mtgban.NM},
		{Set{"10e", "10th Edition"}, "Forest (3) (foil)", "1", "10E#382*", mtgban.NM},
		{Set{"10e", "10th Edition"}, "Goblin token (1/1)", "0", "T10E#4", mtgban.NM},
		{Set{"ctrk", "Star Trek Variants"}, "A Good Day to Die (#438) (Klingon) (foil) (showcase)", "1", "TRK#438*", mtgban.NM},
		{Set{"ctrk", "Star Trek Variants"}, "Forest (#486) (surge foil) (full art)", "6", "TRK#486*", mtgban.NM},
		{Set{"sds", "Star Trek Stardates Cards"}, "Sheoldred, the Apocalypse (borderless)", "0", "SDS#11", mtgban.NM},
		// The nth art in collector number order, accents folded.
		{Set{"all", "Alliances"}, "Agent of Stromgald (2)", "0", "ALL#64b", mtgban.NM},
		{Set{"all", "Alliances"}, "Lim-Dul's High Guard (2)", "0", "ALL#55b", mtgban.NM},
		{Set{"avr", "Avacyn Restored"}, "Spirit token (2) (1/1)", "0", "TAVR#4", mtgban.NM},
		{Set{"dd3_jvc", "Anthology: Jace vs. Chandra"}, "Island (2)", "0", "JVC#31", mtgban.NM},
		// The full-art land is the plain number, its sibling the "a".
		{Set{"bfz", "Battle for Zendikar"}, "Forest (#272b) (full art)", "0", "BFZ#272", mtgban.NM},
	}
	for _, test := range tests {
		l, err := preprocess(b, test.set, test.title, test.finish)
		if err != nil {
			t.Errorf("%s: %v", test.title, err)
			continue
		}
		cardID, err := b.Match(l.card)
		if err != nil {
			t.Errorf("%s: %v", test.title, err)
			continue
		}
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatal(err)
		}
		got := co.SetCode + "#" + co.Number
		if co.Foil {
			got += "*"
		}
		if got != test.want || l.cond != test.cond {
			t.Errorf("%s: got %s %s, want %s %s", test.title, got, l.cond, test.want, test.cond)
		}
	}
}

func TestPreprocessSkipsSigned(t *testing.T) {
	b := realDatastore(t)
	_, err := preprocess(b, Set{"4ed", "4th Edition"}, "Zombie Master (signed)", "0")
	if !errors.Is(err, errSkip) {
		t.Errorf("got %v, want errSkip", err)
	}
}
