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
var gundamNameCode = regexp.MustCompile(`\s*\(([A-Z]+[0-9]*-[0-9]+[a-z]?)\)\s*`)

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
	loc := gundamNameCode.FindStringSubmatchIndex(name)
	if loc != nil {
		// The sell listing carries no number of its own, and this is the
		// same number the buylist files in its own field.
		if variation == "" {
			variation = name[loc[2]:loc[3]]
		}
		for _, wording := range gundamQualifier.FindAllStringSubmatch(name[loc[1]:], -1) {
			variation = strings.TrimSpace(variation + " " + wording[1])
		}
		name = strings.TrimSpace(name[:loc[0]])
	}
	// The token shelf names the art the token wears where the catalog names
	// the token, and the number is what says which it is.
	if strings.HasPrefix(variation, "T-") {
		variation = strings.TrimSpace(variation + " Token")
	}
	return strings.TrimSpace(name), strings.TrimSpace(variation)
}

// gundamQualifier is one bracketed wording, of however many the storefront
// hangs behind the number - "(Alt-Art +)", "(SP)", "(Beam Blast)".
var gundamQualifier = regexp.MustCompile(`\(([^)]*)\)`)

// gundamRarity maps the storefront's spelling of a rarity onto the catalog's.
// This game files a parallel run at the base printing's number and tells the
// two apart by suffixing the rarity, so the column is what picks between
// them - and the storefront abbreviates the suffix the catalog writes out.
var gundamRarity = map[string]string{
	"CP":    "C+",
	"CPP":   "C++",
	"UP":    "U+",
	"RP":    "R+",
	"LGRP":  "LR+",
	"LGRPP": "LR++",
	"U":     "Uncommon",
}

// gundamTier reads the rarity the storefront publishes beside a Gundam card.
func gundamTier(rarity string) string {
	rarity = strings.TrimSpace(rarity)
	if spelled, found := gundamRarity[rarity]; found {
		return spelled
	}
	return rarity
}

// gundamNumberNotes is the collector number as the sell listing's notes write
// it. Every number this catalog carries spells three digits, which is what
// tells one from the art the token shelf spends the same field on.
var gundamNumberNotes = regexp.MustCompile(`^[A-Za-z]+[0-9]*-[0-9]{3}[a-zA-Z]*$`)

// gundamNumber reads the collector number out of a sell listing's notes,
// which spell it in full where the name sometimes drops a digit.
func gundamNumber(notes string) string {
	for _, field := range strings.Fields(notes) {
		field = strings.Trim(field, "()[],.")
		if gundamNumberNotes.MatchString(field) {
			return field
		}
	}
	return ""
}
