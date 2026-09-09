package vocabulary

import (
	"errors"
	"os"
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestCheckReadsTheRules holds each rule against a loader that breaks it and
// one that does not, so the rules hold wherever the tests run.
func TestCheckReadsTheRules(t *testing.T) {
	stated := Published{Tokens: []string{"alternateart", "sp"}, Facts: []string{"boahancock", "tr"}}
	for _, test := range []struct {
		desc   string
		loaded Backend
		want   func(Problems) int
	}{
		{
			desc: "a token the datastore never published",
			loaded: Backend{
				Declared: []string{"special"},
				Labels:   map[string]string{"special": "Special"},
			},
			want: func(p Problems) int { return len(p.Unstated) },
		},
		{
			desc: "a token on a card the datastore states nowhere",
			loaded: Backend{
				Declared: []string{"alternateart"},
				Labels:   map[string]string{"alternateart": "Alternate Art"},
				Worn:     []string{"boahancockgold"},
			},
			want: func(p Problems) int { return len(p.Unstated) },
		},
		{
			desc: "and a mark the datastore does state",
			loaded: Backend{
				Declared: []string{"alternateart"},
				Labels:   map[string]string{"alternateart": "Alternate Art"},
				Worn:     []string{"alternateart", "boahancock", "tr"},
			},
			want: func(p Problems) int { return len(p.Unstated) },
		},
		{
			desc: "a declared token a query could not carry",
			loaded: Backend{
				Declared: []string{"alternate art"},
				Labels:   map[string]string{"alternate art": "Alternate Art"},
			},
			want: func(p Problems) int { return len(p.NotSlugs) },
		},
		{
			desc:   "a token a reader is shown the slug of",
			loaded: Backend{Declared: []string{"alternateart"}},
			want:   func(p Problems) int { return len(p.Unlabelled) },
		},
		{
			desc: "an ordinal a title-caser capitalised",
			loaded: Backend{
				Declared: []string{"sp"},
				Labels:   map[string]string{"sp": "3Rd Anniversary"},
			},
			want: func(p Problems) int { return len(p.Mangled) },
		},
		{
			desc: "and the same ordinal written as one",
			loaded: Backend{
				Declared: []string{"sp"},
				Labels:   map[string]string{"sp": "3rd Anniversary"},
			},
			want: func(p Problems) int { return len(p.Mangled) },
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			found := Check(test.loaded, stated)
			// The clean case of each pair is the one named "and".
			clean := len(test.desc) > 4 && test.desc[:4] == "and "
			if got := test.want(found); (got == 0) != clean {
				t.Errorf("Check() found %d, lines %v", got, found.Lines())
			}
		})
	}
}

// TestLoadersReadWhatIsPublished holds every game's loader to the datastore
// the run carries. A job carries one game, so the others skip; a run
// carrying none says so rather than passing on nothing.
func TestLoadersReadWhatIsPublished(t *testing.T) {
	var read int
	for _, game := range GameNames() {
		path := PathOf(game)
		if path == "" {
			continue
		}
		t.Run(game, func(t *testing.T) {
			stated, err := ReadPublished(path)
			switch {
			case errors.Is(err, os.ErrNotExist):
				t.Skipf("%s: %v", game, err)
			case errors.Is(err, ErrNotDatastore):
				// Said out loud rather than counted as clean.
				t.Skipf("%v", err)
			case err != nil:
				t.Fatal(err)
			}
			loaded, err := ReadLoaded(game, path)
			if err != nil {
				t.Fatal(err)
			}
			read++
			for _, line := range Check(loaded, stated).Lines() {
				t.Errorf("%s: %s", game, line)
			}
		})
	}
	if read == 0 {
		t.Skip("no game datastore is named in this run")
	}
}
