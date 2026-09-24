package mtgseattle

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestMaxConcurrencyOnlyLowers pins the registration override: a caller can
// ask for fewer workers than defaultConcurrency, never more, because the
// site throttles hard past it.
func TestMaxConcurrencyOnlyLowers(t *testing.T) {
	b := &mtgmatcher.Backend{Game: "magic"}

	tests := []struct {
		name string
		opts []mtgban.Option
		want int
	}{
		{"no override", nil, defaultConcurrency},
		{"higher override is ignored", []mtgban.Option{mtgban.WithMaxConcurrency(4)}, defaultConcurrency},
		{"lower override is honored", []mtgban.Option{mtgban.WithMaxConcurrency(1)}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := mtgban.NewScraper(b, "mtgseattle", tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			ms, ok := s.(*MTGSeattle)
			if !ok {
				t.Fatalf("NewScraper returned %T, want *MTGSeattle", s)
			}
			if ms.maxConcurrency != tt.want {
				t.Errorf("maxConcurrency = %d, want %d", ms.maxConcurrency, tt.want)
			}
		})
	}
}
