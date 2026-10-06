package abugames

import "testing"

// TestSinglesFilter pins the one walk: each word quoted once, picture cards
// read only from the old editions.
func TestSinglesFilter(t *testing.T) {
	want := `+language:("English" OR "Italian" OR "Japanese" OR "Phyrexian") -offline_item:true -magic_features:("Artist Signed" OR "Artist Signed Case" OR "Graded" OR "Altered" OR "Miscut" OR "Printing Error") +((*:* -magic_features:"Actual Picture Card" -magic_edition:("Alpha" OR "Beta" OR "Unlimited" OR "Arabian Nights" OR "Antiquities" OR "Legends" OR "The Dark")) OR magic_edition:("Alpha" OR "Beta" OR "Unlimited" OR "Arabian Nights" OR "Antiquities" OR "Legends" OR "The Dark"))`
	got := singlesFilter()
	if got != want {
		t.Errorf("singlesFilter() =\n%s\nwant\n%s", got, want)
	}
}
