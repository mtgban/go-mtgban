package mtgmatcher

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// The errors Match and its helpers return. A caller can tell an input it
// could not place from a datastore that was never loaded, and ErrAliasing
// carries the candidates it could not choose between.
var (
	ErrDatastoreEmpty     = errors.New("datastore is empty")
	ErrCardUnknownID      = errors.New("unknown id")
	ErrCardDoesNotExist   = errors.New("unknown card name")
	ErrCardNotInEdition   = errors.New("unknown edition")
	ErrCardWrongVariant   = errors.New("unknown variant")
	ErrCardMissingVariant = errors.New("missing necessary variant")
	ErrCardWrongFinish    = errors.New("unknown finish")

	// ErrCardUnnamedFinish is a finish name the game cannot place at all, as
	// opposed to one it placed onto a finish the printing is not sold in. It is
	// not evidence of anything - a vendor spelling nobody has taught the game
	// yet reads the same as a typo - so Match falls through to the wording,
	// which is what answered before an id could name a finish.
	ErrCardUnnamedFinish = errors.New("unrecognized finish")

	ErrUnsupported = errors.New("unsupported")
	ErrAliasing    = NewAliasingError()
)

// AliasingError reports that a listing matched several printings and names
// them, so a caller can report the ambiguity rather than guess.
type AliasingError struct {
	Message string
	Dupes   []string
}

// NewAliasingError builds an aliasing error over the candidate uuids.
func NewAliasingError(duplicates ...string) *AliasingError {
	return &AliasingError{
		Message: "aliasing detected",
		Dupes:   duplicates,
	}
}

func (err *AliasingError) Error() string {
	return err.Message
}

// Probe returns the uuids that could not be told apart.
func (err *AliasingError) Probe() []string {
	return err.Dupes
}

// Cards whose names are long enough or odd enough to break naive parsing, kept
// here so tests and callers can reach for them by name.
const (
	LongestCardEver = "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental"
	NightmareCard   = "The Ultimate Nightmare of Wizards of the Coast® Customer Service"
)

// BuyABoxInExpansionSetsDate is when buy-a-box promos began appearing in the
// expansion set rather than in a promos set of their own.
var BuyABoxInExpansionSetsDate = time.Date(2018, time.April, 1, 0, 0, 0, 0, time.UTC)

// PromosForEverybodyYay is when assorted promos began appearing in the
// expansion set itself.
var PromosForEverybodyYay = time.Date(2019, time.October, 1, 0, 0, 0, 0, time.UTC)

// GRNGuilds are the guilds printed in Guilds of Ravnica.
var GRNGuilds = []string{"Boros", "Dimir", "Golgari", "Izzet", "Selesnya"}

// ARNGuilds are the guilds printed in Ravnica Allegiance.
var ARNGuilds = []string{"Azorius", "Gruul", "Orzhov", "Rakdos", "Simic"}

// Regexp for SplitVariants, an optional space and a parenthesis
var re = regexp.MustCompile(` ?\(`)

// SplitVariants returns an array of strings from the parentheses-defined fields
// commonly used to distinguish some cards across editions.
func SplitVariants(str string) []string {
	fields := re.Split(str, -1)
	for i := range fields {
		pos := strings.Index(fields[i], ")")
		if pos > 0 {
			fields[i] = fields[i][:pos]
		}
	}
	if len(fields) > 1 {
		if strings.HasPrefix(str, "Erase (Not the Urza's Legacy One)") ||
			strings.HasPrefix(str, "Hazmat Suit (Used") ||
			strings.HasPrefix(str, "B.O.B.") ||
			(Contains(str, "B.F.M.") && strings.Contains(str, "Big Furry Monster")) {
			fields[0] = fmt.Sprintf("%s (%s)", fields[0], fields[1])
			fields = append(fields[:1], fields[2:]...)
		}
		if strings.HasPrefix(str, "Dwight, Assistant (to the)") {
			fields[0] = fmt.Sprintf("%s (%s) King", fields[0], fields[1])
			fields = append(fields[:1], fields[2:]...)
		}
	}

	// This might have been lost in the split if it was after the ()
	if strings.Contains(strings.ToLower(str), "token") &&
		!strings.Contains(strings.ToLower(fields[0]), "token") {
		fields[0] += " Token"
	}

	return fields
}

var months = []string{
	"january",
	"february",
	"march",
	"april",
	"may",
	"june",
	"july",
	"august",
	"september",
	"october",
	"november",
	"december",
}

// ExtractNumber returns the _first_ collector number below 1993 found in
// the given string, lowercased, or an empty string if there is none.
//
// The number may carry a single character as prefix, or up to two as
// suffix (one letter and one special character), but not both. The rules
// for what is read and what is dropped:
//   - extra letters are ignored while locating the number, but kept in the
//     result
//   - leading # characters, zeroes and parenthesis are stripped away
//   - numbers starting with M are ignored, as they could be confused with
//     core set names
//   - a month name appearing anywhere as a single word returns an empty
//     string, so a date or a day is never read as a number
//   - for a rational number, only the numerator is considered
func ExtractNumber(str string) string {
	return extractNumber(str, 1993)
}

