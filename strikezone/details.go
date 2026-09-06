package strikezone

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The conditions a Details wording opens with, the treatments in the middle,
// and the print run and language it closes with are all this storefront
// spells about a listing: "Near Mint Parallel Foil 1st Edition English".
var (
	detailConditions = []string{
		"Near Mint",
		"Light Play",
		"Medium Play",
		"Heavy Play",
	}
	detailRuns = []string{
		"1st Edition",
		"Unlimited",
	}
	detailLanguages = []string{
		"English",
		"Japanese",
		"Chinese",
		"Korean",
		"German",
		"French",
		"Italian",
		"Spanish",
		"Portuguese",
	}
)

// parseDetails splits a Details wording into the treatment standing between
// the condition and the print run, the run itself, and the closing language.
// The condition stays out of the answer: the caller matches it from the whole
// wording, where the shared grading switch already reads it.
func parseDetails(details string) (treatment, run, language string) {
	rest := strings.TrimSpace(details)
	for _, cond := range detailConditions {
		if strings.HasPrefix(rest, cond) {
			rest = strings.TrimSpace(strings.TrimPrefix(rest, cond))
			break
		}
	}
	for _, lang := range detailLanguages {
		if strings.HasSuffix(rest, lang) {
			language = lang
			rest = strings.TrimSpace(strings.TrimSuffix(rest, lang))
			break
		}
	}
	for _, r := range detailRuns {
		if strings.HasSuffix(rest, r) {
			run = r
			rest = strings.TrimSpace(strings.TrimSuffix(rest, r))
			break
		}
	}
	return rest, run, language
}

var errForeignListing = errors.New("not an English listing")

// preprocessDetails turns a row of the games priced through the generic item
// table into the matcher's input: the name and edition as listed, the Number
// column as the variation, and the Details wording read for what tells the
// printings apart in that game.
//
// The run needs care: the storefront stamps "1st Edition" on every row of
// every modern set, where no such run exists. Passed along regardless, it
// only speaks when the game gives it meaning - the matchers fall back to the
// printing's default when a finish names nothing - so the stamp costs the
// modern rows nothing and tells the WotC-era Pokemon printings apart, where
// the same wording flips between "1st Edition" and "Unlimited" row by row.
func preprocessDetails(game, cardName, edition, number, details string) (*mtgmatcher.InputCard, error) {
	treatment, run, language := parseDetails(details)
	if language != "" && language != "English" {
		return nil, errForeignListing
	}

	// A row often repeats its number in the name after a dash, at times in
	// the fuller promo form the Number column truncates ("Dragapult (Prime)
	// - SWSH132" beside "132", "Briar, Warden of Thorns - HER044"), and the
	// fuller form is the one the catalogs number by.
	// The tail can carry the promo's own labels behind that number,
	// "Hop's Snorlax - 117/159 (GameStop) (Cosmos Holo)", and it can write
	// the number as the card's share of the set, "117/159" for the 117 the
	// catalog numbers. Both have to come off the name to be read at all.
	var labels string
	idx := strings.LastIndex(cardName, " - ")
	if idx != -1 {
		tail := strings.TrimSpace(cardName[idx+3:])
		head, rest, _ := strings.Cut(tail, " ")
		counted, _, share := strings.Cut(head, "/")
		switch {
		case !numberShaped(head):
			// The tail names no number, so it is more of the card's name.
			head = ""
		case share:
			number = counted
		case namesNumber(head, number):
			number = head
		default:
			head = ""
		}
		if head != "" {
			labels = strings.TrimSpace(rest)
			cardName = cardName[:idx]
		}
	}

	switch game {
	case GamePokemon:
		// The gallery subsets number their cards with a prefix the bare
		// Number column drops: GG29 where the row says 029.
		if strings.Contains(edition, "Gallery") {
			prefix := "TG"
			if strings.Contains(edition, "Galarian") {
				prefix = "GG"
			}
			number = prefix + number
		}

		// "Parallel Foil" is this storefront's name for the reverse holo.
		// The run rides in the variation beside the number, where the
		// matcher reads print-run wording, so it can reach the 1st Edition
		// Holofoil of a card whose plain 1st Edition was never printed.
		variation := number
		if labels != "" {
			variation = strings.TrimSpace(variation + " " + labels)
		}
		if run != "" {
			variation = strings.TrimSpace(variation + " " + run)
		}
		// A label names a printing the catalog files in a promo set of its
		// own, so the row's Set column - the set that numbers the card -
		// excludes the very printing the label names. The labels are the
		// more specific of the two, so they speak and the set stands down.
		if labels != "" {
			edition = ""
		}
		finish := ""
		if treatment == "Parallel Foil" {
			finish = "Reverse Holofoil"
		} else if treatment != "Normal" {
			finish = treatment
		}
		return &mtgmatcher.InputCard{
			Name:      cardName,
			Edition:   edition,
			Variation: variation,
			Finish:    finish,
		}, nil
	case GameFleshAndBlood:
		// The datastore crosses the run with the treatment, so the two
		// runs of a card are two printings: "1st Edition Cold Foil".
		finish := treatment
		if run == "Unlimited" {
			finish = "Unlimited Edition " + finish
		} else if run != "" {
			finish = run + " " + finish
		}
		return &mtgmatcher.InputCard{
			Name:      cardName,
			Edition:   edition,
			Variation: number,
			Finish:    finish,
			Foil:      strings.Contains(treatment, "Foil"),
		}, nil
	case GameYuGiOh:
		return &mtgmatcher.InputCard{
			Name:      cardName,
			Edition:   edition,
			Variation: number,
			Foil:      strings.Contains(treatment, "Foil"),
		}, nil
	}
	return nil, errors.New("unsupported game")
}

// numberShaped reports whether a word is a collector number rather than more
// of the card's name: digits, optionally opened by a set's letters, closed by
// a variant letter, and optionally counted out of the set's own size.
var numberShaped = regexp.MustCompile(`^[A-Za-z]*[0-9]+[A-Za-z]*(/[0-9]+)?$`).MatchString

// namesNumber reports whether the number written in a row's name is the one
// its Number column counts, which the storefront closes with a letter of its
// own for the promo runs it files under a number already taken: "198D" for
// the SM198 the Detective Pikachu promo stamps, "117G" for the GameStop 117.
func namesNumber(head, number string) bool {
	number = strings.TrimLeft(number, "0")
	if strings.HasSuffix(head, number) {
		return true
	}
	marked := strings.TrimRightFunc(number, unicode.IsLetter)
	return marked != "" && strings.HasSuffix(head, marked)
}
