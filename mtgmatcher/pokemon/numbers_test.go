package pokemon

import (
	"fmt"
	"strings"
	"testing"
)

// TestPlainNumberIsPlain pins the contract every game keeps: Number is the
// strict number a precise match is made on, and OriginalNumber is that
// number with the game's decorations stripped - never anything more. The
// website reads the two apart, "cn:" against OriginalNumber and "cns:"
// against Number, so a loader that files the wider spelling in
// OriginalNumber leaves the ordinary number search matching nothing while
// the strict one works, which is the wrong way round.
func TestPlainNumberIsPlain(t *testing.T) {
	b := loadBackend(t)

	for uuid, co := range b.UUIDs {
		if co.Number == "" {
			continue
		}
		if len(co.OriginalNumber) > len(co.Number) {
			t.Errorf("%s: OriginalNumber %q is wider than Number %q", uuid, co.OriginalNumber, co.Number)
		}
		if strings.Contains(co.OriginalNumber, "/") {
			t.Errorf("%s: OriginalNumber %q carries a set total", uuid, co.OriginalNumber)
		}
	}
}

// TestPlainNumberDropsThePadding pins that the number a search matches is
// the one a person writes. The catalog pads an ordinal out to three digits
// and 12,549 printings carry the padding, so an OriginalNumber keeping it
// answers "cn:001" and refuses "cn:1" - the strict spelling doing the loose
// spelling's job, which is the whole failure this field exists to avoid.
//
// The padding inside a code is not padding: "SWSH020" is what the card is
// named, and reducing it to "SWSH20" would invent a third spelling that is
// neither the printed one nor a plainer one.
func TestPlainNumberDropsThePadding(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"001", "1"},
		{"01", "1"},
		{"010", "10"},
		{"088a", "88a"},
		{"133", "133"},
		{"SWSH020", "SWSH020"},
		{"TG01", "TG01"},
		{"H32", "H32"},
		{"", ""},
		// A number of nothing but zeros keeps what it had: trimming it away
		// would leave a card no plain-number search could reach.
		{"0", "0"},
		{"000", "000"},
	} {
		if got := plainNumber(tt.in); got != tt.want {
			t.Errorf("plainNumber(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestPaddingIsGoneFromTheDatastore pins the same thing against every
// printing the datastore holds, the unit test above covering only the
// shapes it names.
func TestPaddingIsGoneFromTheDatastore(t *testing.T) {
	b := loadBackend(t)

	for uuid, co := range b.UUIDs {
		if co.Sealed || co.OriginalNumber == "" {
			continue
		}
		if strings.HasPrefix(co.OriginalNumber, "0") {
			t.Errorf("%s: OriginalNumber %q is still padded", uuid, co.OriginalNumber)
		}
	}
}

// TestPrintedFace pins that the number and the total the loader keeps apart
// rejoin into the face the card prints, and that a card printing no total
// stays bare. The emptiness is load-bearing: the verbatim tier reads it to
// let a bare-numbered promo win over its totalled reprint, so filling it in
// from the set's own size would collapse the two.
func TestPrintedFace(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc     string
		uuid     string
		number   string
		plain    string
		total    string
		wantFace string
	}{
		{
			desc:   "a card whose face prints a total keeps it",
			uuid:   "001-102_42346_holo",
			number: "001", plain: "1", total: "102", wantFace: "001/102",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			co, found := b.UUIDs[tt.uuid]
			if !found {
				t.Skipf("%s is not in this datastore", tt.uuid)
			}
			if co.Number != tt.number {
				t.Errorf("Number is %q, want %q", co.Number, tt.number)
			}
			// Number is the number as written, padding and all;
			// OriginalNumber is the one a person types.
			if co.OriginalNumber != tt.plain {
				t.Errorf("OriginalNumber is %q, want %q", co.OriginalNumber, tt.plain)
			}
			if co.SetTotal != tt.total {
				t.Errorf("SetTotal is %q, want %q", co.SetTotal, tt.total)
			}
			if got := printedFace(&co.Card); got != tt.wantFace {
				t.Errorf("printedFace is %q, want %q", got, tt.wantFace)
			}
		})
	}

	t.Run("a promo printing no total stays bare", func(t *testing.T) {
		var checked int
		for _, co := range b.UUIDs {
			if co.SetCode != "SWSD" || co.SetTotal != "" {
				continue
			}
			if got := printedFace(&co.Card); got != co.Number {
				t.Errorf("%s: printedFace is %q, want the bare %q", co.UUID, got, co.Number)
			}
			checked++
		}
		if checked == 0 {
			t.Skip("no totalless printing in this datastore")
		}
	})
}

// TestBaseSetSize pins the set size the builder reads off the totals its
// cards print. A pooled set carries none on purpose: World Championship
// Decks holds cards printed 74/109 beside cards printed 87/101, so there is
// no one size to report and reporting the commonest would be a guess.
func TestBaseSetSize(t *testing.T) {
	b := loadBackend(t)

	var sized int
	for _, set := range b.Sets {
		if set.BaseSetSize > 0 {
			sized++
		}
	}
	if sized == 0 {
		t.Skip("this datastore predates the published set size")
	}

	for _, tt := range []struct {
		code string
		want int
	}{
		{"OBF", 197},
		{"MEG", 132},
		{"PRE", 131},
		{"BS", 102},
		// Pooled: its cards keep the total of wherever they first appeared.
		{"WCD", 0},
	} {
		set, found := b.Sets[tt.code]
		if !found {
			t.Errorf("%s is not a set", tt.code)
			continue
		}
		if set.BaseSetSize != tt.want {
			t.Errorf("%s: BaseSetSize is %d, want %d", tt.code, set.BaseSetSize, tt.want)
		}
	}

	// Whatever a set does report has to be the total its own cards print.
	for code, set := range b.Sets {
		if set.BaseSetSize == 0 {
			continue
		}
		want := fmt.Sprint(set.BaseSetSize)
		for _, card := range set.Cards {
			if card.SetTotal == "" {
				continue
			}
			if strings.TrimLeft(card.SetTotal, "0") != want {
				t.Errorf("%s: set size %s but %s prints %s", code, want, card.UUID, card.SetTotal)
				break
			}
		}
	}
}
