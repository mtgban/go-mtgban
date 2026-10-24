package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Aggregate Secret Lair sets contain products released years apart. Their
// sealed entries must retain their individual release date; an ordinary set's
// products continue to inherit the set date.
func TestSealedProductReleaseDateForAggregateSecretLairSets(t *testing.T) {
	for _, tc := range []struct {
		setCode string
		want    string
	}{
		{"SLD", "2025-11-03"},
		{"SLC", "2025-11-03"},
		{"SLU", "2025-11-03"},
		{"MOM", ""},
	} {
		t.Run(tc.setCode, func(t *testing.T) {
			uuids := map[string]*mtgmatcher.CardObject{}
			generateSealedUUIDs(SealedProduct{
				Name:        "Product",
				ReleaseDate: "2025-11-03",
				SetCode:     tc.setCode,
				UUID:        "product-" + tc.setCode,
			}, uuids, "Edition")

			got := uuids["product-"+tc.setCode].OriginalReleaseDate
			if got != tc.want {
				t.Errorf("OriginalReleaseDate = %q, want %q", got, tc.want)
			}
		})
	}
}
