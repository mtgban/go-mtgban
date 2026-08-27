package yugioh

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestDecoratedRarity pins the tier a set prints only decorated. The Rarity
// Collections sell a Prismatic Collector's Rare and no plain one, so a
// listing saying "Collector's Rare" has still said which of the seven
// printings at its number it means - while a set that prints a tier both
// ways must keep answering the plain wording with the plain printing.
func TestDecoratedRarity(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc       string
		in         mtgmatcher.InputCard
		wantSet    string
		wantNumber string
		wantRarity string
		wantErr    string
	}{
		{
			desc:    "a tier the set prints only decorated is named by its tail",
			in:      mtgmatcher.InputCard{Name: "Alpha, the Master of Beasts", Edition: "25th Anniversary Rarity Collection", Variation: "Collector's Rare"},
			wantSet: "RA01", wantNumber: "RA01-EN022", wantRarity: "Prismatic Collector's Rare",
		},
		{
			desc:    "and so is the other one",
			in:      mtgmatcher.InputCard{Name: "Alpha, the Master of Beasts", Edition: "25th Anniversary Rarity Collection", Variation: "Ultimate Rare"},
			wantSet: "RA01", wantNumber: "RA01-EN022", wantRarity: "Prismatic Ultimate Rare",
		},
		{
			desc:    "the decoration spelled out still names it",
			in:      mtgmatcher.InputCard{Name: "Alpha, the Master of Beasts", Edition: "25th Anniversary Rarity Collection", Variation: "Prismatic Collector's Rare"},
			wantSet: "RA01", wantNumber: "RA01-EN022", wantRarity: "Prismatic Collector's Rare",
		},
		{
			desc:    "a tier printed plain at the same number keeps the plain printing",
			in:      mtgmatcher.InputCard{Name: "Alpha, the Master of Beasts", Edition: "25th Anniversary Rarity Collection", Variation: "Secret Rare"},
			wantSet: "RA01", wantNumber: "RA01-EN022", wantRarity: "Secret Rare",
		},
		{
			desc:    "a number printing a tier both ways answers the plain wording plainly",
			in:      mtgmatcher.InputCard{Name: "Diabellstar the Black Witch", Edition: "Limited Pack World Championship 2025", Variation: "25LP-EN001 Ultra Rare"},
			wantSet: "25LP", wantNumber: "25LP-EN001", wantRarity: "Ultra Rare",
		},
		{
			desc:    "and the decorated wording at that same number picks the decorated one",
			in:      mtgmatcher.InputCard{Name: "Diabellstar the Black Witch", Edition: "Limited Pack World Championship 2025", Variation: "25LP-EN001 Emblazoned Ultra Rare"},
			wantSet: "25LP", wantNumber: "25LP-EN001", wantRarity: "Emblazoned Ultra Rare",
		},
		{
			desc:    "where every printing at the number is decorated, the tail decides",
			in:      mtgmatcher.InputCard{Name: "Exodia the Forbidden One", Edition: "Limited Pack World Championship 2025", Variation: "25LP-EN000 Secret Rare"},
			wantSet: "25LP", wantNumber: "25LP-EN000", wantRarity: "Emblazoned Secret Rare",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Match(%v) = %v, want %q", tt.in, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber || co.Rarity != tt.wantRarity {
				t.Errorf("Match(%v) = %s|%s|%s, want %s|%s|%s", tt.in,
					co.SetCode, co.Number, co.Rarity, tt.wantSet, tt.wantNumber, tt.wantRarity)
			}
		})
	}
}
