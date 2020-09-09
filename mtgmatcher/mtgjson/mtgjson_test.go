package mtgjson

import (
	"os"
	"testing"
)

func TestLoadAllPrintings(t *testing.T) {
	allprintingsPath := os.Getenv("ALLPRINTINGS5_PATH")
	if allprintingsPath == "" {
		t.Errorf("Need ALLPRINTINGS5_PATH variable set to run tests")
		return
	}

	allPrintingsReader, err := os.Open(allprintingsPath)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	defer allPrintingsReader.Close()

	allprints, err := LoadAllPrintings(allPrintingsReader)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if len(allprints.Data) > 0 {
		t.Logf("Loaded %d entries", len(allprints.Data))
		t.Logf("Using version %s", allprints.Meta.Version)
	}
}
