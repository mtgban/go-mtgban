package coolstuffinc

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// tier writes one rarity checkbox the way the advanced search prints it.
func tier(code, label string) string {
	return `<li><input type="checkbox" id="Rarity` + code + `" name="f[Rarity][]" value="` + code +
		`" /> <label for="Rarity` + code + `">` + label + `</label></li>`
}

// TestSinglesRarities pins which tiers a singles search asks for. The
// storefront answers a rarity it does not recognise with nothing rather than
// everything, so a tier left out is a card lost, while a sealed tier let in
// only costs a row the condition parser refuses.
func TestSinglesRarities(t *testing.T) {
	for _, tt := range []struct {
		desc  string
		tiers []string
		want  []string
	}{
		{
			"magic keeps its card tiers and drops the two sealed ones",
			[]string{tier("C", "Common"), tier("MR", "Mythic Rare"), tier("P", "Pack"), tier("B", "Box")},
			[]string{"C", "MR"},
		},
		{
			"yugioh keeps the premium tiers magic never had",
			[]string{tier("MOR", "Mosaic Rare"), tier("QCR", "Quarter Century Secret Rare"), tier("B", "Box")},
			[]string{"MOR", "QCR"},
		},
		{
			"a tier the storefront adds later is asked for unprompted",
			[]string{tier("C", "Common"), tier("XYZ", "Some Rarity Nobody Has Printed Yet")},
			[]string{"C", "XYZ"},
		},
		{
			"the sealed label is matched however it is cased",
			[]string{tier("C", "Common"), tier("B", "BOX"), tier("P", "pack")},
			[]string{"C"},
		},
		{
			"a checkbox with no value names no tier",
			[]string{tier("", ""), tier("R", "Rare")},
			[]string{"R"},
		},
		{
			"a fieldset with nothing in it asks for everything",
			nil,
			nil,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			html := `<fieldset><h2 class="mb10"><b>Rarity</b></h2><ul>` +
				strings.Join(tt.tiers, "") + `</ul></fieldset>`
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
			if err != nil {
				t.Fatalf("parsing the fixture: %v", err)
			}

			got := singlesRarities(doc.Find("fieldset"))
			if len(got) != len(tt.want) {
				t.Fatalf("singlesRarities() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("singlesRarities()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
