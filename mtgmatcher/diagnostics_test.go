package mtgmatcher

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// TestMatchNamesAnIdOnlyInput pins what an id lookup leaves in the log. A
// listing that carries an id and nothing else has no name for the input to
// print, so the line naming it is the only place the card is readable at all;
// the backend running the match is what resolves the id now.
func TestMatchNamesAnIdOnlyInput(t *testing.T) {
	b := candidateTestBackend()
	b.UUIDs["a"].Edition = "Edition A"
	var buf bytes.Buffer
	b.Logger = log.New(&buf, "", 0)

	id, err := b.Match(&InputCard{ID: "a"})
	if err != nil || id != "a" {
		t.Fatalf("id lookup = %q, %v", id, err)
	}
	if got := buf.String(); !strings.Contains(got, "Test Card [Edition A]") {
		t.Errorf("the log does not name the card the id resolved to:\n%s", got)
	}
}

// An input that says what it is keeps its own wording: the datastore is read
// for the cards that have nothing else to be named by, not to correct a
// listing whose name is the thing being diagnosed.
func TestMatchLogsTheNameTheListingGave(t *testing.T) {
	b := candidateTestBackend()
	b.UUIDs["a"].Edition = "Edition A"
	var buf bytes.Buffer
	b.Logger = log.New(&buf, "", 0)

	id, err := b.Match(&InputCard{ID: "a", Name: "Whatever The Shop Called It"})
	if err != nil || id != "a" {
		t.Fatalf("id lookup = %q, %v", id, err)
	}
	got := buf.String()
	if !strings.Contains(got, "Whatever The Shop Called It") {
		t.Errorf("the log dropped the name the listing gave:\n%s", got)
	}
	if strings.Contains(got, "Test Card") {
		t.Errorf("the log replaced the listing's name with the datastore's:\n%s", got)
	}
}
