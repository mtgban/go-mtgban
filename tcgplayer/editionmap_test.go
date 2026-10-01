package tcgplayer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/mtgban/go-tcgplayer"
)

// serveGroups stands in for the API's group listing: count groups counted,
// ids 1 to served handed over a page at a time.
func serveGroups(t *testing.T, count, served int) *tcgplayer.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"access_token": "t", "token_type": "bearer", "expires_in": 86400}`)
	})
	mux.HandleFunc("/catalog/groups", func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var items []string
		if r.URL.Query().Get("limit") != "1" {
			for id := offset + 1; id <= min(offset+tcgplayer.MaxItemsInResponse, served); id++ {
				items = append(items, fmt.Sprintf(`{"groupId": %d, "name": "Set %d"}`, id, id))
			}
		}
		fmt.Fprintf(w, `{"totalItems": %d, "success": true, "errors": [], "results": [%s]}`, count, strings.Join(items, ","))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	token, groups := tcgplayer.TokenURL, tcgplayer.CatalogGroupsURL
	t.Cleanup(func() { tcgplayer.TokenURL, tcgplayer.CatalogGroupsURL = token, groups })
	tcgplayer.TokenURL, tcgplayer.CatalogGroupsURL = srv.URL+"/token", srv.URL+"/catalog/groups"

	client, err := tcgplayer.NewClient("k", "k")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestEditionMapIndexesEveryGroup(t *testing.T) {
	editions, err := EditionMap(context.Background(), serveGroups(t, 150, 150), tcgplayer.CategoryMagic)
	if err != nil {
		t.Fatalf("EditionMap() error = %v, want nil", err)
	}
	if len(editions) != 150 || editions[150].Name != "Set 150" {
		t.Errorf("EditionMap() = %d editions, 150 named %q; want 150, the last named %q", len(editions), editions[150].Name, "Set 150")
	}
}

// A page answering short loses editions with no error of its own, and every
// product filed under them would go unmatched.
func TestEditionMapShortPageIsAnError(t *testing.T) {
	editions, err := EditionMap(context.Background(), serveGroups(t, 150, 140), tcgplayer.CategoryMagic)
	if err == nil {
		t.Errorf("EditionMap() = %d editions and no error, want the short page reported", len(editions))
	}
}
