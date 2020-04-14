package mtgban

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgdb"
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

func GetExchangeRate(currency string) (float64, error) {
	resp, err := http.Get("https://api.exchangeratesapi.io/latest?base=" + currency)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var reply struct {
		Rates struct {
			USD float64 `json:"USD"`
		} `json:"rates"`
	}
	err = json.NewDecoder(resp.Body).Decode(&reply)
	if err != nil {
		return 0, err
	}

	return reply.Rates.USD, nil
}

func Card2card(in *mtgdb.Card) Card {
	return Card{
		Id:   in.Id,
		Name: in.Name,
		Set:  in.Edition,
		Foil: in.Foil,
	}
}
