package mtgban

import "strings"

type Normalizer struct {
	replacer *strings.Replacer
}

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

func (n *Normalizer) Normalize(str string) string {
	str = n.replacer.Replace(str)
	str = strings.TrimSpace(str)
	str = strings.ToLower(str)
	return str
}

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
