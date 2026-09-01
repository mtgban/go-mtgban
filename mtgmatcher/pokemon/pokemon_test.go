package pokemon

import (
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// datastoreOnce loads the datastore the first time a test asks for it. The
// suite used to read and parse the file again on every call.
var datastoreOnce = sync.OnceValues(func() (*mtgmatcher.Backend, error) {
	path := os.Getenv("POKEMON_PATH")
	if path == "" {
		return nil, nil
	}
	f, err := datastore.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
})

func loadBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	b, err := datastoreOnce()
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Skip("POKEMON_PATH not set; skipping Pokemon suite")
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

// TestNameCarriesNumber pins the split of the collector number storefronts
// glue onto the name, and the reason the split has to be conditional: the
// World Championship reprints are named for the number they reprint, so the
// same shape is a real name there and taking it apart would lose the card.
func TestNameCarriesNumber(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"the number glued to the name still reaches the card", mtgmatcher.InputCard{
			Name: "Wingull - 70/100", Edition: "EX Crystal Guardians"}, "70-100_90608"},
		{"a parenthetical beside it is split off too", mtgmatcher.InputCard{
			Name: "Wingull - 70/100 (Reverse Foil)", Edition: "EX Crystal Guardians"}, "70-100_90608_reverse"},
		{"a name that is really spelled that way is left whole", mtgmatcher.InputCard{
			Name: "Torchic - 2004", Edition: "World Championship Decks", Variation: "74/109"}, "74-109_477355"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				co := b.UUIDs[id]
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, co, tt.want)
			}
		})
	}
}

// TestEraPrefixedEdition pins the set an era-prefixed spelling names. The
// catalog numbers a set within its era and storefronts do not, so the two
// spellings agree only after the prefix; without this a listing's edition
// narrows nothing and the card aliases across every set sharing its number.
func TestEraPrefixedEdition(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct{ edition, want string }{
		{"SWSH Darkness Ablaze", "SWSH03: Darkness Ablaze"},
		{"SV Paradox Rift", "SV04: Paradox Rift"},
		{"XY Steam Siege", "XY - Steam Siege"},
		{"BW Dark Explorers", "Dark Explorers"},
		{"Diamond and Pearl Great Encounters", "Great Encounters"},
		// Cardmarket spends no words on the era at all, so the whole name
		// has to be tried before any of it is dropped.
		{"White Flare", "SV: White Flare"},
		{"Rebel Clash", "SWSH02: Rebel Clash"},
		// A set named outright still answers for itself.
		{"Base Set", "Base Set"},
	} {
		in := mtgmatcher.InputCard{Name: "Pikachu", Edition: tt.edition}
		Rules{}.AdjustEdition(b, &in)
		set, err := b.GetSetByName(in.Edition)
		if err != nil {
			t.Errorf("edition %q resolved to %q, which names no set", tt.edition, in.Edition)
			continue
		}
		if set.Name != tt.want {
			t.Errorf("edition %q -> %q, want %q", tt.edition, set.Name, tt.want)
		}
	}
}

