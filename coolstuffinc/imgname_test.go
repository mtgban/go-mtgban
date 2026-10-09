package coolstuffinc

import (
	"slices"
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPreprocessImageLetter pins the image names that write a letter between
// the set code and the number. The surge foil and the pixel art of a Turtles
// card are two printings at two numbers, and reading only the digits files
// both under the pixel art one, pricing the cheaper listing as the dearer
// printing.
func TestPreprocessImageLetter(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

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
			got, err := preprocess(b, tt.name, tt.edition, "", tt.imgURL)
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

// TestPreprocessImageStem pins the image names that put the card's own name
// beside the number, which the set-code-first reading never parses: the
// listing then falls back to the wording, which cannot tell two arts of one
// card apart.
func TestPreprocessImageStem(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	for _, tt := range []struct {
		desc    string
		name    string
		edition string
		variant string
		imgURL  string
		wantSet string
		wantNum string
	}{
		{
			desc:    "the name leads and the shelf gives the set",
			name:    "Boros Guildgate",
			edition: "Guilds of Ravnica",
			imgURL:  "https://s.cf.net/i/BorosGuildgate244.jpg",
			wantSet: "GRN", wantNum: "244",
		},
		{
			// Product-named and set-named images alike carry the base
			// card's number, which the promo pack's own row is not at.
			desc:    "a promo pack image is not the base card's number",
			name:    "Atsushi, the Blazing Sky",
			edition: "Universal Promo Pack",
			variant: "Silver Planeswalker Symbol",
			imgURL:  "https://s.cf.net/i/NEO134.jpg",
			wantSet: "Universal Promo Pack", wantNum: "Silver Planeswalker Symbol",
		},
		{
			desc:    "but a curated promo pack stem names its printing",
			name:    "Terror of the Peaks",
			edition: "Universal Promo Pack",
			variant: "Silver Planeswalker Symbol",
			imgURL:  "https://s.cf.net/i/386443.jpg",
			wantSet: "POTJ", wantNum: "149p",
		},
		{
			desc:    "a name the shelf spells without its accent",
			name:    "Tura Kennerud, Skyknight",
			edition: "Dominaria United: Variants",
			variant: "Stained Glass Frame",
			imgURL:  "https://s.cf.net/i/DMU323.jpg",
			wantSet: "DMU", wantNum: "323",
		},
		{
			desc:    "the name trails the set and number",
			name:    "Roil Eruption",
			edition: "Promo",
			imgURL:  "https://s.cf.net/i/znr389roileruption.jpg",
			wantSet: "ZNR", wantNum: "389",
		},
		{
			// The bare number is the intro pack's; only the stem carrying
			// the stamped printing's "s" may name the prerelease one, so
			// this one is left to the wording.
			desc:    "a prerelease stem without the stamp's letter is left alone",
			name:    "Ivorytusk Fortress",
			edition: "Prerelease Promo",
			variant: "Khans of Tarkir Prerelease Promo",
			imgURL:  "https://s.cf.net/i/pktk179ivorytuskfortress.jpg",
			wantSet: "Prerelease Promo", wantNum: "Khans of Tarkir Prerelease Promo",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := preprocess(b, tt.name, tt.edition, tt.variant, tt.imgURL)
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

// TestPreprocessImageExtendedArt pins the variants shelves whose image names
// the base card's number: when the listing says extended art and the set
// holds one, the image is not trusted; when the set holds none, it stays the
// only evidence there is.
func TestPreprocessImageExtendedArt(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	t.Run("the set holds an extended art printing", func(t *testing.T) {
		card, err := preprocess(b, "Copy Land", "Modern Horizons 3 Commander: Variants", "Extended Art Frame", "https://s.cf.net/i/M3C0099.jpg")
		if err != nil {
			t.Fatal(err)
		}
		id, err := b.Match(card)
		if err != nil {
			t.Fatalf("Match(%+v) = %v", card, err)
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if !isExtendedArt(co.Card) {
			t.Errorf("landed %s %s, which is not extended art", co.SetCode, co.Number)
		}
	})

	t.Run("the set holds none", func(t *testing.T) {
		card, err := preprocess(b, "Detention Chariot", "Aetherdrift: Variants", "Extended Art Frame", "https://s.cf.net/i/DFT0294.jpg")
		if err != nil {
			t.Fatal(err)
		}
		if card.Edition != "DFT" || card.Variation != "294" {
			t.Errorf("preprocess() = %q/%q, want the image's DFT/294", card.Edition, card.Variation)
		}
	})
}

// TestPreprocessFinalFantasyBackground pins the two-sided Final Fantasy
// variants, whose notes name only the colour behind the art: the base card
// answers unless the number or the borderless treatment is asked for.
func TestPreprocessFinalFantasyBackground(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	t.Run("a product-named image takes the borderless treatment", func(t *testing.T) {
		card, err := preprocess(b, "Jill, Shiva's Dominant // Shiva, Warden of Ice", "Final Fantasy Variants", "XVI in Blue Background", "https://s.cf.net/i/414814.jpg")
		if err != nil {
			t.Fatal(err)
		}
		id, err := b.Match(card)
		if err != nil {
			t.Fatalf("Match(%+v) = %v", card, err)
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(co.PromoTypes, "borderless") {
			t.Errorf("landed %s %s, which is not borderless", co.SetCode, co.Number)
		}
	})

	t.Run("a numbered image keeps its number past a ds prefix", func(t *testing.T) {
		card, err := preprocess(b, "Cecil, Dark Knight // Cecil, Redeemed Paladin", "Final Fantasy Variants", "IV in Purple Background", "https://s.cf.net/i/dsfin0380.jpg")
		if err != nil {
			t.Fatal(err)
		}
		if card.Edition != "FIN" || card.Variation != "380" {
			t.Errorf("preprocess() = %q/%q, want FIN/380", card.Edition, card.Variation)
		}
	})
}
