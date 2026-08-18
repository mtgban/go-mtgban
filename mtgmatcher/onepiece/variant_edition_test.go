package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// variantFixture mirrors the shape of the real catalog that makes One
// Piece different from the other games: a variant printing shares its
// collector number with the base card but is filed in another set - the
// alternate arts in PRB-01, the event printings in OP-PR - while
// storefronts file all of them under the base card's set. Brannew is the
// other half of the contract: two plain printings of one number in
// different sets, which only the edition can tell apart. Blast Breath is
// the same shape around an event set, whose name is its base set's with a
// marker in it and whose printings wear no label at all. Bepo is that pair
// again with the marker appended instead, so the two set codes share the
// field a storefront prefix spells and only the wording tells them apart.
// Adio is that pair once more with a two-word marker, so the event name's
// last word sits behind it and no truncation of that name ever reaches the
// base name with it - which is the word a storefront decorating the base
// name reaches for first.
const variantFixture = `{
	"game": "onepiece",
	"sets": {
		"OP01":      {"name": "Romance Dawn", "releaseDate": "2022-12-02"},
		"OP03":      {"name": "Pillars of Strength", "releaseDate": "2023-06-30"},
		"ST-19":     {"name": "Smoker", "releaseDate": "2024-03-08"},
		"OP-PR":     {"name": "One Piece Promotion Cards", "releaseDate": "2022-12-02"},
		"PRB-01":    {"name": "Premium Booster -The Best-", "releaseDate": "2024-08-30"},
		"EB-01":     {"name": "Extra Booster: Memorial Collection", "releaseDate": "2024-05-31"},
		"OP07":      {"name": "500 Years in the Future", "releaseDate": "2024-06-28"},
		"ST-24":     {"name": "Starter Deck 24: GREEN Jewelry Bonney", "releaseDate": "2024-11-01"},
		"PRB-02":    {"name": "Premium Booster -The Best- Vol. 2", "releaseDate": "2025-08-01"},
		"ST-04":     {"name": "Starter Deck 4: Animal Kingdom Pirates", "releaseDate": "2023-02-24"},
		"ST-04 PRE": {"name": "Super Pre-Release Starter Deck 4: Animal Kingdom Pirates", "releaseDate": "2023-02-10"},
		"OP-RP":     {"name": "Revision Pack Cards", "releaseDate": "2024-09-13"},
		"OP14":      {"name": "The Azure Sea's Seven", "releaseDate": "2026-01-16"},
		"OP14 RE":   {"name": "The Azure Sea's Seven Release Event Cards", "releaseDate": "2026-01-09"},
		"OP03 PRE":  {"name": "Pillars of Strength Pre-Release Cards", "releaseDate": "2023-06-16"}
	},
	"cards": [
		{"id": "op01-016_base", "name": "Nami", "number": "OP01-016", "setCode": "OP01", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op01-016_manga", "name": "Nami", "number": "OP01-016", "setCode": "PRB-01", "rarity": "C", "finish": "Normal", "variant": "Manga", "image": "x"},
		{"id": "op01-016_prb2", "name": "Nami", "number": "OP01-016", "setCode": "PRB-02", "rarity": "C", "finish": "Normal", "variant": "Reprint", "image": "x"},
		{"id": "op01-084_base", "name": "Mr.2.Bon.Kurei (Bentham)", "number": "OP01-084", "setCode": "OP01", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op01-084_promo", "name": "Mr.2.Bon.Kurei (Bentham)", "number": "OP01-084", "setCode": "OP-PR", "rarity": "P", "finish": "Normal", "variant": "Store Championship Participation Pack Vol. 2", "image": "x"},
		{"id": "op03-089_base", "name": "Brannew", "number": "OP03-089", "setCode": "OP03", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op03-089_deck", "name": "Brannew", "number": "OP03-089", "setCode": "ST-19", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "eb01-012_base", "name": "Cavendish", "number": "EB01-012", "setCode": "EB-01", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "eb01-012_promo", "name": "Cavendish", "number": "EB01-012", "setCode": "OP-PR", "rarity": "P", "finish": "Normal", "variant": "Treasure Cup 2024", "image": "x"},
		{"id": "st04-016_base", "name": "Blast Breath", "number": "ST04-016", "setCode": "ST-04", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "st04-016_pre", "name": "Blast Breath", "number": "ST04-016", "setCode": "ST-04 PRE", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "st04-016_rp", "name": "Blast Breath", "number": "ST04-016", "setCode": "OP-RP", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op07-031_base", "name": "Bartolomeo", "number": "OP07-031", "setCode": "OP07", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op07-031_st", "name": "Bartolomeo", "number": "OP07-031", "setCode": "ST-24", "rarity": "C", "finish": "Normal", "variant": "Reprint", "image": "x"},
		{"id": "op07-031_prb", "name": "Bartolomeo", "number": "OP07-031", "setCode": "PRB-02", "rarity": "C", "finish": "Normal", "variant": "Reprint", "image": "x"},
		{"id": "momo_base", "name": "Kouzuki Momonosuke", "number": "OP01-031", "setCode": "OP01", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "momo_prb", "name": "Kouzuki Momonosuke", "number": "PRB01-031", "setCode": "PRB-01", "rarity": "C", "finish": "Normal", "variant": "Reprint", "image": "x"},
		{"id": "op14-012_base", "name": "Bepo", "number": "OP14-012", "setCode": "OP14", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op14-012_re", "name": "Bepo", "number": "OP14-012", "setCode": "OP14 RE", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op03-002_base", "name": "Adio", "number": "OP03-002", "setCode": "OP03", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op03-002_pre", "name": "Adio", "number": "OP03-002", "setCode": "OP03 PRE", "rarity": "C", "finish": "Normal", "image": "x"}
	]
}`

func variantBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	b, err := Load(strings.NewReader(variantFixture))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestEditionNamesOneSet pins the snapping of a storefront's spelling onto
// a set name: a family whose members share their wording is told apart by
// the set code the storefront wore in front of it, and by nothing else.
func TestEditionNamesOneSet(t *testing.T) {
	b := variantBackend(t)

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
		err  bool
	}{
		{
			desc: "the set code picks the family member the wording shares",
			in:   mtgmatcher.InputCard{Name: "Nami", Variation: "OP01-016", Edition: "PRB-02: Premium Booster"},
			want: "op01-016_prb2",
		},
		{
			desc: "the other volume resolves through its own set code",
			in:   mtgmatcher.InputCard{Name: "Nami", Variation: "OP01-016", Edition: "PRB-01: Premium Booster"},
			want: "op01-016_manga",
		},
		{
			// A wording that lands on a set name exactly needs no rewriting
			// and gets none - unless the code names a different one, which a
			// storefront truncating "Premium Booster -The Best- Vol. 2" down
			// to the name of PRB-01 does on every listing it files.
			desc: "the set code outranks a wording naming the other volume",
			in:   mtgmatcher.InputCard{Name: "Nami", Variation: "OP01-016", Edition: "PRB-02: Premium Booster -The Best-"},
			want: "op01-016_prb2",
		},
		{
			// Both volumes account for the wording in full, so the shortest
			// name is not an answer: without the code the listing says
			// nothing about which volume it is filed in.
			desc: "negative: a family name without a volume stays ambiguous",
			in:   mtgmatcher.InputCard{Name: "Nami", Variation: "OP01-016", Edition: "Premium Booster"},
			err:  true,
		},
		{
			desc: "a set name spelled a word off still selects its set",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo", Variation: "OP07-031", Edition: "500 Years into the Future"},
			want: "op07-031_base",
		},
		{
			// Two words in common is what a vendor bucket lands on by
			// coincidence, one word from each of two sets, and answering it
			// with either would price a whole shelf as one card.
			desc: "negative: a two-word vendor bucket selects no set",
			in:   mtgmatcher.InputCard{Name: "Brannew", Variation: "OP03-089", Edition: "Premium Cards"},
			err:  true,
		},
		{
			// The event set spells its base set's whole name plus a marker,
			// so a wording spelling the marker too accounts for it in full
			// while leaving the base set three words short.
			desc: "an event set spelled a word off selects the event printing",
			in:   mtgmatcher.InputCard{Name: "Blast Breath", Variation: "ST04-016", Edition: "Super Pre-Release Starter Deck: Animal Kingdom Pirates"},
			want: "st04-016_pre",
		},
		{
			// Spell the base set's words alone and the two are described
			// exactly as well: the marker went unwritten, which is the
			// storefront saying it stocks the ordinary printing.
			desc: "a wording spelling no marker keeps the base set",
			in:   mtgmatcher.InputCard{Name: "Blast Breath", Variation: "ST04-016", Edition: "Animal Kingdom Pirates: Starter Deck 4"},
			want: "st04-016_base",
		},
		{
			// An event set's code opens with its base set's, so the prefix
			// spells the base code for both. The wording is what says which
			// of the pair is meant, and here it spells the marker: a code
			// leaving that word unaccounted for names the other set of the
			// pair, not a truncation of this one.
			desc: "a code short of the wording does not outrank it",
			in:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "OP14 - The Azure Sea's Seven Release"},
			want: "op14-012_re",
		},
		{
			desc: "the same code with no marker spelled keeps the base set",
			in:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "OP14 - The Azure Sea's Seven"},
			want: "op14-012_base",
		},
		{
			// The event set carries every word this wording spells, so the
			// counts alone would hand it the listing. It does not spell them
			// in that set's order: a truncation loses a name's tail, so a
			// wording reading as one opens with the name it cut. This one
			// opens with the coded set's name and hangs a word off the end,
			// which is a storefront decorating the name it wrote.
			desc: "a word appended behind the coded name is not a truncation",
			in:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "OP14 - The Azure Sea's Seven Cards"},
			want: "op14-012_base",
		},
		{
			desc: "a word the event set spells inside its name decorates the coded one",
			in:   mtgmatcher.InputCard{Name: "Adio", Variation: "OP03-002", Edition: "OP03 - Pillars of Strength Cards"},
			want: "op03-002_base",
		},
		{
			desc: "the same pair reached in order still selects the event set",
			in:   mtgmatcher.InputCard{Name: "Adio", Variation: "OP03-002", Edition: "OP03 - Pillars of Strength Pre-Release"},
			want: "op03-002_pre",
		},
		{
			// The volumes share every word of the shorter name, so a word
			// hung off it leaves both a word short and neither can outrank
			// the other: the code is the only thing left saying which.
			desc: "a word appended past a family name keeps the coded volume",
			in:   mtgmatcher.InputCard{Name: "Nami", Variation: "OP01-016", Edition: "PRB-01 - Premium Booster -The Best- Promo"},
			want: "op01-016_manga",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			in := test.in
			uuid, err := b.Match(&in)
			if test.err {
				if err == nil {
					t.Fatalf("Match = %q, want an error", uuid)
				}
				return
			}
			if err != nil {
				t.Fatalf("Match = %v, want %q", err, test.want)
			}
			if uuid != test.want {
				co, _ := b.GetUUID(uuid)
				t.Errorf("Match = %q (%v), want %q", uuid, co, test.want)
			}
		})
	}
}

