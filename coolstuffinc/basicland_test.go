package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPreprocessBasicLand pins the shelf-lettered and letterless basic land
// shape ("Island A", "Mountain") that coolstuffinc.go's error branch
// swallows without logging: the printing its product image names, when that
// names exactly one printing of the basic in the resolved set.
func TestPreprocessBasicLand(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	for _, tt := range []struct {
		desc, name, edition, imgURL, wantSet, wantNum string
	}{
		{
			desc:    "a lettered basic, the set code and number in the image",
			name:    "Forest A",
			edition: "Avacyn Restored",
			imgURL:  "https://res.cloudinary.com/csicdn/image/upload/c_pad,fl_lossy,h_186,q_auto,w_186/v1/Images/Products/mtg%20art/Avacyn%20Restored/full/AVR242.jpg",
			wantSet: "AVR", wantNum: "242",
		},
		{
			desc:    "a letterless basic, the basic's own name and the number",
			name:    "Plains",
			edition: "Hour of Devastation",
			imgURL:  "https://res.cloudinary.com/csicdn/image/upload/c_pad,fl_lossy,h_186,q_auto,w_186/v1/Images/Products/mtg%20art/Hour%20of%20Devastation/full/Plains190.jpg",
			wantSet: "HOU", wantNum: "190",
		},
		{
			// The image names no number, so the letter's position among the
			// set's plain numbers is all there is to read.
			desc:    "a lettered basic with no number in its image",
			name:    "Forest B",
			edition: "Magic 2014",
			imgURL:  "https://s.cf.net/i/foresta.jpg",
			wantSet: "M14", wantNum: "247",
		},
		{
			// The full-art Forest is its own unlettered product, so A is
			// the first of the regular arts and not the full-art 254.
			desc:    "a lettered basic in a set holding a full-art printing",
			name:    "Forest A",
			edition: "Amonkhet",
			imgURL:  "https://s.cf.net/i/forestA.jpg",
			wantSet: "AKH", wantNum: "267",
		},
		{
			// Matched by the letter alone, Battle Royale would answer with
			// a different art than the one CSI letters.
			desc:    "a basic on a shelf lettered in number order",
			name:    "Swamp C",
			edition: "Battle Royale",
			imgURL:  "https://s.cf.net/i/SwampCBRBa.jpg",
			wantSet: "BRB", wantNum: "135",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := preprocess(b, tt.name, tt.edition, "", tt.imgURL)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", tt.name, err)
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%+v) = %v", card, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("%s [%s] -> %s %s, want %s %s",
					tt.name, tt.edition, co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
		})
	}

	t.Run("a full-art basic whose image names no number of its own", func(t *testing.T) {
		card, err := preprocess(b, "Mountain A", "Battle for Zendikar", "", "https://res.cloudinary.com/csicdn/image/upload/c_pad,fl_lossy,h_186,q_auto,w_186/v1/Images/Products/mtg%20art/Battle%20for%20Zendikar/full/MountainA.jpg")
		if err != nil {
			t.Fatalf("preprocess() = %v", err)
		}
		_, err = b.Match(card)
		if err == nil {
			t.Errorf("Match(%+v) unexpectedly succeeded, want an error", card)
		}
	})

	// BFZ 255 (full-art) and 255a (non-full-art) share this stem; nothing
	// says which one "Island A" is, so it must refuse rather than guess.
	t.Run("a basic whose number also names a lettered sibling", func(t *testing.T) {
		card, err := preprocess(b, "Island A", "Battle for Zendikar", "", "https://res.cloudinary.com/csicdn/image/upload/c_pad,fl_lossy,h_186,q_auto,w_186/v1/Images/Products/mtg%20art/Battle%20for%20Zendikar/full/255.jpg")
		if err != nil {
			t.Fatalf("preprocess() = %v", err)
		}
		_, err = b.Match(card)
		if err == nil {
			t.Errorf("Match(%+v) unexpectedly succeeded, want an error", card)
		}
	})
}

func TestBasicLandOrdinal(t *testing.T) {
	for _, tt := range []struct {
		desc   string
		nums   []string
		letter string
		want   string
	}{
		{"second of the plain numbers", []string{"247", "246", "248", "246"}, "B", "247"},
		{"no letter", []string{"1", "2"}, "", ""},
		{"a letter past the last number", []string{"1", "2"}, "C", ""},
		{"a lettered sibling number", []string{"265", "265a", "266"}, "A", ""},
		{"a star sibling number", []string{"101", "102", "102\u2605"}, "A", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := basicLandOrdinal(tt.nums, tt.letter)
			if got != tt.want {
				t.Errorf("basicLandOrdinal(%q, %q) = %q, want %q", tt.nums, tt.letter, got, tt.want)
			}
		})
	}
}
