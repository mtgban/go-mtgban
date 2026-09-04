package mtgmatcher

import (
	"slices"
	"strings"
	"testing"
)

// The "servo" bucket holds two distinct cards, the War of the Spark token
// - filed with its set's token sheet, as TWAR - and L16's "Servo // Thopter"
// through its face name: lookups must resolve to the card actually carrying
// the queried name, regardless of the bucket order the load process
// produced.
func TestPrintings4CardExactName(t *testing.T) {
	if len(GetUUIDs()) == 0 {
		t.Skip("datastore not loaded")
	}

	printings, err := Printings4Card("Servo")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(printings, "TWAR") {
		t.Errorf("Servo printings = %v, expected to contain TWAR", printings)
	}
	if slices.Contains(printings, "L16") {
		t.Errorf("Servo printings = %v, L16 belongs to Servo // Thopter", printings)
	}

	printings, err = Printings4Card("Servo // Thopter")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(printings, "L16") {
		t.Errorf("Servo // Thopter printings = %v, expected to contain L16", printings)
	}

	// The Cat Warrior token and the Cat Warriors card are one letter apart
	// and were a single bucket back when normalization dropped the plural.
	// They hash apart now, and must still answer only for themselves.
	printings, err = Printings4Card("Cat Warriors")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(printings, "LEG") || slices.Contains(printings, "DMU") {
		t.Errorf("Cat Warriors printings = %v, expected LEG without DMU", printings)
	}
	if !defaultBackend.NameIsToken("Cat Warrior") {
		t.Error("the card named exactly Cat Warrior is the DMU token")
	}
	if defaultBackend.NameIsToken("Cat Warriors") {
		t.Error("Cat Warriors is a regular card, not a token")
	}
}

// Token names clashing with a real card name must be excluded from the
// token table no matter the order sets are iterated during load: these
// names used to flip classification from process to process.
func TestIsTokenClashingNames(t *testing.T) {
	if len(GetUUIDs()) == 0 {
		t.Skip("datastore not loaded")
	}
	for _, name := range []string{"Scarecrow", "Spark Elemental", "Spellgorger Weird"} {
		if IsToken(name) {
			t.Errorf("IsToken(%q) = true, but a real card carries this name", name)
		}
	}
}

// oldHasPrinting is the pre-index implementation, kept verbatim as what
// BenchmarkHasPrintingWide measures against: for every printing of the named
// card it scanned the whole set comparing names with Equals.
func oldHasPrinting(name, field, value string, editions ...string) bool {
	if defaultBackend.Sets == nil {
		return false
	}

	var checkFunc func(Card, string) bool
	switch field {
	case "promo_type":
		checkFunc = func(card Card, value string) bool {
			return card.HasPromoType(value)
		}
	case "frame_effect":
		checkFunc = func(card Card, value string) bool {
			return card.HasFrameEffect(value)
		}
	case "border_color":
		checkFunc = func(card Card, value string) bool {
			return card.BorderColor == value
		}
	case "frame_version":
		checkFunc = func(card Card, value string) bool {
			return card.FrameVersion == value
		}
	case "finish":
		checkFunc = func(card Card, value string) bool {
			return card.HasFinish(value)
		}
	default:
		return false
	}

	printings, err := Printings4Card(name)
	if err != nil {
		cc := &InputCard{
			Name: name,
		}
		defaultBackend.rules.AdjustName(&defaultBackend, cc)
		name = cc.Name
		printings, err = Printings4Card(name)
		if err != nil {
			return false
		}
	}
	for _, code := range printings {
		var set *Set
		if len(editions) > 0 {
			set = defaultBackend.Sets[editions[0]]
			if set == nil {
				set, _ = GetSetByName(editions[0])
			}
		}
		if set == nil {
			set = defaultBackend.Sets[code]
			if set == nil {
				continue
			}
		}
		for _, in := range set.Cards {
			if Equals(name, in.Name) && checkFunc(in, value) {
				return true
			}
		}
	}

	return false
}

// The old scans made widely printed cards pathological: every printing
// re-scanned a full set with two normalizations per card.
func BenchmarkHasPrintingWide(b *testing.B) {
	if len(GetUUIDs()) == 0 {
		b.Skip("datastore not loaded")
	}
	b.Run("new", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			HasPrinting("Island", "finish", FinishFoil)
		}
	})
	b.Run("old", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			oldHasPrinting("Island", "finish", FinishFoil)
		}
	})
}

// TestHasPrintingAnswersForTheNamedCard pins what the bucket rewrite of
// hasPrinting broke and e2bdcb9e fixed: a hash bucket holds names that
// normalize the same but belong to different cards - "Mr. 1 (Daz.Bonez)"
// beside "Mr.1 (Daz.Bonez)", since normalization folds punctuation and
// case - and the printings of one must never answer for the other.
//
// The expectation is computed by scanning every card object for the exact
// name, independently of the hash index and of entry4Name, so the test
// keeps its meaning if either is rewritten again. It is data-driven rather
// than pinned to named cards, so a refresh that retires one collision and
// introduces another still exercises the invariant.
func TestHasPrintingAnswersForTheNamedCard(t *testing.T) {
	uuids := GetUUIDs()
	if len(uuids) == 0 {
		t.Skip("datastore not loaded")
	}

	// Group the real card names by the bucket they hash into, keeping
	// only the buckets that hold more than one distinct name.
	namesByBucket := map[string]map[string]bool{}
	for _, uuid := range uuids {
		co, err := GetUUID(uuid)
		if err != nil {
			continue
		}
		norm := Normalize(co.Name)
		if namesByBucket[norm] == nil {
			namesByBucket[norm] = map[string]bool{}
		}
		namesByBucket[norm][co.Name] = true
	}

	checks := []struct{ field, value string }{
		{"finish", FinishNonfoil},
		{"finish", FinishFoil},
		{"finish", FinishEtched},
		// Literal values: the Magic vocabulary lives in the magic
		// package, which this one cannot import.
		{"frame_effect", "extendedart"},
		{"border_color", "borderless"},
	}

	var collisions, compared int
	for _, names := range namesByBucket {
		if len(names) < 2 {
			continue
		}
		collisions++
		for name := range names {
			for _, check := range checks {
				// The oracle: does any card actually named this way carry
				// the property? Full scan, no index involved.
				var want bool
				for _, uuid := range uuids {
					co, err := GetUUID(uuid)
					if err != nil || !strings.EqualFold(co.Name, name) {
						continue
					}
					if cardHasProperty(co.Card, check.field, check.value) {
						want = true
						break
					}
				}
				got := HasPrinting(name, check.field, check.value)
				compared++
				if got != want {
					t.Errorf("HasPrinting(%q, %s, %s) = %v, want %v: answered for a card not named that way",
						name, check.field, check.value, got, want)
				}
			}
		}
	}
	if collisions == 0 {
		t.Fatal("no normalize-equal name collisions in the datastore; the invariant is untested")
	}
	t.Logf("%d colliding buckets, %d combinations compared", collisions, compared)
}

// cardHasProperty answers a hasPrinting check for one card, without the
// name resolution the test is there to exercise.
func cardHasProperty(card Card, field, value string) bool {
	switch field {
	case "finish":
		return card.HasFinish(value)
	case "frame_effect":
		return card.HasFrameEffect(value)
	case "border_color":
		return card.BorderColor == value
	}
	return false
}
