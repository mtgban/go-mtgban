package lorcana

import (
	"strings"
	"testing"
)

// TestNumberlessCards pins that a card the datastore files with no number
// carries none. The number is a string here because that is the only
// spelling an absence survives: an integer gives a card that prints nothing
// the same 0 that "Bruno Madrigal - Undetected Uncle" really prints, and 173
// puzzle inserts, lore cards and oversized components used to land on it,
// one crowd per set.
func TestNumberlessCards(t *testing.T) {
	b := loadDatastore(t)

	// A build that predates the number leaving those cards still has them
	// on 0, and says so by carrying no total either.
	base, err := b.GetUUID("1")
	if err != nil {
		t.Skipf("uuid 1 is not in this datastore: %s", err)
	}
	if base.SetTotal == "" {
		t.Skip("LORCANA_PATH predates the numberless products; rebuild the datastore")
	}

	var numberless, zero int
	var zeroNames []string
	// By card rather than by printing: a card sold in two finishes wears
	// two uuids and would be counted twice.
	seen := map[string]bool{}
	for _, co := range b.UUIDs {
		card := co.Name + "|" + co.SetCode + "|" + co.Number
		if co.Sealed || seen[card] {
			continue
		}
		seen[card] = true
		switch co.Number {
		case "":
			numberless++
		case "0":
			zero++
			zeroNames = append(zeroNames, co.Name)
		}
	}

	// One card prints a 0 and keeps it. Any crowd beside it is the
	// absence being spelled as a number again.
	if zero > 1 {
		if len(zeroNames) > 5 {
			zeroNames = zeroNames[:5]
		}
		t.Errorf("%d cards on number 0, want the one that prints it; first few: %v",
			zero, zeroNames)
	}
	if zero == 1 && !strings.HasPrefix(zeroNames[0], "Bruno Madrigal") {
		t.Errorf("the card numbered 0 is %q, want Bruno Madrigal", zeroNames[0])
	}
	if numberless == 0 {
		t.Error("no card carries an absent number; the inserts print none")
	}
	t.Logf("%d cards carry no number, %d carry a 0", numberless, zero)
}

// TestCollectorNumberBothSpellings pins that either datastore loads. Lorcana
// published the number as an integer and now publishes it as the string
// every other game writes, and the two cannot be swapped in one step: a
// reader taking only one of them is down for as long as the other half of
// the deploy takes.
func TestCollectorNumberBothSpellings(t *testing.T) {
	for _, tt := range []struct{ desc, in, want string }{
		{"the string every other game writes", `"23"`, "23"},
		{"the integer Lorcana published", `23`, "23"},
		{"the card that really prints 0", `0`, "0"},
		{"and its string spelling", `"0"`, "0"},
		{"a number no integer could hold", `"23a"`, "23a"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			var got collectorNumber
			if err := got.UnmarshalJSON([]byte(tt.in)); err != nil {
				t.Fatalf("UnmarshalJSON(%s) = %v", tt.in, err)
			}
			if string(got) != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
