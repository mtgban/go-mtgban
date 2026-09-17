package tcgplayer

import (
	"testing"

	"github.com/mtgban/go-tcgplayer"
)

func TestIsUnsupportedProduct(t *testing.T) {
	tests := []struct {
		name        string
		unsupported bool
	}{
		{"Code Card - 30th Celebration Booster Pack", true},
		{"Code Card - 30th Celebration Elite Trainer Box", true},
		{"Code Card - 30th Celebration Pokemon Center Elite Trainer Box", true},
		{"Code Card - 30th Celebration Battle Deck [Espeon ex]", true},
		{"Mewtwo ex", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := tcgplayer.Product{Name: tt.name}
			if got := isUnsupportedProduct(&product); got != tt.unsupported {
				t.Fatalf("isUnsupportedProduct(%q) = %v, want %v", tt.name, got, tt.unsupported)
			}
		})
	}
}
