package lorcana

import "testing"

// TestSetTotal pins the denominator the face prints, which for Lorcana is
// not always a size: a set numbers its promos from 1 alongside its own
// cards and prints the run in place of the size, so "1/P1" sits beside
// "1/204" and the number alone names both. 155 of the game's (set, number)
// pairs named two cards for exactly that reason.
//
// The datastore carries it; a build that predates the field leaves every
// card without one, and this skips rather than failing over a stale file.
func TestSetTotal(t *testing.T) {
	b := loadDatastore(t)

	// A build predating the field leaves a card of the set without a
	// total. Its promos still have one, read back off promoGrouping, so
	// the promos are the wrong thing to ask.
	base, err := b.GetUUID("1")
	if err != nil {
		t.Skipf("uuid 1 is not in this datastore: %s", err)
	}
	if base.SetTotal == "" {
		t.Skip("LORCANA_PATH predates the total field; rebuild the datastore")
	}

	// Set 1 number 1 is five cards, and the total is the whole of what
	// tells them apart - two of them share a name as well as a number.
	for _, tt := range []struct {
		name string
		uuid string
		want string
	}{
		// The set's own card, printing the set's size
		{"Ariel - On Human Legs", "1", "204"},
		// Four promo runs, each numbered from 1 in its own space
		{"Mickey Mouse - Brave Little Tailor", "659", "P1"},
		{"Dragon Fire", "1187_holofoil", "C1"},
		{"Mickey Mouse - Friendly Face", "1191_holofoil", "D23"},
		{"Ariel - Sonic Warrior", "3237_holofoil", "CC1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			co, err := b.GetUUID(tt.uuid)
			if err != nil {
				t.Skipf("uuid %s is not in this datastore: %s", tt.uuid, err)
			}
			if co.SetTotal != tt.want {
				t.Errorf("%s: SetTotal is %q, want %q", co.Name, co.SetTotal, tt.want)
			}
		})
	}
}
