package mtgmatcher

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"testing"
)

type MatchTest struct {
	Id   string `json:"uuid,omitempty"`
	Err  string `json:"error,omitempty"`
	Desc string `json:"description"`

	In InputCard `json:"input"`

	Wildcard bool `json:"wildcard,omitempty"`
}

type DatastoreProperty struct {
	EnvVariable  string
	TestDataFile string
}

type TestProperty struct {
	Backend      cardBackend
	MatchTests   []MatchTest
	TestDataFile string
}

var Datastores = []DatastoreProperty{
	{
		EnvVariable:  "ALLPRINTINGS5_PATH",
		TestDataFile: "matcher_test_data.json",
	},
}

var UpdateTests = flag.Bool("u", false, "Update test ids while running")

var MatchTestSet []TestProperty

func TestMain(m *testing.M) {
	for _, datastoreProp := range Datastores {
		if os.Getenv(datastoreProp.EnvVariable) == "" {
			log.Println("Need", datastoreProp.EnvVariable, "variable set to run some tests")
			continue
		}

		tp := loadTestSet(datastoreProp)
		MatchTestSet = append(MatchTestSet, tp)
	}
	if len(MatchTestSet) == 0 {
		log.Fatalln("No tests configured")
	}

	// Keep one set for compatibility with other tests
	SetGlobalDatastore(MatchTestSet[0].Backend)

	SetGlobalLogger(log.New(os.Stderr, "", 0))

	os.Exit(m.Run())
}

func loadTestSet(datastoreProp DatastoreProperty) TestProperty {
	var tp TestProperty

	datastorePath := os.Getenv(datastoreProp.EnvVariable)

	datastoreReader, err := os.Open(datastorePath)
	if err != nil {
		log.Fatalln(err)
	}
	defer datastoreReader.Close()

	datastore, err := LoadAllPrintings(datastoreReader)
	if err != nil {
		log.Fatalln(err)
	}

	tp.Backend = datastore.Load()
	tp.TestDataFile = datastoreProp.TestDataFile

	testDataReader, err := os.Open(tp.TestDataFile)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.NewDecoder(testDataReader).Decode(&tp.MatchTests)
	if err != nil {
		log.Fatalln(err)
	}

	// Close the file right away so that it can be modified later
	testDataReader.Close()

	return tp
}

func TestMatch(t *testing.T) {
	for _, testSet := range MatchTestSet {
		testMatch(t, testSet)
	}
}

func testMatch(t *testing.T, testSet TestProperty) {
	SetGlobalDatastore(testSet.Backend)

	var shouldUpdateTests bool

	for i, probe := range testSet.MatchTests {
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
					testSet.MatchTests[i].Id = cardId
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
		fileWriter, err := os.Create(testSet.TestDataFile)
		if err != nil {
			t.Errorf("FAIL: Unable to update test data file: %s", err.Error())
			return
		}
		enc := json.NewEncoder(fileWriter)
		enc.SetIndent("", "    ")
		err = enc.Encode(testSet.MatchTests)
		if err != nil {
			t.Errorf("FAIL: Error while updating test data file: %s", err.Error())
			return
		}
	}
}

// This benchmark function just runs the Match tests b.N times
func BenchmarkMatch(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, test := range MatchTestSet[0].MatchTests {
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
