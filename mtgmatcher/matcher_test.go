package mtgmatcher_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

type MatchTest struct {
	Id   string `json:"uuid,omitempty"`
	Err  string `json:"error,omitempty"`
	Desc string `json:"description"`

	In mtgmatcher.InputCard `json:"input"`

	Wildcard bool `json:"wildcard,omitempty"`
}

type DatastoreProperty struct {
	EnvVariable  string
	TestDataFile string
}

type TestProperty struct {
	Backend      *mtgmatcher.Backend
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

	mtgmatcher.SetGlobalDatastore(MatchTestSet[0].Backend)
	mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0))

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

	datastore, err := magic.Load(datastoreReader)
	if err != nil {
		log.Fatalln(err)
	}

	tp.Backend = datastore
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

func runMatch(b *mtgmatcher.Backend, test MatchTest) (string, error) {
	card := test.In
	card.PromoWildcard = test.Wildcard

	cardId, err := b.Match(&card)
	if err == nil && test.Err != "" {
		return cardId, fmt.Errorf("expected error: %s", test.Err)
	}
	if err != nil {
		if test.Err == "" {
			return cardId, fmt.Errorf("unexpected error: %s", err.Error())
		}
		if test.Err != err.Error() {
			return cardId, fmt.Errorf("mismatched error: expected '%s', got '%s'", test.Err, err.Error())
		}
	} else if cardId != test.Id {
		return cardId, fmt.Errorf("id mismatch: expected '%s', got '%s'", test.Id, cardId)
	}

	return cardId, nil
}

func TestMatch(t *testing.T) {
	for _, testSet := range MatchTestSet {
		testMatch(t, testSet)
	}
}

func testMatch(t *testing.T, testSet TestProperty) {
	b := testSet.Backend

	var shouldUpdateTests bool

	for i, probe := range testSet.MatchTests {
		test := probe
		t.Run(test.Desc, func(t *testing.T) {
			// Need to run tests sequentially if we're updating them
			if !*UpdateTests {
				t.Parallel()
			}
			cardId, err := runMatch(b, test)
			if err != nil {
				if test.Err == "" {
					if *UpdateTests {
						t.Logf("NOTE: Updating test result from '%s' to '%s'", test.Id, cardId)
						testSet.MatchTests[i].Id = cardId
						shouldUpdateTests = true
						return
					}

					co, _ := b.GetUUID(cardId)
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
		for _, testSet := range MatchTestSet {
			backend := testSet.Backend
			for _, test := range testSet.MatchTests {
				_, err := runMatch(backend, test)
				if err != nil {
					b.Errorf("FAIL: %s", err.Error())
				} else {
					b.Log("PASS:", test.Desc)
				}
			}
		}
	}
}
