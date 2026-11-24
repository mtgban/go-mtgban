package cardtrader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestBlueprintsForGameIDOrder pins that a pooled fetch returns the
// blueprints in catalog order however the expansions finish, and skips and
// names the one that fails again when asked alone.
func TestBlueprintsForGameIDOrder(t *testing.T) {
	expansions := `[{"id":1,"game_id":26,"name":"One"},{"id":2,"game_id":26,"name":"Two"},` +
		`{"id":9,"game_id":1,"name":"Magic"},{"id":3,"game_id":26,"name":"Three"},` +
		`{"id":4,"game_id":26,"name":"Four"},{"id":5,"game_id":26,"name":"Five"}]`

	// The first expansion answers only once the third has, so the two
	// finish out of catalog order.
	thirdServed := make(chan struct{})
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		body := expansions
		id := req.URL.Query().Get("expansion_id")
		switch id {
		case "":
		case "1":
			<-thirdServed
			body = `[{"id":10},{"id":11}]`
		case "3":
			defer close(thirdServed)
			body = `[{"id":30},{"id":31}]`
		case "4":
			body = `{"extra":{"message":"boom"}}`
		default:
			body = fmt.Sprintf(`[{"id":%s0},{"id":%s1}]`, id, id)
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

	blueprints, _, err := BlueprintsForGameID(context.Background(), client, GamePalworld, "", 4, logf)
	if !errors.Is(err, ErrIncomplete) || !strings.Contains(err.Error(), "lost 1 of 5 expansions: 4 Four") {
		t.Errorf("got error %v, want the lost expansion named", err)
	}
	var ids []int
	for _, bp := range blueprints {
		ids = append(ids, bp.ID)
	}
	want := []int{10, 11, 20, 21, 30, 31, 50, 51}
	if !slices.Equal(ids, want) {
		t.Errorf("got blueprints %v, want %v", ids, want)
	}
	wantLogs := []string{"refetching 4 Four due to boom", "skipping 4 Four due to boom"}
	if !slices.Equal(logs, wantLogs) {
		t.Errorf("got logs %q, want %q", logs, wantLogs)
	}
}

// TestBlueprintsForGameIncomplete pins, against a server, that only a wanted
// expansion failing twice reads as lost: not one that answers when asked again,
// one with no blueprints or none published yet, or one the target edition
// leaves out.
func TestBlueprintsForGameIncomplete(t *testing.T) {
	const expansions = `[{"id":1,"game_id":23,"code":"one","name":"One"},{"id":2,"game_id":23,"code":"two","name":"Broken"},` +
		`{"id":3,"game_id":23,"code":"three","name":"Empty"},{"id":4,"game_id":23,"code":"four","name":"Unpublished"},` +
		`{"id":5,"game_id":23,"code":"five","name":"Flaky"},{"id":6,"game_id":23,"code":"six","name":"Gone"},{"id":7,"game_id":1,"code":"m","name":"Magic"}]`
	var flaky atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("expansion_id") {
		case "":
			fmt.Fprint(w, expansions)
		case "1":
			fmt.Fprint(w, `[{"id":10}]`)
		case "3":
			fmt.Fprint(w, `[]`)
		case "4":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error_code":"not_found","errors":[],"extra":{"message":"Data is not ready for blueprints_exp_4"}}`)
		case "5":
			if flaky.Add(1)%2 == 1 {
				http.Error(w, "slow down", http.StatusTooManyRequests)
				return
			}
			fmt.Fprint(w, `[{"id":50}]`)
		case "6":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error_code":"not_found","errors":[],"extra":{"message":"no such page"}}`)
		default:
			http.Error(w, "broken", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Scheme = "http"
		req.URL.Host = server.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})
	client := &CTAuthClient{client: &http.Client{Transport: transport}}

	for _, test := range []struct {
		edition string
		want    []int
		lost    string
	}{
		{"", []int{10, 50}, "lost 2 of 6 expansions: 2 Broken, 6 Gone"},
		{"One", []int{10}, ""},
		{"Broken", nil, "lost 1 of 1 expansions: 2 Broken"},
		{"Gone", nil, "lost 1 of 1 expansions: 6 Gone"},
		{"Empty", nil, ""},
		{"Unpublished", nil, ""},
		{"Flaky", []int{50}, ""},
	} {
		blueprints, _, err := BlueprintsForGame(context.Background(), client, matcher.GameGundam, test.edition, t.Logf)
		var ids []int
		for _, bp := range blueprints {
			ids = append(ids, bp.ID)
		}
		if !slices.Equal(ids, test.want) {
			t.Errorf("edition %q: got blueprints %v, want %v", test.edition, ids, test.want)
		}
		if test.lost == "" {
			if err != nil {
				t.Errorf("edition %q: %v", test.edition, err)
			}
			continue
		}
		if !errors.Is(err, ErrIncomplete) || !strings.Contains(err.Error(), test.lost) {
			t.Errorf("edition %q: got error %v, want %s", test.edition, err, test.lost)
		}
	}
}
