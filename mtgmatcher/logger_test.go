package mtgmatcher

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// TestBackendLogsToItsOwnLogger pins that a backend reports its diagnostics
// to the logger it names, so one datastore's diagnostics stay out of
// another's. There is no package logger left for a backend naming none to
// fall back to.
func TestBackendLogsToItsOwnLogger(t *testing.T) {
	var own bytes.Buffer
	b := candidateTestBackend()
	b.Logger = log.New(&own, "", 0)
	b.Logf("from %s", "the backend's own logger")

	if got := own.String(); !strings.Contains(got, "from the backend's own logger") {
		t.Errorf("the backend's own logger received nothing:\n%s", got)
	}
}
