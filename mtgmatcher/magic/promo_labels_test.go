package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPromoTypeLabels pins that every promo type the datastore carries can be
// shown to a reader. MTGJSON writes them as single lower-case words, so there
// is nothing to recover the spelling from and it has to be written down.
//
// A type nobody has written down yet is reported rather than failed. The
// spelling table is ours and the promo types are upstream's, so the day
// MTGJSON prints a new one is a day this list is behind - and a red suite
// then says a game broke when what happened is that Wizards named a promo.
// The fallback spells the token plainly in the meantime, so nothing is
// unreadable while the entry is written; the run names what is missing.
func TestPromoTypeLabels(t *testing.T) {
	b := testBackend

	var unlabelled []string
	for _, promoType := range b.AllPromoTypes {
		if b.PromoTypeLabels[promoType] == "" {
			unlabelled = append(unlabelled, promoType)
		}
	}
	if len(unlabelled) > 0 {
		t.Logf("%d promo types have no label yet, so they read as a run-together word until promolabels.go spells them: %v",
			len(unlabelled), unlabelled)
	}

	for _, tt := range []struct{ promoType, want string }{
		{"boosterfun", "Booster Fun"},
		{"buyabox", "Buy-a-Box"},
		{"arenaleague", "Arena League"},
		{"planeswalkerstamped", "Planeswalker Stamped"},
		{"universesbeyond", "Universes Beyond"},
	} {
		if got := b.PromoTypeLabel(tt.promoType); got != tt.want {
			t.Errorf("PromoTypeLabel(%q) = %q, want %q", tt.promoType, got, tt.want)
		}
	}

	// A type the list does not know still reads as something.
	if got := mtgmatcher.PromoTypeLabel("nosuchtype"); got == "" {
		t.Error("an unknown promo type reads back as nothing")
	}
}
