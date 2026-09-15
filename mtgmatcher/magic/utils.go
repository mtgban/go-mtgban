package magic

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// ExtractWCDNumber returns a World Championship collector number, which
// carries the player's deck code and may mark a sideboard card.
func ExtractWCDNumber(str, prefix string, sideboard bool) string {
	fields := strings.FieldsSeq(str)
	for field := range fields {
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

// dropSetCodes removes from a World Championship variation every field this
// backend knows as a set code, since the listing names the set the deck's
// card was printed from before the collector number ("2001 Tom van de Logt
// 7ED 337") and ExtractNumber reads whatever comes first. A field ending in
// a lowercase "a" is kept, as a collector number may be spelled that way
// (30a).
func dropSetCodes(b *mtgmatcher.Backend, variation string) string {
	var kept []string
	for field := range strings.FieldsSeq(variation) {
		_, err := b.GetSet(field)
		if err == nil && !strings.HasSuffix(field, "a") {
			continue
		}
		kept = append(kept, field)
	}
	return strings.Join(kept, " ")
}

// IsDFCSameName reports whether a double-faced card carries the same name on
// both halves.
func IsDFCSameName(name string) bool {
	if !strings.Contains(name, " // ") {
		return false
	}
	left := name[:len(name)/2-2]
	right := name[len(name)/2+2:]
	return left == right
}
