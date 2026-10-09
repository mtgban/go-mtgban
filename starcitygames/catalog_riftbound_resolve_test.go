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
// Garden printing that has nothing to do with Unleashed. Prefilter checks that
// direct match against the edition rather than trusting it for not being
// promo-only - one of its two printings sits in a real, non-promo set. Trusting
// it fails every fixture as "unknown variant"; the fixtures are copied from a
// captured CI run verbatim.
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

// TestRiftboundPromoSkuEvent pins the promo skus whose generic "Promotional
// Cards" set says nothing: the event and set in the sku place them. Jinx,
// Rebel #202 exists in both Organized Play and the plain promotional set,
// and only the release-event prefix names the first. The Spiritforged runes
// "R01b" share their wording with the Vendetta runes, which a Vendetta sku
// keeps.
func TestRiftboundPromoSkuEvent(t *testing.T) {
	b := withGameDatastore(t, "riftbound", "RIFTBOUND_PATH")

	for _, tt := range []struct {
		sku, name, number string
		wantSet, wantNo   string
	}{
		{"SGL-RIFT-PRM-RLS_OGN_202-ENF", "Jinx - Rebel", "202", "OPP", "202"},
		{"SGL-RIFT-PRM-NN_SFD_R01b-ENF", "Fury Rune", "R01b", "SFD", "R1b"},
		{"SGL-RIFT-PRM-NN_SFD_R04b-ENF", "Body Rune", "R04b", "SFD", "R4b"},
		{"SGL-RIFT-PRM-NN_VEN_R04b-ENF", "Body Rune", "R04b", "OPP", "R4b"},
		{"SGL-RIFT-PRM-NN_SFD_007-ENF", "Gem Jammer", "007", "OPP", "7"},
	} {
		t.Run(tt.sku, func(t *testing.T) {
			p := CatalogProduct{
				SKU: tt.sku, Name: tt.name, Game: "Riftbound",
				Set: "Promotional Cards", Finish: "Foil", FinishGroup: "Foil",
				CollectorNumber: tt.number, ProductType: ProductTypeSingles,
			}
			id, err := resolveProduct(b, GameRiftbound, p)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.sku, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNo {
				t.Errorf("%s resolved to %s #%s, want %s #%s", tt.sku, co.SetCode, co.Number, tt.wantSet, tt.wantNo)
			}
		})
	}
}
