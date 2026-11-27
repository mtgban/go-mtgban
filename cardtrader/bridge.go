package cardtrader

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/mtgban/go-mtgban/matcher"
)

// SealedBridge maps every Cardmarket product id to the TCGplayer id of the
// same product, read off the game's blueprints - the one source linking the
// two marketplaces' ids - with the corrections tcgIDOverrides keeps for the
// ids Card Trader sends wrong or not at all. The blueprints cover singles and
// sealed alike, so the same bridge serves both cardmarket scrapers.
//
// It is strict: an expansion that fails to fetch fails the call, with how
// many were skipped and the first, and so does a game with no blueprints at
// all, since a short map reads as links the store withdrew.
// SealedBridgeLogged keeps what fetched instead.
func SealedBridge(ctx context.Context, token string, game matcher.Game) (map[int]int, error) {
	if token == "" {
		return nil, errors.New("missing Card Trader token")
	}
	return strictSealedBridge(ctx, NewCTAuthClient(token), game)
}

// strictSealedBridge is SealedBridge over a client already built.
func strictSealedBridge(ctx context.Context, client *CTAuthClient, game matcher.Game) (map[int]int, error) {
	var skips expansionSkips
	blueprints, _, err := BlueprintsForGame(ctx, client, game, "", skips.logf)
	if err != nil {
		return nil, err
	}
	err = skips.err()
	if err != nil {
		return nil, err
	}
	if len(blueprints) == 0 {
		return nil, fmt.Errorf("no blueprints for %q", game)
	}
	return bridgeFromBlueprints(blueprints), nil
}

// SealedBridgeLogged is SealedBridge, tolerant where SealedBridge is strict:
// an expansion that fails to fetch is left out of the map, with a line to
// logf, and the rest is returned. A nil logf logs nothing.
func SealedBridgeLogged(ctx context.Context, token string, game matcher.Game, logf func(string, ...any)) (map[int]int, error) {
	if token == "" {
		return nil, errors.New("missing Card Trader token")
	}
	return sealedBridge(ctx, NewCTAuthClient(token), game, logf)
}

// expansionSkips counts the expansions a blueprint walk skipped, every line
// the walk hands its logger being one, and keeps the first.
type expansionSkips struct {
	mu    sync.Mutex
	count int
	first string
}

// logf is the walk's logger; the pool's workers call it concurrently.
func (s *expansionSkips) logf(format string, a ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.count++
	if s.count == 1 {
		s.first = fmt.Sprintf(format, a...)
	}
}

// err names the skips, or is nil when there were none.
func (s *expansionSkips) err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count == 0 {
		return nil
	}
	return fmt.Errorf("expansions failed to fetch: %d, the first: %s", s.count, s.first)
}

// sealedBridge is SealedBridgeLogged over a client already built.
func sealedBridge(ctx context.Context, client *CTAuthClient, game matcher.Game, logf func(string, ...any)) (map[int]int, error) {
	blueprints, _, err := BlueprintsForGame(ctx, client, game, "", logf)
	if err != nil {
		return nil, err
	}
	return bridgeFromBlueprints(blueprints), nil
}

// bridgeFromBlueprints is SealedBridge's map, from blueprints already
// fetched.
func bridgeFromBlueprints(blueprints []Blueprint) map[int]int {
	bridge := map[int]int{}
	for _, bp := range blueprints {
		tcgID := bp.TCGplayerProductID()
		if tcgID == 0 {
			continue
		}
		for _, mkmID := range bp.CardMarketIDs {
			bridge[mkmID] = tcgID
		}
	}
	return bridge
}
