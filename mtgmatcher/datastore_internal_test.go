package mtgmatcher

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/simplecloud"
)

var (
	internalDatastoreOnce sync.Once
	internalDatastoreErr  error
)

// realDatastore installs the Magic datastore the first time a test inside the
// package asks for it, reusing what the suite beside it installed where that
// got there first.
//
// It reaches simplecloud directly rather than internal/datastore, which
// decodes and so imports the package this file is part of. The decoder is
// reached through Open, which is local, and "magic" is registered by
// mtgmatcher/magic - imported by the external test files, which share this
// test binary.
func realDatastore(t *testing.T) {
	t.Helper()
	internalDatastoreOnce.Do(func() {
		if len(GetAllSets()) > 0 {
			return
		}
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		id, key := os.Getenv("B2_APPLICATION_KEY_ID"), os.Getenv("B2_APPLICATION_KEY")
		if id == "" && key == "" {
			id, key = os.Getenv("B2_APPLICATION_KEY_ID_DATASTORE"), os.Getenv("B2_APPLICATION_KEY_DATASTORE")
		}
		reader, err := simplecloud.Open(context.Background(), path,
			simplecloud.WithB2Credentials(id, key),
			simplecloud.WithConcurrentDownloads(20))
		if err != nil {
			internalDatastoreErr = err
			return
		}
		defer reader.Close()

		backend, err := Open("magic", reader)
		if err != nil {
			internalDatastoreErr = err
			return
		}
		SetGlobalDatastore(backend)
	})
	if internalDatastoreErr != nil {
		t.Fatal(internalDatastoreErr)
	}
	if len(GetAllSets()) == 0 {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
}
