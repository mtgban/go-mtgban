package coolstuffinc

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestNotePlace pins the finishing place a buy row names only in its note.
// The catalog labels these printings by the event and the place, so the
// event alone left the deck's own card at the same number answering.
func TestNotePlace(t *testing.T) {
	b := readGameDatastore(t, "onepiece", "ONEPIECE_PATH")

	for _, tt := range []struct {
		name, notes, number, wantID string
	}{
		{"Nefeltari Vivi - 009 (OP03 Pre-Release Tournament)", "ST01-009 Participant Luffy in Back NO STAMP", "ST01-009", "st01-009_497407"},
		{"Nami - 007 (Tournament Pack Vol. 3)", "ST01-007 Participant White Border Holding Money", "ST01-007", "st01-007_496999"},
		{"Nefeltari Vivi - 009", "ST01-009", "ST01-009", "st01-009_288238"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := &mtgmatcher.InputCard{
				Name:      tt.name,
				Edition:   "ST01 - Starter Deck: Straw Hat Crew",
				Variation: eventNamed(strings.TrimSpace(tt.number + " " + nameQualifiers(tt.name) + " " + onePieceNotePlace.FindString(tt.notes))),
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match = %v", err)
			}
			if id != tt.wantID {
				t.Errorf("Match = %q, want %q", id, tt.wantID)
			}
		})
	}
}
