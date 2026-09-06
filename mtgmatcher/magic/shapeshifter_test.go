package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestShapeshifterByEditionCode pins that the card and the token sharing
// the name are told apart when the storefront names the edition by its
// code, the way Card Kingdom does, as well as by its name.
func TestShapeshifterByEditionCode(t *testing.T) {
	for _, tt := range []struct {
		edition string
		layout  string
	}{
		{"4ED", "normal"},
		{"ATQ", "normal"},
		{"Fourth Edition", "normal"},
		{"TC15", "token"},
		{"Commander 2015 Tokens", "token"},
	} {
		t.Run(tt.edition, func(t *testing.T) {
			cardID, err := mtgmatcher.Match(&mtgmatcher.InputCard{
				Name:    "Shapeshifter",
				Edition: tt.edition,
			})
			if err != nil {
				t.Fatalf("Match(Shapeshifter, %s) = %v", tt.edition, err)
			}
			co, _ := mtgmatcher.GetUUID(cardID)
			if co.Layout != tt.layout {
				t.Errorf("Match(Shapeshifter, %s) = %v, want a %s", tt.edition, co, tt.layout)
			}
		})
	}
}
