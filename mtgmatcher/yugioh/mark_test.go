package yugioh

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestTierByMarkSlugs pins that a mark is read as the token it is, whatever
// case or spacing the datastore wrote it in.
func TestTierByMarkSlugs(t *testing.T) {
	cards := []mtgmatcher.Card{
		{UUID: "blue", Watermark: "Blue Ink"},
		{UUID: "red", Watermark: "red"},
	}
	marked := tierByMark("dl18-en002 rare blue ink", cards)
	if len(marked) != 1 || marked[0].UUID != "blue" {
		t.Errorf("tierByMark kept %v, want the blue ink alone", marked)
	}
}
