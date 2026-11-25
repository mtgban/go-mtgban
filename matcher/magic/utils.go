package magic

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/matcher"
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
		// Nothing past the deck code, zeros and sideboard mark is no number
		if num == "" {
			continue
		}
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
func dropSetCodes(b *matcher.Backend, variation string) string {
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

// parseWorldChampPrefix returns the deck code for the player named in the
// text, and whether the card was in their sideboard.
func parseWorldChampPrefix(variation string) (string, bool) {
	players := map[string]string{
		"Aeo Paquette":         "ap",
		"Alex Borteh":          "ab",
		"Antoine Ruel":         "ar",
		"Ben Rubin":            "br",
		"Bertrand Lestree":     "bl",
		"Brian Hacker":         "bh",
		"Brian Kibler":         "bk",
		"Brian Selden":         "bs",
		"Brian Seldon":         "bs",
		"Carlos Romao":         "cr",
		"Daniel Zink":          "dz",
		"Dave Humpherys":       "dh",
		"Eric Tam":             "et",
		"Gabriel Nassif":       "gn",
		"George Baxter":        "gb",
		"Jakub Slemr":          "js",
		"Jan Tomcani":          "jt",
		"Janosch Kuhn":         "jk",
		"Janosch Kuehn":        "jk",
		"Jon Finkel":           "jf",
		"Julien Nuijten":       "jn",
		"Kai Budde":            "kb",
		"Leon Lindback":        "ll",
		"Leon Linback":         "ll",
		"Manuel Bevand":        "mb",
		"Mark Justice":         "mj",
		"Mark Le Pine":         "mlp",
		"Matt Linde":           "ml",
		"Michael Locanto":      "ml",
		"Michael Loconto":      "ml",
		"Nicolas Labarre":      "nl",
		"Paul McCabe":          "pm",
		"Peer Kroger":          "pk",
		"Preston Poulter":      "pp",
		"Randy Buehler":        "rb",
		"Raphael Levy":         "rl",
		"Shawn Regnier":        "shr",
		"Shawn Hammer Regnier": "shr",
		"Sim Han How":          "shh",
		"Svend Geertsen":       "sg",
		"Tom van de Logt":      "tvdl",
		"Wolfgang Eder":        "we",
	}

	// We cannot use HasPrefix for the second check due to mlp/ml aliasing
	variation = strings.ToLower(variation)
	idx := strings.IndexFunc(variation, func(c rune) bool {
		return unicode.IsDigit(c)
	})
	// Iterate over the player list and check if their name or their initials are present
	for player, tag := range players {
		if matcher.Contains(variation, player) || (idx > -1 && variation[:idx] == tag) {
			sb := strings.Contains(variation, "sb") || strings.Contains(variation, "sideboard")
			return tag, sb
		}
	}
	return "", false
}

// commanderKeywordEdition returns the commander set whose keyword the edition
// holds, or nothing. The longest of several keywords answers, as the most
// specific, and the first in alphabetical order among equally long ones.
func commanderKeywordEdition(b *matcher.Backend, edition string) string {
	edition = strings.ToLower(edition)
	best := ""
	for key := range b.CommanderKeywordMap {
		if !strings.Contains(edition, strings.ToLower(key)) {
			continue
		}
		if len(key) > len(best) || (len(key) == len(best) && key < best) {
			best = key
		}
	}
	return b.CommanderKeywordMap[best]
}

// parseCommanderEdition returns the Commander edition the text names, or an
// empty string when it names none.
func parseCommanderEdition(b *matcher.Backend, edition, variant string) string {
	if !strings.Contains(edition, "Commander") {
		return ""
	}

	// An edition already naming a carried token set is exact: parsing it
	// down to the commander set it stems from would lose the tokens
	if strings.Contains(strings.ToLower(edition), "token") {
		_, found := b.NormalizedSets[matcher.Normalize(edition)]
		if found {
			return ""
		}
	}

	// Append a custom display tag to avoid including the main set during filtering
	if strings.Contains(edition, "Display") || strings.Contains(edition, "Thick") ||
		strings.Contains(variant, "Display") || strings.Contains(variant, "Thick") {
		return edition + " Display"
	}

	// Legends series
	if strings.Contains(edition, "Legends") {
		if edition == "Commander Legends" {
			return "Commander Legends"
		} else if strings.Contains(edition, "Baldur's Gate") {
			edition = "Commander Legends: Battle for Baldur's Gate"
			return edition
		}
	}
	// Double Strixhaven
	if strings.Contains(edition, "Strixhaven") {
		if strings.Contains(edition, "Secret") {
			return "Secrets of Strixhaven Commander"
		}
		return "Commander 2021"
	}

	// Well-known extra tags
	switch {
	case strings.Contains(edition, "Launch"):
		return "Commander 2011 Launch Party"
	case strings.Contains(edition, "Arsenal"):
		return "Commander's Arsenal"
	case strings.Contains(edition, "Ikoria"):
		return "Commander 2020"
	case strings.Contains(edition, "Starter"):
		return "Starter Commander Decks"
	}
	ed := commanderKeywordEdition(b, edition)
	if ed != "" {
		// Bundle promos retain the commander set and its collector numbers.
		if strings.Contains(edition, "Promo") || (strings.Contains(variant, "Promo") && !matcher.Contains(variant, "Bundle")) {
			ed += " Promos"
		}
		return ed
	}

	// Collection series
	if strings.Contains(edition, "Collection") {
		for _, color := range []string{"Green", "Black"} {
			if strings.Contains(edition, color) {
				return "Commander Collection: " + color
			}
		}
	}

	// Check Anthology, but decouple from volume 2
	if strings.Contains(edition, "Anthology") {
		for _, tag := range []string{"2018", "II", "Vol"} {
			if strings.Contains(edition, tag) {
				return "Commander Anthology Volume II"
			}
		}
		return "Commander Anthology"
	}

	// Is there a year available?
	year := matcher.ExtractYear(edition)
	if year != "" {
		return "Commander " + year
	}

	// Special fallbacks
	switch edition {
	case "Commander",
		"Commander Decks",
		"Commander Singles":
		return "Commander 2011"
	}

	return ""
}
