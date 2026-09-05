package mtgmatcher_test

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A name holds its articles, so the part of one that a person remembers finds
// it. The article step used to drop an interior " the " from a name and keep a
// leading one in a query, and the two then disagreed: the Secret Lair stored as
// "...turtleslastronin" could not be found by "the last ronin".
func TestSearchSealedContainsHoldsAnInteriorArticle(t *testing.T) {
	var checked int
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		at := strings.Index(co.Name, " The ")
		if at < 0 {
			continue
		}
		checked++

		// From the article to the end of the name, which is how someone
		// searches for the part of a product they remember.
		fragment := co.Name[at+1:]
		found, err := mtgmatcher.SearchSealedContains(fragment)
		if err != nil {
			t.Errorf("%q holds %q and was not found: %v", co.Name, fragment, err)
			continue
		}

		var hit bool
		for _, id := range found {
			if id == uuid {
				hit = true
				break
			}
		}
		if !hit {
			t.Errorf("%q holds %q, which found %d products, none of them it",
				co.Name, fragment, len(found))
		}
	}
	if checked == 0 {
		t.Skip("no product here is named around an interior article")
	}
	t.Logf("%d products named around an interior article, all found by it", checked)
}

// The same for cards, which are searched the same way.
func TestSearchContainsHoldsAnInteriorArticle(t *testing.T) {
	uuids := mtgmatcher.GetUUIDs()
	var checked int
	for i := 0; i < len(uuids) && checked < 40; i += 13 {
		co, err := mtgmatcher.GetUUID(uuids[i])
		if err != nil {
			continue
		}
		at := strings.Index(co.Name, " the ")
		if at < 0 {
			at = strings.Index(co.Name, " The ")
		}
		if at < 0 {
			continue
		}
		checked++

		fragment := co.Name[at+1:]
		found, err := mtgmatcher.SearchContains(fragment)
		if err != nil {
			t.Errorf("%q holds %q and was not found: %v", co.Name, fragment, err)
			continue
		}
		var hit bool
		for _, id := range found {
			if id == uuids[i] {
				hit = true
				break
			}
		}
		if !hit {
			t.Errorf("%q holds %q, which found %d cards, none of them it",
				co.Name, fragment, len(found))
		}
	}
	if checked == 0 {
		t.Skip("no card here is named around an article")
	}
}

// And a whole name still finds itself.
func TestSearchContainsStillFindsAWholeName(t *testing.T) {
	uuids := mtgmatcher.GetUUIDs()
	for i := 0; i < len(uuids); i += 997 {
		co, err := mtgmatcher.GetUUID(uuids[i])
		if err != nil {
			continue
		}
		found, err := mtgmatcher.SearchContains(co.Name)
		if err != nil {
			t.Fatalf("%q does not find itself: %v", co.Name, err)
		}
		var hit bool
		for _, id := range found {
			if id == uuids[i] {
				hit = true
				break
			}
		}
		if !hit {
			t.Errorf("%q found %d cards, none of them it", co.Name, len(found))
		}
	}
}

// An article on its own is a word like any other, and names it: it does not
// come out empty, which every name would hold.
func TestSearchContainsKeepsAnArticleOnlyQuery(t *testing.T) {
	found, err := mtgmatcher.SearchContains("the")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) >= len(mtgmatcher.GetUUIDs()) {
		t.Errorf("searching for %q answered with %d of the %d printings there are",
			"the", len(found), len(mtgmatcher.GetUUIDs()))
	}
}

// A prefix finds the name it starts. Cut at a word: a name that carries "of
// the" kept those spaces while a rule shielded it from the article step, so
// this used to depend on where the cut fell.
func TestSearchHasPrefixStillFindsAName(t *testing.T) {
	uuids := mtgmatcher.GetUUIDs()
	var checked int
	for i := 0; i < len(uuids); i += 997 {
		co, err := mtgmatcher.GetUUID(uuids[i])
		if err != nil {
			continue
		}
		first, _, multiword := strings.Cut(co.Name, " ")
		if !multiword || first == "" {
			continue
		}
		checked++

		found, err := mtgmatcher.SearchHasPrefix(first)
		if err != nil {
			t.Errorf("%q is not found by the word it starts with, %q: %v", co.Name, first, err)
			continue
		}
		var hit bool
		for _, id := range found {
			if id == uuids[i] {
				hit = true
				break
			}
		}
		if !hit {
			t.Errorf("%q is not among the %d found by %q", co.Name, len(found), first)
		}
	}
	if checked == 0 {
		t.Skip("no multi-word name in this sample")
	}
}

// Glimpse the Unthinkable and Glimpse, the Unthinkable are two cards a comma
// apart, and normalizing drops commas. The article step used to be what kept
// them apart, by way of a rule shielding the second from itself; the playtest
// card carries a qualifier instead, the way every other playtest card whose
// name clashes with a real one does.
func TestAClashingPlaytestNameStaysApart(t *testing.T) {
	real, err := mtgmatcher.SearchEquals("Glimpse the Unthinkable")
	if err != nil {
		t.Fatal(err)
	}
	playtest, err := mtgmatcher.SearchEquals("Glimpse, the Unthinkable Playtest")
	if err != nil {
		t.Fatalf("the playtest card is not named apart from the card it clashes with: %v", err)
	}

	if mtgmatcher.Normalize("Glimpse the Unthinkable") == mtgmatcher.Normalize("Glimpse, the Unthinkable Playtest") {
		t.Fatal("the two names normalize the same, so neither can be matched")
	}

	held := map[string]bool{}
	for _, id := range real {
		held[id] = true
	}
	for _, id := range playtest {
		if held[id] {
			t.Errorf("%s answers to both names", id)
		}
	}
	if len(playtest) != 1 {
		t.Errorf("the playtest card has %d printings, want the one", len(playtest))
	}
}
