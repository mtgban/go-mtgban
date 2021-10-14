package mtgmatcher

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

var ErrDatastoreEmpty = errors.New("datastore is empty")
var ErrCardUnknownId = errors.New("unknown id")
var ErrCardDoesNotExist = errors.New("unknown card name")
var ErrCardNotInEdition = errors.New("unknown edition")
var ErrCardWrongVariant = errors.New("unknown variant")
var ErrCardMissingVariant = errors.New("missing necessary variant")
var ErrAliasing = newAliasingError()

type AliasingError struct {
	message string
	dupes   []string
}

func newAliasingError() *AliasingError {
	return &AliasingError{
		message: "aliasing detected",
	}
}

func (err *AliasingError) Error() string {
	return err.message
}

func (err *AliasingError) Probe() []string {
	return err.dupes
}

const LongestCardEver = "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental"
const NightmareCard = "The Ultimate Nightmare of Wizards of the Coast® Customer Service"

// Date since any card could be Prerelease Promo
var NewPrereleaseDate = time.Date(2014, time.September, 1, 0, 0, 0, 0, time.UTC)

// Date since BuyABox cards are found in the expansion set instead of Promos
var BuyABoxInExpansionSetsDate = time.Date(2018, time.April, 1, 0, 0, 0, 0, time.UTC)

// Date in which random promos can be in the expansion set
var PromosForEverybodyYay = time.Date(2019, time.October, 1, 0, 0, 0, 0, time.UTC)

// Date since BuyABox cards are not unique any more
var BuyABoxNotUniqueDate = time.Date(2020, time.September, 1, 0, 0, 0, 0, time.UTC)

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
	if len(fields) > 1 {
		if strings.HasPrefix(str, "Erase (Not the Urza's Legacy One)") ||
			strings.HasPrefix(str, "Hazmat Suit (Used") ||
			(Contains(str, "B.F.M.") && strings.Contains(str, "Big Furry Monster")) {
			fields[0] = fmt.Sprintf("%s (%s)", fields[0], fields[1])
			fields = append(fields[:1], fields[2:]...)
		}
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

// ExtractNumber returns as string with _first_ number below 1993 found in a
// given string, or an empty string if none could be found.
// The input string may have a single character as prefix or suffix (but not both),
// which will be ignored while determining the number portion, but preserved,
// in lowercase if suffix, or as-is if prefix.
// Any leading # characters or parenthesis are stripped away.
// Numbers starting with M are ignored because they could be confused
// with core set names.
// If a month name is detected anywhere as a single word in the input string,
// an empty string is returned, to prevent confusing a number with a date or day.
// If a rational number is provided, only the numerator part is considered.
func ExtractNumber(str string) string {
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

		num := strings.TrimLeft(field, "0")
		val, err := strconv.Atoi(num)
		if err == nil && val < 1993 {
			return num
		}
		if len(num) > 1 {
			if unicode.IsLetter(rune(num[len(num)-1])) ||
				strings.HasSuffix(num, mtgjson.SuffixSpecial) ||
				strings.HasSuffix(num, mtgjson.SuffixVariant) {
				// Strip any extra characters at the end
				trimmed := num
				if unicode.IsLetter(rune(num[len(num)-1])) {
					trimmed = num[:len(num)-1]
				} else {
					trimmed = strings.TrimSuffix(trimmed, mtgjson.SuffixSpecial)
					trimmed = strings.TrimSuffix(trimmed, mtgjson.SuffixVariant)
				}
				// Try converting to an integer number
				val, err = strconv.Atoi(trimmed)
				if err == nil && val < 1993 {
					return strings.ToLower(num)
				}
			}
			if !unicode.IsDigit(rune(num[0])) && num[0] != 'M' {
				val, err = strconv.Atoi(strings.TrimLeft(num[1:], "0"))
				if err == nil && val < 1993 {
					return num
				}
			}
		}
	}
	return ""
}

// Specialized version of ExtractNumber, suited for parsing WCD numbers
func extractWCDNumber(str, prefix string, sideboard bool) string {
	fields := strings.Fields(str)
	for _, field := range fields {
		field = strings.Replace(field, "(", "", -1)
		field = strings.Replace(field, ")", "", -1)

		if !strings.HasPrefix(field, prefix) {
			continue
		}

		num := strings.TrimPrefix(field, prefix)
		if sideboard {
			num = strings.Replace(num, "sb", "", 1)
		}
		num = strings.TrimLeft(num, "0")
		if unicode.IsLetter(rune(num[len(num)-1])) {
			num = num[:len(num)-1]
		}
		val, err := strconv.Atoi(num)
		if err == nil && val < 1993 {
			// Special way to discard any leading zeros without rebuilding manually
			field = strings.Replace(field, prefix+"00", prefix, 1)
			return strings.Replace(field, prefix+"0", prefix, 1)
		}
	}

	return ""
}

// ExtractYear returns as string with _first_ year after 1993 found in a
// given string, or an empty string if nothing is found.  It takes care
// of some special characters like parenthesis (ignored) and abbreviations
// (so '06 becomes 2006).
func ExtractYear(str string) string {
	fields := strings.Fields(str)
	for _, field := range fields {
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

// Strip input string of dollar sign and commas, convert it to a normal float
func ParsePrice(priceStr string) (float64, error) {
	priceStr = strings.Replace(priceStr, "$", "", 1)
	priceStr = strings.Replace(priceStr, ",", "", -1)
	priceStr = strings.TrimSpace(priceStr)
	return strconv.ParseFloat(priceStr, 64)
}
