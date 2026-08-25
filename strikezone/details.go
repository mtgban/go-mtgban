package strikezone

import (
	"errors"
	"strings"

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
	if idx := strings.LastIndex(cardName, " - "); idx != -1 {
		tail := strings.TrimSpace(cardName[idx+3:])
		if !strings.Contains(tail, " ") &&
			strings.HasSuffix(tail, strings.TrimLeft(number, "0")) {
			number = tail
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
		if run != "" {
			variation = strings.TrimSpace(variation + " " + run)
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
