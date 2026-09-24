package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Prefilter's Shapeshifter->token rename must stand aside when the named
// edition (by set code or set name) holds a real, non-token Shapeshifter of
// its own.
func TestShapeshifterTokenRenameKeepsRealCardByEdition(t *testing.T) {
	realDatastore(t)
	b := testBackend

	for _, tc := range []struct {
		edition string
		want    string
	}{
		{"4ED", "53b8c030-d92e-5d30-96a6-0ae6ff682aeb"},
		{"5ED", "9a040a5b-4d0f-59b6-bce6-7ceb53f8f97a"},
		{"ATQ", "484f1f05-d8e9-5ef3-a8b8-07a9edc23f5e"},
		{"Renaissance", "800d535e-6894-5995-affa-bbc23345caac"},
	} {
		in := &mtgmatcher.InputCard{Name: "Shapeshifter", Edition: tc.edition}
		id, err := b.Match(in)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.edition, err)
			continue
		}
		if id != tc.want {
			t.Errorf("%s: id = %q, want %q", tc.edition, id, tc.want)
		}
	}

	// A set that genuinely holds only the token (no real Shapeshifter
	// card) must still rename, same as before.
	in := &mtgmatcher.InputCard{Name: "Shapeshifter", Edition: "2XM"}
	Rules{}.Prefilter(b, in)
	if in.Name != "Shapeshifter Token" {
		t.Errorf("2XM: Name = %q, want %q", in.Name, "Shapeshifter Token")
	}
}
