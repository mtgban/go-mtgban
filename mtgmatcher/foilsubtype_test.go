package mtgmatcher

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
)

// The sub-type this printing is sold in beyond its primary foil, and the
// prefix that marks a listing the game does not price.
const (
	subtypeFinish     = "pillars"
	unsupportedPrefix = "Unsupported "
)

// noopRules is the smallest GameRules a hand-built Backend can carry: every
// hook leaves the input alone except the two this file leans on.
type noopRules struct{}

func (noopRules) Prefilter(b *Backend, inCard *InputCard)     {}
func (noopRules) AdjustName(b *Backend, inCard *InputCard)    {}
func (noopRules) AdjustEdition(b *Backend, inCard *InputCard) {}

func (noopRules) FilterPrintings(b *Backend, inCard *InputCard, editions []string) []string {
	return editions
}

// FilterCards keeps every candidate, so a name printed twice aliases the way
// a real game's number filter does when the caller sent no number to filter
// on. A variation naming the sub-type re-keys the copy's FoilUUIDs onto the
// primary foil, which is how Lorcana's own rules hand the wording's choice to
// output().
func (noopRules) FilterCards(b *Backend, inCard *InputCard, cardSet map[string][]Card) []Card {
	var out []Card
	for _, cards := range cardSet {
		for _, card := range cards {
			_, found := card.FoilUUIDs[subtypeFinish]
			if found && Contains(inCard.Variation, subtypeFinish) {
				card.FoilUUIDs = maps.Clone(card.FoilUUIDs)
				card.FoilUUIDs[FinishFoil] = card.FoilUUIDs[subtypeFinish]
			}
			out = append(out, card)
		}
	}
	slices.SortFunc(out, func(a, b Card) int {
		return strings.Compare(a.Number, b.Number)
	})
	return out
}

func (noopRules) IsUnsupported(b *Backend, inCard *InputCard) bool {
	return strings.HasPrefix(inCard.Name, unsupportedPrefix)
}

func (noopRules) IsSpecificUnsupported(b *Backend, inCard *InputCard) bool { return false }

func (noopRules) MissingPromoTag(b *Backend, inCard *InputCard, co *CardObject) bool { return false }

// ResolveFinish is the optional hook Match reaches for when an id lands on a
// printing sold in a sub-type: the wording picks which foil, the flag still
// decides whether a foil was asked for at all - the same division of labour
// the game's own rules apply downstream.
func (noopRules) ResolveFinish(b *Backend, inCard *InputCard, co *CardObject) string {
	if !inCard.Foil || !Contains(inCard.Variation, subtypeFinish) {
		return ""
	}
	return co.FoilUUIDs[subtypeFinish]
}

// subtypeBackend builds the shape a foil sub-type takes in a real datastore:
// one printing stored once per finish, each uuid filed under the finish key
// that reaches it, with a sub-type key beyond the three the caller's flags can
// name. A second printing carries the same name, so a caller who sends no
// collector number leaves the two of them aliasing. A third carries a name of
// its own and a sub-type of its own, standing in for an id that names a
// printing the aliasing candidates never contain.
func subtypeBackend() *Backend {
	cards := []Card{
		{
			Name:      "Some Printing",
			Number:    "15",
			SetCode:   "SET",
			Language:  "English",
			Printings: []string{"SET"},
			Finishes:  []string{FinishNonfoil, FinishFoil},
			FoilUUIDs: map[string]string{
				FinishNonfoil: "1951",
				FinishFoil:    "1951_f",
				subtypeFinish: "1951_pillars",
			},
		},
		{
			Name:      "Some Printing",
			Number:    "233",
			SetCode:   "SET",
			Language:  "English",
			Printings: []string{"SET"},
			Finishes:  []string{FinishNonfoil, FinishFoil},
			FoilUUIDs: map[string]string{
				FinishNonfoil: "2151",
				FinishFoil:    "2151_f",
			},
		},
		{
			Name:      "Other Printing",
			Number:    "300",
			SetCode:   "SET",
			Language:  "English",
			Printings: []string{"SET"},
			Finishes:  []string{FinishNonfoil, FinishFoil},
			FoilUUIDs: map[string]string{
				FinishNonfoil: "3151",
				FinishFoil:    "3151_f",
				subtypeFinish: "3151_pillars",
			},
		},
	}

	var b Backend
	b.Sets = map[string]*Set{"SET": {Name: "Some Set", Code: "SET", Cards: cards}}
	b.UUIDs = map[string]*CardObject{}
	b.CanonicalNames = map[string]string{}
	b.Hashes = map[string][]string{}
	for _, card := range cards {
		b.CanonicalNames[Normalize(card.Name)] = card.Name
		for finish, uuid := range card.FoilUUIDs {
			co := CardObject{Card: card, Edition: "Some Set", Foil: finish != FinishNonfoil}
			co.UUID = uuid
			// The primary foil records the source's own foil name, not the key
			// it is filed under - the reason Finish cannot tell a sub-type
			// apart.
			co.Finish = finish
			if finish == FinishFoil {
				co.Finish = "silver"
			}
			b.UUIDs[uuid] = &co
			b.Hashes[Normalize(card.Name)] = append(b.Hashes[Normalize(card.Name)], uuid)
		}
	}
	b.SetRules(noopRules{})
	return &b
}

