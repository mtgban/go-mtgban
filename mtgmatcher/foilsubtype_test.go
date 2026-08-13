package mtgmatcher

import "testing"

// noopRules is the smallest GameRules a hand-built Backend can carry: every
// hook leaves the input alone, so Match runs its own pipeline and reaches its
// natural errors instead of the nil-rules short circuit.
type noopRules struct{}

func (noopRules) Prefilter(b *Backend, inCard *InputCard)     {}
func (noopRules) AdjustName(b *Backend, inCard *InputCard)    {}
func (noopRules) AdjustEdition(b *Backend, inCard *InputCard) {}

func (noopRules) FilterPrintings(b *Backend, inCard *InputCard, editions []string) []string {
	return editions
}

func (noopRules) FilterCards(b *Backend, inCard *InputCard, cardSet map[string][]Card) []Card {
	return nil
}

func (noopRules) IsUnsupported(b *Backend, inCard *InputCard) bool         { return false }
func (noopRules) IsSpecificUnsupported(b *Backend, inCard *InputCard) bool { return false }

func (noopRules) MissingPromoTag(b *Backend, inCard *InputCard, co *CardObject) bool { return false }

// subtypeBackend builds the shape a foil sub-type takes in a real datastore:
// one printing stored once per finish, each uuid filed under the finish key
// that reaches it, with a sub-type key ("pillars") beyond the three the
// caller's flags can name. The card name is deliberately absent from
// CanonicalNames, so the text path errors and each test sees what Match does
// with the id's answer on its own.
func subtypeBackend() *Backend {
	card := Card{
		Name:     "Some Printing",
		Number:   "15",
		SetCode:  "SET",
		Finishes: []string{FinishNonfoil, FinishFoil},
		FoilUUIDs: map[string]string{
			FinishNonfoil: "1951",
			FinishFoil:    "1951_f",
			"pillars":     "1951_pillars",
		},
	}

	var b Backend
	b.Sets = map[string]*Set{"SET": {Name: "Some Set", Code: "SET"}}
	b.UUIDs = map[string]*CardObject{}
	for _, entry := range []struct {
		uuid   string
		foil   bool
		finish string
	}{
		{"1951", false, FinishNonfoil},
		// The primary foil records the source's own foil name, not the key it
		// is filed under - the reason Finish cannot tell a sub-type apart.
		{"1951_f", true, "silver"},
		{"1951_pillars", true, "pillars"},
	} {
		co := CardObject{Card: card, Edition: "Some Set", Foil: entry.foil}
		co.UUID = entry.uuid
		co.Finish = entry.finish
		b.UUIDs[entry.uuid] = &co
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
