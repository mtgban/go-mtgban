package starcitygames

import "testing"

// TestRiftboundEpithetOnlyChampions pins Unleashed's alternate-art and
// signature champions that the gallery files under their epithet alone
// ("Green Father", not "Ivern, Green Father") while SCG's storefront still
// writes the champion-first dash spelling ("Ivern - Green Father").
//
// Normalizing strips both the dash and a comma, so that dash spelling
// collides with an entirely different, unrelated card: "Ivern, Green
// Father" is itself a real, non-promotional name, carried by a Secret
// Garden printing that has nothing to do with Unleashed. Prefilter used to
// trust that direct match and return immediately, because it is not
// promo-only - one of its two printings sits in a real, non-promo set. The
// fixtures are copied from a captured CI run verbatim; every one of them
// failed as "unknown variant" before the edition-conflict retry was added.
func TestRiftboundEpithetOnlyChampions(t *testing.T) {
	b := withGameDatastore(t, "riftbound", "RIFTBOUND_PATH")

	for _, tt := range []struct {
		sku, name, number string
		wantNumber        string
	}{
		{sku: "SGL-RIFT-UNL-195-ENF", name: "Ivern - Green Father", number: "195", wantNumber: "195"},
		{sku: "SGL-RIFT-UNL-233-ENA", name: "Ivern - Green Father", number: "233", wantNumber: "233"},
		{sku: "SGL-RIFT-UNL-233s-ENA", name: "Ivern - Green Father", number: "233s", wantNumber: "233*"},
		{sku: "SGL-RIFT-UNL-189-ENF", name: "Lillia - Bashful Bloom", number: "189", wantNumber: "189"},
		{sku: "SGL-RIFT-UNL-230-ENA", name: "Lillia - Bashful Bloom", number: "230", wantNumber: "230"},
		{sku: "SGL-RIFT-UNL-230s-ENA", name: "Lillia - Bashful Bloom", number: "230s", wantNumber: "230*"},
	} {
		t.Run(tt.sku, func(t *testing.T) {
			p := CatalogProduct{
				SKU: tt.sku, Name: tt.name, Game: "Riftbound", Set: "Unleashed",
				Finish: "Foil", FinishGroup: "Foil", CollectorNumber: tt.number,
				ProductType: ProductTypeSingles,
			}
			id, err := resolveProduct(b, GameRiftbound, p)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.sku, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != "UNL" || co.Number != tt.wantNumber {
				t.Errorf("%s resolved to %s #%s, want UNL #%s", tt.sku, co.SetCode, co.Number, tt.wantNumber)
			}
		})
	}
}

// TestRiftboundAmbiguousOrganizedPlayPromoRefuses pins the one case in the
// whole Riftbound datastore where two different promotional sets carry the
// same name at the same number - Organized Play's and the plain
// Promotional Cards set's own "Jinx, Rebel" #202 - with nothing else
// distinguishing them: identical rarity, identical finish, no promo type on
// either. SCG's own catalog set bucket ("Promotional Cards") does not say
// which either. Resolving would mean guessing; refusing is correct.
func TestRiftboundAmbiguousOrganizedPlayPromoRefuses(t *testing.T) {
	b := withGameDatastore(t, "riftbound", "RIFTBOUND_PATH")

	p := CatalogProduct{
		SKU: "SGL-RIFT-PRM-RLS_OGN_202-ENF", Name: "Jinx - Rebel", Game: "Riftbound",
		Set: "Promotional Cards", Finish: "Foil", FinishGroup: "Foil",
		CollectorNumber: "202", ProductType: ProductTypeSingles,
	}
	if _, err := resolveProduct(b, GameRiftbound, p); err == nil {
		t.Fatalf("resolveProduct(%s) resolved, want a refusal", p.SKU)
	}
}
