package magic

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastoretest"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type MatchTest struct {
	ID   string `json:"uuid,omitempty"`
	Err  string `json:"error,omitempty"`
	Desc string `json:"description"`

	In mtgmatcher.InputCard `json:"input"`

	Wildcard bool `json:"wildcard,omitempty"`
}

const testDataFile = "testdata/magic_test_data.json"

var UpdateTests = flag.Bool("u", false, "Update test ids while running")

var testBackend *mtgmatcher.Backend
var matchTests []MatchTest

func TestMain(m *testing.M) {
	datastorePath := os.Getenv("ALLPRINTINGS5_PATH")
	if datastorePath == "" {
		log.Fatalln("Need ALLPRINTINGS5_PATH variable set to run this suite")
	}

	datastoreReader, err := datastoretest.Open(datastorePath)
	if err != nil {
		log.Fatalln(err)
	}
	defer datastoreReader.Close()

	testBackend, err = Load(datastoreReader)
	if err != nil {
		log.Fatalln(err)
	}

	testDataReader, err := os.Open(testDataFile)
	if err != nil {
		log.Fatalln(err)
	}
	err = json.NewDecoder(testDataReader).Decode(&matchTests)
	if err != nil {
		log.Fatalln(err)
	}
	// Close the file right away so that it can be modified later
	testDataReader.Close()

	// The auxiliary lookups of the filter callbacks resolve through the
	// global datastore (see doc.go), and TestVariants reads it too.
	mtgmatcher.SetGlobalDatastore(testBackend)
	mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0))

	os.Exit(m.Run())
}

func runMatch(b *mtgmatcher.Backend, test MatchTest) (string, error) {
	card := test.In
	card.PromoWildcard = test.Wildcard

	cardID, err := b.Match(&card)
	if err == nil && test.Err != "" {
		return cardID, fmt.Errorf("expected error: %s", test.Err)
	}
	if err != nil {
		if test.Err == "" {
			return cardID, fmt.Errorf("unexpected error: %s", err.Error())
		}
		if test.Err != err.Error() {
			return cardID, fmt.Errorf("mismatched error: expected '%s', got '%s'", test.Err, err.Error())
		}
	} else if cardID != test.ID {
		return cardID, fmt.Errorf("id mismatch: expected '%s', got '%s'", test.ID, cardID)
	}

	return cardID, nil
}

func TestMatch(t *testing.T) {
	var shouldUpdateTests bool

	for i, probe := range matchTests {
		test := probe
		t.Run(test.Desc, func(t *testing.T) {
			// Need to run tests sequentially if we're updating them
			if !*UpdateTests {
				t.Parallel()
			}
			cardID, err := runMatch(testBackend, test)
			if err != nil {
				if test.Err == "" {
					if *UpdateTests {
						t.Logf("NOTE: Updating test result from '%s' to '%s'", test.ID, cardID)
						matchTests[i].ID = cardID
						shouldUpdateTests = true
						return
					}

					co, _ := testBackend.GetUUID(cardID)
					t.Errorf("FAIL: %s (%v)", err.Error(), co)
					return
				}

				t.Errorf("FAIL: %s", err.Error())
				return
			}

			t.Log("PASS:", test.Desc)
		})
	}

	if shouldUpdateTests {
		fileWriter, err := os.Create(testDataFile)
		if err != nil {
			t.Errorf("FAIL: Unable to update test data file: %s", err.Error())
			return
		}
		enc := json.NewEncoder(fileWriter)
		enc.SetIndent("", "    ")
		err = enc.Encode(matchTests)
		if err != nil {
			t.Errorf("FAIL: Error while updating test data file: %s", err.Error())
			return
		}
	}
}

// This benchmark function just runs the Match tests b.N times
func BenchmarkMatch(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, test := range matchTests {
			_, err := runMatch(testBackend, test)
			if err != nil {
				b.Errorf("FAIL: %s", err.Error())
			} else {
				b.Log("PASS:", test.Desc)
			}
		}
	}
}

// BenchmarkMatchQuiet runs the match set with the logger left as
// production leaves it - discarding - so the measurement is of the
// matching rather than of writing the matcher's narration to stderr.
func BenchmarkMatchQuiet(b *testing.B) {
	saved := mtgmatcher.Logger
	mtgmatcher.SetGlobalLogger(log.New(io.Discard, "", log.LstdFlags))
	b.Cleanup(func() { mtgmatcher.SetGlobalLogger(saved) })

	b.ReportAllocs()
	for b.Loop() {
		for _, test := range matchTests {
			_, err := runMatch(testBackend, test)
			if err != nil {
				b.Errorf("FAIL: %s", err.Error())
			}
		}
	}
}
