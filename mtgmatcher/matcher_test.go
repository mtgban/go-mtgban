package mtgmatcher

import (
	"encoding/json"
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
	for _, probe := range MatchTests {
		test := probe
		t.Run(test.Desc, func(t *testing.T) {
			t.Parallel()

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
				t.Errorf("FAIL: Id mismatch: expected '%s', got '%s'", test.Id, cardId)
				return
			}

			t.Log("PASS:", test.Desc)
		})
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
