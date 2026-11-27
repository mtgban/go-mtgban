package matcher

import (
	"maps"
	"testing"
)

func sealedIDsBackend() *Backend {
	return &Backend{
		UUIDs:          map[string]*CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*Set{
			"OGN": {Name: "Origins", Code: "OGN"},
		},
	}
}

func TestAddSealedIdentifiers(t *testing.T) {
	b := sealedIDsBackend()
	ids := map[string]string{"tcgplayerProductId": "5", "mcmId": "7", "csiId": "C1", "scgId": ""}
	b.AddSealed("ogn-box", "Booster Box", "OGN", "", ids)

	want := map[string]string{"tcgplayerProductId": "5", "mcmId": "7", "csiId": "C1"}
	co, err := b.GetUUID("ogn-box")
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(co.Identifiers, want) {
		t.Errorf("Identifiers = %v, want %v", co.Identifiers, want)
	}
	products := b.Sets["OGN"].SealedProduct
	if len(products) != 1 || !maps.Equal(products[0].Identifiers, want) {
		t.Errorf("SealedProduct = %+v, want one product with %v", products, want)
	}

	// The caller's map is not the one stored
	ids["mcmId"] = "changed"
	if co.Identifiers["mcmId"] != "7" {
		t.Errorf(`Identifiers["mcmId"] = %q after a write to the caller's map, want "7"`, co.Identifiers["mcmId"])
	}
}

func TestAddSealedNoIdentifiers(t *testing.T) {
	b := sealedIDsBackend()
	b.AddSealed("ogn-box", "Booster Box", "OGN", "", nil)
	b.AddSealed("ogn-bundle", "Bundle", "OGN", "", map[string]string{"mcmId": ""})

	for _, uuid := range []string{"ogn-box", "ogn-bundle"} {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.Identifiers != nil {
			t.Errorf("%s Identifiers = %v, want nil", uuid, co.Identifiers)
		}
	}
	for _, product := range b.Sets["OGN"].SealedProduct {
		if product.Identifiers != nil {
			t.Errorf("SealedProduct %s Identifiers = %v, want nil", product.UUID, product.Identifiers)
		}
	}
}

func TestSealedIdentifiers(t *testing.T) {
	for _, tt := range []struct {
		name        string
		identifiers map[string]string
		tcgPlayerID int
		want        map[string]string
	}{
		{"map only", map[string]string{"mcmId": "7"}, 0, map[string]string{"mcmId": "7"}},
		{"int only", nil, 5, map[string]string{"tcgplayerProductId": "5"}},
		{"map wins", map[string]string{"tcgplayerProductId": "6", "mcmId": "7"}, 5, map[string]string{"tcgplayerProductId": "6", "mcmId": "7"}},
		{"both merged", map[string]string{"mcmId": "7"}, 5, map[string]string{"tcgplayerProductId": "5", "mcmId": "7"}},
		{"neither", nil, 0, nil},
		{"empty in the map", map[string]string{"tcgplayerProductId": ""}, 5, map[string]string{"tcgplayerProductId": "5"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := SealedIdentifiers(tt.identifiers, tt.tcgPlayerID)
			if !maps.Equal(got, tt.want) {
				t.Errorf("SealedIdentifiers(%v, %d) = %v, want %v", tt.identifiers, tt.tcgPlayerID, got, tt.want)
			}
		})
	}

	// The entry's own map is left as it was
	identifiers := map[string]string{"mcmId": "7"}
	SealedIdentifiers(identifiers, 5)
	if len(identifiers) != 1 {
		t.Errorf("identifiers = %v after the call, want only mcmId", identifiers)
	}
}
