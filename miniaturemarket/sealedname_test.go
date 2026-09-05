package miniaturemarket

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
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

// TestSealedNameFleshAndBlood pins the rewrites against listings read off
// the storefront, and the shapes they have to leave alone.
func TestSealedNameGundam(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want string
	}{
		// The deck number moves to the front, where the canonical names
		// lead with it, the same way One Piece's does.
		{
			"GUNDAM Card Game: Clan Unity [ST06] - Starter Deck",
			"Starter Deck 06: Clan Unity",
		},
		{
			"GUNDAM Card Game: Iron Bloom [ST05] - Starter Deck",
			"Starter Deck 05: Iron Bloom",
		},
		// Everything else runs the set name straight into what it is sold
		// as, and the pack count the storefront hangs off it falls away.
		{
			"GUNDAM Card Game: Freedom Ascension [GD05] - Booster Box (24)",
			"Freedom Ascension Booster Box",
		},
		{
			"GUNDAM Card Game: Freedom Ascension [GD05] - Booster Pack",
			"Freedom Ascension Booster Pack",
		},
		{
			"GUNDAM Card Game: Freedom Ascension [SC01] - Deck Build Box (New Arrival)",
			"Freedom Ascension Deck Build Box",
		},
		// The Premium Collections carry a letter after their number, which
		// One Piece's own pattern does not allow for. The catalog holds no
		// row for either, so they resolve to nothing - but the code still
		// has to come off, or the name would not even be asked properly.
		{
			"GUNDAM Card Game: Gundam Assemble Premium Collection [PC01A] - Iron Blooded Orphans",
			"Gundam Assemble Premium Collection Iron Blooded Orphans",
		},
	} {
		if got := sealedName(GameGundam, tt.in); got != tt.want {
			t.Errorf("%s:\n got  %q\n want %q", tt.in, got, tt.want)
		}
	}
}

func TestSealedNameFleshAndBlood(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want string
	}{
		// The canon runs the set name into what it is sold as, and spells
		// no pack count.
		{
			"Flesh & Blood TCG: High Seas - Booster Box (24)",
			"High Seas Booster Box",
		},
		{
			"Flesh & Blood TCG: Super Slam - Booster Pack",
			"Super Slam Booster Pack",
		},
		// An unlimited printing is a bracketed edition at the end.
		{
			"Flesh & Blood TCG: Monarch Unlimited Ed - Booster Box",
			"Monarch Booster Box [Unlimited Edition]",
		},
		// An Armory deck takes a colon before its hero.
		{
			"Flesh & Blood TCG: Armory Deck - Azalea",
			"Armory Deck: Azalea",
		},
		// A chapter's full set is named for what it is.
		{
			"Flesh & Blood TCG: Silver Age Chapter 2 Deck - Set of 5",
			"Silver Age Chapter 2 Deck Display",
		},
		// The Silver Age decks keep both their dash and their
		// parenthetical: the canon spells them the same way, so the
		// rewrites above must not reach them.
		{
			"Flesh & Blood TCG: Silver Age Chapter 3 Deck - Briar (Elemental Runeblade)",
			"Silver Age Chapter 3 Deck - Briar (Elemental Runeblade)",
		},
		// A trailing decoration is left for the resolve retry to drop, so
		// that the parenthetical above survives this step.
		{
			"Flesh & Blood TCG: Usurp the Shadow Throne - Booster Pack (Preorder)",
			"Usurp the Shadow Throne Booster Pack (Preorder)",
		},
	} {
		got := sealedName(GameFleshAndBlood, tt.in)
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
	if err := datastore.Load(path); err != nil {
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

// TestExtraWords pins what the fallback is allowed to forgive, which is the
// half of the rule that needs no datastore: the candidate must account for
// every word the storefront said, and what it adds beyond them is what gets
// weighed.
func TestExtraWords(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		candidate string
		vendor    string
		want      []string
		wantOK    bool
	}{
		{
			"a hero spelled out adds the rest of their name",
			"Silver Age Chapter 3 Deck - Blaze Firemind (Wizard)",
			"Silver Age Chapter 3 Deck - Blaze (Wizard)",
			[]string{"firemind"}, true,
		},
		{
			"the words may be reordered, since only the set of them counts",
			"Silver Age: Usurp the Shadow Throne Deck - Viserai Between Worlds",
			"Usurp the Shadow Throne: Silver Age Deck - Viserai",
			[]string{"between", "worlds"}, true,
		},
		{
			"a case says everything its box does, and one word more",
			"High Seas Booster Box Case",
			"High Seas Booster Box",
			[]string{"case"}, true,
		},
		{
			"a word the storefront said and the candidate lacks is a refusal",
			"High Seas Booster Box",
			"High Seas Booster Box Case",
			nil, false,
		},
		{
			"an exact spelling adds nothing",
			"Super Slam Booster Pack",
			"Super Slam Booster Pack",
			nil, true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, ok := extraWords(sealedWords(tt.candidate), sealedWords(tt.vendor))
			if ok != tt.wantOK {
				t.Fatalf("extraWords ok = %v, want %v", ok, tt.wantOK)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("extraWords = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extraWords = %v, want %v", got, tt.want)
					break
				}
			}
		})
	}
}

// TestResolveByNamedCard exercises the half that needs the datastore: the
// forgiveness is granted only where a card accounts for the added words, and
// the case a box is not is what that rule exists to refuse.
func TestResolveByNamedCard(t *testing.T) {
	path := os.Getenv("FLESHANDBLOOD_PATH")
	if path == "" {
		t.Skip("FLESHANDBLOOD_PATH not set")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	// A hero the storefront names by their first word only.
	uuid, err := resolveByNamedCard("Silver Age Chapter 3 Deck - Blaze (Wizard)")
	if err != nil {
		t.Fatalf("the hero's epithet was not forgiven: %v", err)
	}
	co, cerr := mtgmatcher.GetUUID(uuid)
	if cerr != nil || co.Name != "Silver Age Chapter 3 Deck - Blaze Firemind (Wizard)" {
		t.Errorf("resolved to %v, want the Blaze Firemind deck", co)
	}

	// A case adds a word to its box, and that word is no card: forgiving it
	// would price a case as the box inside it.
	if _, err := resolveByNamedCard("High Seas Booster Box"); err == nil {
		t.Error("a box resolved onto something that says more than it does")
	}
}

// TestResolveListingKeepsTheResolvedAnswer pins that a name the resolver
// answers only once its decoration is trimmed keeps that answer. The fallback
// runs after the resolver, and running it over an answer already found both
// discarded the answer and left nothing to report the failure with.
func TestResolveListingKeepsTheResolvedAnswer(t *testing.T) {
	path := os.Getenv("FLESHANDBLOOD_PATH")
	if path == "" {
		t.Skip("FLESHANDBLOOD_PATH not set")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	mm := NewScraperSealed(GameFleshAndBlood)
	uuid, drop := mm.resolveListing("", "Flesh & Blood TCG: Usurp the Shadow Throne - Booster Pack (Preorder)")
	if drop != "" || uuid == "" {
		t.Fatalf("resolveListing = (%q, %q), want the booster pack", uuid, drop)
	}
	co, err := mtgmatcher.GetUUID(uuid)
	if err != nil || co.Name != "Usurp the Shadow Throne Booster Pack" {
		t.Errorf("resolved to %v, want the Usurp the Shadow Throne Booster Pack", co)
	}
}
