package vegassingles

import "testing"

// TestUniqueCopy pins which listings name one particular card rather than a
// printing. The storefront writes the copy's own id behind the word, and that
// id is the whole discriminator: Magic has a set called "Unique and
// Miscellaneous Promos", so every product in it says the word and none of
// them may be refused for it.
func TestUniqueCopy(t *testing.T) {
	for _, display := range []string{
		"Ahri - Inquisitive (Signature) (227*/221) - Spiritforged Foil (Unique) (011842)",
		"Ahri - Nine-Tailed Fox (Overnumbered) (303/298) - Origins Foil Unique 81136",
		"Kha'Zix - Voidreaver (Signature) (236*/219) - Unleashed Foil Unique (93094)",
		"Teemo - Swift Scout (Alternate Art) (263a/298) - Riftbound Promotional Cards Foil (Unique) 54353",
		"Teemo - Swift Scout (Signature) (307*/298) - Unique (390545)",
	} {
		if _, err := preprocess(VSProduct{DisplayName: display}, GameRiftbound); err == nil {
			t.Errorf("%s: read as a printing, want refused as one copy", display)
		}
	}

	for _, display := range []string{
		"Behold the Sinister Six! (UMP-002) - Unique and Miscellaneous Promos Foil",
		"Bird Token (JP FIN Exclusive) (UMP-010) - Unique and Miscellaneous Promos",
		"Day of Black Sun (UMP-002) - Unique and Miscellaneous Promos",
		// The ordinary listing standing beside a refused copy is the one
		// that must keep pricing the printing.
		"Vayne - Hunter (Signature) (223*/221) - Spiritforged Foil",
	} {
		if uniqueCopy.MatchString(display) {
			t.Errorf("%s: read as one copy, want read as a printing", display)
		}
	}
}
