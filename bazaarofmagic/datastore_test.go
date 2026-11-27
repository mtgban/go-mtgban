package bazaarofmagic

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/magic"
)

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
		backend, err := datastore.Read("magic", path)
		if err != nil {
			datastoreErr = err
			return
		}
		datastoreBackend = backend
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if datastoreBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return datastoreBackend
}
