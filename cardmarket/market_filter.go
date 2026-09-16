package cardmarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// marketFilterParams are the offline pre-filter's thresholds, one row per
// game it applies to. Magic, Pokemon and YuGiOh need it to fit a nightly
// scrape budget at all; measured against each game's own price snapshot
// (see banHost - a game earlier missing from this table simply because
// the snapshot was fetched from the wrong host reads as "nothing passed
// the filter," not as "unfiltered"), Lorcana, Riftbound, Flesh and Blood
// and One Piece all fit their budget unfiltered too, but only clear this
// filter on 9-26% of their own priced uuids - the rest is bulk commons a
// live call is wasted on, and every call spends the one daily allowance
// this app token shares across every game. One Piece's own snapshot host
// briefly rejected every request with "invalid or expired signature" -
// a separate backend deployment than every other game's subdomain,
// confirmed by its own x-do-app-origin header - fixed server-side and
// re-measured clean afterward.
//
// Magic's tighter minDiff guards the arbit and mismatch legs' percentage
// spread against cent-level noise on bulk cards, where a $0.02 vs $0.10
// listing reads as an "80% spread" that means nothing; every other game
// has enough budget headroom that a floor alone is enough.
var marketFilterParams = map[int]struct {
	floor   float64
	minDiff float64
}{
	cm.GameMagic:         {floor: 3.0, minDiff: 2.0},
	cm.GamePokemon:       {floor: 1.0},
	cm.GameYuGiOh:        {floor: 1.0},
	cm.GameLorcana:       {floor: 1.0},
	cm.GameRiftbound:     {floor: 1.0},
	cm.GameFleshAndBlood: {floor: 1.0},
	cm.GameOnePiece:      {floor: 1.0},
}

// marketFilterRequired names the games in marketFilterParams whose own
// catalog does not fit a nightly budget unfiltered at all - Magic, Pokemon
// and YuGiOh (see marketFilterParams's own comment on the measured sizes).
// Load refuses to run one of these without a BanPriceKey. The other four
// games in marketFilterParams fit their budget either way, so a missing key
// there costs call volume, not correctness, and Load runs them unfiltered
// instead of refusing.
var marketFilterRequired = map[int]bool{
	cm.GameMagic:   true,
	cm.GamePokemon: true,
	cm.GameYuGiOh:  true,
}

// marketFilterVendors is the buylist vendor the Arbit leg reads from, first
// available. Card Kingdom's own data is Magic-only in mtgban's feed; Star
// City Games' and CSI's both cover every game marketFilterParams does, CSI
// more completely - measured directly against each game's own snapshot,
// not assumed from Magic's.
var marketFilterVendors = []string{"CK", "SCG", "CSI"}

// marketCandidateThreshold is the flat Cardmarket trend price above which a
// card is worth a live call on its own, no spread required.
const marketCandidateThreshold = 7.0

// marketCandidates computes the pre-filter's candidate set for the uuids of
// whichever game's datastore is currently loaded (mtgmatcher.GetUUIDs()):
// Cardmarket's own trend price over $7; or an Arbit-style spread - a US
// buylist vendor's price against that same trend price - over 20%; or a
// Mismatch-style spread - Cardmarket's trend against TCG market retail -
// over 80%. What decides whether a live call is worth spending is whether
// Cardmarket's own guide price looks likely to be stale or wrong, so
// Cardmarket's trend price anchors every leg, not TCG market - a card
// where TCG and a US buylist disagree wildly says nothing about whether
// Cardmarket's own listings are worth polling. The second and third legs
// are both guarded by the game's price floor and minimum absolute
// difference (see marketFilterParams), on the side of the spread that is
// not Cardmarket's trend, since an unguarded percentage spread is
// dominated by cent-level noise on bulk cards.
//
// A game marketFilterParams does not cover returns nil - unfiltered, not
// empty - which Load reads as "price every candidate this game has."
func marketCandidates(gameID int, snap *banSnapshot) map[string]bool {
	if _, filtered := marketFilterParams[gameID]; !filtered {
		return nil
	}
	candidates := map[string]bool{}
	for _, uuid := range mtgmatcher.GetUUIDs() {
		if marketCandidate(gameID, uuid, snap) {
			candidates[uuid] = true
		}
	}
	return candidates
}

