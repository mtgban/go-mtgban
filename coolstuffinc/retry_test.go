package coolstuffinc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// TestSearchSurvivesTransientFailures pins that a search whose first
// request fails is retried rather than abandoned. The first page of a
// search stands for a whole edition: abandoning it drops every row of
// that edition silently, which a run of the storefront's 445 Magic
// editions did twice in one evening.
func TestSearchSurvivesTransientFailures(t *testing.T) {
	const page = `<html><span id="nextLink"><a href="/sq/12345&page=2"></a></span></html>`

	for _, tt := range []struct {
		desc string
		fail func(w http.ResponseWriter)
	}{
		{"the storefront answers with a server error", func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusInternalServerError)
		}},
		{"the storefront drops the connection", func(w http.ResponseWriter) {
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				return
			}
			conn, _, err := hijacker.Hijack()
			if err == nil {
				_ = conn.Close()
			}
		}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			var attempts int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				if attempts < 3 {
					tt.fail(w)
					return
				}
				_, _ = w.Write([]byte(page))
			}))
			defer srv.Close()

			saved := csiSearchURL
			csiSearchURL = srv.URL
			defer func() { csiSearchURL = saved }()

			result, err := Search(context.Background(), newCSIHTTPClient(), GameMagic, "Coldsnap", false, nil)
			if err != nil {
				t.Fatalf("Search() after %d attempts: %v", attempts, err)
			}
			if result.NextLink != "/sq/12345&page=2" {
				t.Errorf("NextLink = %q, want %q", result.NextLink, "/sq/12345&page=2")
			}
			if attempts != 3 {
				t.Errorf("served %d attempts, want 3", attempts)
			}
		})
	}
}

// TestSearchPageSurvivesForbidden pins that a results page answered 403 is
// asked for again: giving up there loses the rest of the shelf.
func TestSearchPageSurvivesForbidden(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`<html><span id="nextLink"><a href="/sq/12345&page=3"></a></span></html>`))
	}))
	defer srv.Close()

	doc, err := fetchSearchPage(context.Background(), newCSIHTTPClient(), srv.URL)
	if err != nil {
		t.Fatalf("fetchSearchPage() after %d attempts: %v", attempts, err)
	}
	next := searchNextLink(doc)
	if next != "/sq/12345&page=3" {
		t.Errorf("next link = %q, want /sq/12345&page=3", next)
	}
}

// TestSearchResendsForbidden pins that a search answered 403 is sent again
// with its whole form, and fails with the status once attempts run out.
func TestSearchResendsForbidden(t *testing.T) {
	const page = `<html><span id="nextLink"><a href="/sq/12345&page=2"></a></span></html>`
	client := newCSIHTTPClient(mtgban.WithHTTPRetryWait(time.Millisecond, time.Millisecond))

	for _, tt := range []struct {
		desc   string
		field  string
		value  string
		search func() (*SearchResult, error)
	}{
		{"singles", "f[ItemSet][]", "Coldsnap", func() (*SearchResult, error) {
			return Search(context.Background(), client, GameMagic, "Coldsnap", false, nil)
		}},
		{"sealed", "name", "booster", func() (*SearchResult, error) {
			return searchSealed(context.Background(), client, GameLorcana, "booster")
		}},
	} {
		for _, always := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s, refused every time: %t", tt.desc, always), func(t *testing.T) {
				var bodies []string
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, _ := io.ReadAll(r.Body)
					bodies = append(bodies, r.Method+" "+string(body))
					if always || len(bodies) < 2 {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					_, _ = w.Write([]byte(page))
				}))
				defer srv.Close()

				saved := csiSearchURL
				csiSearchURL = srv.URL
				defer func() { csiSearchURL = saved }()

				result, err := tt.search()
				want := 2
				if always {
					want = 5
					if err == nil || !strings.Contains(err.Error(), "unexpected 403 status code") {
						t.Errorf("search answered %v, want the status in an error", err)
					}
				} else if err != nil {
					t.Errorf("search answered %v", err)
				} else if result.NextLink != "/sq/12345&page=2" {
					t.Errorf("NextLink = %q, want /sq/12345&page=2", result.NextLink)
				}

				if len(bodies) != want {
					t.Fatalf("served %d attempts, want %d", len(bodies), want)
				}
				form, err := url.ParseQuery(strings.TrimPrefix(bodies[0], "POST "))
				if err != nil || form.Get(tt.field) != tt.value {
					t.Errorf("first attempt carried %q, %v", bodies[0], err)
				}
				for i, body := range bodies {
					if body != bodies[0] {
						t.Errorf("attempt %d carried %q, want %q", i+1, body, bodies[0])
					}
				}
			})
		}
	}
}

// TestForbiddenRetriesOnlyReads pins that no method but GET and POST is
// retried on a 403.
func TestForbiddenRetriesOnlyReads(t *testing.T) {
	for _, method := range []string{
		http.MethodGet, http.MethodPost, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete,
	} {
		resp := &http.Response{StatusCode: http.StatusForbidden, Request: &http.Request{Method: method}}
		retry, err := retryForbidden(context.Background(), resp, nil)
		want := method == http.MethodGet || method == http.MethodPost
		if retry != want || err != nil {
			t.Errorf("%s: retry %t, %v, want %t", method, retry, err, want)
		}
	}
}

// storeTransport answers every request in process.
type storeTransport func(req *http.Request) (status int, body string)

