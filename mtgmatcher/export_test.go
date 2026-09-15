package mtgmatcher

import "testing"

// RealDatastore hands the external test package the Magic datastore the
// internal one loads, so the two suites sharing this binary read one copy.
func RealDatastore(t *testing.T) *Backend {
	t.Helper()
	return realDatastore(t)
}
