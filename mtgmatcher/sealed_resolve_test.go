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
			// Pokemon really is filed under sets named "Blister
			// Exclusives" and "EX Dragon", which donate those words to
			// the pooled set vocabulary and so let a vendor say them
			// about any product.
			"BLE": {Name: "Blister Exclusives", Code: "BLE"},
			"DR":  {Name: "EX Dragon", Code: "DR"},
			// Yu-Gi-Oh! numbers its sequels, and every sequel's set name
			// donates its number to the pooled set vocabulary the way
			// "promotion" is donated above. A set whose whole name sits
			// inside another's does the same with its own extra word.
			"HA01": {Name: "Hidden Arsenal", Code: "HA01"},
			"HA05": {Name: "Hidden Arsenal 5: Steelswarm Invasion", Code: "HA05"},
			"SR14": {Name: "Structure Deck: Fire Kings", Code: "SR14"},
			"SDOK": {Name: "Structure Deck: Onslaught of the Fire Kings", Code: "SDOK"},
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
		// Two runs of one product: a name saying neither reaches both
		// and settles on neither.
		"ven-elite-1e":  {"Vendetta Elite Box [1st Edition]", "VEN"},
		"ven-elite-unl": {"Vendetta Elite Box [Unlimited Edition]", "VEN"},
		// The same pair, spelled the way a catalog spells it when it is
		// careless: one run gets the word "Edition" and the other does
		// not. They are no less two runs for it.
		"ven-duel-1e":  {"Vendetta Duel Box [1st Edition]", "VEN"},
		"ven-duel-unl": {"Vendetta Duel Box [Unlimited]", "VEN"},
		// One run only: nothing else answers to a name saying no run.
		"sfd-elite-1e": {"Spiritforged Elite Box [1st Edition]", "SFD"},
		// A run beside the plain product it reprints.
		"ogn-elite":    {"Origins Elite Box", "OGN"},
		"ogn-elite-1e": {"Origins Elite Box [1st Edition]", "OGN"},
		// A first edition the catalog brackets a run onto, and the
		// sequel that is a whole other product. The number is the only
		// thing the vendor says that tells them apart, and the sequel's
		// catalog name carries words no storefront repeats - so the
		// sequel is out of a terse vendor's reach and the first edition
		// must not answer in its place. The SDOK set above carries no
		// product at all, which is the real shape of the third case:
		// the vendor names a set the datastore has nothing sealed for.
		"ha01-display": {"Hidden Arsenal - Booster Box [Unlimited Edition]", "HA01"},
		"ha05-display": {"Hidden Arsenal 5: Steelswarm Invasion - Booster Box [1st Edition]", "HA05"},
		"sr14-deck":    {"Fire Kings Structure Deck [1st Edition]", "SR14"},
		// Two blisters that differ only by the creature on the pack.
		"ble-blister-z": {"2-Pack Blister [Zapdos]", "BLE"},
		"ble-blister-a": {"2-Pack Blister [Articuno]", "BLE"},
		// A storefront's own edition of a product, which that storefront
		// names for the set and nothing else. The two below say which
		// storefront and are two products, so they stay unforgiven.
		"ogn-etb-store":  {"Origins Trainer Box (Exclusive)", "OGN"},
		"ogn-etb-retail": {"Origins Trainer Box (Retail Exclusive)", "OGN"},
		"ogn-etb-eu":     {"Origins Trainer Box (EU Exclusive)", "OGN"},
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

