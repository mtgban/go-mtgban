package magic

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The multiverse ids are integers like TCGplayer's, so their space stays out
// of the space-blind chain: the same integer must be reachable as a
// multiverse id by name and never as one bare.
func TestMultiverseIdentifiers(t *testing.T) {
	uuid := testBackend.ConvertID(mtgmatcher.IDSpaceMultiverse, "29728")
	if uuid == "" {
		t.Fatal("multiverse id 29728 did not resolve")
	}
	co, err := testBackend.GetUUID(uuid)
	if err != nil {
		t.Fatal(err)
	}
	if co.Name != "Aboshan's Desire" || co.SetCode != "ODY" {
		t.Errorf("multiverse id 29728 resolved to %s (%s), want Aboshan's Desire (ODY)", co.Name, co.SetCode)
	}

	// Bare, the integer stays in the TCGplayer space or resolves nowhere.
	bare, err := testBackend.MatchID("29728")
	if err == nil {
		co, err = testBackend.GetUUID(bare)
		if err == nil && co.Identifiers["tcgplayerProductId"] != "29728" {
			t.Error("bare 29728 resolved outside the TCGplayer space")
		}
	} else if !errors.Is(err, mtgmatcher.ErrCardUnknownID) {
		t.Errorf("bare 29728: %s", err)
	}
}
