package mtgban

import (
	"errors"
	"strings"
	"testing"
)

// TestDuplicateBuylistReportNamesTheCollision pins that a duplicate names the
// entry it collides with. The report is how a reader tells which two listings
// landed on one card - the urls and ids in it are the whole diagnostic - and
// printing every entry on the card instead of the one that conflicts buried
// that: a card bid on in five grades printed all five, every time.
func TestDuplicateBuylistReportNamesTheCollision(t *testing.T) {
	bl := BuylistRecord{}
	for _, grade := range []string{"NM", "SP", "MP", "HP", "PO"} {
		entry := BuylistEntry{
			Quantity:   1,
			Conditions: grade,
			BuyPrice:   10,
			URL:        "https://example.com/" + grade,
			VendorName: "BANNED",
		}
		if err := bl.Add("A", &entry); err != nil {
			t.Fatalf("%s: %v", grade, err)
		}
	}

	again := BuylistEntry{
		Quantity:   1,
		Conditions: "NM",
		BuyPrice:   10,
		URL:        "https://example.com/NM",
		VendorName: "BANNED",
	}
	err := bl.Add("A", &again)
	if !errors.Is(err, ErrDuplicateEntry) {
		t.Fatalf("the duplicate was accepted: %v", err)
	}

	// The grade that collided is named, and the four it did not are not.
	if !strings.Contains(err.Error(), "example.com/NM") {
		t.Errorf("the report does not name the listing it collides with:\n%s", err)
	}
	for _, grade := range []string{"SP", "MP", "HP", "PO"} {
		if strings.Contains(err.Error(), "example.com/"+grade) {
			t.Errorf("the report carries the %s entry, which is not the collision:\n%s", grade, err)
		}
	}
	// Four lines: the reason, the card, the new entry and the one it hit.
	if got := strings.Count(err.Error(), "\n"); got != 3 {
		t.Errorf("the report spans %d newlines, want 3:\n%s", got, err)
	}
}