// ExtractNumberAny returns the first number in the input whatever its length,
// where ExtractNumber caps how many digits it will accept.
func ExtractNumberAny(str string) string {
	return extractNumber(str, math.MaxInt32)
}

func extractNumber(str string, threshold int) string {
	fields := strings.Fields(str)
	for _, field := range fields {
		for _, month := range months {
			if Equals(field, month) {
				return ""
			}
		}
	}

	fields = strings.Fields(str)
	for _, field := range fields {
		field = strings.Replace(field, "(", "", -1)
		field = strings.Replace(field, ")", "", -1)
		field = strings.Replace(field, "#", "", -1)

		if strings.Contains(field, "/") && strings.Count(field, "/") == 1 {
			subfields := strings.Split(field, "/")
			field = strings.TrimSpace(subfields[0])
		}

		// Skip any ordinal number that would be caught up in the check below
		ordinal := strings.ToLower(field)
		if strings.HasSuffix(ordinal, "th") ||
			strings.HasSuffix(ordinal, "st") ||
			strings.HasSuffix(ordinal, "nd") ||
			strings.HasSuffix(ordinal, "rd") {
			continue
		}

		// Skip tags that could be confused with set codes
		// unless it ends with "a" (ie 30A)
		_, err := GetSet(field)
		if err == nil && !strings.HasSuffix(field, "a") {
			continue
		}

		num := strings.TrimLeft(field, "0")
		val, err := strconv.Atoi(num)
		if err == nil && val < threshold {
			return num
		}
		if len(num) > 1 {
			if !unicode.IsDigit(rune(num[len(num)-1])) {
				trimmed := num

				// Remove any suffix
				index := -1
				for i, r := range num {
					if !unicode.IsDigit(r) {
						index = i
						break
					}
				}
				if index > 0 {
					trimmed = num[:index]
				}

				// Try converting to an integer number
				val, err = strconv.Atoi(trimmed)
				if err == nil && val < threshold {
					return strings.ToLower(num)
				}
			}
			if !unicode.IsDigit(rune(num[0])) && num[0] != 'M' {
				val, err = strconv.Atoi(strings.TrimLeft(num[1:], "0"))
				if err == nil && val < threshold {
					return strings.ToLower(num)
				}
			}
			// Check for the PLST numbering system
			if strings.Contains(field, "-") {
				subfields := strings.Split(field, "-")
				if len(subfields) == 2 {
					parsed := ExtractNumber(subfields[1])
					if parsed != "" {
						return subfields[0] + "-" + strings.TrimLeft(subfields[1], "0")
					}
				}
			}
		}
	}
	return ""
}

var reNumerical = regexp.MustCompile(`\d+`)

// ExtractNumberValue returns the first run of digits in the input with leading
// zeroes stripped, so that it survives strconv.Atoi.
func ExtractNumberValue(str string) string {
	return strings.TrimLeft(reNumerical.FindString(str), "0")
}

// ExtractYear returns as string with _first_ year after 1993 found in a
// given string, or an empty string if nothing is found.  It takes care
// of some special characters like parenthesis (ignored) and abbreviations
// (so '06 becomes 2006).
func ExtractYear(str string) string {
	fields := strings.FieldsSeq(str)
	for field := range fields {
		// Drop characters that could interfere with the numeric part
		field = strings.Replace(field, "(", "", -1)
		field = strings.Replace(field, ")", "", -1)
		field = strings.Replace(field, ":", "", -1)

		// Handle abbreviations, checking if year is before or after 2000
		if strings.Contains(field, "'") || strings.HasPrefix(field, "M") {
			probe := "'"
			if strings.HasPrefix(field, "M") {
				probe = "M"
			}
			yearIndex := strings.Index(field, probe)
			yearStr := field[yearIndex+1:]

			// If it not a number maybe it's a common apostrophe,
			// keep iterating over the other fields
			val, err := strconv.Atoi(strings.TrimLeft(yearStr, "0"))
			if err == nil {
				field = strings.Replace(field, probe, "20", 1)
				if val >= 93 {
					field = strings.Replace(field, "20", "19", 1)
				}
			}
		}

		num := strings.TrimLeft(field, "0")
		val, err := strconv.Atoi(num)
		if err == nil && val >= 1993 {
			return num
		}
	}
	return ""
}

// Cut splits the input string in two segments, stripping any whitespace
// before or after the cut, if present.
func Cut(in, tag string) []string {
	splits := strings.SplitN(in, tag, 2)
	if len(splits) > 1 {
		splits[0] = strings.TrimSpace(splits[0])
		splits[1] = strings.TrimSpace(tag + splits[1])
	}
	return splits
}

// ParsePrice reads a price written for people, with a dollar sign and
// thousands separators, as a float.
func ParsePrice(priceStr string) (float64, error) {
	priceStr = strings.Replace(priceStr, "$", "", 1)
	priceStr = strings.Replace(priceStr, ",", "", -1)
	priceStr = strings.TrimSpace(priceStr)
	return strconv.ParseFloat(priceStr, 64)
}