// TestEditionKeepsVariantPrintings pins the interaction between the
// edition and the variant tiering. CoolStuffInc names the base card's set
// for every printing of it, so an edition that resolves would otherwise
// narrow the candidates to that set and delete the very printing the rest
// of the wording asks for - before FilterCards ever tiers them.
func TestEditionKeepsVariantPrintings(t *testing.T) {
	b := variantBackend(t)

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
		err  bool
	}{
		{
			desc: "qualifier reaches the variant filed in another set",
			in:   mtgmatcher.InputCard{Name: "Nami (OP01-016) (Manga)", Edition: "OP01 - Romance Dawn"},
			want: "op01-016_manga",
		},
		{
			desc: "the same wording without an edition is unchanged",
			in:   mtgmatcher.InputCard{Name: "Nami (OP01-016) (Manga)"},
			want: "op01-016_manga",
		},
		{
			desc: "a bare input still keeps the base printing",
			in:   mtgmatcher.InputCard{Name: "Nami", Variation: "OP01-016", Edition: "OP01 - Romance Dawn"},
			want: "op01-016_base",
		},
		{
			desc: "a letter tail reaches the variant despite the edition",
			in:   mtgmatcher.InputCard{Name: "Mr.2.Bon.Kurei (Bentham)", Variation: "OP01-084a", Edition: "OP01 - Romance Dawn"},
			want: "op01-084_promo",
		},
		{
			desc: "an event qualifier reaches the promo despite the edition",
			in:   mtgmatcher.InputCard{Name: "Mr.2.Bon.Kurei (Bentham) (Store Championship Participation Pack Vol. 2)", Variation: "OP01-084", Edition: "OP01 - Romance Dawn"},
			want: "op01-084_promo",
		},
		{
			// A storefront's edition need not equal the set name to have
			// narrowed the candidates: "Memorial Collection" is merely
			// contained in "Extra Booster: Memorial Collection", and the
			// looser match deleted the promo just as thoroughly.
			desc: "a partially spelled edition still reaches the promo",
			in:   mtgmatcher.InputCard{Name: "Cavendish (Treasure Cup 2024)", Variation: "EB01-012", Edition: "EB01 - Memorial Collection"},
			want: "eb01-012_promo",
		},
		{
			// The edition must keep doing the job it was widened for:
			// telling two plain printings of one number apart.
			desc: "the edition still separates two base printings",
			in:   mtgmatcher.InputCard{Name: "Brannew", Variation: "OP03-089", Edition: "OP03 - Pillars of Strength"},
			want: "op03-089_base",
		},
		{
			desc: "the other set's reprint resolves through its own edition",
			in:   mtgmatcher.InputCard{Name: "Brannew", Variation: "OP03-089", Edition: "ST-19 - Smoker"},
			want: "op03-089_deck",
		},
		{
			desc: "negative: without an edition the two base printings alias",
			in:   mtgmatcher.InputCard{Name: "Brannew", Variation: "OP03-089"},
			err:  true,
		},
		{
			// One label printed in several sets: the widened pool holds both
			// Reprints, and only the edition can hand back the named one.
			desc: "the edition tiebreaks a label shared across sets",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo (Reprint)", Variation: "OP07-031", Edition: "PRB-02 - Premium Booster -The Best- Vol. 2"},
			want: "op07-031_prb",
		},
		{
			desc: "the same label resolves to the other set through its edition",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo (Reprint)", Variation: "OP07-031", Edition: "ST-24 - Starter Deck 24: GREEN Jewelry Bonney"},
			want: "op07-031_st",
		},
		{
			// cardtrader demands the variant with a letter tail and names the
			// variant's own set; the tail widens the pool past the edition,
			// and the tiebreak brings it back.
			desc: "a letter tail with the variant's own edition stays in that set",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo", Variation: "OP07-031a", Edition: "ST-24 - Starter Deck 24: GREEN Jewelry Bonney"},
			want: "op07-031_st",
		},
		{
			desc: "the base printing still resolves through its own edition",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo", Variation: "OP07-031", Edition: "OP07 - 500 Years in the Future"},
			want: "op07-031_base",
		},
		{
			desc: "negative: a shared label with a foreign edition stays ambiguous",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo (Reprint)", Variation: "OP07-031", Edition: "OP07 - 500 Years in the Future"},
			err:  true,
		},
		{
			// The reprint is filed under a number of its own, so the label it
			// wears says nothing about the number being asked for: reading it
			// as a demand for a variant would unpin an edition that was
			// answering the listing perfectly, and the widened pool hands back
			// the base printing - a promo priced as the base common.
			desc: "negative: a label at another number cannot unpin the edition",
			in:   mtgmatcher.InputCard{Name: "Kouzuki Momonosuke", Variation: "OP01-031 Reprint", Edition: "PRB-01: Premium Booster -The Best-"},
			err:  true,
		},
		{
			desc: "the same label at its own number still reaches the reprint",
			in:   mtgmatcher.InputCard{Name: "Kouzuki Momonosuke", Variation: "PRB01-031 Reprint", Edition: "OP01 - Romance Dawn"},
			want: "momo_prb",
		},
		{
			desc: "the base printing of that number is unchanged",
			in:   mtgmatcher.InputCard{Name: "Kouzuki Momonosuke", Variation: "OP01-031", Edition: "OP01 - Romance Dawn"},
			want: "momo_base",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			in := test.in
			uuid, err := b.Match(&in)
			if test.err {
				if err == nil {
					t.Fatalf("Match = %q, want an error", uuid)
				}
				return
			}
			if err != nil {
				t.Fatalf("Match = %v, want %q", err, test.want)
			}
			if uuid != test.want {
				co, _ := b.GetUUID(uuid)
				t.Errorf("Match = %q (%v), want %q", uuid, co, test.want)
			}
		})
	}
}

