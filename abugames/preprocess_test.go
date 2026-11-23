package abugames

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/magic"
)

// TestMain loads the datastore once for the whole package. A backend is
// 2.9GB resident, and a test that loads its own holds a second one for as
// long as the binary runs.
var (
	datastoreOnce    sync.Once
	datastoreErr     error
	datastoreBackend *matcher.Backend
)

// realDatastore reads the Magic datastore the first time a test asks for it,
// and skips where the run carries none.
func realDatastore(t *testing.T) *matcher.Backend {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		b, err := datastore.Read("magic", path)
		if err != nil {
			datastoreErr = err
			return
		}
		datastoreBackend = b
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if datastoreBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return datastoreBackend
}

// TestSecretLairNumberOverStaleCardNumber guards the case where ABU's
// card_number disagrees with the collector number in the title. Secret Lair
// numbers can be >= 1993, which the year-capped ExtractNumber can't see; a stale
// card.Number ("1933") must not clobber the authoritative title number ("7010").
func TestSecretLairNumberOverStaleCardNumber(t *testing.T) {
	tests := []struct {
		name    string
		number  string // ABU card_number
		wantSet string
		wantNum string
	}{
		{"stale card_number", "1933", "SLD", "7010"},
		{"matching card_number", "7010", "SLD", "7010"},
		{"empty card_number", "", "SLD", "7010"},
	}

	b := realDatastore(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := preprocess(b, &ABUCard{
				DisplayTitle: "Counterspell (Secret Lair 7010) - FOIL",
				Edition:      "Secret Lair Drop",
				Number:       tt.number,
				Language:     []string{"English"},
			})
			if err != nil {
				t.Fatalf("preprocess: %v", err)
			}
			id, err := b.Match(in)
			if err != nil {
				t.Fatalf("match (variation %q): %v", in.Variation, err)
			}
			co, _ := b.GetUUID(id)
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("got %s #%s, want %s #%s (variation %q)", co.SetCode, co.Number, tt.wantSet, tt.wantNum, in.Variation)
			}
		})
	}
}

// TestForeignBlackBorder pins foreign Revised and 4th Edition listings, whose
// title alone says the border: a black-bordered listing lands on the set its
// language was printed in, its land letter intact, and a white-bordered one,
// which the catalog does not carry, is refused.
func TestForeignBlackBorder(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		title, edition, language string
		wantSet, wantNumber      string
	}{
		{"Control Magic (BB 1994)", "Revised", "Italian", "FBB", "52"},
		{"Control Magic (BB 1994)", "Revised", "German", "FBBDE", "52"},
		{"Island (A Purple) (BB 1994)", "Revised", "Italian", "FBB", "295"},
		{"Island (C Dark Purple) (BB 1994)", "Revised", "French", "FBBFR", "297"},
		{"Control Magic (WB)", "Revised", "Italian", "", ""},
		{"Strip Mine (BB)", "4th Edition", "Korean", "4BBKO", "363"},
		{"Strip Mine", "4th Edition", "Chinese Traditional", "4BBZHT", "363"},
		{"Strip Mine (BB)", "4th Edition", "Portuguese", "4BBPT", "363"},
	} {
		card := ABUCard{DisplayTitle: tt.title, Edition: tt.edition, Language: []string{tt.language}}
		in, err := preprocess(b, &card)
		var cardID string
		if err == nil {
			cardID, err = matchCard(b, &card, in)
		}
		if tt.wantSet == "" {
			if err == nil {
				t.Errorf("%s in %s = %s, want refused", tt.title, tt.language, cardID)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s in %s: %v", tt.title, tt.language, err)
			continue
		}
		co, _ := b.GetUUID(cardID)
		if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
			t.Errorf("%s in %s = %s #%s, want %s #%s", tt.title, tt.language, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
		}
	}
}
