package pokemon

import (
	"os"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func loadBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	path := os.Getenv("POKEMON_PATH")
	if path == "" {
		t.Skip("POKEMON_PATH not set; skipping Pokemon suite")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestFinishIdentity pins the invariant the per-printing entries exist for:
// one uuid per finish, and no uuid answering for two. Pokemon crosses two
// axes, so a product can carry six entries, and folding any of them together
// would file two sku prices under one uuid.
func TestFinishIdentity(t *testing.T) {
	b := loadBackend(t)

	for uuid, co := range b.UUIDs {
		if co.Sealed {
			continue
		}
		if co.Finish == "" {
			t.Errorf("%s carries no finish", uuid)
			continue
		}
		// The uuid the printing's own finish resolves to is the printing
		// itself; anything else means two finishes share one entry.
		if got := co.FoilUUIDs[co.Finish]; got != uuid {
			t.Errorf("%s is finish %q but that finish answers %q", uuid, co.Finish, got)
		}
	}

	// Within a product, no two finishes may name one uuid.
	byProduct := map[string]map[string]string{}
	for uuid, co := range b.UUIDs {
		if co.Sealed {
			continue
		}
		key := co.Identifiers["tcgplayerProductId"]
		if key == "" {
			key = trimFinishSuffix(uuid)
		}
		if byProduct[key] == nil {
			byProduct[key] = map[string]string{}
		}
		if other, seen := byProduct[key][uuid]; seen && other != co.Finish {
			t.Errorf("uuid %s answers for finishes %q and %q", uuid, other, co.Finish)
		}
		byProduct[key][uuid] = co.Finish
	}
}

// TestFinishVocabulary pins the names the game places, both axes and their
// crossings, plus the abbreviations storefronts write them with. A name the
// game cannot place has to stay unplaced rather than borrow another
// finish's uuid.
func TestFinishVocabulary(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
	}{
		{"Normal", mtgmatcher.FinishNonfoil},
		{"Holofoil", finishHolofoil},
		{"holo", finishHolofoil},
		{"Reverse Holofoil", finishReverseHolofoil},
		{"Reverse Holo", finishReverseHolofoil},
		{"1st Edition", finish1stEdition},
		{"1st Edition Holofoil", finish1stEditionHolo},
		{"Unlimited", finishUnlimited},
		{"Unlimited Holofoil", finishUnlimitedHolo},
		{"Cold Foil", ""},
		{"Rainbow Pillars", ""},
	} {
		if got := canonicalFinish(tt.name); got != tt.want {
			t.Errorf("canonicalFinish(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestPromoFlag pins what "is:promo" answers with. Two things say a set is
// promotional and they cover different ground: TCGplayer names most of them,
// and the league, championship and blister groups hand their cards out
// without saying so in the name, which the products' own rarity records.
func TestPromoFlag(t *testing.T) {
	b := loadBackend(t)

	byName := map[string]bool{}
	for _, co := range b.UUIDs {
		// Sealed product hangs off the same set and is never a promo
		// printing, so reading it would answer for the cards.
		if co.Sealed {
			continue
		}
		set := b.Sets[co.SetCode]
		if set == nil {
			continue
		}
		byName[set.Name] = co.IsPromo
	}

	for _, tt := range []struct {
		set  string
		want bool
	}{
		{"WoTC Promo", true},
		{"Nintendo Promos", true},
		{"SM Promos", true},
		{"Base Set", false},
		{"Jungle", false},
	} {
		got, found := byName[tt.set]
		if !found {
			t.Errorf("set %q carries no printing", tt.set)
			continue
		}
		if got != tt.want {
			t.Errorf("set %q is:promo = %v, want %v", tt.set, got, tt.want)
		}
	}
}

// TestPromoTags pins that the labels telling sibling printings apart are
// declared, since the site prints a qualifier only when the backend says it
// is real, and that the finish names the catalog also writes as qualifiers
// stay out of the declaration.
func TestPromoTags(t *testing.T) {
	b := loadBackend(t)

	for _, tag := range []string{"fullart", "staff", "prerelease"} {
		if !slices.Contains(b.AllPromoTypes, tag) {
			t.Errorf("promo type %q is not declared, so nothing will print it", tag)
		}
	}
	for _, tag := range []string{"holofoil", "reverseholofoil", "normal"} {
		if slices.Contains(b.AllPromoTypes, tag) {
			t.Errorf("finish %q is declared as a promo type", tag)
		}
	}
}

// TestFinishSelection pins how a listing's own wording reaches a printing.
// The wording names axes rather than a printing, so the entry that answers
// is the one naming everything asked for and the least beside it: "1st
// Edition" reaches the 1st Edition Holofoil on a card sold in no other
// first-edition printing.
func TestFinishSelection(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc      string
		in        mtgmatcher.InputCard
		wantSuffx string
	}{
		{"bare number keeps the default", mtgmatcher.InputCard{
			Name: "Alakazam", Edition: "Base Set", Variation: "001/102"}, "_holo"},
		{"the finish field names the crossing", mtgmatcher.InputCard{
			Name: "Alakazam", Edition: "Base Set", Variation: "001/102", Finish: "1st Edition Holofoil"}, "_1eholo"},
		{"the wording names the run alone", mtgmatcher.InputCard{
			Name: "Alakazam", Edition: "Base Set", Variation: "001/102 1st Edition"}, "_1eholo"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, found := b.UUIDs[id]
			if !found {
				t.Fatalf("Match returned unknown uuid %s", id)
			}
			if want := canonicalFinish(finishForSuffix(tt.wantSuffx)); co.Finish != want {
				t.Errorf("Match(%v) = %s (finish %q), want finish %q", tt.in, id, co.Finish, want)
			}
		})
	}
}

// finishForSuffix names the printing an id suffix stands for, so the table
// above can read the way the builder's ids do.
func finishForSuffix(suffix string) string {
	switch suffix {
	case "_holo":
		return "Holofoil"
	case "_1eholo":
		return "1st Edition Holofoil"
	case "_reverse":
		return "Reverse Holofoil"
	}
	return "Normal"
}

// TestPromoTypeLabels pins that a token can be read back as the words it was
// made from. The token is what a search query carries; the label is what a
// reader is shown, and title-casing the token cannot put back the spaces it
// dropped.
func TestPromoTypeLabels(t *testing.T) {
	b := loadBackend(t)

	if len(b.PromoTypeLabels) != len(b.AllPromoTypes) {
		t.Errorf("%d tags declared but %d labelled", len(b.AllPromoTypes), len(b.PromoTypeLabels))
	}
	for _, tag := range b.AllPromoTypes {
		if b.PromoTypeLabel(tag) == "" {
			t.Errorf("tag %q reads back as nothing", tag)
		}
	}
	for _, tt := range []struct{ tag, want string }{
		{"fullart", "Full Art"},
		{"pokemoncenterexclusive", "Pokemon Center Exclusive"},
		// The artist names the World Championship decks reprint under,
		// which title-casing spells on its own.
		{"shintaroito", "Shintaro Ito"},
		// The cases that say why the words are looked up rather than
		// guessed: title-casing gives "Charizard Gx" and "Bw Black Star
		// Promos".
		{"charizardgx", "Charizard GX"},
		{"bwblackstarpromos", "BW Black Star Promos"},
	} {
		if got := b.PromoTypeLabel(tt.tag); got != tt.want {
			t.Errorf("PromoTypeLabel(%q) = %q, want %q", tt.tag, got, tt.want)
		}
	}
	// An unknown token still reads as something rather than empty.
	if got := b.PromoTypeLabel("nosuchtag"); got == "" {
		t.Error("an undeclared tag reads back as nothing")
	}
}