// Wrapper for the deprecated strings.Title
// abc -> Abc
// ABC -> Abc
// ordinalSuffix matches an English ordinal's letters where a title-caser has
// capitalised them. The digit in front is what makes them wrong: nothing else
// about "St" or "Th" says it should be lower case.
var ordinalSuffix = regexp.MustCompile(`([0-9])(St|Nd|Rd|Th)\b`)

// smallWords are the words an English title leaves in lower case when they
// fall inside the phrase: "Legacy of the Duelist", "Back to Duel". They are
// only small in that position - a phrase opening with one still opens with a
// capital, so "the sacred cards" stays "The Sacred Cards".
var smallWords = map[string]bool{
	"a":    true,
	"an":   true,
	"and":  true,
	"as":   true,
	"at":   true,
	"but":  true,
	"by":   true,
	"for":  true,
	"in":   true,
	"nor":  true,
	"of":   true,
	"on":   true,
	"or":   true,
	"the":  true,
	"to":   true,
	"with": true,
}

// opensPhrase reports whether a word ends on a mark that hands what follows
// a phrase of its own, where a small word is no longer inside one and opens
// its phrase capitalised: "Tag Force: The World", "Duel of Destiny - The
// Movie".
func opensPhrase(word string) bool {
	last, _ := utf8.DecodeLastRuneInString(word)
	switch last {
	case ':', ';', '.', '!', '?', '-', '\u2013', '\u2014':
		return true
	}
	return false
}

// Title capitalizes a phrase the way a title is written, replacing the
// deprecated strings.Title: abc -> Abc, and ABC -> Abc. It also puts back
// down what a plain title-caser gets wrong, the letter after a digit in an
// ordinal and the small words sitting inside a phrase rather than opening
// one.
func Title(str string) string {
	titled := cases.Title(language.English).String(str)
	// A title-caser capitalises the letter after a digit, so "1st place"
	// comes back "1St Place" and "10th anniversary" "10Th Anniversary".
	// Only an ordinal is put back down: "3d text" keeps its "3D Text",
	// where the capital is the one wanted.
	titled = ordinalSuffix.ReplaceAllStringFunc(titled, strings.ToLower)

	// A title-caser capitalises every word, including the ones a title
	// leaves down: "duel of destiny" comes back "Duel Of Destiny". The
	// first word is never one of them, whatever it is, and neither is the
	// word a mark hands a new phrase to.
	words := strings.Split(titled, " ")
	for i := 1; i < len(words); i++ {
		if opensPhrase(words[i-1]) || !smallWords[strings.ToLower(words[i])] {
			continue
		}
		words[i] = strings.ToLower(words[i])
	}
	return strings.Join(words, " ")
}

// greatestCommonDivisor exists for the multiple below.
func greatestCommonDivisor(a, b int) int {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

// leastCommonMultiple is folded over a color-balanced sheet's weights:
// scaled by it, every subsheet keeps its proportions in integers.
func leastCommonMultiple(a, b int) int {
	return a * b / greatestCommonDivisor(a, b)
}

// CardReleaseDate returns the date the card's set was released.
func (b *Backend) CardReleaseDate(cardID string) (time.Time, error) {
	co, err := b.GetUUID(cardID)
	if err != nil {
		return time.Time{}, err
	}
	releaseDate := co.OriginalReleaseDate

	if releaseDate == "" {
		set, err := b.GetSet(co.SetCode)
		if err != nil {
			return time.Time{}, err
		}
		return set.ReleaseDateTime, nil
	}

	return time.Parse("2006-01-02", releaseDate)
}

// CardReleaseDate returns the release date of the card's set, from the default
// datastore.
func CardReleaseDate(cardID string) (time.Time, error) {
	return defaultBackend.CardReleaseDate(cardID)
}

// promoHeadings are the headings storefronts file promotional printings
// under without saying which set issued them. They name no set in any game:
// the heading spans every promotional printing a game has, while a set
// carrying that name holds only the products a catalog groups there.
//
// Built on first use rather than at init: Normalize memoizes through a map
// this package sets up in its own init, and a package-level initializer
// would reach it before that runs.
var promoHeadings = sync.OnceValue(func() map[string]bool {
	out := map[string]bool{}
	for _, heading := range []string{
		"Promo", "Promos", "Promo Cards",
		"Promotional", "Promotionals", "Promotional Cards",
	} {
		out[Normalize(heading)] = true
	}
	return out
})

// IsPromoHeading reports whether an edition is one of the headings a
// storefront files promotional printings under rather than a set name.
//
// A game reading one has to decide what to do with it, and the answers
// differ: Lorcana clears it, so the heading cannot narrow to the one set
// wearing its name and drop the promos upstream files in the set they
// reprint; Riftbound keeps it as a gate on the promo-only names while
// refusing to let it choose a set. What they share is the question, which
// is why only the question lives here.
func IsPromoHeading(edition string) bool {
	return promoHeadings()[Normalize(edition)]
}