// TestSealedQualifierTokens pins which words a storefront may leave unsaid. A
// bracket holding a card of the game's own is the creature printed beside a
// product, which no storefront repeats; a bracket holding nothing but a print
// run is the run, which a terse storefront leaves to the shelf. A bracket
// holding anything else is the product's identity - the number of copies, the
// placing a promo was handed out for - and forgiving those would merge
// products that differ by nothing else.
func TestSealedQualifierTokens(t *testing.T) {
	for _, tt := range []struct {
		name string
		want []string
	}{
		// The creature on the box says nothing about which deck it is.
		{`Origins Theme Deck - "Storm Rider" [Zapdos]`, []string{"zapdos"}},
		{`Origins Theme Deck - "Aurora Blast" [Articuno]`, []string{"articuno"}},
		// A print run is forgivable; two runs both being forgivable is
		// what leaves a name saying neither unresolved.
		{"Fossil Booster Pack [1st Edition]", []string{"1st", "edition"}},
		{"Fossil Booster Pack [Unlimited Edition]", []string{"unlimited", "edition"}},
		// A run named alongside anything else is not one of these: the
		// rest of the bracket is identity we cannot read.
		{"Fossil Booster Pack [1st Edition North American English]", nil},
		// A count and a placing are the product's identity.
		{"Origins Booster Pack [Set of 4]", nil},
		{"Origins Promo [1st Place]", nil},
		// The catalog's own edition is forgivable; a parenthetical saying
		// which storefront is the product's identity.
		{"Origins Trainer Box (Exclusive)", []string{"exclusive"}},
		{"Origins Trainer Box (Retail Exclusive)", nil},
		{"Origins Trainer Box (EU Exclusive)", nil},
		// A word the name also carries outside the brackets is doing the
		// product's own work there, so it is not forgiven.
		{"Pikachu Collection [Pikachu]", nil},
		{"1st Edition Booster Pack [1st Edition]", nil},
		// Nothing bracketed, nothing forgiven.
		{"XY Phantom Forces Booster Pack", nil},
	} {
		got := sealedQualifierTokens(sealedResolveBackend().sealedQualifierGroups(tt.name))
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

// TestResolveSealedForgivesQualifier pins the resolutions the forgiveness is
// for: a storefront names a theme deck by its set and its own name, where the
// catalog also names the creature printed beside it, and it names a product
// without saying which run when the catalog carries only one.
func TestResolveSealedForgivesQualifier(t *testing.T) {
	b := sealedResolveBackend()

	for _, tt := range []struct{ desc, vendor, want string }{
		{
			desc:   "the creature beside a deck is not the deck",
			vendor: "Origins: Storm Rider Theme Deck",
			want:   "ogn-deck-storm",
		},
		{
			desc:   "and the other deck is still the other deck",
			vendor: "Origins Aurora Blast Theme Deck",
			want:   "ogn-deck-blast",
		},
		{
			// The whole of what a print-run bracket says is which run it
			// is, and here there is only one to be.
			desc:   "a name saying no run reaches the only run there is",
			vendor: "Spiritforged Elite Box",
			want:   "sfd-elite-1e",
		},
		{
			// Both share every word the vendor said, so the run decides,
			// and the vendor named none: the product that needed no
			// forgiving is the one it described.
			desc:   "the plain product outranks the run that reprints it",
			vendor: "Origins Elite Box 6x",
			want:   "ogn-elite",
		},
		{
			// The catalog is the only one that calls it exclusive. Two
			// siblings say which storefront and are two other products,
			// and neither is reachable by a name saying no storefront.
			desc:   "the catalog's own exclusive mark is not the product",
			vendor: "Origins Trainer Box",
			want:   "ogn-etb-store",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			uuid, err := b.ResolveSealed(tt.vendor)
			if err != nil {
				t.Fatalf("ResolveSealed(%q) = %v, want %q", tt.vendor, err, tt.want)
			}
			if uuid != tt.want {
				t.Errorf("ResolveSealed(%q) = %q, want %q", tt.vendor, uuid, tt.want)
			}
		})
	}

	for _, tt := range []struct{ desc, vendor string }{
		{
			desc:   "two runs, neither named",
			vendor: "Vendetta Elite Box",
		},
		{
			// The catalog spells one run in two words and the other in
			// one. Counting the words unsaid rather than the brackets
			// would hand the product to whichever it spelled shorter.
			desc:   "two runs the catalog spells at different lengths",
			vendor: "Vendetta Duel Box",
		},
		{
			// The blisters share more of the vendor's wording than the
			// plain booster pack does, and they tie with each other.
			// Ranking by what went unsaid before what was said would
			// hand a blister's price to the booster pack.
			desc:   "a blister must not fall back on the plain pack",
			vendor: "Origins Dragon 2-Pack Blister",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			uuid, err := b.ResolveSealed(tt.vendor)
			if err == nil {
				t.Fatalf("ResolveSealed(%q) = %q (%s), want no match", tt.vendor, uuid, b.UUIDs[uuid].Name)
			}
		})
	}
}

// TestSealedQuantityTokens pins which numbers a name says as a count. A
// storefront writing "Hidden Arsenal 5" is naming the fifth Hidden Arsenal,
// not five of anything, and a number that leads a parenthetical a container
// closes is the opposite.
func TestSealedQuantityTokens(t *testing.T) {
	for _, tt := range []struct {
		name string
		want []string
	}{
		// A number opening a parenthetical the counted thing closes.
		{"Hidden Arsenal Booster Box (18 Booster)", []string{"18"}},
		{"Structure Deck: Albaz Strike Display (8 Structure Decks)", []string{"8"}},
		{"Crucible of War - Unlimited Case (4 Booster Boxes)", []string{"4"}},
		// The multiplier says it is a count on its own.
		{"Origins Case (6x Booster Box)", []string{"6x"}},
		// A sequel number, in the body of the name where a count never is.
		{"Hidden Arsenal 5 Booster Box", nil},
		{"Duelist Pack: Yusei Fudo 2 Booster", nil},
		{"151 Pokemon Center Elite Trainer Box", nil},
		// The sequel is still no count for standing beside one.
		{"Hidden Arsenal 5 Booster Box (18 Booster)", []string{"18"}},
		// A parenthetical a container does not close counts nothing.
		{"Legendary Collection (2024 Reprint)", nil},
	} {
		got := sealedQuantityTokens(tt.name)
		if len(got) != len(tt.want) {
			t.Errorf("sealedQuantityTokens(%q) = %v, want %v", tt.name, got, tt.want)
			continue
		}
		for _, tok := range tt.want {
			if !got[tok] {
				t.Errorf("sealedQuantityTokens(%q) = %v, missing %q", tt.name, got, tok)
			}
		}
	}
}