// An id that already names the foil sub-type has made the choice the flags
// could not: Match must hand it back whether or not a name came along with
// it, instead of sending it down a text path that can only do worse.
func TestMatchKeepsSubtypeIdWithName(t *testing.T) {
	b := subtypeBackend()

	for _, name := range []string{"", "Some Printing"} {
		inCard := &InputCard{
			Id:        "1951_pillars",
			Name:      name,
			Edition:   "Some Set",
			Variation: "15",
			Foil:      true,
		}
		cardId, err := b.Match(inCard)
		if err != nil || cardId != "1951_pillars" {
			t.Errorf("Match with name %q = %q, %v, want 1951_pillars, nil", name, cardId, err)
		}
	}
}

// The id names the printing and the wording names the finish: Match asks the
// game's rules to put the two together, and must never answer the primary
// foil where the wording named the sub-type, which would file two of the
// printing's skus under one uuid.
func TestMatchReadsTheSubtypeOffTheIdsPrinting(t *testing.T) {
	b := subtypeBackend()

	for _, probe := range []struct {
		variation string
		want      string
	}{
		{"", "1951_f"},
		{"Pillars", "1951_pillars"},
	} {
		inCard := &InputCard{
			Id:        "1951",
			Name:      "Some Printing",
			Edition:   "Some Set",
			Variation: probe.variation,
			Foil:      true,
		}
		cardId, err := b.Match(inCard)
		if err != nil || cardId != probe.want {
			t.Errorf("Match with variation %q = %q, %v, want %q, nil", probe.variation, cardId, err, probe.want)
		}
	}
}

// Where the wording names no sub-type there is nothing left to ask, and the
// id's answer stands - the way it already does for a printing with no
// sub-type at all. A text path that would have errored never gets to downgrade
// or discard it, whatever the name says.
func TestMatchKeepsTheIdsAnswerWithoutWording(t *testing.T) {
	b := subtypeBackend()

	for _, probe := range []struct {
		desc      string
		name      string
		variation string
	}{
		{"a name printed twice", "Some Printing", ""},
		{"a number matching nothing", "Some Printing", "99999"},
		{"a name the game does not know", "No Such Printing", ""},
		{"a listing the game does not price", unsupportedPrefix + "Some Printing", ""},
	} {
		inCard := &InputCard{
			Id:        "1951",
			Name:      probe.name,
			Edition:   "Some Set",
			Variation: probe.variation,
			Foil:      true,
		}
		cardId, err := b.Match(inCard)
		if err != nil || cardId != "1951_f" {
			t.Errorf("Match with %s = %q, %v, want 1951_f, nil", probe.desc, cardId, err)
		}
	}
}

// The sub-type is read off the printing the id named, never off the
// candidates a name happens to alias on. Answering with one of them - the
// first dupe, say - would price a different card altogether.
func TestMatchPrefersTheIdsPrintingOverTheAliasingCandidates(t *testing.T) {
	b := subtypeBackend()

	for _, probe := range []struct {
		variation string
		want      string
	}{
		{"", "3151_f"},
		{"Pillars", "3151_pillars"},
	} {
		inCard := &InputCard{
			Id:        "3151",
			Name:      "Some Printing",
			Edition:   "Some Set",
			Variation: probe.variation,
			Foil:      true,
		}
		cardId, err := b.Match(inCard)
		if err != nil || cardId != probe.want {
			t.Errorf("Match of a foreign id with variation %q = %q, %v, want %q, nil", probe.variation, cardId, err, probe.want)
		}
	}
}

// The hook only ever answers a printing an id named: it is no general rescue,
// so a name that aliases on its own, or alongside an id naming nothing, still
// errors, and so does a listing the game refuses to price.
func TestMatchKeepsTheTextPathError(t *testing.T) {
	b := subtypeBackend()

	for _, id := range []string{"", "9999"} {
		inCard := &InputCard{
			Id:      id,
			Name:    "Some Printing",
			Edition: "Some Set",
			Foil:    true,
		}
		cardId, err := b.Match(inCard)
		var alias *AliasingError
		if !errors.As(err, &alias) {
			t.Errorf("Match with id %q = %q, %v, want an aliasing error", id, cardId, err)
		}
	}

	inCard := &InputCard{
		Name:    unsupportedPrefix + "Some Printing",
		Edition: "Some Set",
		Foil:    true,
	}
	cardId, err := b.Match(inCard)
	if err != ErrUnsupported {
		t.Errorf("Match of an unsupported listing = %q, %v, want %v", cardId, err, ErrUnsupported)
	}
}
