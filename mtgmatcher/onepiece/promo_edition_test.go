package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoEditionFixture mirrors the shape Cool Stuff Inc's buylist has: an
// event printing filed in the promo set under the base card's collector
// number, with the event's name shortened to a parenthetical on the product
// name and the base card's set written in the edition field. The Zoro pair is
// the same shape around a label whose front is a word the wording leaves
// behind, and the Sanji pair is a label named just as plainly by a set that
// hands nothing out.
//
// The collector numbers, product ids and event name are the catalog's.
const promoEditionFixture = `{
	"game": "onepiece",
	"sets": {
		"OP09":   {"name": "Emperors in the New World", "releaseDate": "2024-12-27"},
		"OP-PR":  {"name": "One Piece Promotion Cards", "releaseDate": "2022-12-02"},
		"PRB-02": {"name": "Premium Booster -The Best- Vol. 2", "releaseDate": "2025-08-01"}
	},
	"cards": [
		{"id": "op09-050_596981", "name": "Nami", "number": "OP09-050", "setCode": "OP09", "rarity": "R", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 596981}},
		{"id": "op09-050_596981_foil", "name": "Nami", "number": "OP09-050", "setCode": "OP09", "rarity": "R", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 596981}},
		{"id": "op09-050_597214", "name": "Nami", "number": "OP09-050", "setCode": "OP09", "rarity": "R", "finish": "Normal", "variant": "Alternate Art", "image": "x", "externalLinks": {"tcgPlayerId": 597214}},
		{"id": "op09-050_619216_foil", "name": "Nami", "number": "OP09-050", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "variant": "Championship 25 26 Regionals Season 1", "image": "x", "externalLinks": {"tcgPlayerId": 619216}},
		{"id": "op09-051_596982", "name": "Roronoa Zoro", "number": "OP09-051", "setCode": "OP09", "rarity": "R", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 596982}},
		{"id": "op09-051_596982_foil", "name": "Roronoa Zoro", "number": "OP09-051", "setCode": "OP09", "rarity": "R", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 596982}},
		{"id": "op09-051_619217_foil", "name": "Roronoa Zoro", "number": "OP09-051", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "variant": "Gold Foil Winner Pack", "image": "x", "externalLinks": {"tcgPlayerId": 619217}},
		{"id": "op09-052_596983", "name": "Sanji", "number": "OP09-052", "setCode": "OP09", "rarity": "R", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 596983}},
		{"id": "op09-052_619218", "name": "Sanji", "number": "OP09-052", "setCode": "PRB-02", "rarity": "R", "finish": "Normal", "variant": "Best Selection Vol. 6 Reprint", "image": "x", "externalLinks": {"tcgPlayerId": 619218}}
	]
}`

func promoEditionBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	b, err := Load(strings.NewReader(promoEditionFixture))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestPromoSetBegunByWording pins the redirection: a wording spelling a run of
// a promo printing's label names the set that printing is filed in, whatever
// set the edition field names, and every guard that keeps a wording from
// naming one.
func TestPromoSetBegunByWording(t *testing.T) {
	b := promoEditionBackend(t)

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{
			// The listing that started this: a $1250 buy price on the promo,
			// answered with the base set's common until the edition moved.
			desc: "a shortened event name reaches the promo set",
			in: mtgmatcher.InputCard{
				Name: "Nami (Gold in Border) (Championship 25-26)", Variation: "OP09-050",
				Edition: "OP09 - Emperors in the New World", Foil: true,
			},
			want: "op09-050_619216_foil",
		},
		{
			desc: "the event name spelled in full still reaches it",
			in: mtgmatcher.InputCard{
				Name: "Nami (Championship 25 26 Regionals Season 1)", Variation: "OP09-050",
				Edition: "OP09 - Emperors in the New World", Foil: true,
			},
			want: "op09-050_619216_foil",
		},
		{
			desc: "a bare listing keeps the base printing",
			in: mtgmatcher.InputCard{
				Name: "Nami", Variation: "OP09-050",
				Edition: "OP09 - Emperors in the New World",
			},
			want: "op09-050_596981",
		},
		{
			desc: "the base set's own variant is unmoved",
			in: mtgmatcher.InputCard{
				Name: "Nami (Alternate Art)", Variation: "OP09-050",
				Edition: "OP09 - Emperors in the New World",
			},
			want: "op09-050_597214",
		},
		{
			// One word is what a wording lands on by coincidence, and this
			// one is spending it on the artwork rather than on the event.
			desc: "one word of a label names no set",
			in: mtgmatcher.InputCard{
				Name: "Roronoa Zoro (Gold in Border)", Variation: "OP09-051",
				Edition: "OP09 - Emperors in the New World", Foil: true,
			},
			want: "op09-051_596982_foil",
		},
		{
			// A storefront shortening a name keeps a run of it, not always
			// the first: the event is what it names and the treatment the
			// event's card is printed on is what it leaves behind.
			desc: "a run the label spells further in reaches the promo set",
			in: mtgmatcher.InputCard{
				Name: "Roronoa Zoro (Winner Pack)", Variation: "OP09-051",
				Edition: "OP09 - Emperors in the New World", Foil: true,
			},
			want: "op09-051_619217_foil",
		},
		{
			desc: "two words opening the label do reach the promo set",
			in: mtgmatcher.InputCard{
				Name: "Roronoa Zoro (Gold Foil)", Variation: "OP09-051",
				Edition: "OP09 - Emperors in the New World", Foil: true,
			},
			want: "op09-051_619217_foil",
		},
		{
			// Only a set that hands its cards out is reached this way: the
			// storefronts name the base card's set for a promo, and they name
			// an ordinary product's set for an ordinary product.
			desc: "a set that hands nothing out is not reached",
			in: mtgmatcher.InputCard{
				Name: "Sanji (Best Selection)", Variation: "OP09-052",
				Edition: "OP09 - Emperors in the New World",
			},
			want: "op09-052_596983",
		},
		{
			desc: "and is still reached by an edition naming it",
			in: mtgmatcher.InputCard{
				Name: "Sanji (Best Selection Vol. 6 Reprint)", Variation: "OP09-052",
				Edition: "PRB-02 - Premium Booster -The Best- Vol. 2",
			},
			want: "op09-052_619218",
		},
		{
			// A bare tail is the same number in every set of the game, so it
			// cannot say which printing a label belongs to.
			desc: "a bare tail number names no set",
			in: mtgmatcher.InputCard{
				Name: "Nami (Championship 25-26)", Variation: "050",
				Edition: "OP09 - Emperors in the New World",
			},
			want: "op09-050_596981",
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

// TestSlugsRunOf pins the reading itself: a run of whole words the label
// spells somewhere, never one word and never words the label does not carry.
func TestSlugsRunOf(t *testing.T) {
	slug := mtgmatcher.PromoTypeSlug("Championship 25 26 Regionals Season 1")

	tests := []struct {
		desc    string
		wording string
		want    bool
	}{
		{"the storefront's abbreviation", "OP09-050 Championship 25-26", true},
		{"a longer run of the same front", "Championship 25 26 Regionals", true},
		{"a full spelling holds every run", "Championship 25 26 Regionals Season 1", true},
		{"a run the label spells further in", "Regionals Season", true},
		{"one word is not a run", "Championship", false},
		{"nor a run the label never spells", "Gold in Border", false},
		{"nor an empty wording", "", false},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if got := slugsRunOf(test.wording, []string{slug}); got != test.want {
				t.Errorf("slugsRunOf(%q) = %v, want %v", test.wording, got, test.want)
			}
		})
	}
}

