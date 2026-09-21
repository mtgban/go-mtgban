package coolstuffinc

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRequestsCarryAUserAgent pins that the clients this package builds
// name an agent on every request. Cool Stuff Inc answers Go's own agent
// with the bare site chrome under a 200, which parses as a page holding
// no rows rather than as an error, so a request that goes out without
// the header loses a whole shelf and says nothing.
func TestRequestsCarryAUserAgent(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.UserAgent()
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	resp, err := newCSIHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("Do(): %v", err)
	}
	resp.Body.Close()

	if got != csiUserAgent {
		t.Errorf("User-Agent = %q, want %q", got, csiUserAgent)
	}
}
