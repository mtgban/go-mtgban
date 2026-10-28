package starcitygames

import "testing"

// TestRiftboundChampionNames pins that SCG's catalog names of two Riftbound
// champions resolve to their printings. Fixtures copied from a captured CI
// run verbatim.
func TestRiftboundChampionNames(t *testing.T) {
	b := withGameDatastore(t, "riftbound", "RIFTBOUND_PATH")

	for _, tt := range []struct {
		sku, name, set, number, finish string
		wantSet, wantNumber            string
	}{
		{
			sku: "SGL-RIFT-SFD-179-ENF", name: "Corina Veraza", set: "Spiritforged", number: "179",
			finish: "Foil", wantSet: "SFD", wantNumber: "179",
		},
		{
			sku: "SGL-RIFT-VEN-060-ENN", name: "Sky Cruiser", set: "Vendetta", number: "060",
			finish: "Non-foil", wantSet: "VEN", wantNumber: "60",
		},
	} {
		t.Run(tt.sku, func(t *testing.T) {
			p := CatalogProduct{
				SKU: tt.sku, Name: tt.name, Game: "Riftbound", Set: tt.set,
				Finish: tt.finish, FinishGroup: tt.finish, CollectorNumber: tt.number,
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
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
				t.Errorf("%s resolved to %s #%s, want %s #%s", tt.sku, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}
