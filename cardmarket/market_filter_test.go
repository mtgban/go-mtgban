package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestBanHost pins the subdomain a game's own price snapshot is fetched
// from - confirmed live that mtgban publishes one snapshot per game
// subdomain, not one shared snapshot, after querying www.mtgban.com (the
// host sealedev's own Magic-only loader hardcodes) for Pokemon and
// getting back a real 200 with none of Pokemon's own uuids in it.
func TestBanHost(t *testing.T) {
	tests := []struct {
		game mtgban.Game
		want string
	}{
		{mtgban.GameMagic, "magic"},
		{mtgban.GamePokemon, "pokemon"},
		{mtgban.GameYuGiOh, "yugioh"},
		{mtgban.GameFleshAndBlood, "fleshandblood"},
		{mtgban.GameOnePiece, "onepiece"},
		{mtgban.GameLorcana, "lorcana"},
		{mtgban.GameRiftbound, "riftbound"},
	}
	for _, tt := range tests {
		if got := banHost(tt.game); got != tt.want {
			t.Errorf("banHost(%s) = %q, want %q", tt.game, got, tt.want)
		}
	}
}

// TestMarketFilterParamsCoverage pins which games are filtered: measured
// live against each game's own snapshot (once the wrong-host bug was
// found and fixed - see banHost), every non-Magic game clears this
// filter on only 9-26% of its own priced uuids, so it is worth applying
// wherever it can be measured, not only where the catalog would
// otherwise miss its budget.
func TestMarketFilterParamsCoverage(t *testing.T) {
	want := map[cm.Game]bool{
		cm.GameMagic:         true,
		cm.GamePokemon:       true,
		cm.GameYuGiOh:        true,
		cm.GameLorcana:       true,
		cm.GameRiftbound:     true,
		cm.GameFleshAndBlood: true,
		cm.GameOnePiece:      true,
	}
	if len(marketFilterParams) != len(want) {
		t.Fatalf("marketFilterParams has %d games, want %d", len(marketFilterParams), len(want))
	}
	for gameID := range want {
		if _, ok := marketFilterParams[gameID]; !ok {
			t.Errorf("gameID %d is missing from marketFilterParams", gameID)
		}
	}
}

// TestMarketFilterRequiredCoverage pins which of the seven filtered games
// truly require BanPriceKey to run at all - Magic, Pokemon and YuGiOh,
// whose own catalogs don't fit a nightly budget unfiltered (see
// marketFilterParams) - against the other four, which clear their own
// budget unfiltered too and only lose call-volume savings, not the ability
// to run, when the key is missing (see Load and Market.banPriceKey).
func TestMarketFilterRequiredCoverage(t *testing.T) {
	want := map[cm.Game]bool{
		cm.GameMagic:   true,
		cm.GamePokemon: true,
		cm.GameYuGiOh:  true,
	}
	if len(marketFilterRequired) != len(want) {
		t.Fatalf("marketFilterRequired has %d games, want %d", len(marketFilterRequired), len(want))
	}
	for gameID := range want {
		if !marketFilterRequired[gameID] {
			t.Errorf("gameID %d should require BanPriceKey", gameID)
		}
		if _, filtered := marketFilterParams[gameID]; !filtered {
			t.Errorf("gameID %d requires BanPriceKey but is not in marketFilterParams", gameID)
		}
	}
}

