package coolstuffinc

import "testing"

// TestBuylistVariation pins what the buylist tells the matcher about a
// printing beyond its name. The qualifier lives in a free-text note the sell
// listing spends its variation on, and the numbers written in that note name
// other products rather than this one.
func TestBuylistVariation(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		product CSIPriceEntry
		want    string
	}{
		{"the number alone when there is no note",
			CSIPriceEntry{Number: "001/072"}, "001/072"},
		{"the note carries the qualifier",
			CSIPriceEntry{Number: "074/217", Notes: "Love Ball Foil"}, "074/217 Love Ball Foil"},
		{"a stamped promo says so in the note",
			CSIPriceEntry{Number: "SM198", Notes: "Detective Pikachu Stamped"}, "SM198 Detective Pikachu Stamped"},
		{"the note's own numbers name other products",
			CSIPriceEntry{Number: "SVP107", Notes: "Can be Pikachu 2, 19, 41, or 45"}, "SVP107 Can be Pikachu or"},
		{"a note and no number of its own",
			CSIPriceEntry{Notes: "Prerelease"}, "Prerelease"},
		{"a note describing a reprint names the other set",
			CSIPriceEntry{Number: "24/53", Notes: "25th Anniversary Stamp WOTC Black Star Promo Reprint"}, "24/53"},
		{"and so does one that names a real set",
			CSIPriceEntry{Number: "4/102", Notes: "25th Anniversary Stamp Base Set Reprint"}, "4/102"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := buylistVariation(tt.product); got != tt.want {
				t.Errorf("buylistVariation(%q, %q) = %q, want %q", tt.product.Number, tt.product.Notes, got, tt.want)
			}
		})
	}
}
