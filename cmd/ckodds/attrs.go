package main

import (
	"slices"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// reprintSetTypes are the set types whose reprints lower CK's price of the
// older printings: Modern Horizons-type sets, Commander and Masters.
var reprintSetTypes = []string{"draft_innovation", "commander", "masters"}

// A reprint counts for the printings released at least this long before it.
const reprintAfter = 30 * 24 * time.Hour

// attrs are what the rules read about a CK product's card, beyond its
// prices.
type attrs struct {
	// Exception is the Reserved List and sets released through 1994,
	// measured apart.
	Exception bool
	// SetReleased is the release of the printing's set; zero when unknown.
	SetReleased time.Time
	// Reprints are the releases of the sets of reprintSetTypes that printed
	// the card again later, oldest first.
	Reprints []time.Time
}

// attrsOf reads a card's attrs; printings are the set codes its name was
// printed in.
func attrsOf(b *mtgmatcher.Backend, co *mtgmatcher.CardObject, printings []string) attrs {
	var a attrs
	set := b.Sets[co.SetCode]
	if set != nil {
		a.SetReleased, _ = time.Parse(time.DateOnly, set.ReleaseDate)
	}
	a.Exception = co.IsReserved || (!a.SetReleased.IsZero() && a.SetReleased.Year() <= 1994)
	a.Reprints = reprintsOf(b.Sets, printings, a.SetReleased)
	return a
}

// reprintsOf are the releases, oldest first, of the sets of reprintSetTypes
// among printings released more than reprintAfter after released.
func reprintsOf(sets map[string]*mtgmatcher.Set, printings []string, released time.Time) []time.Time {
	if released.IsZero() {
		return nil
	}
	var out []time.Time
	for _, code := range printings {
		set := sets[code]
		if set == nil || !slices.Contains(reprintSetTypes, set.Type) {
			continue
		}
		date, err := time.Parse(time.DateOnly, set.ReleaseDate)
		if err != nil || date.Sub(released) <= reprintAfter {
			continue
		}
		out = append(out, date)
	}
	slices.SortFunc(out, func(a, b time.Time) int { return a.Compare(b) })
	return slices.Compact(out)
}

// latestReprint is the newest reprint released on or before day, or zero.
func (a attrs) latestReprint(day time.Time) time.Time {
	var latest time.Time
	for _, r := range a.Reprints {
		if !r.After(day) {
			latest = r
		}
	}
	return latest
}
