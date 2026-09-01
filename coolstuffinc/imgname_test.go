package coolstuffinc

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPreprocessImageLetter pins the image names that write a letter between
// the set code and the number. The surge foil and the pixel art of a Turtles
// card are two printings at two numbers, and reading only the digits filed
// both under the pixel art one: the cheaper listing was priced as the dearer
// printing.
func TestPreprocessImageLetter(t *testing.T) {
	path := os.Getenv("ALLPRINTINGS5_PATH")
	if path == "" {
		t.Skip("Need ALLPRINTINGS5_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		desc    string
		name    string
		edition string
		imgURL  string
		wantSet string
		wantNum string
	}{
		{
			desc:    "the letter marks the treatment, not the set",
			name:    "Ninja Pizza (Surge Foil)",
			edition: "Teenage Mutant Ninja Turtles Commander Variants",
			imgURL:  "https://s.cf.net/i/TMCS0032.jpg",
			wantSet: "TMC", wantNum: "32",
		},
		{
			// The number is the one the digits already spell, so the
			// pixel art keeps answering for itself.
			desc:    "and the plain number is left where it was",
			name:    "Ninja Pizza (Pixel Art Surge Foil)",
			edition: "Teenage Mutant Ninja Turtles Commander",
			imgURL:  "https://s.cf.net/i/TMC0093.jpg",
			wantSet: "TMC", wantNum: "93",
		},
		{
			desc:    "the storefront writes the letter in lower case too",
			name:    "Ash Barrens (Surge Foil)",
			edition: "Teenage Mutant Ninja Turtles Commander Variants",
			imgURL:  "https://s.cf.net/i/tmcs0060.jpg",
			wantSet: "TMC", wantNum: "60",
		},
		{
			desc:    "a run of letters reads the same way",
			name:    "Mountain (Ripple Foil)",
			edition: "Modern Horizons 3 Commander: Ripple Foil Variants",
			imgURL:  "https://s.cf.net/i/MH3R0503.jpg",
			wantSet: "MH3", wantNum: "503",
		},
		{
			// "UPP" names the promo pack rather than a treatment, so the
			// number behind it belongs to a set of its own and the
			// edition is left to place the card instead.
			desc:    "the promo pack keeps its own printing",
			name:    "Simulacrum Synthesizer",
			edition: "Universal Promo Pack",
			imgURL:  "https://s.cf.net/i/BIGUPP0006.jpg",
			wantSet: "Universal Promo Pack", wantNum: "",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := preprocess(tt.name, tt.edition, "", tt.imgURL)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", tt.imgURL, err)
			}
			if got.Edition != tt.wantSet || got.Variation != tt.wantNum {
				t.Errorf("preprocess(%q) = %q/%q, want %q/%q",
					tt.imgURL, got.Edition, got.Variation, tt.wantSet, tt.wantNum)
			}
		})
	}
}
