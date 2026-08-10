package mtgmatcher

import "testing"

// The fixture and every expectation below come from the real Cardmarket
// and StarCityGames catalogs as of 2026-08-10, run against the published
// Riftbound and Lorcana datastores during tuning. The negative cases are
// actual mismatches earlier drafts produced; they are the contract.
func sealedResolveBackend() *Backend {
	b := &Backend{
		UUIDs: map[string]*CardObject{},
		Sets: map[string]*Set{
			"OGN": {Name: "Origins", Code: "OGN"},
			"SFD": {Name: "Spiritforged", Code: "SFD"},
			"VEN": {Name: "Vendetta", Code: "VEN"},
			"1":   {Name: "The First Chapter", Code: "1"},
			"6":   {Name: "Azurite Sea", Code: "6"},
			"9":   {Name: "Fabled", Code: "9"},
		},
	}
	for uuid, name := range map[string]string{
		"ogn-pack":       "Origins - Booster Pack",
		"ogn-display":    "Origins - Booster Display",
		"ogn-case":       "Origins - Booster Display Case",
		"ogn-sleeved":    "Origins - Sleeved Booster Pack",
		"sfd-pack":       "Spiritforged - Booster Pack",
		"sfd-nexus":      "Spiritforged - Nexus Night Promo Pack",
		"ven-pack":       "Vendetta - Booster Pack",
		"ven-showdown":   "Showdown Decks: Zed vs Shen",
		"ven-showdown-d": "Showdown Decks: Zed vs Shen Display",
		"tfc-pack":       "Disney Lorcana: The First Chapter Booster Pack",
		"tfc-display":    "Disney Lorcana: The First Chapter Booster Box",
		"tfc-case":       "Disney Lorcana: The First Chapter Booster Box Case",
		"tfc-trove":      "Disney Lorcana: The First Chapter Illumineer's Trove",
		"stitch-gift":    "Disney Lorcana: Stitch Collector's Gift Set",
		"fabled-pack":    "Disney Lorcana: Fabled Booster Pack",
		"quest-trouble":  "Disney Lorcana: Illumineer's Quest: Deep Trouble",
		"fabled-display": "Disney Lorcana: Fabled Booster Box",
	} {
		b.UUIDs[uuid] = &CardObject{
			Card:   Card{UUID: uuid, Name: name},
			Sealed: true,
		}
		b.AllSealedUUIDs = append(b.AllSealedUUIDs, uuid)
	}
	return b
}

func TestResolveSealed(t *testing.T) {
	b := sealedResolveBackend()

	tests := []struct {
		desc string
		name string
		want string
	}{
		{
			desc: "marketplace box is our display",
			name: "Spiritforged Booster Box",
			want: "", // no spiritforged display in the fixture: unique-or-nothing
		},
		{
			desc: "bare booster is the pack",
			name: "Vendetta Booster",
			want: "ven-pack",
		},
		{
			desc: "versus folds onto vs, display variant stays apart",
			name: "Riftbound: League of Legends TCG - Vendetta Showdown Decks - Zed versus Shen",
			want: "ven-showdown",
		},
		{
			desc: "box folds onto display",
			name: "Origins Booster Box",
			want: "ogn-display",
		},
		{
			desc: "count decorations are safe extras",
			name: "The First Chapter 4 Booster Box Case",
			want: "tfc-case",
		},
		{
			desc: "case case picks the most specific candidate",
			name: "Origins Case (6x Booster Box)",
			want: "ogn-case",
		},
		{
			desc: "set-name decoration is a safe extra",
			name: "Lorcana: Azurite Sea - Stitch Collector's Gift Set",
			want: "stitch-gift",
		},
		{
			desc: "vendor prefix and punctuation are transparent",
			name: "Lorcana: The First Chapter Illumineer's Trove",
			want: "tfc-trove",
		},
		{
			desc: "promo booster resolves to the promo pack, not the plain one",
			name: "Spiritforged Nexus Night Promo Booster",
			want: "sfd-nexus",
		},
		{
			desc: "negative: prerelease pack must not resolve to the booster pack",
			name: "Fabled Prerelease Pack",
			want: "",
		},
		{
			desc: "negative: participation booster is a different product",
			name: "Fabled Participation Booster",
			want: "",
		},
		{
			desc: "negative: a promo booster of a set without one stays unresolved",
			name: "Vendetta Nexus Night Promo Booster",
			want: "",
		},
		{
			desc: "negative: gold victor's pack is not the quest box",
			name: "Illumineer's Quest: Deep Trouble Gold Victor's Pack",
			want: "",
		},
		{
			desc: "negative: sleeved wording must not reach the plain pack",
			name: "Vendetta Sleeved Booster",
			want: "",
		},
		{
			desc: "negative: a case without a case product stays unresolved",
			name: "Riftbound: League of Legends TCG - Vendetta Booster Case",
			want: "",
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			uuid, err := b.ResolveSealed(test.name)
			if test.want == "" {
				if err == nil {
					co := b.UUIDs[uuid]
					t.Fatalf("ResolveSealed(%q) = %q (%s), want no match", test.name, uuid, co.Name)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveSealed(%q) = %v, want %q", test.name, err, test.want)
			}
			if uuid != test.want {
				t.Errorf("ResolveSealed(%q) = %q, want %q", test.name, uuid, test.want)
			}
		})
	}
}

func TestSealedIsLanguageVariant(t *testing.T) {
	for name, want := range map[string]bool{
		"Origins Booster Box (Chinese, Slim)":                       true,
		"The First Chapter Japanese Booster Box":                    true,
		"Rise of the Floodborn Simplified Chinese Slim Booster Box": true,
		"Origins Booster Box (French)":                              true,
		"Origins Booster Box":                                       false,
		"The First Chapter Illumineer's Trove":                      false,
	} {
		if got := SealedIsLanguageVariant(name); got != want {
			t.Errorf("SealedIsLanguageVariant(%q) = %t, want %t", name, got, want)
		}
	}
}
