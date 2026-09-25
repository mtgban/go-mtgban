package mtgmatcher

import (
	"slices"
	"strings"
)

// The finishes every game shares, spelled the way the matcher spells every
// finish name (FinishSlug): lower case, no separators. The datastore games
// name the rest as TCGplayer prices them (Finishes), Magic as mtgjson does.
// The names are the keys of Card.FoilUUIDs and the value of
// CardObject.Finish, so one spelling reaches a printing from every source.
const (
	FinishNonfoil = "nonfoil"
	FinishFoil    = "foil"

	// FinishEtched is Magic's alone - only mtgjson prints one - and Magic
	// is what places the name. It lives here because the flag form of the
	// id lookup has a bit for it, predating a game naming its own finishes.
	FinishEtched = "etched"
)

// NormalizeFinish spells a finish name the way finish names are spelled here,
// dropping the separators and the case a vendor writes it with, so "Cold
// Foil", "cold foil" and "COLD-FOIL" are one name.
func NormalizeFinish(name string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// FinishSlug is the name every finish is keyed and asked for by: the name
// TCGplayer prices the printing under, without case or separators, and
// FinishNonfoil for the plain one TCGplayer calls "Normal". The datastore
// ends the printing's id in the same spelling, but this reads the finish the
// entry publishes, never the id. A name TCGplayer adds later is keyed by its
// own spelling before Finishes has a row for it.
func FinishSlug(name string) string {
	slug := NormalizeFinish(name)
	if slug == "normal" {
		return FinishNonfoil
	}
	return slug
}

// The print runs a printing name can open with.
const (
	RunUnlimited  = "Unlimited"
	Run1stEdition = "1st Edition"
	RunLimited    = "Limited"
)

// runOrder is the order a finish naming no run reaches a product sold only in
// runs: the unlimited run is the plain default, as the flags have it.
var runOrder = []string{RunUnlimited, Run1stEdition, RunLimited}

// Finish is one printing TCGplayer prices the singles of a datastore game
// under, and what the matcher needs to know about it.
type Finish struct {
	// Slug is the name the printing is keyed by, FinishSlug of TCGplayer's.
	Slug string
	// TCGplayer is the printing's name in the catalog.
	TCGplayer string
	// Label spells the printing for a reader. It keeps the slug's words, so
	// FinishSlug reads a label back as it reads a TCGplayer name.
	Label string
	// Run is the print run the name opens with, "" for a set printed once.
	Run string
	// Treatment is the slug of the same printing with its run taken off.
	Treatment string
	// Foil is the flag a storefront raises for the printing.
	Foil bool
}

// Finishes are the printings TCGplayer prices the singles of the eight
// datastore games under, every one their catalogs define. Magic's finishes
// are mtgjson's and are not here. The rows naming no run are in the order a
// bare flag prefers them (DefaultPrinting): Rainbow Foil over Cold Foil, Cold
// Foil over Holofoil, Holofoil over Reverse Holofoil.
var Finishes = []Finish{
	{"nonfoil", "Normal", "Normal", "", "nonfoil", false},
	{"foil", "Foil", "Foil", "", "foil", true},
	{"rainbowfoil", "Rainbow Foil", "Rainbow Foil", "", "rainbowfoil", true},
	{"coldfoil", "Cold Foil", "Cold Foil", "", "coldfoil", true},
	{"holofoil", "Holofoil", "Holofoil", "", "holofoil", true},
	{"reverseholofoil", "Reverse Holofoil", "Reverse Holofoil", "", "reverseholofoil", true},
	{"1stedition", "1st Edition", "1st Edition", Run1stEdition, "nonfoil", false},
	{"unlimited", "Unlimited", "Unlimited", RunUnlimited, "nonfoil", false},
	{"limited", "Limited", "Limited", RunLimited, "nonfoil", false},
	{"1steditionholofoil", "1st Edition Holofoil", "1st Edition Holofoil", Run1stEdition, "holofoil", true},
	{"unlimitedholofoil", "Unlimited Holofoil", "Unlimited Holofoil", RunUnlimited, "holofoil", true},
	{"1steditionnormal", "1st Edition Normal", "1st Edition Normal", Run1stEdition, "nonfoil", false},
	{"1steditionrainbowfoil", "1st Edition Rainbow Foil", "1st Edition Rainbow Foil", Run1stEdition, "rainbowfoil", true},
	{"1steditioncoldfoil", "1st Edition Cold Foil", "1st Edition Cold Foil", Run1stEdition, "coldfoil", true},
	{"unlimitededitionnormal", "Unlimited Edition Normal", "Unlimited Edition Normal", RunUnlimited, "nonfoil", false},
	{"unlimitededitionrainbowfoil", "Unlimited Edition Rainbow Foil", "Unlimited Edition Rainbow Foil", RunUnlimited, "rainbowfoil", true},
}

var finishesBySlug = func() map[string]Finish {
	out := make(map[string]Finish, len(Finishes))
	for _, finish := range Finishes {
		out[finish.Slug] = finish
	}
	return out
}()

// DefaultPrinting is the printing a bare flag answers with, of a product's
// printings keyed by finish: of its foils, or of the rest, the plainest.
func DefaultPrinting[T any](printings map[string]T, foil bool) (T, bool) {
	finish := plainest(printings, func(slug string) bool {
		return IsFoilFinish(slug) == foil
	})
	printing, found := printings[finish]
	return printing, found && finish != ""
}

// plainest is the finish of the printings that keep accepts which Finishes
// ranks first: the treatment it lists first, in its plainest run - none, then
// Unlimited, 1st Edition, Limited. A finish the table has no row for comes
// after the rest.
func plainest[T any](printings map[string]T, keep func(string) bool) string {
	var best string
	var bestRank []int
	for finish := range printings {
		if finish == "" || !keep(finish) {
			continue
		}
		rank := []int{len(Finishes), len(runOrder) + 1}
		if row, found := finishesBySlug[finish]; found {
			rank = []int{treatmentRank[row.Treatment], slices.Index(runOrder, row.Run) + 1}
		}
		order := slices.Compare(rank, bestRank)
		if best == "" || order < 0 || order == 0 && finish < best {
			best, bestRank = finish, rank
		}
	}
	return best
}

// treatmentRank is where each treatment's own row sits in Finishes.
var treatmentRank = func() map[string]int {
	out := map[string]int{}
	for i, finish := range Finishes {
		if finish.Run == "" {
			out[finish.Slug] = i
		}
	}
	return out
}()

// FinishOf is the table's row for a slug, and false for one it has none for.
func FinishOf(slug string) (Finish, bool) {
	finish, found := finishesBySlug[slug]
	return finish, found
}

// TCGplayerFinish is the name TCGplayer prices a finish under, "" for a slug
// the table has no row for.
func TCGplayerFinish(slug string) string {
	return finishesBySlug[slug].TCGplayer
}

// FinishLabel spells a finish for a reader, "" for a slug the table has no
// row for.
func FinishLabel(slug string) string {
	return finishesBySlug[slug].Label
}

// IsFoilFinish reports whether a storefront calls the finish a foil. A
// printing TCGplayer prices past Normal that the table has no row for yet is
// taken for a treatment, and so for a foil.
func IsFoilFinish(slug string) bool {
	if finish, found := finishesBySlug[slug]; found {
		return finish.Foil
	}
	return slug != ""
}

// PrintingFinish names the printing a source describes by its print run and
// its treatment ("1st Edition", "Rainbow Foil"), or the treatment alone where
// TCGplayer prices no such printing: "Unlimited Edition Cold Foil" is a
// shelf's wording rather than a printing, and its cold foil is the
// treatment's, wherever the card was printed in one.
func PrintingFinish(run, treatment string) string {
	name := strings.TrimSpace(run + " " + treatment)
	if _, found := finishesBySlug[FinishSlug(name)]; found {
		return name
	}
	return treatment
}

// otherRun answers a finish a printing is not sold in with the one it is sold
// in that differs from it by print run alone. A finish naming no run reaches
// a product sold only in runs through its unlimited run, then its first
// edition; a run named on a card never printed in it is dropped, since it
// says nothing the treatment does not. One run never answers for another,
// being a different printing at a different price. Only a finish this
// datastore sells somewhere is answered: another game's run names nothing.
func (b *Backend) otherRun(card *Card, slug string) string {
	want, found := FinishOf(slug)
	if !found || !b.knownFinishes[slug] {
		return ""
	}
	bare := map[string]string{}
	runs := map[string]map[string]string{}
	for _, uuid := range card.FoilUUIDs {
		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		have, found := FinishOf(co.Finish)
		if !found {
			continue
		}
		if have.Run == "" {
			bare[have.Treatment] = uuid
			continue
		}
		if runs[have.Treatment] == nil {
			runs[have.Treatment] = map[string]string{}
		}
		runs[have.Treatment][have.Run] = uuid
	}
	if want.Run == "" {
		for _, run := range runOrder {
			if uuid := runs[want.Treatment][run]; uuid != "" {
				return uuid
			}
		}
		return ""
	}
	// A treatment sold in runs alone files the product in them
	for treatment := range runs {
		if bare[treatment] == "" {
			return ""
		}
	}
	if bare[want.Treatment] == "" || b.printedIn(card, want) {
		return ""
	}
	return bare[want.Treatment]
}

// printedIn reports whether the card, by name and number, is sold in the
// finish on another product: Base Set's unlimited Alakazam is sold as plain
// Holofoil, and its 1st Edition on the Shadowless product beside it.
func (b *Backend) printedIn(card *Card, finish Finish) bool {
	for _, uuid := range b.Hashes[Normalize(card.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Number != card.Number {
			continue
		}
		if have, found := FinishOf(co.Finish); found && have.Run == finish.Run && have.Treatment == finish.Treatment {
			return true
		}
	}
	return false
}
