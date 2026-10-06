package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The Un-Known Event playtest cards load and answer to their TCGplayer
// products, while the ones punning on a real card, or reprinted from another
// playtest set, leave that card alone.
func TestUnknownEventCards(t *testing.T) {
	realDatastore(t)
	b := testBackend

	for _, tc := range []struct {
		product, name, set string
	}{
		{"485115", "Unclaimed Cat", "UNK"},
		{"532622", "Rampant, Growth Playtest", "UNK"},
		{"532632", "Ransack, the Lab Playtest", "UNK"},
		// Named like a different card, with no table row of its own
		{"496930", "Joven and Chandler Playtest", "UNK"},
		// MTGJSON files this product on UNK's Fast // Furious too
		{"240150", "Fast // Furious", "MH2"},
	} {
		uuid, err := b.MatchID(tc.product)
		if err != nil {
			t.Errorf("product %s: %v", tc.product, err)
			continue
		}
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != tc.set || co.Name != tc.name {
			t.Errorf("product %s: got %s %q, want %s %q", tc.product, co.SetCode, co.Name, tc.set, tc.name)
		}
	}

	for _, tc := range []struct {
		name, edition, set string
	}{
		{"Rampant Growth", "Tenth Edition", "10E"},
		{"Ransack the Lab", "Modern Horizons", "MH1"},
		{"Fast // Furious", "Modern Horizons 2", "MH2"},
		{"Math is for Blockers (Plane)", "Secret Lair Showcase Planes", "PSSC"},
		// A playtest card the Unknown Event shares with Mystery Booster 2
		{"Mox Poison", "Mystery Booster Playtest Cards", "MB2"},
		{"Mox Poison", "", "MB2"},
		{"Mox Poison", "Unknown Event", "UNK"},
		// A pun spelled exactly, in wording that names its set
		{"Clear, the Mind", "Unknown Event", "UNK"},
	} {
		uuid, err := b.Match(&mtgmatcher.InputCard{Name: tc.name, Edition: tc.edition})
		if err != nil {
			t.Errorf("%s [%s]: %v", tc.name, tc.edition, err)
			continue
		}
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != tc.set {
			t.Errorf("%s [%s]: landed in %s, want %s", tc.name, tc.edition, co.SetCode, tc.set)
		}
	}
}
