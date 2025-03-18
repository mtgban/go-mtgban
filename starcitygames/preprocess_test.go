package starcitygames

import (
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

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

	err = mtgmatcher.LoadDatastore(allPrintingsReader)
	if err != nil {
		log.Fatalln(err)
	}

	mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0))

	os.Exit(m.Run())
}

type SKUTest struct {
	Name string
	In   string
	Out  string
	Err  string
}

var SKUTests = []SKUTest{
	{
		In:  "SGL-MTG-FDN2-428-ENF1",
		Out: "f91b4613-1633-5428-90ca-420174ceb533",
	},
	{
		In:  "SGL-MTG-PWSB-ELD2_291-ENN1",
		Out: "728fa13b-e7e3-5193-a7a3-d18439bb7ee5",
	},
	{
		In:  "SGL-MTG-LTR4-741-ENF1",
		Out: "9d1b6dca-03ce-5c5c-b0d4-ee294bb89421",
	},
	{
		In:  "SGL-MTG-MPS3-001-ENF1",
		Out: "cb4a260d-8f98-5e81-b881-6727d5323917",
	},
}

func TestSKU(t *testing.T) {
	for _, probe := range SKUTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out, err := ProcessSKU(test.Name, test.In)
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
			}
			if out.Id != test.Out {
				t.Errorf("FAIL %s: Expected '%s' got '%s' (%s)", test.In, test.Out, out.Id, out)
				return
			}
			t.Log("PASS:", test.In)
		})
	}
}