// TestPromoMarkerNamesNoSet pins what counts as a promo line naming a set.
// The remainder is handed to a normalized comparison, so a marker trailing
// nothing a set name could be matched against selects nothing: whatever a
// storefront left after its marker, the event printing is reached only by
// wording that survives the normalizing.
func TestPromoMarkerNamesNoSet(t *testing.T) {
	b := variantBackend(t)

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{
			desc: "a promo line naming the set reaches the event printing",
			in:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "Promos: The Azure Sea's Seven"},
			want: "op14-012_re",
		},
		{
			desc: "a bare marker keeps the regular printing",
			in:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "Promos:"},
			want: "op14-012_base",
		},
		{
			// Punctuation is what the normalizing drops first, so each of
			// these remainders reaches the comparison as the empty needle
			// every set name contains - the bare marker again, spelled with
			// something in it.
			desc: "a marker trailing a dash keeps the regular printing",
			in:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "Promos: -"},
			want: "op14-012_base",
		},
		{
			desc: "a marker trailing a stop keeps the regular printing",
			in:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "Promo - ."},
			want: "op14-012_base",
		},
		{
			// The normalizing drops a bare "s" along with the apostrophe, so
			// a guard counting words would still call this a set name.
			desc: "a marker trailing a possessive keeps the regular printing",
			in:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "Promo: 's"},
			want: "op14-012_base",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			in := test.in
			uuid, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match = %v, want %q", err, test.want)
			}
			if uuid != test.want {
				co, _ := b.GetUUID(uuid)
				t.Errorf("Match = %q (%v), want %q", uuid, co, test.want)
			}
		})
	}
}
