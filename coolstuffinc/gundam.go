package coolstuffinc

import (
	"regexp"
	"strings"
)

// gundamShelfCode is the set code this storefront opens a Gundam shelf with,
// "GD01 - Newtype Rising". The catalog names the set by its title alone, so
// the code narrows nothing and leaves the shelf naming no set at all - and
// this game prints the same card at the same number in GD01, its beta
// edition and the deck build box, so a shelf that narrows nothing leaves the
// number aliasing three ways.
var gundamShelfCode = regexp.MustCompile(`^[A-Za-z]{1,4}-?[0-9]{1,2}\s*-\s+`)

// gundamShelf reads the set a Gundam shelf names.
func gundamShelf(shelf string) string {
	return strings.TrimSpace(gundamShelfCode.ReplaceAllString(shelf, ""))
}

// gundamNameCode is the collector number this storefront writes inside the
// name as well as in its own field. It is also the divider: what stands
// before it is the card's own name, parentheticals and all, and what stands
// after it is the wording naming the run.
var gundamNameCode = regexp.MustCompile(`\s*\([A-Z]+[0-9]*-[0-9]+[a-z]?\)\s*`)

// gundamCard splits what the storefront writes into the card's own name and
// the wording that picks between the printings sharing its number.
//
// The split is by position rather than by shape. A card whose own name ends
// in a parenthetical - "GQuuuuuuX (Omega Psycommu)", "Unicorn Gundam
// (Destroy Mode)" - is indistinguishable from a run's wording by looking at
// the brackets alone, and taking the last one asks for a card the catalog
// does not have. The number is written between the two, so it says which is
// which.
func gundamCard(name, number string) (string, string) {
	variation := number
	if loc := gundamNameCode.FindStringIndex(name); loc != nil {
		for _, wording := range gundamQualifier.FindAllStringSubmatch(name[loc[1]:], -1) {
			variation = strings.TrimSpace(variation + " " + wording[1])
		}
		name = strings.TrimSpace(name[:loc[0]])
	}
	// The token shelf names the art the token wears where the catalog names
	// the token, and the number is what says which it is.
	if strings.HasPrefix(number, "T-") {
		variation = strings.TrimSpace(variation + " Token")
	}
	return strings.TrimSpace(name), strings.TrimSpace(variation)
}

// gundamQualifier is one bracketed wording, of however many the storefront
// hangs behind the number - "(Alt-Art +)", "(SP)", "(Beam Blast)".
var gundamQualifier = regexp.MustCompile(`\(([^)]*)\)`)
