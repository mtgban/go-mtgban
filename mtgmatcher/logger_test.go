package mtgmatcher

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// TestNilLoggerFallsBackToThePackageLogger pins what the Logger field's
// fallback is worth while it lasts: a backend that leaves the field nil
// reports through the package logger, which is one sink every such backend
// in the process shares. This half goes when the package logger does.
func TestNilLoggerFallsBackToThePackageLogger(t *testing.T) {
	var shared bytes.Buffer
	installPackageLogger(t, &shared)

	first := candidateTestBackend()
	second := candidateTestBackend()
	first.Logf("from %s", "the first backend")
	second.Log("from the second backend")

	got := shared.String()
	if !strings.Contains(got, "from the first backend") ||
		!strings.Contains(got, "from the second backend") {
		t.Errorf("both nil-logger backends should report to the package logger:\n%s", got)
	}
}

// A backend that names a logger of its own reports there instead, so one
// datastore's diagnostics stay out of another's.
func TestBackendLogsToItsOwnLogger(t *testing.T) {
	var shared, own bytes.Buffer
	installPackageLogger(t, &shared)

	b := candidateTestBackend()
	b.Logger = log.New(&own, "", 0)
	b.Logf("from %s", "the backend's own logger")

	if got := own.String(); !strings.Contains(got, "from the backend's own logger") {
		t.Errorf("the backend's own logger received nothing:\n%s", got)
	}
	if shared.Len() != 0 {
		t.Errorf("the package logger received it too:\n%s", shared.String())
	}
}

// installPackageLogger points the package logger at buf for the duration of
// the test, and puts the previous one back after it.
func installPackageLogger(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	previous := Logger
	t.Cleanup(func() { SetGlobalLogger(previous) })
	SetGlobalLogger(log.New(buf, "", 0))
}
