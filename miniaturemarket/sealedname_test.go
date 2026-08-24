package miniaturemarket

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

func TestSealedNameOnePiece(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want string
	}{
		// The deck number moves to the front, where the canonical names
		// lead with it, and the storefront decorations fall away.
		{
			"One Piece TCG: BLUE Kuzan [ST-33] - Starter Deck (Preorder)",
			"Starter Deck 33: BLUE Kuzan",
		},
		{
			"One Piece TCG: RED Monkey.D.Luffy [ST-31] - Starter Deck (Preorder)",
			"Starter Deck 31: RED Monkey.D.Luffy",
		},
		// A non-deck code restates a set the name already spells, so it
		// only falls away.
		{
			"One Piece TCG: Extra Booster Heroines Edition [EB-03] - Booster Box (24 Packs) (Preorder)",
			"Extra Booster Heroines Edition - Booster Box",
		},
		{
			"One Piece TCG: The Time of Battle [OP-16] - Booster Pack",
			"The Time of Battle - Booster Pack",
		},
		// A name with nothing to rewrite passes through whole.
		{
			"One Piece TCG: Heroines Gift Collection",
			"Heroines Gift Collection",
		},
	} {
		got := sealedName(GameOnePiece, tt.in)
		if got != tt.want {
			t.Errorf("%q:\n got  %q\n want %q", tt.in, got, tt.want)
		}
	}
}

// The other games' names already read like their canon: nothing moves.
func TestSealedNameOtherGamesUntouched(t *testing.T) {
	name := "Riftbound: League of Legends TCG - Origins Booster Box (Preorder)"
	if got := sealedName(GameRiftbound, name); got != name {
		t.Errorf("riftbound name rewritten: %q", got)
	}
}

// TestResolveListing pins that every listing this scraper does not price
// says why. The reasons are what a run reports, and a run reporting none of
// them while pricing nothing is indistinguishable from a catalog with
// nothing in it.
func TestResolveListing(t *testing.T) {
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		t.Skip("LORCANA_PATH not set")
	}
	if err := mtgmatcher.LoadDatastoreFile(path); err != nil {
		t.Fatal(err)
	}

	mm := NewScraperSealed(GameLorcana)
	for _, tt := range []struct {
		desc, id, listed string
		wantDrop         bool
		drop             string
	}{
		{"a listing with no name at all", "1", "   ", true, "unnamed listing"},
		{"a listing of another language's printing", "2", "Lorcana TCG: Azurite Sea Booster Pack (Japanese)", true, "language variant"},
		{"a name the resolver turns down", "3", "Lorcana TCG: No Such Product Exists", true, "unknown card name"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			uuid, drop := mm.resolveListing(tt.id, tt.listed)
			if tt.wantDrop {
				if drop != tt.drop {
					t.Errorf("resolveListing(%q) dropped with %q, want %q", tt.listed, drop, tt.drop)
				}
				return
			}
			if drop != "" || uuid == "" {
				t.Errorf("resolveListing(%q) = (%q, %q), want a uuid", tt.listed, uuid, drop)
			}
		})
	}

	// A Magic listing routes through the id alone, and one the datastore
	// does not carry says so rather than vanishing.
	magic := NewScraperSealed(GameMagic)
	if _, drop := magic.resolveListing("nosuchid", "Whatever Box"); drop != "no datastore id" {
		t.Errorf("an unmapped Magic listing dropped with %q", drop)
	}

	// A name the resolver does answer comes back with no reason at all,
	// read off a product the loaded datastore actually holds so the case
	// does not depend on which build of the file is in use.
	sealed := mtgmatcher.GetSealedUUIDs()
	if len(sealed) == 0 {
		t.Skip("datastore holds no sealed products")
	}
	co, err := mtgmatcher.GetUUID(sealed[0])
	if err != nil {
		t.Fatal(err)
	}
	if uuid, drop := mm.resolveListing("5", co.Name); uuid == "" || drop != "" {
		t.Errorf("resolveListing(%q) = (%q, %q), want a uuid", co.Name, uuid, drop)
	}

}
