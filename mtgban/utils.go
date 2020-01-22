package mtgban

import (
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgjson"
)

type LogCallbackFunc func(format string, a ...interface{})

type SetCheckFunc func(set mtgjson.Set) bool
type NumberCheckFunc func(set mtgjson.Set, card mtgjson.Card) bool

var NewPrereleaseDate = time.Date(2014, time.September, 1, 0, 0, 0, 0, time.UTC)

// Type to normalize the various name differences across vendors
type Normalizer struct {
	replacer *strings.Replacer
}

// NewNormalizer initializes a Normalizer with default rules.
func NewNormalizer() *Normalizer {
	return &Normalizer{
		replacer: strings.NewReplacer(
			// Quotes and commas and whatnot
			"''", "",
			"“", "",
			"”", "",
			"\"", "",
			"'", "",
			"-", "",
			"’", "",
			",", "",
			"®", "",
			":", "",
			"~", "",

			// Accented characters
			"â", "a",
			"á", "a",
			"à", "a",
			"é", "e",
			"í", "i",
			"ö", "o",
			"ú", "u",
			"û", "u",

			// Almost everbody spells aether differently
			"AEther", "Aether",
			"Æther", "Aether",

			// Common typos
			" s ", "s ",
		)}
}

// Normalize uses the rules defined in NewNormalized to replace uncommon
// elements of card names, producing an easy to compare string.
func (n *Normalizer) Normalize(str string) string {
	str = n.replacer.Replace(str)
	str = strings.TrimSpace(str)
	str = strings.ToLower(str)
	return str
}

// SplitVariants returns an array of strings from the parentheses-defined fields
// commonly used to distinguish some cards across editions.
func SplitVariants(str string) []string {
	fields := strings.Split(str, " (")
	for i := range fields {
		pos := strings.Index(fields[i], ")")
		if pos > 0 {
			fields[i] = fields[i][:pos]
		}
	}
	return fields
}