// TestQualifiedNameWidening pins the name the catalog bakes a disambiguator
// into being reached from the bare spelling a storefront writes. The
// collector number is what says which qualified spelling is meant, so the
// guard is the last case: a bare name the set already answers for at that
// number is never widened onto its qualified sibling.
func TestQualifiedNameWidening(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"the set's own qualifier", mtgmatcher.InputCard{
			Name: "Accelgor - 8/101", Edition: "BW Plasma Blast"}, "8-101_83461"},
		{"a promo run's qualifier", mtgmatcher.InputCard{
			Name: "Archeops - SWSH272", Edition: "SWSH Promos"}, "swsh272_451847_holo"},
		{"a bracketed qualifier", mtgmatcher.InputCard{
			Name: "Professor's Research - 085/086", Edition: "SV Black Bolt"}, "085-086_642533"},
		{"the bare name answering the number keeps it", mtgmatcher.InputCard{
			Name: "Magnezone - 47/135", Edition: "BW Plasma Storm"}, "47-135_87119"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestSetCodePrefixedNumber pins the set code a storefront glues onto a
// collector number the catalog writes bare. The promo sets number their
// cards "001".."282", so without this every "SVP016" listing compares a
// number the catalog never writes and the whole edition goes unmatched.
func TestSetCodePrefixedNumber(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"the promo set's own code", mtgmatcher.InputCard{
			Name: "Ampharos ex", Edition: "SV Promos", Variation: "SVP016"}, "016_484397_holo"},
		{"another era's promo set", mtgmatcher.InputCard{
			Name: "Alakazam", Edition: "ME Promos", Variation: "MEP003"}, "003_654597_holo"},
		{"the qualified name is reached through it too", mtgmatcher.InputCard{
			Name: "Baxcalibur", Edition: "SV Promos", Variation: "SVP019"}, "019_501885_holo"},
		// The prefix has to be the set's own code: stripping letters
		// freely would read the promo set's SVP016 as the 016 of every
		// Scarlet & Violet set there is.
		{"another set's prefix does not open its numbering", mtgmatcher.InputCard{
			Name: "Tarountula", Edition: "SV01: Scarlet & Violet Base Set", Variation: "SVP016"}, ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if tt.want == "" {
				if err == nil {
					t.Errorf("Match(%v) = %s (%v), want no match", tt.in, id, b.UUIDs[id])
				}
				return
			}
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestBaseSetEdition pins which "Base Set" an edition means. Three eras head
// their first set that way, and dropping the leading words instead lands all
// of them on 1999's set of that literal name, which prices a Platinum card
// as a Base Set one.
func TestBaseSetEdition(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct{ edition, want string }{
		{"Diamond and Pearl Base Set", "Diamond and Pearl"},
		{"Platinum Base Set", "Platinum"},
		{"Expedition Base Set", "Expedition"},
		// The head has to name a set by itself, so a print run written in
		// front of the name still means 1999's set.
		{"1st Edition Base Set", "Base Set"},
		{"Base Set", "Base Set"},
		// Only the trailing "Base Set" comes off: dropping trailing words
		// in general would resolve this to "Diamond & Pearl".
		{"Diamond and Pearl Stormfront", "Stormfront"},
	} {
		in := mtgmatcher.InputCard{Name: "Pikachu", Edition: tt.edition}
		Rules{}.AdjustEdition(b, &in)
		set, err := b.GetSetByName(in.Edition)
		if err != nil {
			t.Errorf("edition %q resolved to %q, which names no set", tt.edition, in.Edition)
			continue
		}
		if set.Name != tt.want {
			t.Errorf("edition %q -> %q, want %q", tt.edition, set.Name, tt.want)
		}
	}
}

// TestSecondNumberInWording pins which collector number wins when a listing
// carries two. Cool Stuff Inc's buylist prepends its own catalogue field,
// which is empty, "0" or "001" for whole editions and occasionally names a
// different card entirely, and the number the listing prints lands after it.
// Reading only the first token is what put every Dark Explorers and Ancient
// Origins row out of reach.
func TestSecondNumberInWording(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"a padded field before the real number", mtgmatcher.InputCard{
			Name: "Ampharos-EX - 27/98", Edition: "XY Ancient Origins", Variation: "001"}, "27-98_101448_holo"},
		{"a field naming a number the set does not have", mtgmatcher.InputCard{
			Name: "Drowzee - 74a/147", Edition: "Aquapolis", Variation: "74"}, "074a-147_84971"},
		{"a field naming another set's card", mtgmatcher.InputCard{
			Name: "Swablu - SH5", Edition: "Platinum Base Set", Variation: "132"}, "sh5_89659_reverse"},
		// The order is the safety property, and these two pin it. The feed
		// has swapped the two numbers of Plasma Blast's pair of Palkia EX
		// printings between the name and the field, so this row's field
		// names the other row's answer and reading it first prices it as
		// the sibling.
		{"the name wins over the field it was swapped with", mtgmatcher.InputCard{
			Name: "Palkia-EX - 66/101", Edition: "BW Plasma Blast", Variation: "100/101"}, "66-101_87915_holo"},
		// And where the edition names no set every printing of the name
		// competes, so the field's bare "5" reaches a McDonald's promo
		// numbered 005/012 as readily as the Nintendo promo numbered 005,
		// while the name's "005" is one of them verbatim.
		{"a field the promo sets all answer to", mtgmatcher.InputCard{
			Name: "Mudkip - 005", Edition: "Nintendo Black Star Promos", Variation: "5"}, "005_87608_holo"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestDecoratedNameSplit pins the number surviving the decorations written
// around it. A parenthetical is lifted out of the middle of the name rather
// than truncating it, because the number written after it is often the only
// thing telling the printing apart; and the dashed segments are read from
// the right, so a number with wording behind it is still found.
func TestDecoratedNameSplit(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		// Four quarter cards share this name and only the number says
		// which; cutting at the parenthesis leaves them aliased.
		{"the number written after a parenthetical", mtgmatcher.InputCard{
			Name: "Greninja V-Union (Bottom Left) - SWSH157", Edition: "SWSH Promos"}, "swsh157_248889_holo"},
		{"the number with wording behind it", mtgmatcher.InputCard{
			Name: "Dragonite - 5/20 - Normal Holo", Edition: "Dragon Vault"}, "5-20_84915_holo"},
		{"the wording behind it still speaks", mtgmatcher.InputCard{
			Name: "Braixen - 9/39 - NON-HOLO", Edition: "XY Kalos Starter"}, "9-39_83949"},
		// The catalog spells a few promos with the number inside the name,
		// which an ordinary decorated listing collides with once its
		// parenthetical is gone. The head is what says which was meant.
		{"a promo spelled with its number does not capture the listing", mtgmatcher.InputCard{
			Name: "Bouffalant (Non-Holo) - 119/142", Edition: "SV Stellar Crown", Variation: "119/142"}, "119-142_567345_holo"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestSetTotalBreaksTheTie pins the set total deciding between reprints. The
// number folds down to its digits to survive the padding storefronts vary,
// which throws away the one part that tells "1/149" from "1/106"; without
// the total these rows can only alias.
func TestSetTotalBreaksTheTie(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"the edition names no set", mtgmatcher.InputCard{
			Name: "Caterpie", Edition: "Sun & Moon", Variation: "1/149"}, "1-149_126872"},
		{"and the reprint keeps its own total", mtgmatcher.InputCard{
			Name: "Caterpie", Edition: "XY Flashfire", Variation: "1/106"}, "1-106_91134"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestPromoEraEdition pins the promo set an era's heading means. The catalog
// spells them "<era>: <title> Promo Cards" and the storefronts "<era>
// Promos", which agree on nothing after the era, so the row narrows to no
// set and aliases against the Jumbo reprint carrying the same number.
func TestPromoEraEdition(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct{ edition, want string }{
		{"SV Promos", "SV: Scarlet & Violet Promo Cards"},
		{"SWSH Promos", "SWSH: Sword & Shield Promo Cards"},
		{"ME Promos", "ME: Mega Evolution Promo"},
		// An era the catalog names outright still answers for itself.
		{"XY Promos", "XY Promos"},
		{"SM Promos", "SM Promos"},
	} {
		in := mtgmatcher.InputCard{Name: "Pikachu", Edition: tt.edition}
		Rules{}.AdjustEdition(b, &in)
		set, err := b.GetSetByName(in.Edition)
		if err != nil {
			t.Errorf("edition %q resolved to %q, which names no set", tt.edition, in.Edition)
			continue
		}
		if set.Name != tt.want {
			t.Errorf("edition %q -> %q, want %q", tt.edition, set.Name, tt.want)
		}
	}

	// The promo set and the Jumbo reprint carry the same name and number,
	// so the edition is the only thing that can pick between them.
	in := mtgmatcher.InputCard{Name: "Blastoise VMAX", Edition: "SWSH Promos", Variation: "SWSH103"}
	id, err := b.Match(&in)
	if err != nil {
		t.Fatalf("Match(%v) = %v", in, err)
	}
	if want := "swsh103_234299_holo"; id != want {
		t.Errorf("Match(%v) = %s (%v), want %s", in, id, b.UUIDs[id], want)
	}
}

// TestSubsetOfEdition pins the collections a set prints inside itself being
// reachable from the parent's name, which is the only name a storefront
// files them under. The negative is the point of the name test: a set code
// spelled like a subset's is not one.
func TestSubsetOfEdition(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"a Radiant Collection card", mtgmatcher.InputCard{
			Name: "Cinccino - RC19/RC25", Edition: "BW Legendary Treasures"}, "rc19-rc25_84321_holo"},
		{"a Trainer Gallery card", mtgmatcher.InputCard{
			Name: "Flareon - TG01/TG30", Edition: "SWSH Brilliant Stars"}, "tg01-tg30_264210_holo"},
		// CL-23323 is "Trading Card Game Classic", not a part of Call of
		// Legends, and it carries a Gyarados of its own at 007/034.
		{"an unrelated set sharing the code shape", mtgmatcher.InputCard{
			Name: "Gyarados - 7/95", Edition: "Call of Legends", Variation: "7"}, "7-95_85999_holo"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestYearReprintDoesNotTie pins the prefix fallback getting past the World
// Championship reprints. They spell the name they reprint plus the year and
// keep its collector number, so they answer every prefix search for a bare
// name and tie with the spelling the storefront meant, which the ambiguity
// guard reads as "the listing does not say".
func TestYearReprintDoesNotTie(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"the qualified spelling wins the tie", mtgmatcher.InputCard{
			Name: "Colress Machine - 119/135", Edition: "BW Plasma Storm"}, "119-135_84391"},
		{"and again where the name is a phrase", mtgmatcher.InputCard{
			Name: "Shadow Triad - 102/116", Edition: "BW Plasma Freeze"}, "102-116_89096"},
		// The year has to be all the reprint adds. This name's qualifier
		// opens with one and it is not a reprint of anything.
		{"a qualifier that opens with a year is not one", mtgmatcher.InputCard{
			Name: "Championship Arena - 028", Edition: "Nintendo Black Star Promos", Variation: "28"}, "028_84163"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestReprintedByYear pins the predicate directly, because the pipeline
// cannot reach every branch of it: Prefilter strips a parenthetical before
// AdjustName runs, so of the datastore names that end in a year without the
// dash, none arrives here still spelled that way. The dash requirement is a
// guard against names the catalog does not hold today, and only a test of
// the predicate itself can hold it to that.
func TestReprintedByYear(t *testing.T) {
	for _, tt := range []struct {
		desc, name, candidate string
		want                  bool
	}{
		{"the year the deck was played", "Colress Machine", "Colress Machine - 2013", true},
		{"punctuation in the name still answers", "Farfetch'd", "Farfetchd - 2005", true},
		{"a different name that happens to end in a year", "Colress", "Colress Machine - 2013", false},
		{"a year glued to the words before it", "Victory Medal", "Victory Medal 2006", false},
		{"a year inside a parenthetical", "Paradise Resort", "Paradise Resort (World Championships 2023)", false},
		{"a year opening a qualifier rather than closing the name", "Championship Arena", "Championship Arena - 2000 Promo", false},
		{"no year at all", "Colress Machine", "Colress Machine (Team Plasma)", false},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := reprintedByYear(tt.name, tt.candidate); got != tt.want {
				t.Errorf("reprintedByYear(%q, %q) = %v, want %v", tt.name, tt.candidate, got, tt.want)
			}
		})
	}
}
