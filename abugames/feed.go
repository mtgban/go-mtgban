package abugames

import "strings"

var singlesLanguages = []string{"English", "Italian", "Japanese", "Phyrexian"}

// oldEditions are walked with their picture-card copies, which are most of
// ABU's stock there; elsewhere a picture card is a duplicate listing.
var oldEditions = []string{"Alpha", "Beta", "Unlimited", "Arabian Nights", "Antiquities", "Legends", "The Dark"}

// oneOffLabels mark a single copy listed on its own, by serial. Its price is
// not the printing's, so none of them is read.
var oneOffLabels = []string{"Artist Signed", "Artist Signed Case", "Graded", "Altered", "Miscut", "Printing Error"}

// anyOf is a Solr group matching any of words, each quoted once.
func anyOf(words []string) string {
	quoted := make([]string, 0, len(words))
	for _, word := range words {
		quoted = append(quoted, `"`+word+`"`)
	}
	return "(" + strings.Join(quoted, " OR ") + ")"
}

// singlesFilter selects every singles listing read, for one walk, inside
// the singles catalog abuBaseURL already holds it to.
func singlesFilter() string {
	old := anyOf(oldEditions)
	return `+language:` + anyOf(singlesLanguages) +
		` -offline_item:true` +
		` -magic_features:` + anyOf(oneOffLabels) +
		` +((*:* -magic_features:"Actual Picture Card" -magic_edition:` + old + `) OR magic_edition:` + old + `)`
}
