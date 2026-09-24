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