// TestResolveSealedForgivenessRunsOneWay pins the bound on the forgiveness: a
// candidate reached only because its brackets were forgiven must account for
// every word the vendor said, out of its own name, its own set, or a count the
// name marks as one. Without the bound a sequel folds onto the product it is a
// sequel to, because the pooled set vocabulary carries the sequel's number and
// the sibling set's extra word.
func TestResolveSealedForgivenessRunsOneWay(t *testing.T) {
	b := sealedResolveBackend()

	for _, tt := range []struct{ desc, vendor, want string }{
		{
			// The count the catalog spells out, on the product that
			// needs its run forgiven: still reached.
			desc:   "a quantity in parentheses is still a count",
			vendor: "Hidden Arsenal Booster Box (18 Booster)",
			want:   "ha01-display",
		},
		{
			desc:   "and the sequel reaches the sequel",
			vendor: "Hidden Arsenal 5: Steelswarm Invasion Booster Box",
			want:   "ha05-display",
		},
		{
			// Nothing here needs forgiving - the case carries no
			// bracket - so the bare count stays a safe extra.
			desc:   "a bare count still reaches a name said in full",
			vendor: "The First Chapter 4 Booster Box Case",
			want:   "tfc-case",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			uuid, err := b.ResolveSealed(tt.vendor)
			if err != nil {
				t.Fatalf("ResolveSealed(%q) = %v, want %q", tt.vendor, err, tt.want)
			}
			if uuid != tt.want {
				t.Errorf("ResolveSealed(%q) = %q, want %q", tt.vendor, uuid, tt.want)
			}
		})
	}

	for _, tt := range []struct{ desc, vendor string }{
		{
			desc:   "a sequel must not fold onto the first edition",
			vendor: "Hidden Arsenal 5 Booster Box",
		},
		{
			desc:   "nor when the catalog spells a real count beside it",
			vendor: "Hidden Arsenal 5 Booster Box (18 Booster)",
		},
		{
			// "Onslaught" is a set word, but of the other set: the one
			// this product is filed under does not say it.
			desc:   "a sibling set's own word is not filing noise",
			vendor: "Structure Deck: Onslaught of the Fire Kings",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			uuid, err := b.ResolveSealed(tt.vendor)
			if err == nil {
				t.Fatalf("ResolveSealed(%q) = %q (%s), want no match", tt.vendor, uuid, b.UUIDs[uuid].Name)
			}
		})
	}
}

// TestResolveSealedWithHint pins what the storefront's own shelf is allowed to
// settle. CardTrader names both runs of a Flesh and Blood set the same and
// tells them apart by the expansion it files them under, so the shelf is the
// only thing that says which run a blueprint is - and it says it only where
// the product name alone had already given up.
func TestResolveSealedWithHint(t *testing.T) {
	b := sealedResolveBackend()

	for _, tt := range []struct{ desc, vendor, hint, want string }{
		{
			desc:   "the shelf names the run the product name omits",
			vendor: "Vendetta Elite Box",
			hint:   "Vendetta - Unlimited",
			want:   "ven-elite-unl",
		},
		{
			// CardTrader spells the first run "First" where the catalog
			// brackets it "1st".
			desc:   "and the other run, under the storefront's spelling",
			vendor: "Vendetta Elite Box",
			hint:   "Vendetta - First",
			want:   "ven-elite-1e",
		},
		{
			desc:   "a shelf that names no run settles nothing",
			vendor: "Vendetta Elite Box",
			hint:   "Vendetta",
			want:   "",
		},
		{
			desc:   "no shelf at all is ResolveSealed",
			vendor: "Vendetta Elite Box",
			hint:   "",
			want:   "",
		},
		{
			// The name reached one answer on its own. A shelf naming the
			// other run does not get to move it - it would be answering
			// for a product the datastore does not carry.
			desc:   "a shelf cannot move an answer the name already reached",
			vendor: "Spiritforged Elite Box",
			hint:   "Spiritforged - Unlimited",
			want:   "sfd-elite-1e",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			uuid, err := b.ResolveSealedWithHint(tt.vendor, tt.hint)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("ResolveSealedWithHint(%q, %q) = %q (%s), want no match",
						tt.vendor, tt.hint, uuid, b.UUIDs[uuid].Name)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveSealedWithHint(%q, %q) = %v, want %q", tt.vendor, tt.hint, err, tt.want)
			}
			if uuid != tt.want {
				t.Errorf("ResolveSealedWithHint(%q, %q) = %q, want %q", tt.vendor, tt.hint, uuid, tt.want)
			}
		})
	}
}
