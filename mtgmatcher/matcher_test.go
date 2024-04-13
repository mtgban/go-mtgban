package mtgmatcher

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

type MatchTest struct {
	Id   string `json:"uuid,omitempty"`
	Err  string `json:"error,omitempty"`
	Desc string `json:"description"`
	In   Card   `json:"input"`

	Wildcard bool `json:"wildcard,omitempty"`
}

const TestDataFile = "matcher_test_data.json"

var UpdateTests = flag.Bool("u", false, "Update test ids while running")

var MatchTests []MatchTest

func TestMain(m *testing.M) {
	allprintingsPath := os.Getenv("ALLPRINTINGS5_PATH")
	if allprintingsPath == "" {
		log.Fatalln("Need ALLPRINTINGS5_PATH variable set to run tests")
	}

	allPrintingsReader, err := os.Open(allprintingsPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer allPrintingsReader.Close()

	allprints, err := mtgjson.LoadAllPrintings(allPrintingsReader)
	if err != nil {
		log.Fatalln(err)
	}

	NewDatastore(allprints)

	// Close the file right away so that it can be modified later
	testDataReader, err := os.Open(TestDataFile)
	if err != nil {
		testDataReader.Close()
		log.Fatalln(err)
	}

	err = json.NewDecoder(testDataReader).Decode(&MatchTests)
	testDataReader.Close()
	if err != nil {
		log.Fatalln(err)
	}

	SetGlobalLogger(log.New(os.Stderr, "", 0))

	os.Exit(m.Run())
}

func TestMatch(t *testing.T) {
	var shouldUpdateTests bool

	for i, probe := range MatchTests {
		test := probe
		t.Run(test.Desc, func(t *testing.T) {
			// Need to run tests sequentially if we're updating them
			if !*UpdateTests {
				t.Parallel()
			}
			card := test.In
			card.promoWildcard = test.Wildcard
			cardId, err := Match(&card)
			if err == nil && test.Err != "" {
				t.Errorf("FAIL: Expected error: %s", test.Err)
				return
			}
			if err != nil {
				if test.Err == "" {
					t.Errorf("FAIL: Unexpected error: %s", err.Error())
					return
				}
				if test.Err != err.Error() {
					t.Errorf("FAIL: Mismatched error: expected '%s', got '%s'", test.Err, err.Error())
					return
				}
			} else if cardId != test.Id {
				if *UpdateTests {
					t.Logf("NOTE: Updating test result from '%s' to '%s'", test.Id, cardId)
					MatchTests[i].Id = cardId
					shouldUpdateTests = true
					return
				}
				co, _ := GetUUID(cardId)
				t.Errorf("FAIL: Id mismatch: expected '%s', got '%s' (%v)", test.Id, cardId, co)
				return
			}

			t.Log("PASS:", test.Desc)
		})
	}

	if shouldUpdateTests {
		fileWriter, err := os.Create(TestDataFile)
		if err != nil {
			t.Errorf("FAIL: Unable to update test data file: %s", err.Error())
			return
		}
		enc := json.NewEncoder(fileWriter)
		enc.SetIndent("", "    ")
		err = enc.Encode(MatchTests)
		if err != nil {
			t.Errorf("FAIL: Error while updating test data file: %s", err.Error())
			return
		}
	}
}

// This benchmark function just runs the Match tests b.N times
func BenchmarkMatch(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, test := range MatchTests {
			card := test.In
			card.promoWildcard = test.Wildcard
			cardId, err := Match(&card)
			if err == nil && test.Err != "" {
				b.Errorf("FAIL: Expected error: %s", test.Err)
				return
			}
			if err != nil {
				if test.Err == "" {
					b.Errorf("FAIL: Unexpected error: %s", err.Error())
					return
				}
				if test.Err != err.Error() {
					b.Errorf("FAIL: Mismatched error: expected '%s', got '%s'", test.Err, err.Error())
					return
				}
			} else if cardId != test.Id {
				b.Errorf("FAIL: Id mismatch: expected '%s', got '%s'", test.Id, cardId)
				return
			}

			b.Log("PASS:", test.Desc)
		}

	}
}
