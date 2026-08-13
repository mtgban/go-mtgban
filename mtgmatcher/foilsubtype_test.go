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

// subtypeBackend builds the shape a foil sub-type takes in a real datastore:
// one printing stored once per finish, each uuid filed under the finish key
// that reaches it, with a sub-type key beyond the three the caller's flags can
// name. A second printing carries the same name, so a caller who sends no
// collector number leaves the two of them aliasing.
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
	}

	var b Backend
	b.Sets = map[string]*Set{"SET": {Name: "Some Set", Code: "SET", Cards: cards}}
	b.UUIDs = map[string]*CardObject{}
	b.CanonicalNames = map[string]string{Normalize(cards[0].Name): cards[0].Name}
	b.Hashes = map[string][]string{}
	for _, card := range cards {
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

// Aliasing is the wording naming the finish but not the printing, against an
// id that named the printing but not the finish. Match must put the two
// together - and never answer the primary foil where the wording named the
// sub-type, which would file two of the printing's skus under one uuid.
func TestMatchResolvesAliasingWithIdPrinting(t *testing.T) {
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

// The id only breaks a tie: it cannot answer where there is no tie to break,
// and a listing the game refuses to price is not a tie at all.
func TestMatchKeepsTheTextPathError(t *testing.T) {
	b := subtypeBackend()

	inCard := &InputCard{
		Name:    "Some Printing",
		Edition: "Some Set",
		Foil:    true,
	}
	cardId, err := b.Match(inCard)
	var alias *AliasingError
	if !errors.As(err, &alias) {
		t.Errorf("Match without id = %q, %v, want an aliasing error", cardId, err)
	}

	inCard = &InputCard{
		Id:      "1951",
		Name:    unsupportedPrefix + "Some Printing",
		Edition: "Some Set",
		Foil:    true,
	}
	cardId, err = b.Match(inCard)
	if err != ErrUnsupported {
		t.Errorf("Match of an unsupported listing = %q, %v, want %v", cardId, err, ErrUnsupported)
	}
}
