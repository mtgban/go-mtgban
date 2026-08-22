package mtgmatcher

import "testing"

// The fixture and every expectation below come from the real Cardmarket
// and StarCityGames catalogs as of 2026-08-10, run against the published
// Riftbound and Lorcana datastores during tuning. The negative cases are
// actual mismatches earlier drafts produced; they are the contract.
func sealedResolveBackend() *Backend {
	b := &Backend{
		UUIDs: map[string]*CardObject{},
		// The cards a sealed name may be decorated with; a bracket only
		// forgives what the game itself has a card for.
		CanonicalNames: map[string]string{
			Normalize("Zapdos"):   "Zapdos",
			Normalize("Articuno"): "Articuno",
			Normalize("Pikachu"):  "Pikachu",
		},
		Sets: map[string]*Set{
			"OGN": {Name: "Origins", Code: "OGN"},
			"SFD": {Name: "Spiritforged", Code: "SFD"},
			"VEN": {Name: "Vendetta", Code: "VEN"},
			"1":   {Name: "The First Chapter", Code: "1"},
			"6":   {Name: "Azurite Sea", Code: "6"},
			"9":   {Name: "Fabled", Code: "9"},
			// One Piece: two sets each have a Box Promotion Pack, and a
			// third is named for the promos, donating "promotion" to the
			// pooled set vocabulary.
			"OP01":  {Name: "Romance Dawn", Code: "OP01"},
			"OP02":  {Name: "Paramount War", Code: "OP02"},
			"OP-PR": {Name: "One Piece Promotion Cards", Code: "OP-PR"},
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
	// Products filed under a set: the set a product belongs to is what
	// makes the set words in a vendor's name its own rather than noise.
	for uuid, product := range map[string]struct{ name, setCode string }{
		"op01-display":  {"Romance Dawn - Booster Box", "OP01"},
		"op01-boxpromo": {"Box Promotion Pack", "OP01"},
		"op02-pack":     {"Paramount War - Booster Pack", "OP02"},
		"op02-display":  {"Paramount War - Booster Box", "OP02"},
		"op02-case":     {"Paramount War - Booster Box Case", "OP02"},
		"op02-boxpromo": {"Box Promotion Pack", "OP02"},
		"oppr-winner-1": {"Winner Pack Vol. 1", "OP-PR"},
		// A catalog decorates a theme deck with the creature printed
		// beside it, which no storefront repeats; the two decks differ
		// only by their own names.
		"ogn-deck-storm": {`Origins Theme Deck - "Storm Rider" [Zapdos]`, "OGN"},
		"ogn-deck-blast": {`Origins Theme Deck - "Aurora Blast" [Articuno]`, "OGN"},
		// A run is not decoration: these two are different products, and
		// nothing else answers to the name without one.
		"ven-elite-1e":  {"Vendetta Elite Box [1st Edition]", "VEN"},
		"ven-elite-unl": {"Vendetta Elite Box [Unlimited Edition]", "VEN"},
	} {
		b.UUIDs[uuid] = &CardObject{
			Card:   Card{UUID: uuid, Name: product.name, SetCode: product.setCode},
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
		{
			// The promo handed out with a box purchase. Its words are a
			// rearrangement of the box's own, so counting tokens made the
			// box look like the better answer - and "promotion" passed as
			// filing noise because another set is named for the promos.
			desc: "box promotion booster is the promo, not the box",
			name: "Paramount War Box Promotion Booster",
			want: "op02-boxpromo",
		},
		{
			// Same product name in two sets: the set the vendor names is
			// the one that owns it.
			desc: "box promotion booster picks the set it is filed under",
			name: "Romance Dawn Box Promotion Booster",
			want: "op01-boxpromo",
		},
		{
			desc: "the plain box still resolves to the box",
			name: "Paramount War Booster Box",
			want: "op02-display",
		},
		{
			desc: "the case still outranks the box it contains",
			name: "Paramount War Booster Box Case (12x Booster Box)",
			want: "op02-case",
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
		// Cardmarket marks the ones it names no language for this way.
		"Romance Dawn Booster Box (Non-English)":                        true,
		"Romance Dawn Booster Box Case (12x Booster Box) (Non-English)": true,
		// The word alone is not the mark: this is the English product.
		"English Booster Box":                   false,
		"Romance Dawn Booster Box (Pre-Errata)": false,
	} {
		if got := SealedIsLanguageVariant(name); got != want {
			t.Errorf("SealedIsLanguageVariant(%q) = %t, want %t", name, got, want)
		}
	}
}

// TestSealedTokensFoldAccents pins that an accented letter reads as a letter
// rather than a word break. The token pattern is plain ASCII, so without the
// fold "Pokémon" splits into "pok" and "mon" and matches nothing the catalog
// spells without the accent - which is how every catalog but CardTrader's
// and Cardmarket's spells it.
func TestSealedTokensFoldAccents(t *testing.T) {
	for _, tt := range []struct{ accented, plain string }{
		{"Pokémon Center Elite Trainer Box", "Pokemon Center Elite Trainer Box"},
		{"Mythical Pokémon Collection", "Mythical Pokemon Collection"},
	} {
		got := sealedTokens(tt.accented)
		want := sealedTokens(tt.plain)
		if !tokensEqual(got, want) {
			t.Errorf("sealedTokens(%q) = %v, want the same as %q = %v", tt.accented, got, tt.plain, want)
		}
	}
}

// TestSealedQualifierTokens pins which words a catalog name only decorates a
// product with. A bracket holding a card of the game's own is the creature
// printed beside a product, which no storefront repeats; a bracket holding
// anything else is the product's identity - the print run, the number of
// copies, the placing a promo was handed out for - and forgiving those would
// merge products that differ by nothing else.
func TestSealedQualifierTokens(t *testing.T) {
	for _, tt := range []struct {
		name string
		want []string
	}{
		// The creature on the box says nothing about which deck it is.
		{`Origins Theme Deck - "Storm Rider" [Zapdos]`, []string{"zapdos"}},
		{`Origins Theme Deck - "Aurora Blast" [Articuno]`, []string{"articuno"}},
		// Only a card of the game's own is decoration. A print run, a
		// count and a placing are the product's identity.
		{"Fossil Booster Pack [1st Edition]", nil},
		{"Origins Booster Pack [Set of 4]", nil},
		{"Origins Promo [1st Place]", nil},
		// A word the name also carries outside the brackets is doing the
		// product's own work there, so it is not forgiven.
		{"Pikachu Collection [Pikachu]", nil},
		// Nothing bracketed, nothing forgiven.
		{"XY Phantom Forces Booster Pack", nil},
	} {
		got := sealedResolveBackend().sealedQualifierTokens(tt.name)
		if len(got) != len(tt.want) {
			t.Errorf("sealedQualifierTokens(%q) = %v, want %v", tt.name, got, tt.want)
			continue
		}
		for _, tok := range tt.want {
			if !got[tok] {
				t.Errorf("sealedQualifierTokens(%q) = %v, missing %q", tt.name, got, tok)
			}
		}
	}
}

// TestResolveSealedForgivesQualifier pins the resolution the forgiveness is
// for: a storefront names a theme deck by its set and its own name, where the
// catalog also names the creature printed beside it. A print run is not
// forgiven, so a name saying neither run stays ambiguous rather than
// picking one.
func TestResolveSealedForgivesQualifier(t *testing.T) {
	b := sealedResolveBackend()

	for _, tt := range []struct{ vendor, want string }{
		{"Origins: Storm Rider Theme Deck", "ogn-deck-storm"},
		{"Origins Aurora Blast Theme Deck", "ogn-deck-blast"},
	} {
		uuid, err := b.ResolveSealed(tt.vendor)
		if err != nil {
			t.Errorf("ResolveSealed(%q) = %v", tt.vendor, err)
			continue
		}
		if uuid != tt.want {
			t.Errorf("ResolveSealed(%q) = %q, want %q", tt.vendor, uuid, tt.want)
		}
	}

	// Both runs answer to a name saying neither, so it resolves to none.
	if _, err := b.ResolveSealed("Vendetta Elite Box"); err == nil {
		t.Error("a name naming no print run resolved to one of the two anyway")
	}
}