func (f storeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	err := req.Context().Err()
	if err != nil {
		return nil, err
	}
	rec := httptest.NewRecorder()
	status, body := f(req)
	rec.WriteHeader(status)
	_, _ = rec.WriteString(body)
	resp := rec.Result()
	resp.Request = req
	return resp, nil
}

// TestLostRequestFailsLoad pins that a retail walk missing any of its
// requests fails and dates nothing, whichever request was refused.
func TestLostRequestFailsLoad(t *testing.T) {
	const form = `<fieldset><h2 class="mb10"><b>Item Set</b></h2><div class="toggleTable"><ul>` +
		`<li><input type="checkbox" value="Coldsnap"></li><li><input type="checkbox" value="Ice Age"></li>` +
		`</ul></div></fieldset>`
	const links = `<div class="search-result-links">1 - 25 of 60 Results</div>`

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, tt := range []struct {
		desc   string
		sealed bool
		game   matcher.Game
		ctx    context.Context
		refuse func(req *http.Request) bool
		want   string
	}{
		{"singles form", false, matcher.GameMagic, context.Background(), func(req *http.Request) bool {
			return req.URL.Query().Get("s") != ""
		}, "unexpected 404 status code"},
		{"one edition", false, matcher.GameMagic, context.Background(), func(req *http.Request) bool {
			return req.FormValue("f[ItemSet][]") == "Ice Age"
		}, "1 of 2 editions lost, first: Ice Age: unexpected 404 status code"},
		{"cancelled on the first edition", false, matcher.GameMagic, ctx, func(req *http.Request) bool {
			if req.Method == http.MethodPost {
				cancel()
			}
			return req.Method == http.MethodPost
		}, "2 of 2 editions lost: context canceled"},
		{"one sealed query", true, matcher.GameLorcana, context.Background(), func(req *http.Request) bool {
			return req.FormValue("name") == "deck"
		}, "1 of 10 searches lost, first: deck: unexpected 404 status code"},
		{"sealed page count", true, matcher.GameMagic, context.Background(), func(req *http.Request) bool {
			return true
		}, "unexpected 404 status code"},
		{"one sealed page", true, matcher.GameMagic, context.Background(), func(req *http.Request) bool {
			return req.URL.Query().Get("page") == "2"
		}, "1 of 3 pages lost, first: page 2: unexpected 404 status code"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			client := &http.Client{Transport: storeTransport(func(req *http.Request) (int, string) {
				if tt.refuse(req) {
					return http.StatusNotFound, "<html></html>"
				}
				return http.StatusOK, "<html>" + form + links + pagingRow + "</html>"
			})}
			b := &matcher.Backend{Game: tt.game}

			var err error
			var date time.Time
			if tt.sealed {
				csi := Sealed{client: client, shelf: csiGames[tt.game], backend: b, maxConcurrency: 1, disableBuylist: true}
				err = csi.Load(tt.ctx)
				date = csi.inventoryDate
			} else {
				csi := Coolstuffinc{client: client, shelf: csiGames[tt.game], backend: b, maxConcurrency: 1, disableBuylist: true}
				err = csi.Load(tt.ctx)
				date = csi.inventoryDate
			}
			if !errors.Is(err, mtgban.ErrInventoryLoad) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load answered %v, want %q", err, tt.want)
			}
			if !date.IsZero() {
				t.Errorf("dated a short walk %v", date)
			}
		})
	}
}

// TestWholeWalkLoads pins that a retail walk losing nothing loads and is
// dated.
func TestWholeWalkLoads(t *testing.T) {
	client := &http.Client{Transport: storeTransport(func(*http.Request) (int, string) {
		return http.StatusOK, `<html><fieldset><h2 class="mb10"><b>Item Set</b></h2><div class="toggleTable"><ul>` +
			`<li><input type="checkbox" value="Coldsnap"></li></ul></div></fieldset>` +
			`<div class="search-result-links">1 - 25 of 60 Results</div>` + pagingRow + `</html>`
	})}

	// Naming the edition allows a walk that prices nothing
	singles := Coolstuffinc{
		client: client, shelf: GameMagic, backend: &matcher.Backend{Game: matcher.GameMagic},
		targetEdition: "Coldsnap", disableBuylist: true,
	}
	err := singles.Load(context.Background())
	if err != nil || singles.inventoryDate.IsZero() {
		t.Errorf("singles Load answered %v, dated %v", err, singles.inventoryDate)
	}

	for _, game := range []matcher.Game{matcher.GameMagic, matcher.GameLorcana} {
		sealed := Sealed{client: client, shelf: csiGames[game], backend: &matcher.Backend{Game: game}, disableBuylist: true}
		err := sealed.Load(context.Background())
		if err != nil || sealed.inventoryDate.IsZero() {
			t.Errorf("%s sealed Load answered %v, dated %v", game, err, sealed.inventoryDate)
		}
	}
}

// TestBuylistEditionsReportsARefusal pins that a refused edition list is
// an error rather than a map naming no editions.
func TestBuylistEditionsReportsARefusal(t *testing.T) {
	client := &http.Client{Transport: storeTransport(func(*http.Request) (int, string) {
		return http.StatusNotFound, "<html></html>"
	})}
	_, err := LoadBuylistEditions(context.Background(), client, GameMagic)
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("LoadBuylistEditions() = %v, want the status in an error", err)
	}
}
