package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// SCG braces anything that is not a normal card, and preprocess read
// that as "token" and suffixed the name. Only creature tokens are named
// after something else (the creature); the rest are named after what
// they are, so the suffix invented names nothing carries - every
// Unfinity sticker sheet resolved to nothing because of it.
func TestPreprocessBracedNonTokens(t *testing.T) {
	if len(mtgmatcher.GetUUIDs()) == 0 {
		t.Skip("datastore not loaded")
	}

	tests := []struct {
		desc     string
		hitName  string
		wantName string
	}{
		{
			desc:     "a sticker sheet keeps its own name",
			hitName:  "{Eldrazi Guacamole Tightrope Sticker Sheet}",
			wantName: "Eldrazi Guacamole Tightrope Sticker Sheet",
		},
		{
			desc:     "an emblem keeps its own name",
			hitName:  "{Arlinn Kord Emblem}",
			wantName: "Arlinn Kord Emblem",
		},
		{
			desc:     "a name already saying Token is left alone",
			hitName:  "{Golem Token}",
			wantName: "Golem Token",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			card, err := preprocess(Hit{
				Name:            test.hitName,
				SetName:         "Unfinity",
				Language:        "English",
				CollectorNumber: "1",
				Variants:        []Variant{{Sku: "SGL-MTG-UNF-S01-ENN"}},
			})
			if err != nil {
				t.Fatalf("preprocess: %v", err)
			}
			if card.Name != test.wantName {
				t.Errorf("name = %q, want %q", card.Name, test.wantName)
			}
		})
	}
}