// marketCandidate applies the filter described on marketCandidates to one
// uuid, split out so the arithmetic can be pinned directly against a
// synthetic snapshot without needing a loaded datastore behind it.
func marketCandidate(gameID int, uuid string, snap *banSnapshot) bool {
	params, filtered := marketFilterParams[gameID]
	if !filtered {
		return true
	}
	mkm := snap.retail(uuid, "MKMTrend")
	if mkm == 0 {
		return false
	}
	if mkm > marketCandidateThreshold {
		return true
	}
	if mkm >= params.floor {
		if bl := snap.firstBuylist(uuid, marketFilterVendors); bl != 0 {
			diff := bl - mkm
			if diff >= params.minDiff && 100*diff/mkm > 20 {
				return true
			}
		}
	}
	if tcg := snap.retail(uuid, "TCGMarket"); tcg != 0 && tcg >= params.floor {
		diff := mkm - tcg
		if diff >= params.minDiff && 100*diff/tcg > 80 {
			return true
		}
	}
	return false
}

// banAPIURL is the same endpoint sealedev's own price loader reads,
// per-game: mtgban publishes one price snapshot per subdomain
// (pokemon.mtgban.com, yugioh.mtgban.com, ...), each carrying only that
// game's own prices - www.mtgban.com (sealedev's own host, hardcoded
// there since it only ever prices Magic) carries Magic's alone. Querying
// it for another game answers with a real 200 and an empty result: no
// error, just nothing that game's own uuids are keyed under.
const banAPIURL = "https://%s.mtgban.com/api/mtgban/all.json?tag=tags&conds=true&sig=%s"

// banHost is the subdomain a game's own price snapshot is published
// under, lowercase, matching the game input every workflow and bucket
// path already spells it with.
func banHost(game mtgban.Game) string {
	return strings.ToLower(string(game))
}

// banPrice is one store's price for one uuid, as the mtgban price API
// answers it on the wire: "regular" for a plain uuid, "foil" for a
// "_f"-suffixed one. This is not sealedev's own BANPriceResponse shape (a
// "conditions" map keyed by grade) - that struct decodes nothing back
// against a live fetch of this same endpoint, verified directly rather than
// assumed; whatever query mode it was written for, it is not this one.
type banPrice struct {
	Regular float64 `json:"regular"`
	Foil    float64 `json:"foil"`
}

// value reads the price the wire actually populated. The API answers a
// plain uuid's price under "regular" and an "_f"-suffixed uuid's under
// "foil" - never both for the same uuid - so Regular winning when both are
// somehow set is a defensive tiebreak, not a real choice: every caller here
// looks a plain uuid up, so it is the field that should be populated.
func (p *banPrice) value() float64 {
	if p == nil {
		return 0
	}
	if p.Regular != 0 {
		return p.Regular
	}
	return p.Foil
}

// banSnapshot is the mtgban price API's response, trimmed to the fields
// marketCandidates reads plus Error - a request the API refuses (an
// invalid or expired sig, say) still answers 200 with an error body
// instead of the two fields above, which would otherwise decode as a
// real but empty snapshot and read as "nothing passed the filter" rather
// than "the fetch itself failed."
type banSnapshot struct {
	Error   string                          `json:"error"`
	Retail  map[string]map[string]*banPrice `json:"retail"`
	Buylist map[string]map[string]*banPrice `json:"buylist"`
}

func (snap *banSnapshot) retail(uuid, source string) float64 {
	return snap.Retail[uuid][source].value()
}

func (snap *banSnapshot) firstBuylist(uuid string, sources []string) float64 {
	for _, source := range sources {
		if v := snap.Buylist[uuid][source].value(); v != 0 {
			return v
		}
	}
	return 0
}

// loadBanSnapshot fetches game's own price snapshot, the one the offline
// pre-filter reads. sig authenticates it - bantool reads it from the
// BAN_API_KEY env var, the same key sealedev's own price loader uses.
func loadBanSnapshot(ctx context.Context, game mtgban.Game, sig string) (*banSnapshot, error) {
	link := fmt.Sprintf(banAPIURL, banHost(game), sig)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mtgban price API returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseBanSnapshot(data, game)
}

// parseBanSnapshot decodes one game's price snapshot body, split out from
// loadBanSnapshot so the two ways a fetch can go wrong without ever
// returning a body-level error - a rejected sig, or a game the API
// genuinely has no prices for yet - are pinned directly, without a live
// fetch to reach them. Both answer a real 200: a rejected request with an
// error body instead of the two fields marketCandidates reads, and a
// legitimate empty response no differently from one that simply is not
// published. Trusting either as "an empty result, nothing passed the
// filter" is the same silent-failure shape the empty-body-on-error bug in
// go-cardmarket's own get() was.
func parseBanSnapshot(data []byte, game mtgban.Game) (*banSnapshot, error) {
	var snap banSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	if snap.Error != "" {
		return nil, errors.New(snap.Error)
	}
	if len(snap.Retail) == 0 {
		return nil, fmt.Errorf("the price snapshot for %s carries no retail prices at all", game)
	}
	return &snap, nil
}