// TestSlugsRunOfShortenings pins the three shapes a storefront shortens a
// label into, each drawn from a Cool Stuff Inc buylist listing that answered
// with the base card before the run was looked for past the label's front.
func TestSlugsRunOfShortenings(t *testing.T) {
	tests := []struct {
		desc    string
		wording string
		label   string
		want    bool
	}{
		{
			desc:    "the collection the promo was sold in is dropped",
			wording: "116 Best Selection Vol. 5",
			label:   "Premium Card Collection Best Selection Vol 5",
			want:    true,
		},
		{
			desc:    "so is the year it was handed out in",
			wording: "038 PSA Magazine promo",
			label:   "2025 PSA Magazine promo",
			want:    true,
		},
		{
			desc:    "and the treatment the signature is printed on",
			wording: "012 OP05 1st Anniversary Gold-Stamped Signature",
			label:   "Alternate Art Gold Stamped Signature",
			want:    true,
		},
		{
			desc:    "but prose about the artwork names no label",
			wording: "060 Strings, Foil",
			label:   "Parallel",
			want:    false,
		},
		{
			desc:    "and neither does a single word it happens to share",
			wording: "035 Sword",
			label:   "Sword Alternate Art",
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			slug := mtgmatcher.PromoTypeSlug(test.label)
			if got := slugsRunOf(test.wording, []string{slug}); got != test.want {
				t.Errorf("slugsRunOf(%q, %q) = %v, want %v", test.wording, slug, got, test.want)
			}
		})
	}
}

// eventSetFixture mirrors the shape a starter deck's pre-release printing
// has: filed in a set of its own whose name is the base set's with the event
// written in front of it, sharing the base card's name, number and empty
// label so nothing but the set name tells the two apart.
//
// The collector numbers, product ids and set names are the catalog's.
const eventSetFixture = `{
	"game": "onepiece",
	"sets": {
		"ST-01":     {"name": "Starter Deck 1: Straw Hat Crew", "releaseDate": "2022-12-02"},
		"ST-01-PRE": {"name": "Super Pre-Release Starter Deck 1: Straw Hat Crew", "releaseDate": "2022-10-28"}
	},
	"cards": [
		{"id": "st01-001_288228_foil", "name": "Monkey.D.Luffy", "number": "ST01-001", "setCode": "ST-01", "rarity": "L", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 288228}},
		{"id": "st01-001_412113_foil", "name": "Monkey.D.Luffy", "number": "ST01-001", "setCode": "ST-01-PRE", "rarity": "L", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 412113}}
	]
}`

// TestEventSetNamedByWording pins the reading of an event set a storefront
// names across two fields: the event on the card and the base set in the
// edition. Neither names the set on its own, and the edition alone is a
// perfect match for the base set, so the base card answered a listing that
// had said which printing it was - a $840 buy price filed on the card the
// pre-release printing reprints.
func TestEventSetNamedByWording(t *testing.T) {
	b, err := Load(strings.NewReader(eventSetFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{
			desc: "the event written on the card reaches the set it is filed in",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "ST01-001 Super Pre-Release",
				Edition: "ST01 - Starter Deck: Straw Hat Crew", Foil: true,
			},
			want: "st01-001_412113_foil",
		},
		{
			desc: "a listing naming no event is the base printing",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "ST01-001",
				Edition: "ST01 - Starter Deck: Straw Hat Crew", Foil: true,
			},
			want: "st01-001_288228_foil",
		},
		{
			// The two printings are told apart by nothing but the set name,
			// so a wording short of the whole marker has said nothing.
			desc: "and so is one spelling the marker only in part",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "ST01-001 Pre-Release",
				Edition: "ST01 - Starter Deck: Straw Hat Crew", Foil: true,
			},
			want: "st01-001_288228_foil",
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
