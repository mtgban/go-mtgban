package abugames

import "testing"

// TestSinglesFilter pins the one walk: each word quoted once, the foreign
// border languages only from their editions, picture cards read only from the
// old editions, slabs only in stock and never as plain.
func TestSinglesFilter(t *testing.T) {
	want := `+(language:("English" OR "Italian" OR "Japanese" OR "Phyrexian") OR (+language:("French" OR "German" OR "Spanish" OR "Portuguese" OR "Korean" OR "Chinese Traditional") +magic_edition:("Revised" OR "4th Edition" OR "Renaissance"))) -offline_item:true -magic_features:("Artist Signed" OR "Artist Signed Case" OR "Altered" OR "Miscut" OR "Printing Error") -location_section:"Signed" +((*:* -magic_features:("Actual Picture Card" OR "Graded") -magic_edition:("Alpha" OR "Beta" OR "Unlimited" OR "Arabian Nights" OR "Antiquities" OR "Legends" OR "The Dark")) OR (+magic_edition:("Alpha" OR "Beta" OR "Unlimited" OR "Arabian Nights" OR "Antiquities" OR "Legends" OR "The Dark") -magic_features:"Graded") OR (+magic_features:"Graded" +quantity:[1 TO *]))`
	got := singlesFilter()
	if got != want {
		t.Errorf("singlesFilter() =\n%s\nwant\n%s", got, want)
	}
}