// TestParseBanSnapshotBodyLevelErrors pins the two ways a fetch can fail
// without ever returning a non-200 status - confirmed live against
// mtgban.com's real endpoints (a rejected sig for one game, an empty
// retail map that would otherwise read as "nothing passed the filter").
func TestParseBanSnapshotBodyLevelErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "a rejected signature answers 200 with an error body",
			body:    `{"error": "invalid or expired signature"}`,
			wantErr: true,
		},
		{
			name:    "an empty retail map is refused rather than read as a real but empty filter result",
			body:    `{"retail": {}, "buylist": {}}`,
			wantErr: true,
		},
		{
			name:    "a genuine snapshot decodes cleanly",
			body:    `{"retail": {"u1": {"TCGMarket": {"regular": 5}}}, "buylist": {}}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseBanSnapshot([]byte(tt.body), mtgban.GamePokemon)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBanSnapshot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBanPriceDecodesEtched pins the wire key itself, not just the field:
// the snapshot answers an etched printing under "etched", a third key beside
// "regular" and "foil", and reading only the first two made every etched
// price decode as zero - which marketCandidate refuses as "not priced".
func TestBanPriceDecodesEtched(t *testing.T) {
	const body = `{"retail":{"u":{"MKMTrend":{"etched":11.95}}},"buylist":{}}`
	snap, err := parseBanSnapshot([]byte(body), mtgban.GameMagic)
	if err != nil {
		t.Fatal(err)
	}
	if got := snap.retail("u", "MKMTrend"); got != 11.95 {
		t.Errorf("retail() = %v, want 11.95", got)
	}
}

func TestBanPriceValue(t *testing.T) {
	tests := []struct {
		name string
		p    *banPrice
		want float64
	}{
		{"nil is zero", nil, 0},
		{"a plain uuid reads regular", &banPrice{Regular: 1.5}, 1.5},
		{"a foil uuid reads foil", &banPrice{Foil: 2.5}, 2.5},
		{"an etched printing reads etched", &banPrice{Etched: 3.5}, 3.5},
		{"regular wins when somehow both are set", &banPrice{Regular: 1, Foil: 2}, 1},
		{"foil wins over etched when somehow both are set", &banPrice{Foil: 2, Etched: 3}, 2},
		{"none set is zero", &banPrice{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.value(); got != tt.want {
				t.Errorf("value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshotFirstAvailable(t *testing.T) {
	snap := &banSnapshot{
		Buylist: map[string]map[string]*banPrice{
			"has-ck":     {"CK": {Regular: 8}, "CSI": {Regular: 20}},
			"only-csi":   {"CSI": {Regular: 5}},
			"zero-price": {"CK": {Regular: 0}, "CSI": {Regular: 3}},
		},
	}
	order := []string{"CK", "SCG", "CSI"}

	if got := snap.firstBuylist("has-ck", order); got != 8 {
		t.Errorf("firstBuylist(has-ck) = %v, want 8 (CK first)", got)
	}
	if got := snap.firstBuylist("only-csi", order); got != 5 {
		t.Errorf("firstBuylist(only-csi) = %v, want 5 (falls through to CSI)", got)
	}
	if got := snap.firstBuylist("zero-price", order); got != 3 {
		t.Errorf("firstBuylist(zero-price) = %v, want 3 (a zero CK price is not available)", got)
	}
	if got := snap.firstBuylist("missing", order); got != 0 {
		t.Errorf("firstBuylist(missing) = %v, want 0", got)
	}
}

// TestMarketCandidatesThresholds pins the three-way filter's arithmetic
// directly, without going through a backend's GetUUIDs() - the uuids here
// are plain labels, not real ones, which is fine: marketCandidates never
// resolves them to a card, only reads them back out of the snapshot.
func TestMarketCandidatesThresholds(t *testing.T) {
	// Magic's params: $3 floor, $2 minimum absolute difference.
	var gameID cm.Game = 1 // stand-in; marketFilterParams is keyed by
	// cm.GameMagic in production, but the test below swaps the table for
	// one keyed the same way so the arithmetic is exercised without
	// depending on go-cardmarket's own constant values. The type is the
	// library's now; the number deliberately still is not.
	saved := marketFilterParams
	marketFilterParams = map[cm.Game]struct {
		floor   float64
		minDiff float64
	}{
		gameID: {floor: 3.0, minDiff: 2.0},
	}
	defer func() { marketFilterParams = saved }()

	retail := func(mkm float64, tcg float64) map[string]*banPrice {
		m := map[string]*banPrice{"MKMTrend": {Regular: mkm}}
		if tcg != 0 {
			m["TCGMarket"] = &banPrice{Regular: tcg}
		}
		return m
	}
	buylist := func(source string, price float64) map[string]*banPrice {
		return map[string]*banPrice{source: {Regular: price}}
	}

	tests := []struct {
		name string
		snap *banSnapshot
		want bool
	}{
		{
			name: "flat price over the $7 threshold is a candidate on its own",
			snap: &banSnapshot{Retail: map[string]map[string]*banPrice{"u": retail(7.01, 0)}},
			want: true,
		},
		{
			name: "flat price at exactly $7 is not (strictly greater)",
			snap: &banSnapshot{Retail: map[string]map[string]*banPrice{"u": retail(7.00, 0)}},
			want: false,
		},
		{
			name: "no Cardmarket trend price at all is never a candidate",
			snap: &banSnapshot{
				Retail:  map[string]map[string]*banPrice{"u": {}},
				Buylist: map[string]map[string]*banPrice{"u": buylist("CK", 100)},
			},
			want: false,
		},
		{
			name: "an arbit spread over 20% with enough absolute gap and floor is a candidate",
			// mkm=4, buylist=5: spread=25%, diff=$1 - fails the $2 min diff
			snap: &banSnapshot{
				Retail:  map[string]map[string]*banPrice{"u": retail(4, 0)},
				Buylist: map[string]map[string]*banPrice{"u": buylist("CK", 5)},
			},
			want: false,
		},
		{
			name: "an arbit spread clearing both the percentage and the absolute gap",
			// mkm=4, buylist=6.5: spread=62.5%, diff=$2.5 - both clear
			snap: &banSnapshot{
				Retail:  map[string]map[string]*banPrice{"u": retail(4, 0)},
				Buylist: map[string]map[string]*banPrice{"u": buylist("CK", 6.5)},
			},
			want: true,
		},
		{
			name: "an arbit spread below the game's price floor is dropped even if the spread is huge",
			// mkm=1 (below the $3 floor), buylist=10: spread=900%
			snap: &banSnapshot{
				Retail:  map[string]map[string]*banPrice{"u": retail(1, 0)},
				Buylist: map[string]map[string]*banPrice{"u": buylist("CK", 10)},
			},
			want: false,
		},
		{
			name: "the buylist vendor fallback reaches CSI when CK and SCG have nothing",
			// mkm=4, CSI buylist=6.5: same numbers as the clearing case above
			snap: &banSnapshot{
				Retail:  map[string]map[string]*banPrice{"u": retail(4, 0)},
				Buylist: map[string]map[string]*banPrice{"u": buylist("CSI", 6.5)},
			},
			want: true,
		},
		{
			name: "a mismatch spread over 80% against TCG market is a candidate",
			// mkm=6, tcg=3 (at the floor): spread=100%, diff=$3
			snap: &banSnapshot{
				Retail: map[string]map[string]*banPrice{"u": retail(6, 3)},
			},
			want: true,
		},
		{
			name: "a mismatch spread guarded by the floor on TCG market's side",
			// mkm=4, tcg=0.5 (below the $3 floor): spread would be 600%
			snap: &banSnapshot{
				Retail: map[string]map[string]*banPrice{"u": retail(4, 0.5)},
			},
			want: false,
		},
		{
			name: "neither leg clears is not a candidate",
			snap: &banSnapshot{
				Retail:  map[string]map[string]*banPrice{"u": retail(3.5, 3.2)},
				Buylist: map[string]map[string]*banPrice{"u": buylist("CK", 3.6)},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := marketCandidate(gameID, "u", tt.snap)
			if got != tt.want {
				t.Errorf("candidate = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarketCandidatesUnfilteredGame(t *testing.T) {
	if got := marketCandidates(&mtgmatcher.Backend{}, 999999, &banSnapshot{}); got != nil {
		t.Errorf("marketCandidates for an unfiltered game = %v, want nil", got)
	}
}
