package yugioh

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// qualifierFixture sells one number twice, once plain and once under a
// qualifier the catalog wrote into the product name; the second file sells
// the same uuids with no qualifier at all.
const qualifierFixture = `{
	"game": "yugioh",
	"sets": {"RA04": {"name": "Quarter Century Bonanza", "releaseDate": "2025-11-21"}},
	"cards": [
		{"id": "ra04-en106_1_unlimited", "name": "Dark Magician", "number": "RA04-EN106", "setCode": "RA04", "rarity": "Ultra Rare", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 1}},
		{"id": "ra04-en106_2_unlimited", "name": "Dark Magician", "number": "RA04-EN106", "setCode": "RA04", "rarity": "Ultra Rare", "finish": "Unlimited", "variant": "Arkana", "image": "x", "externalLinks": {"tcgPlayerId": 2}}
	]
}`

// TestQualifiersAreTheBackends pins that a printing's qualifier is the
// backend's that loaded it: a second backend loaded from a file selling the
// same uuids without the qualifier knows nothing of it, where a map shared
// by every backend of the process kept answering with the first file's.
func TestQualifiersAreTheBackends(t *testing.T) {
	qualified, err := Load(strings.NewReader(qualifierFixture))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Load(strings.NewReader(strings.ReplaceAll(qualifierFixture, `"variant": "Arkana", `, "")))
	if err != nil {
		t.Fatal(err)
	}
	in := mtgmatcher.InputCard{Name: "Dark Magician", Edition: "Quarter Century Bonanza", Variation: "RA04-EN106 Arkana"}
	got, err := qualified.Match(&in)
	if err != nil || got != "ra04-en106_2_unlimited" {
		t.Errorf("the qualified backend answered %q, %v; want the Arkana printing", got, err)
	}
	in = mtgmatcher.InputCard{Name: "Dark Magician", Edition: "Quarter Century Bonanza", Variation: "RA04-EN106"}
	got, err = qualified.Match(&in)
	if err != nil || got != "ra04-en106_1_unlimited" {
		t.Errorf("the qualified backend answered the bare wording %q, %v; want the plain printing", got, err)
	}
	var alias *mtgmatcher.AliasingError
	_, err = plain.Match(&in)
	if !errors.As(err, &alias) {
		t.Errorf("the backend loaded without qualifiers answered %v, want two printings it cannot tell apart", err)
	}
}
