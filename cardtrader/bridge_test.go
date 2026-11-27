package cardtrader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// TestBridgeFromBlueprints pins the Cardmarket to TCGplayer map read off the
// blueprints: every Cardmarket id a blueprint sells as, none for a blueprint
// with no TCGplayer id, and the override where Card Trader sends none.
func TestBridgeFromBlueprints(t *testing.T) {
	blueprints := []Blueprint{
		{ID: 1, TCGplayerID: 10, CardMarketIDs: []int{100, 101}},
		{ID: 2, CardMarketIDs: []int{200}},
		// One Piece DON!! promo, Bandai Fest | Elbaph: no id sent at all.
		{ID: 391163, CardMarketIDs: []int{300}},
		{ID: 4, TCGplayerID: 40},
	}
	want := map[int]int{100: 10, 101: 10, 300: 692132}
	got := bridgeFromBlueprints(blueprints)
	if !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSealedBridgeNeedsToken pins that a missing token fails before any fetch.
func TestSealedBridgeNeedsToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bridge, err := SealedBridge(ctx, "", matcher.GameOnePiece)
	if err == nil {
		t.Errorf("got %d links, want an error", len(bridge))
	}
	if errors.Is(err, context.Canceled) {
		t.Error("reached the network")
	}
}

// TestSealedBridgeLogsSkips pins that an expansion failing to fetch reaches
// the caller's logger, as bantool's log line, and is left out of the map.
func TestSealedBridgeLogsSkips(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		body := `[{"id":1,"game_id":15,"name":"One"},{"id":2,"game_id":15,"name":"Two"}]`
		switch req.URL.Query().Get("expansion_id") {
		case "1":
			body = `[{"id":10,"tcg_player_id":100,"card_market_ids":[1000]}]`
		case "2":
			body = `{"extra":{"message":"boom"}}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	client := &CTAuthClient{client: &http.Client{Transport: transport}}

	var mu sync.Mutex
	var logs []string
	logf := func(format string, a ...any) {
		mu.Lock()
		defer mu.Unlock()
		logs = append(logs, fmt.Sprintf(format, a...))
	}

	bridge, err := sealedBridge(context.Background(), client, matcher.GameOnePiece, logf)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]int{1000: 100}
	if !maps.Equal(bridge, want) {
		t.Errorf("got %v, want %v", bridge, want)
	}
	wantLogs := []string{"skipping 2 Two due to boom"}
	if !slices.Equal(logs, wantLogs) {
		t.Errorf("got logs %q, want %q", logs, wantLogs)
	}
}

// TestSealedBridgeLoggedNeedsToken pins the logged variant's token check.
func TestSealedBridgeLoggedNeedsToken(t *testing.T) {
	bridge, err := SealedBridgeLogged(context.Background(), "", matcher.GameOnePiece, t.Logf)
	if err == nil {
		t.Errorf("got %d links, want an error", len(bridge))
	}
}
